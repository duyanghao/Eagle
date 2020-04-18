package btclient

import (
	"bytes"
	"fmt"
	"github.com/duyanghao/eagle/pkg/utils/lrucache"
	"github.com/duyanghao/eagle/pkg/utils/process"
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
	"github.com/duyanghao/eagle/pkg/constants"
	distdigests "github.com/opencontainers/go-digest"
	log "github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
)

type Config struct {
	EnableUpload      bool
	EnableSeeding     bool
	IncomingPort      int
	UploadRateLimit   int
	DownloadRateLimit int
	CacheLimitSize    int64
}

type idInfo struct {
	Id       string
	InfoHash string
	Started  bool
	Count    int
}

// BtEngine backed by anacrolix/torrent
type BtEngine struct {
	sync.RWMutex
	lruCache   *lrucache.LruCache
	client     *torrent.Client
	httpClient *http.Client
	config     *Config
	idInfos    map[string]*torrent.Torrent // image ID -> InfoHash
	rootDir    string
	trackers   []string
	seeders    []string

	torrentDir string
	dataDir    string
}

func NewBtEngine(root string, trackers, seeders []string, c *Config) *BtEngine {
	dataDir := path.Join(root, "data")
	torrentDir := path.Join(root, "torrents")
	if c == nil {
		c = &Config{
			EnableUpload:      true,
			EnableSeeding:     true,
			IncomingPort:      50007,
			UploadRateLimit:   constants.DefaultUploadRateLimit,
			DownloadRateLimit: constants.DefaultDownloadRateLimit,
		}
	}
	return &BtEngine{
		rootDir:    root,
		trackers:   trackers,
		seeders:    seeders,
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

func (e *BtEngine) Run() error {
	if err := os.MkdirAll(e.dataDir, 0700); err != nil && !os.IsExist(err) {
		return nil
	}
	if err := os.MkdirAll(e.torrentDir, 0700); err != nil && !os.IsExist(err) {
		return nil
	}

	if e.client != nil {
		e.client.Close()
		time.Sleep(1 * time.Second)
	}

	c := e.config
	if c.IncomingPort <= 0 {
		return fmt.Errorf("Invalid incoming port (%d)", c.IncomingPort)
	}
	tc := torrent.NewDefaultClientConfig()
	tc.DataDir = e.dataDir
	tc.NoUpload = !c.EnableUpload
	tc.Seed = c.EnableSeeding
	tc.DisableUTP = true
	tc.ListenPort = c.IncomingPort
	tc.UploadRateLimiter = rate.NewLimiter(rate.Limit(c.UploadRateLimit), constants.DefaultRateLimitBurst)
	tc.DownloadRateLimiter = rate.NewLimiter(rate.Limit(c.DownloadRateLimit), constants.DefaultRateLimitBurst)
	client, err := torrent.NewClient(tc)
	if err != nil {
		return err
	}

	e.client = client

	// create lruCache
	e.lruCache, err = lrucache.NewLRU(c.CacheLimitSize, e.DeleteTorrent)
	if err != nil {
		log.Errorf("Create lruCache for p2p client failed, %v", err)
		return err
	}

	files, err := ioutil.ReadDir(e.dataDir)
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
			tf := e.GetFilePath(id)
			if _, err = os.Lstat(tf); err != nil {
				return
			}

			if err = e.StartSeed(id); err != nil {
				log.Errorf("Start seed %s failed: %v", id, err)
				return
			}
			e.lruCache.CreateIfNotExists(id)
			e.lruCache.SetComplete(id, f.Size())
		}(f)
	}
	/*go func() {
	        for {
	                time.Sleep(time.Second * 10)
	                e.lruCache.Output()
	        }
	}()*/
	return nil
}

func (e *BtEngine) GetTorrentFromSeeder(r *http.Request, blobUrl string) ([]byte, error) {
	// construct encoded endpoint
	Url, err := url.Parse(fmt.Sprintf("http://%s", e.seeders[0]))
	if err != nil {
		return nil, err
	}
	Url.Path += r.URL.Path
	endpoint := Url.String()
	log.Debugf("GetTorrentFromSeeder endpoint: %s", endpoint)
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Location", r.URL.Host)
	// use httpClient to send request
	rsp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	// close the connection to reuse it
	defer rsp.Body.Close()
	// check status code
	if rsp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GetTorrentFromSeeder rsp error: %v", rsp)
	}
	// parse rsp body
	t, err := ioutil.ReadAll(rsp.Body)
	return t, err
}

func (e *BtEngine) downloadLayer(req *http.Request, blobUrl string) (int64, error) {
	digest := blobUrl[strings.LastIndex(blobUrl, "/")+1:]
	id := distdigests.Digest(digest).Encoded()
	log.Debugf("Start leeching layer %s", id)
	t, err := e.GetTorrentFromSeeder(req, blobUrl)
	if err != nil {
		log.Errorf("Get torrent data from seeder for %s failed: %v", id, err)
		return -1, err
	}
	// Load torrent data
	reader := bytes.NewBuffer(t)
	metaInfo, err := metainfo.Load(reader)
	if err != nil {
		return -1, fmt.Errorf("Load torrent file failed: %v", err)
	}
	info, err := metaInfo.UnmarshalInfo()
	if err != nil {
		return -1, fmt.Errorf("UnmarshalInfo failed: %v", err)
	}
	progress := process.NewProgressDownload(id, int(info.TotalLength()), os.Stdout)
	// Download layer file
	if err := e.StartLeecher(id, metaInfo, progress); err != nil {
		log.Errorf("Download layer %s failed: %v", id, err)
		return info.TotalLength(), err
	} else {
		log.Infof("Download layer %s success", id)
	}
	return info.TotalLength(), nil
}

