package btclient

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/duyanghao/eagle/pkg/utils"
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
	distdigests "github.com/opencontainers/go-digest"
	log "github.com/sirupsen/logrus"
)

var (
	ErrBtEngineNotStart = fmt.Errorf("BT engine not started")
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

// BtEngine backed by anacrolix/torrent
type BtEngine struct {
	mut sync.Mutex

	client     *torrent.Client
	httpClient *http.Client
	config     *Config
	ts         map[string]*Torrent // InfoHash -> torrent

	idInfos  map[string]*idInfo // image ID -> InfoHash
	rootDir  string
	trackers []string
	seeders  []string

	torrentDir string
	dataDir    string

	started bool
}

func NewBtEngine(root string, trackers, seeders []string, c *Config) *BtEngine {
	dataDir := path.Join(root, "data")
	torrentDir := path.Join(root, "torrents")
	if c == nil {
		c = &Config{
			EnableUpload:      true,
			EnableSeeding:     true,
			IncomingPort:      50007,
			UploadRateLimit:   DefaultUploadRateLimit,
			DownloadRateLimit: DefaultDownloadRateLimit,
		}
	}
	return &BtEngine{
		rootDir:    root,
		trackers:   trackers,
		seeders:    seeders,
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
	/*tc := torrent.ClientConfig{
		DataDir:    e.dataDir,
		NoUpload:   !c.EnableUpload,
		Seed:       c.EnableSeeding,
		DisableUTP: true,
		ListenHost: func(string) string { return "0.0.0.0" },
		ListenPort: c.IncomingPort,
		DisableIPv6: true,
		DhtStartingNodes: func(network string) dht.StartingNodesGetter {
			return func() ([]dht.Addr, error) { return dht.GlobalBootstrapAddrs(network) }
		},
	}*/
	client, err := torrent.NewClient(tc)
	if err != nil {
		return err
	}

	e.client = client

	// for StartSeed
	e.started = true

	files, err := ioutil.ReadDir(e.dataDir)
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
		tf := e.GetTorrentFilePath(id)
		if _, err = os.Lstat(tf); err != nil {
			continue
		}

		if err = e.StartSeed(id); err != nil {
			log.Errorf("Start seed %s failed: %v", id, err)
		}
	}

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

func (e *BtEngine) DownloadLayer(req *http.Request, blobUrl string) (string, error) {
	digest := blobUrl[strings.LastIndex(blobUrl, "/")+1:]
	id := distdigests.Digest(digest).Encoded()
	// Check whether or not the layer exists
	layerPath := e.GetFilePath(id)
	if _, err := os.Stat(layerPath); err == nil {
		log.Debugf("layer %s exists already, let's return directly ...", id)
		return layerPath, nil
	}
	log.Debugf("Start leeching layer %s", id)
	t, err := e.GetTorrentFromSeeder(req, blobUrl)
	if err != nil {
		log.Errorf("Get torrent data from seeder for %s failed: %v", id, err)
		return "", err
	}
	// Load torrent data
	reader := bytes.NewBuffer(t)
	metaInfo, err := metainfo.Load(reader)
	if err != nil {
		return "", fmt.Errorf("Load torrent file failed: %v", err)
	}
	info, err := metaInfo.UnmarshalInfo()
	if err != nil {
		return "", fmt.Errorf("UnmarshalInfo failed: %v", err)
	}
	progress := utils.NewProgressDownload(id, int(info.TotalLength()), os.Stdout)
	// Download layer file
	if err := e.StartLeecher(id, metaInfo, progress); err != nil {
		log.Errorf("Download layer %s failed: %v", id, err)
		return "", err
	} else {
		log.Infof("Download layer %s success", id)
	}
	return layerPath, nil
}

func (e *BtEngine) Started() bool {
	e.mut.Lock()
	defer e.mut.Unlock()
	return e.started
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
	if !e.started {
		return ErrBtEngineNotStart
	}

	e.mut.Lock()
	defer e.mut.Unlock()

	info, ok := e.idInfos[id]
	if ok && info.Started {
		info.Count++
		return nil
	}

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

	t := e.addTorrent(tt)
	go func() {
		<-t.tt.GotInfo()
		err = e.startTorrent(t.InfoHash)
		if err != nil {
			log.Errorf("Start torrent %v failed: %v", t.InfoHash, err)
		} else {
			log.Infof("Start torrent %v success", t.InfoHash)
		}
	}()

	e.idInfos[id] = &idInfo{
		Id:       id,
		InfoHash: t.InfoHash,
		Started:  true,
	}
	return nil
}

func (e *BtEngine) StartLeecher(id string, metaInfo *metainfo.MetaInfo, p *utils.ProgressDownload) error {
	if !e.started {
		return ErrBtEngineNotStart
	}

	e.mut.Lock()

	info, ok := e.idInfos[id]
	if ok && info.Started {
		info.Count++
		e.mut.Unlock()
		return nil
	}

	tt, err := e.client.AddTorrent(metaInfo)
	if err != nil {
		e.mut.Unlock()
		return fmt.Errorf("Add torrent failed: %v", err)
	}

	t := e.addTorrent(tt)
	go func() {
		<-t.tt.GotInfo()
		err = e.startTorrent(t.InfoHash)
		if err != nil {
			log.Errorf("start torrent %v failed: %v", t.InfoHash, err)
		} else {
			log.Infof("start torrent %v success", t.InfoHash)
		}
	}()

	e.idInfos[id] = &idInfo{
		Id:       id,
		InfoHash: t.InfoHash,
		Started:  true,
	}

	e.mut.Unlock()

	if p != nil {
		log.Debugf("Waiting bt download %s complete", id)
		p.WaitComplete(t.tt)
		log.Infof("Bt download %s completed", id)
	}
	return nil
}

func (e *BtEngine) createTorrent(id string) error {
	info := metainfo.Info{
		PieceLength: 4 * 1024 * 1024,
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

func (e *BtEngine) addTorrent(tt *torrent.Torrent) *Torrent {
	ih := tt.InfoHash().HexString()
	torrent, ok := e.ts[ih]
	if !ok {
		torrent = &Torrent{
			InfoHash: ih,
			Name:     tt.Name(),
			tt:       tt,
		}
		e.ts[ih] = torrent
	}
	return torrent
}

func (e *BtEngine) getTorrent(infohash string) (*Torrent, error) {
	ih, err := str2ih(infohash)
	if err != nil {
		return nil, err
	}
	t, ok := e.ts[ih.HexString()]
	if !ok {
		return t, fmt.Errorf("Missing torrent %x", ih)
	}
	return t, nil
}

// Start downloading
func (e *BtEngine) startTorrent(infohash string) error {
	t, err := e.getTorrent(infohash)
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
func (e *BtEngine) stopTorrent(infohash string) error {
	t, err := e.getTorrent(infohash)
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
