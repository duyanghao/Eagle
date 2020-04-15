package bt

import (
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"
	distdigests "github.com/opencontainers/go-digest"
	log "github.com/sirupsen/logrus"
)

var (
	ErrSeederNotStart = fmt.Errorf("Seeder not started")
)

const DefaultUploadRateLimit = 50 * 1024 * 1024 // 50Mb/s
const DefaultDownloadRateLimit = 50 * 1024 * 1024

type Config struct {
	EnableUpload      bool
	EnableSeeding     bool
	IncomingPort      int
	UploadRateLimit   int
	DownloadRateLimit int
}

type Status struct {
	Id        string `json:"id"`
	State     string `json:"state"`
	Completed int64  `json:"completed"`
	TotalLen  int64  `json:"totallength"`
	Seeding   bool   `json:"seeding"`
}

type idInfo struct {
	Id       string
	InfoHash string
	Started  bool
	Count    int
}

// Seeder backed by anacrolix/torrent
type Seeder struct {
	mut sync.Mutex

	client     *torrent.Client
	httpClient *http.Client
	config     *Config
	ts         map[string]*Torrent // InfoHash -> torrent

	idInfos  map[string]*idInfo // image ID -> InfoHash
	rootDir  string
	trackers []string

	torrentDir string
	dataDir    string

	started bool
}

func NewSeeder(root string, trackers []string, c *Config) *Seeder {
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
		dataDir:    dataDir,
		torrentDir: torrentDir,
		config:     c,
		ts:         map[string]*Torrent{},
		idInfos:    map[string]*idInfo{},
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
	tc := torrent.ClientConfig{
		DataDir:    s.dataDir,
		NoUpload:   !c.EnableUpload,
		Seed:       c.EnableSeeding,
		DisableUTP: true,
	}
	tc.SetListenAddr("0.0.0.0:" + strconv.Itoa(c.IncomingPort))
	client, err := torrent.NewClient(&tc)
	if err != nil {
		return err
	}

	s.client = client

	// for StartSeed
	s.started = true

	files, err := ioutil.ReadDir(s.dataDir)
	if err != nil {
		return err
	}

	for _, f := range files {
		if filepath.Ext(f.Name()) != ".layer" {
			continue
		}
		ss := strings.Split(f.Name(), ".")
		if len(ss) != 2 {
			log.Errorf("Found invalid layer file %s", f.Name())
			continue
		}

		id := ss[0]
		tf := s.GetTorrentFilePath(id)
		if _, err = os.Lstat(tf); err != nil {
			continue
		}

		if err = s.StartSeed(id); err != nil {
			log.Errorf("Start seed %s failed: %v", id, err)
		}
	}

	return nil
}

