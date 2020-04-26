package bt

import (
	"context"
	"fmt"
	"github.com/duyanghao/eagle/pkg/constants"
	pb "github.com/duyanghao/eagle/proto/metainfo"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/bencode"
	"github.com/anacrolix/torrent/metainfo"
	"github.com/duyanghao/eagle/pkg/utils/lrucache"
	"github.com/duyanghao/eagle/pkg/utils/process"
	distdigests "github.com/opencontainers/go-digest"
	log "github.com/sirupsen/logrus"
)

const DefaultUploadRateLimit = 50 * 1024 * 1024 // 50Mb/s
const DefaultDownloadRateLimit = 50 * 1024 * 1024

type Config struct {
	EnableUpload      bool
	EnableSeeding     bool
	IncomingPort      int
	UploadRateLimit   int
	DownloadRateLimit int
	CacheLimitSize    int64
}

// Seeder backed by anacrolix/torrent
type Seeder struct {
	sync.RWMutex
	lruCache   *lrucache.LruCache
	client     *torrent.Client
	httpClient *http.Client
	config     *Config
	idInfos    map[string]*torrent.Torrent // layer digest -> Torrent
	rootDir    string
	trackers   []string
	origin     string

	torrentDir string
	dataDir    string
}

func NewSeeder(root, origin string, trackers []string, c *Config) *Seeder {
	dataDir := path.Join(root, "data")
	torrentDir := path.Join(root, "torrents")
	if c == nil {
		c = &Config{
			EnableUpload:      true,
			EnableSeeding:     true,
			IncomingPort:      50017,
			UploadRateLimit:   DefaultUploadRateLimit,
			DownloadRateLimit: DefaultDownloadRateLimit,
		}
	}
	return &Seeder{
		rootDir:    root,
		trackers:   trackers,
		origin:     origin,
		dataDir:    dataDir,
		torrentDir: torrentDir,
		config:     c,
		idInfos:    make(map[string]*torrent.Torrent),
		httpClient: &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
				DialContext: (&net.Dialer{
					Timeout:   30 * time.Second,
					KeepAlive: 30 * time.Second,
					DualStack: true,
				}).DialContext,
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 100,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}
}

func (s *Seeder) Run() error {
	if err := os.MkdirAll(s.dataDir, 0700); err != nil && !os.IsExist(err) {
		return nil
	}
	if err := os.MkdirAll(s.torrentDir, 0700); err != nil && !os.IsExist(err) {
		return nil
	}

	if s.client != nil {
		s.client.Close()
		time.Sleep(1 * time.Second)
	}

	c := s.config
	if c.IncomingPort <= 0 {
		return fmt.Errorf("Invalid incoming port (%d)", c.IncomingPort)
	}

	tc := torrent.NewDefaultClientConfig()
	tc.DataDir = s.dataDir
	tc.NoUpload = !c.EnableUpload
	tc.Seed = c.EnableSeeding
	tc.DisableUTP = true
	tc.ListenPort = c.IncomingPort

	client, err := torrent.NewClient(tc)
	if err != nil {
		return err
	}

	s.client = client

	// create lruCache
	s.lruCache, err = lrucache.NewLRU(c.CacheLimitSize, s.DeleteTorrent)
	if err != nil {
		log.Errorf("Create lruCache for seeder failed, %v", err)
		return err
	}

	files, err := ioutil.ReadDir(s.dataDir)
	if err != nil {
		return err
	}

	for _, f := range files {
		go func(f os.FileInfo) {
			if filepath.Ext(f.Name()) != ".layer" {
				return
			}
			ss := strings.Split(f.Name(), ".")
			if len(ss) != 2 {
				log.Errorf("Found invalid layer file %s", f.Name())
				return
			}

			id := ss[0]
			tf := s.GetFilePath(id)
			if _, err = os.Lstat(tf); err != nil {
				return
			}

			if err = s.StartSeed(id); err != nil {
				log.Errorf("Start seed %s failed: %v", id, err)
				return
			}
			s.lruCache.CreateIfNotExists(id)
			s.lruCache.SetComplete(id, f.Size())
		}(f)
	}

	go func() {
		for {
			time.Sleep(time.Minute * 2)
			s.lruCache.Output()
		}
	}()

	return nil
}

