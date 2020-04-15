package btclient

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"fmt"
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
	ErrBtEngineNotStart = fmt.Errorf("BT engine not started")
	ErrIdNotExist       = fmt.Errorf("ID not exist")
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
	tc := torrent.ClientConfig{
		DataDir:    e.dataDir,
		NoUpload:   !c.EnableUpload,
		Seed:       c.EnableSeeding,
		DisableUTP: true,
	}
	tc.SetListenAddr("0.0.0.0:" + strconv.Itoa(c.IncomingPort))
	client, err := torrent.NewClient(&tc)
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
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
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
	progress := NewProgressDownload(id, int(info.TotalLength()), os.Stdout)
	// Download layer file
	if err := e.StartLeecher(id, metaInfo, progress); err != nil {
		log.Errorf("Download layer %s failed: %v", id, err)
		return "", err
	} else {
		log.Infof("Download layer %s success", id)
	}
	return e.GetFilePath(id), nil
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

func (e *BtEngine) GetTorrent(id string) ([]byte, error) {
	var b bytes.Buffer

	if !e.started {
		return nil, ErrBtEngineNotStart
	}

	info, ok := e.idInfos[id]
	if !ok {
		return nil, fmt.Errorf("Get torrent for %s not founded", id)
	}

	t, err := e.getTorrent(info.InfoHash)
	if err != nil {
		return nil, fmt.Errorf("Get torrent for %s failed: %v", err)
	}

	m := t.tt.Metainfo()
	w := bufio.NewWriter(&b)
	if err = m.Write(w); err != nil {
		return nil, fmt.Errorf("Write metainfo %s error: %v", id, err)
	}
	return b.Bytes(), nil
}

func (e *BtEngine) GetStatus(id string) (*Status, error) {
	if !e.started {
		return nil, ErrBtEngineNotStart
	}

	info, ok := e.idInfos[id]
	if !ok {
		return nil, ErrIdNotExist
	}

	t, err := e.getTorrent(info.InfoHash)
	if err != nil {
		return nil, fmt.Errorf("Get status for %s failed: %v", id, err)
	}

	t.Update()
	return &Status{
		Id:        id,
		State:     t.State.String(),
		Completed: t.Downloaded,
		TotalLen:  t.Size,
		Seeding:   t.Seeding,
	}, nil
}

func (e *BtEngine) GetAllStatus() ([]Status, error) {
	if !e.started {
		return nil, ErrBtEngineNotStart
	}

	e.mut.Lock()
	defer e.mut.Unlock()

	var ss []Status
	for id, info := range e.idInfos {
		t, err := e.getTorrent(info.InfoHash)
		if err != nil {
			log.Errorf("Get status for %s failed: %v", id, err)
			continue
		}

		t.Update()
		ss = append(ss, Status{
			Id:        id,
			State:     t.State.String(),
			Completed: t.Downloaded,
			TotalLen:  t.Size,
			Seeding:   t.Seeding,
		})
	}
	log.Debugf("All status: %v", ss)
	return ss, nil
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

func (e *BtEngine) StopTorrent(id string) error {
	if !e.started {
		return ErrBtEngineNotStart
	}

	e.mut.Lock()
	defer e.mut.Unlock()

	info, ok := e.idInfos[id]
	if !ok || (ok && !info.Started) {
		return nil
	}

	info.Count--
	if info.Count > 0 {
		return nil
	}

	infoHash := info.InfoHash
	if err := e.stopTorrent(infoHash); err != nil {
		return fmt.Errorf("Stop torrent failed: %v", err)
	}

	info.Started = false
	return nil
}

func (e *BtEngine) StartLeecher(id string, metaInfo *metainfo.MetaInfo, p *ProgressDownload) error {
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
		p.waitComplete(t)
		log.Infof("Bt download %s completed", id)
	}
	return nil
}

func (e *BtEngine) DeleteTorrent(id string) error {
	if !e.started {
		return ErrBtEngineNotStart
	}

	e.mut.Lock()
	defer e.mut.Unlock()

	info, ok := e.idInfos[id]
	if !ok {
		return nil
	}

	if info.Started {
		return fmt.Errorf("Id %s torrent is still started, stop it first", id)
	}

	infoHash := info.InfoHash
	if err := e.deleteTorrent(infoHash); err != nil {
		return fmt.Errorf("Delete torrent failed: %v", err)
	}
	delete(e.idInfos, id)

	// Remove data file and torrent file
	dfn := e.GetFilePath(id)
	if err := os.Remove(dfn); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("Remove data file %s failed: %v", dfn, err)
	}

	tfn := e.GetTorrentFilePath(id)
	if err := os.Remove(tfn); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("Remove torrent file %s failed: %v", tfn, err)
	}
	return nil
}

func (e *BtEngine) createTorrent(id string) error {
	mi := metainfo.MetaInfo{
		Announce: e.trackers[0],
	}
	mi.SetDefaults()
	info, err := mi.UnmarshalInfo()
	if err != nil {
		return fmt.Errorf("UnmarshalInfo failed: %v", err)
	}
	info.PieceLength = 1024 * 1024 //1MB

	f := e.GetFilePath(id)
	err = info.BuildFromFilePath(f)
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

func (e *BtEngine) addMagnet(magnetURI string) (*Torrent, error) {
	//adds the torrent but does not start it
	tt, err := e.client.AddMagnet(magnetURI)
	if err != nil {
		return nil, err
	}
	t := e.addTorrent(tt)

	go func() {
		<-t.tt.GotInfo()
		e.startTorrent(t.InfoHash)
	}()

	return t, nil
}

//GetTorrents moves torrents out of the anacrolix/torrent
//and into the local cache
func (e *BtEngine) GetTorrents() map[string]*Torrent {
	e.mut.Lock()
	defer e.mut.Unlock()

	if e.client == nil {
		return nil
	}
	for _, t := range e.ts {
		t.Update()
	}
	return e.ts
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
	//update torrent fields using underlying torrent
	torrent.Update()
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

func (e *BtEngine) deleteTorrent(infohash string) error {
	t, err := e.getTorrent(infohash)
	if err != nil {
		return err
	}
	delete(e.ts, t.InfoHash)
	ih, _ := str2ih(infohash)
	if tt, ok := e.client.Torrent(ih); ok {
		tt.Drop()
	}
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