func (s *Seeder) getDataFromOrigin(r *http.Request) ([]byte, error) {
	// construct encoded endpoint
	Url, err := url.Parse(fmt.Sprintf("http://%s", r.URL.Host))
	if err != nil {
		return nil, err
	}
	Url.Path += r.URL.Path
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

func (s *Seeder) getMetaData(r *http.Request, id string) error {
	s.mut.Lock()
	defer s.mut.Unlock()
	// step1 - check whether or not the torrent exists
	torrentFile := s.GetTorrentFilePath(id)
	if _, err := os.Stat(torrentFile); err == nil {
		log.Debugf(fmt.Sprintf("torrent of layer: %s exists alyready, let's return directly ...", id))
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	// step2 - get data from origin
	log.Debugf(fmt.Sprintf("torrent of layer: %s not found, let's fetch data from origin ...", id))
	data, err := s.getDataFromOrigin(r)
	if err != nil {
		return err
	}
	// step3 - generate layerFile
	log.Debugf(fmt.Sprintf("generate dataFile of layer: %s ...", id))
	layerFile := s.GetFilePath(id)
	err = ioutil.WriteFile(layerFile, data, 0644)
	if err != nil {
		return err
	}
	// step4 - start seed
	log.Debugf(fmt.Sprintf("start to seed layer: %s ...", id))
	return s.StartSeed(id)
}

func (s *Seeder) GetMetaData(w http.ResponseWriter, r *http.Request) {
	log.Debugf("access: %s", r.URL.String())
	if !s.started {
		http.Error(w, ErrSeederNotStart.Error(), http.StatusInternalServerError)
		return
	}
	digest := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
	id := distdigests.Digest(digest).Encoded()
	log.Debugf("start get metadata of layer %s", id)
	err := s.getMetaData(r, id)
	if err != nil {
		http.Error(w, fmt.Sprintf("get torrent failed: %v", err), http.StatusInternalServerError)
		return
	}
	torrentFile := s.GetTorrentFilePath(id)
	f, err := os.Open(torrentFile)
	if err != nil {
		http.Error(w, fmt.Sprintf("open torrent failed: %v", err), http.StatusInternalServerError)
		return
	}
	if _, err := io.Copy(w, f); err != nil {
		http.Error(w, fmt.Sprintf("copy torrent failed: %v", err), http.StatusInternalServerError)
		return
	}
}

func (s *Seeder) Started() bool {
	s.mut.Lock()
	defer s.mut.Unlock()
	return s.started
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
	if !s.started {
		return ErrSeederNotStart
	}

	s.mut.Lock()
	defer s.mut.Unlock()

	info, ok := s.idInfos[id]
	if ok && info.Started {
		info.Count++
		return nil
	}

	tf := s.GetTorrentFilePath(id)
	if _, err := os.Lstat(tf); err != nil {
		// Torrent file not exist, create it
		log.Debugf("Create torrent file for %s", id)
		if err = s.createTorrent(id); err != nil {
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

	t := s.addTorrent(tt)
	go func() {
		<-t.tt.GotInfo()
		err = s.startTorrent(t.InfoHash)
		if err != nil {
			log.Errorf("Start torrent %v failed: %v", t.InfoHash, err)
		} else {
			log.Infof("Start torrent %v success", t.InfoHash)
		}
	}()

	s.idInfos[id] = &idInfo{
		Id:       id,
		InfoHash: t.InfoHash,
		Started:  true,
	}
	return nil
}

func (s *Seeder) createTorrent(id string) error {
	mi := metainfo.MetaInfo{
		Announce: s.trackers[0],
	}
	mi.SetDefaults()
	info, err := mi.UnmarshalInfo()
	if err != nil {
		return fmt.Errorf("UnmarshalInfo failed: %v", err)
	}
	info.PieceLength = 4 * 1024 * 1024 //4MB, similar to dragonfly

	f := s.GetFilePath(id)
	err = info.BuildFromFilePath(f)
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

func (s *Seeder) addTorrent(tt *torrent.Torrent) *Torrent {
	ih := tt.InfoHash().HexString()
	torrent, ok := s.ts[ih]
	if !ok {
		torrent = &Torrent{
			InfoHash: ih,
			Name:     tt.Name(),
			tt:       tt,
		}
		s.ts[ih] = torrent
	}
	return torrent
}

func (s *Seeder) getTorrent(infohash string) (*Torrent, error) {
	ih, err := str2ih(infohash)
	if err != nil {
		return nil, err
	}
	t, ok := s.ts[ih.HexString()]
	if !ok {
		return t, fmt.Errorf("Missing torrent %x", ih)
	}
	return t, nil
}

// Start downloading
func (s *Seeder) startTorrent(infohash string) error {
	t, err := s.getTorrent(infohash)
	if err != nil {
		return err
	}
	if t.State == Started {
		return fmt.Errorf("Already started")
	}
	t.State = Started
	if t.tt.Info() != nil {
		t.tt.DownloadAll()
	}
	return nil
}

// Drop torrent
func (s *Seeder) stopTorrent(infohash string) error {
	t, err := s.getTorrent(infohash)
	if err != nil {
		return err
	}
	if t.State == Dropped {
		return fmt.Errorf("Already stopped")
	}
	//there is no stop - kill underlying torrent
	t.tt.Drop()
	t.State = Dropped
	return nil
}

func str2ih(str string) (metainfo.Hash, error) {
	var ih metainfo.Hash
	e, err := hex.Decode(ih[:], []byte(str))
	if err != nil {
		return ih, fmt.Errorf("Invalid hex string")
	}
	if e != 20 {
		return ih, fmt.Errorf("Invalid length")
	}
	return ih, nil
}