// getDataFromOrigin get layer from remote origin
func (s *Seeder) getDataFromOrigin(reqUrl string) ([]byte, error) {
	// construct encoded endpoint
	//origin := r.Header.Get("Location")
	Url, err := url.Parse(fmt.Sprintf("http://%s", s.origin))
	if err != nil {
		return nil, err
	}
	Url.Path += reqUrl
	endpoint := Url.String()
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	// use httpClient to send request
	rsp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	// close the connection to reuse it
	defer rsp.Body.Close()
	// check status code
	if rsp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GetDataFromOrigin rsp error: %v", rsp)
	}
	// parse rsp body
	t, err := ioutil.ReadAll(rsp.Body)
	return t, err
}

// getMetaData generates layer file and its relevant torrent
func (s *Seeder) getMetaData(reqUrl, id string) (int, error) {
	// step1 - get data from origin
	log.Debugf(fmt.Sprintf("torrent of layer: %s not found, let's fetch data from origin ...", id))
	data, err := s.getDataFromOrigin(reqUrl)
	if err != nil {
		log.Errorf("get torrent of layer: %s failed, error: %v", id, err)
		return len(data), err
	}
	// step2 - generate layerFile
	log.Debugf(fmt.Sprintf("generate dataFile of layer: %s ...", id))
	layerFile := s.GetFilePath(id)
	err = ioutil.WriteFile(layerFile, data, 0644)
	if err != nil {
		return len(data), err
	}
	// step3 - start seed
	log.Debugf(fmt.Sprintf("start to seed layer: %s ...", id))
	return len(data), s.StartSeed(id)
}

// getMetaDataSync generates layer file and its relevant torrent only once for each of layer
func (s *Seeder) getMetaDataSync(reqUrl, id string) error {
	// get only once each of layer
	torrentFile := s.GetTorrentFilePath(id)
	layerFile := s.GetFilePath(id)
Loop:
	entry, exist := s.lruCache.Get(id)
Execute:
	if exist {
		if entry.Completed {
			if _, err := os.Stat(torrentFile); err != nil {
				log.Errorf("failed to find torrent file of cached layer: %s, try to remove its relevant records", id)
				s.lruCache.Remove(id)
				return err
			}
			if _, err := os.Stat(layerFile); err != nil {
				log.Errorf("failed to find data file of cached layer: %s, try to remove its relevant records", id)
				s.lruCache.Remove(id)
				return err
			}
			log.Infof("layer: %s has been cached, return directly", id)
			return nil
		}
		// wait
		for {
			select {
			case <-entry.Done:
				log.Debugf("layer: %s cache updated, try to get it again...", id)
				goto Loop
			}
		}
	}
	entry, exist = s.lruCache.CreateIfNotExists(id)
	if exist {
		goto Execute
	} else { // get layer from origin
		size, err := s.getMetaData(reqUrl, id)
		if err != nil {
			log.Errorf("getMetaData layer: %s failed, %v, try to remove its relevant records ...", id, err)
			os.Remove(torrentFile)
			os.Remove(layerFile)
			s.lruCache.Remove(id)
		} else {
			log.Infof("getMetaData layer: %s successfully, try to update status ...", id)
			s.lruCache.SetComplete(id, int64(size))
		}
		return err
	}
}

// GetMetaData get torrent of layer
func (s *Seeder) GetMetaInfo(ctx context.Context, metaInfoReq *pb.MetaInfoRequest) (*pb.MetaInfoReply, error) {
	log.Debugf("access: %s", metaInfoReq.Url)
	digest := metaInfoReq.Url[strings.LastIndex(metaInfoReq.Url, "/")+1:]
	id := distdigests.Digest(digest).Encoded()
	log.Debugf("start to get metadata of layer %s", id)
	err := s.getMetaDataSync(metaInfoReq.Url, id)
	if err != nil {
		return nil, fmt.Errorf("get metainfo from origin failed: %v", err)
	}
	torrentFile := s.GetTorrentFilePath(id)
	content, err := ioutil.ReadFile(torrentFile)
	if err != nil {
		return nil, fmt.Errorf("read metainfo file failed: %v", err)
	}
	return &pb.MetaInfoReply{Metainfo: content}, nil
}