func (e *BtEngine) downloadLayerSync(req *http.Request, blobUrl string) (string, error) {
	// get only once each of layer
	digest := blobUrl[strings.LastIndex(blobUrl, "/")+1:]
	id := distdigests.Digest(digest).Encoded()
	layerFile := e.GetFilePath(id)
	torrentFile := e.GetTorrentFilePath(id)
Loop:
	entry, exist := e.lruCache.Get(id)
Execute:
	if exist {
		if entry.Completed {
			if _, err := os.Stat(layerFile); err != nil {
				log.Errorf("failed to find data file of cached layer: %s, try to remove its relevant records", id)
				e.lruCache.Remove(id)
				return layerFile, err
			}
			log.Infof("layer: %s has been cached, return directly", id)
			return layerFile, nil
		}
		// wait
		for {
			select {
			case <-entry.Done:
				log.Debugf("layer: %s cache generated, switch to normal process", id)
				goto Loop
			}
		}
	}
	entry, exist = e.lruCache.CreateIfNotExists(id)
	if exist {
		goto Execute
	} else { // get layer from origin
		size, err := e.downloadLayer(req, blobUrl)
		if err != nil {
			log.Errorf("download layer: %s failed, %v, try to remove its relevant records ...", id, err)
			os.Remove(torrentFile)
			os.Remove(layerFile)
			e.lruCache.Remove(id)
		} else {
			log.Infof("download layer: %s successfully, try to update status ...", id)
			e.lruCache.SetComplete(id, int64(size))
		}
		return layerFile, err
	}
}

func (e *BtEngine) DownloadLayer(req *http.Request, blobUrl string) (string, error) {
	return e.downloadLayerSync(req, blobUrl)
}

func (e *BtEngine) RootDir() string {
	return e.rootDir
}

func (e *BtEngine) GetTorrentFilePath(id string) string {
	return path.Join(e.rootDir, "torrents", id+".torrent")
}

func (e *BtEngine) GetFilePath(id string) string {
	return path.Join(e.rootDir, "data", id+".layer")
}

func (e *BtEngine) StartSeed(id string) error {
	tf := e.GetTorrentFilePath(id)
	if _, err := os.Lstat(tf); err != nil {
		// Torrent file not exist, create it
		log.Debugf("Create torrent file for %s", id)
		if err = e.createTorrent(id); err != nil {
			return err
		}
	}

	metaInfo, err := metainfo.LoadFromFile(tf)
	if err != nil {
		return fmt.Errorf("Load torrent file failed: %v", err)
	}

	tt, err := e.client.AddTorrent(metaInfo)
	if err != nil {
		return fmt.Errorf("Add torrent failed: %v", err)
	}

	e.addTorrent(id, tt)
	go func() {
		<-tt.GotInfo()
		if tt.Info() != nil {
			tt.DownloadAll()
		}
		log.Infof("Start torrent %v of layer %s success", tt.InfoHash(), id)
	}()

	return nil
}

func (e *BtEngine) StartLeecher(id string, metaInfo *metainfo.MetaInfo, p *process.ProgressDownload) error {
	tt, err := e.client.AddTorrent(metaInfo)
	if err != nil {
		return fmt.Errorf("Add torrent failed: %v", err)
	}

	e.addTorrent(id, tt)
	go func() {
		<-tt.GotInfo()
		if tt.Info() != nil {
			tt.DownloadAll()
		}
		log.Infof("start torrent %v of layer %s success", tt.InfoHash(), id)
	}()

	if p != nil {
		log.Debugf("Waiting bt download %s complete", id)
		p.WaitComplete(tt)
		log.Infof("Bt download %s completed", id)
	}
	return nil
}

func (e *BtEngine) createTorrent(id string) error {
	info := metainfo.Info{
		PieceLength: constants.DefaultMetaInfoPieceLength,
	}
	f := e.GetFilePath(id)
	err := info.BuildFromFilePath(f)
	if err != nil {
		return fmt.Errorf("Create torrent file for %s failed: %v", f, err)
	}
	mi := metainfo.MetaInfo{
		Announce: e.trackers[0],
	}
	mi.SetDefaults()
	mi.InfoBytes, err = bencode.Marshal(&info)
	if err != nil {
		return fmt.Errorf("Create torrent file for %s failed: %v", f, err)
	}

	tfn := e.GetTorrentFilePath(id)
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

func (e *BtEngine) addTorrent(id string, tt *torrent.Torrent) {
	e.Lock()
	defer e.Unlock()
	e.idInfos[id] = tt
}

func (e *BtEngine) DeleteTorrent(id string) {
	// remove info and bt torrent records
	e.deleteTorrent(id)

	// remove data file asynchronously
	go func() {
		tfn := e.GetTorrentFilePath(id)
		if err := os.Remove(tfn); err != nil && !os.IsNotExist(err) {
			log.Errorf("Remove torrent file %s failed: %v", tfn, err)
		}

		dfn := e.GetFilePath(id)
		if err := os.Remove(dfn); err != nil && !os.IsNotExist(err) {
			log.Errorf("Remove layer file %s failed: %v", dfn, err)
		}
	}()
}

func (e *BtEngine) deleteTorrent(id string) {
	e.Lock()
	defer e.Unlock()
	if tt, ok := e.idInfos[id]; ok {
		tt.Drop()
	}
	delete(e.idInfos, id)
}