func (s *Seeder) RootDir() string {
	return s.rootDir
}

func (s *Seeder) GetTorrentFilePath(id string) string {
	return path.Join(s.rootDir, "torrents", id+".torrent")
}

func (s *Seeder) GetFilePath(id string) string {
	return path.Join(s.rootDir, "data", id+".layer")
}

func (s *Seeder) StartSeed(id string) error {
	tf := s.GetTorrentFilePath(id)
	if _, err := os.Lstat(tf); err != nil {
		// Torrent file not exist, create it
		log.Debugf("Create torrent file for %s", id)
		if err = s.createTorrent(id); err != nil {
			log.Errorf("Create torrent file for %s, error: %v", id, err)
			return err
		}
	}

	metaInfo, err := metainfo.LoadFromFile(tf)
	if err != nil {
		return fmt.Errorf("Load torrent file failed: %v", err)
	}

	tt, err := s.client.AddTorrent(metaInfo)
	if err != nil {
		return fmt.Errorf("Add torrent failed: %v", err)
	}

	s.addTorrent(id, tt)
	go func() {
		<-tt.GotInfo()
		if tt.Info() != nil {
			tt.DownloadAll()
		}
		log.Infof("Start torrent %v of layer %s success", tt.InfoHash(), id)
	}()

	p := process.NewProgressDownload(id, int(tt.Info().TotalLength()), os.Stdout)
	if p != nil {
		log.Debugf("Waiting bt download %s complete", id)
		p.WaitComplete(tt)
		log.Infof("Bt download %s completed", id)
	}

	return nil
}

func (s *Seeder) DeleteTorrent(id string) {
	// remove info and bt torrent records
	s.deleteTorrent(id)

	// remove data file and torrent file asynchronously
	go func() {
		tfn := s.GetTorrentFilePath(id)
		if err := os.Remove(tfn); err != nil && !os.IsNotExist(err) {
			log.Errorf("Remove torrent file %s failed: %v", tfn, err)
		}

		dfn := s.GetFilePath(id)
		if err := os.Remove(dfn); err != nil && !os.IsNotExist(err) {
			log.Errorf("Remove layer file %s failed: %v", dfn, err)
		}
	}()
}

func (s *Seeder) deleteTorrent(id string) {
	s.Lock()
	defer s.Unlock()
	if tt, ok := s.idInfos[id]; ok {
		tt.Drop()
	}
	delete(s.idInfos, id)
}

func (s *Seeder) createTorrent(id string) error {
	info := metainfo.Info{
		PieceLength: constants.DefaultMetaInfoPieceLength,
	}
	f := s.GetFilePath(id)
	err := info.BuildFromFilePath(f)
	if err != nil {
		return fmt.Errorf("Create torrent file for %s failed: %v", f, err)
	}
	mi := metainfo.MetaInfo{
		Announce: s.trackers[0],
	}
	mi.SetDefaults()
	mi.InfoBytes, err = bencode.Marshal(&info)
	if err != nil {
		return fmt.Errorf("Create torrent file for %s failed: %v", f, err)
	}

	tfn := s.GetTorrentFilePath(id)
	tFile, err := os.Create(tfn)
	if err != nil {
		return fmt.Errorf("Create torrent file %s failed: %v", tfn, err)
	}
	defer tFile.Close()

	if err = mi.Write(tFile); err != nil {
		return fmt.Errorf("Write torrent file %s failed: %v", tfn, err)
	}

	log.Infof("Create torrent file %s success", tfn)
	return nil
}

func (s *Seeder) addTorrent(id string, tt *torrent.Torrent) {
	s.Lock()
	defer s.Unlock()
	s.idInfos[id] = tt
}
