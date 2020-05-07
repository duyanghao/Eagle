// Copyright 2020 duyanghao
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
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
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/bencode"
	"github.com/anacrolix/torrent/metainfo"
	"github.com/duyanghao/eagle/lib/backend"
	"github.com/duyanghao/eagle/lib/backend/fsbackend"
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
	DownloadTimeout   time.Duration
}

// Seeder backed by anacrolix/torrent
type Seeder struct {
	sync.RWMutex
	lruCache   *lrucache.LruCache
	client     *torrent.Client
	httpClient *http.Client
	config     *Config
	idInfos    map[string]*torrent.Torrent // layer digest -> Torrent
	trackers   []string
	origin     string
	storage    backend.Storage
}

func NewSeeder(root, storage, origin string, trackers []string, c *Config) (*Seeder, error) {
	if c == nil {
		c = &Config{
			EnableUpload:      true,
			EnableSeeding:     true,
			IncomingPort:      50017,
			UploadRateLimit:   DefaultUploadRateLimit,
			DownloadRateLimit: DefaultDownloadRateLimit,
		}
	}
	// Create storage backend
	backendCfg := fsbackend.Config{RootDirectory: root}
	s, err := backend.GetStorageBackend(storage, backendCfg, nil)
	if err != nil {
		return nil, err
	}
	return &Seeder{
		trackers: trackers,
		origin:   origin,
		config:   c,
		idInfos:  make(map[string]*torrent.Torrent),
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
		storage: s,
	}, nil
}

func (s *Seeder) Run() error {
	c := s.config
	if c.IncomingPort <= 0 {
		return fmt.Errorf("Invalid incoming port (%d)", c.IncomingPort)
	}

	tc := torrent.NewDefaultClientConfig()
	tc.DataDir = s.storage.GetDataDir()
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

	files, err := s.storage.List(s.storage.GetDataDir())
	if err != nil {
		return err
	}

	for _, f := range files {
		go func(f *backend.FileInfo) {
			if filepath.Ext(f.Name) != ".layer" {
				return
			}
			ss := strings.Split(f.Name, ".")
			if len(ss) != 2 {
				log.Errorf("Found invalid layer file %s", f.Name)
				return
			}

			id := ss[0]
			df := s.storage.GetFilePath(id)

			if _, err = s.storage.Stat(df); err != nil {
				return
			}

			if err = s.StartSeed(context.Background(), id); err != nil {
				log.Errorf("Start seed %s failed: %v", id, err)
				return
			}
			s.lruCache.CreateIfNotExists(id)
			s.lruCache.SetComplete(id, f.Length)
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
func (s *Seeder) getMetaData(ctx context.Context, reqUrl, id string) (int, error) {
	// step1 - get data from origin
	log.Debugf("Torrent of layer: %s not found, let's fetch data from origin ...", id)
	data, err := s.getDataFromOrigin(reqUrl)
	if err != nil {
		log.Errorf("get torrent of layer: %s failed, error: %v", id, err)
		return len(data), err
	}
	// step2 - generate layerFile
	log.Debugf("Start to generate dataFile of layer: %s ...", id)
	layerFile := s.storage.GetFilePath(id)
	err = s.storage.Upload(layerFile, data)
	if err != nil {
		return len(data), err
	}
	// step3 - start seed
	log.Debugf("Start to seed layer: %s ...", id)
	return len(data), s.StartSeed(ctx, id)
}

// getMetaDataSync generates layer file and its relevant torrent only once for each of layer
func (s *Seeder) getMetaDataSync(reqUrl, id string) error {
	// get only once each of layer
	torrentFile := s.storage.GetTorrentFilePath(id)
	layerFile := s.storage.GetFilePath(id)
Loop:
	entry, exist := s.lruCache.Get(id)
Execute:
	if exist {
		if entry.Completed {
			if _, err := s.storage.Stat(torrentFile); err != nil {
				log.Errorf("Failed to find torrent file of cached layer: %s, try to remove its relevant records", id)
				s.lruCache.Remove(id)
				return err
			}
			if _, err := s.storage.Stat(layerFile); err != nil {
				log.Errorf("Failed to find data file of cached layer: %s, try to remove its relevant records", id)
				s.lruCache.Remove(id)
				return err
			}
			log.Infof("Layer: %s has been cached, return directly", id)
			return nil
		}
		// wait
		for {
			select {
			case <-entry.Done:
				log.Debugf("Layer: %s cache updated, try to get it again...", id)
				goto Loop
			}
		}
	}
	entry, exist = s.lruCache.CreateIfNotExists(id)
	if exist {
		goto Execute
	} else { // get layer from origin
		var err error
		errChan := make(chan error, 1)
		sizeChan := make(chan int, 1)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		go func() {
			size, err := s.getMetaData(ctx, reqUrl, id)
			sizeChan <- size
			errChan <- err
		}()
		select {
		case err = <-errChan:
			size := <-sizeChan
			if err != nil {
				log.Errorf("GetMetaData layer: %s failed, %v, try to remove its relevant records ...", id, err)
				s.storage.Delete(torrentFile)
				s.storage.Delete(layerFile)
				s.lruCache.Remove(id)
			} else {
				log.Infof("GetMetaData layer: %s successfully, try to update status ...", id)
				s.lruCache.SetComplete(id, int64(size))
			}
		case <-time.After(s.config.DownloadTimeout * time.Second):
			err = fmt.Errorf("GetMetaData layer: %s timeout %s", id, s.config.DownloadTimeout)
			log.Errorf("GetMetaData layer: %s timeout %s, %v, try to remove its relevant records ...", id, s.config.DownloadTimeout, err)
			s.storage.Delete(torrentFile)
			s.storage.Delete(layerFile)
			s.lruCache.Remove(id)
		}
		return err
	}
}

// GetMetaData get torrent of layer
func (s *Seeder) GetMetaInfo(ctx context.Context, metaInfoReq *pb.MetaInfoRequest) (*pb.MetaInfoReply, error) {
	log.Debugf("Access: %s", metaInfoReq.Url)
	digest := metaInfoReq.Url[strings.LastIndex(metaInfoReq.Url, "/")+1:]
	id := distdigests.Digest(digest).Encoded()
	log.Debugf("Start to get metadata of layer %s", id)
	err := s.getMetaDataSync(metaInfoReq.Url, id)
	if err != nil {
		return nil, fmt.Errorf("Get metainfo from origin failed: %v", err)
	}
	torrentFile := s.storage.GetTorrentFilePath(id)
	content, err := s.storage.Download(torrentFile)
	if err != nil {
		return nil, fmt.Errorf("Download metainfo file failed: %v", err)
	}
	return &pb.MetaInfoReply{Metainfo: content}, nil
}

// StartSeed seeds relevant blob
func (s *Seeder) StartSeed(ctx context.Context, id string) error {
	tf := s.storage.GetTorrentFilePath(id)
	if _, err := s.storage.Stat(tf); err != nil {
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
		p.WaitComplete(ctx, tt)
	}

	return nil
}

func (s *Seeder) DeleteTorrent(id string) {
	// remove info and bt torrent records
	s.deleteTorrent(id)

	// remove data file and torrent file asynchronously
	go func() {
		tf := s.storage.GetTorrentFilePath(id)
		if err := s.storage.Delete(tf); err != nil && !os.IsNotExist(err) {
			log.Errorf("Remove torrent file %s failed: %v", tf, err)
		}

		df := s.storage.GetFilePath(id)
		if err := s.storage.Delete(df); err != nil && !os.IsNotExist(err) {
			log.Errorf("Remove layer file %s failed: %v", df, err)
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
	f := s.storage.GetFilePath(id)
	err := info.BuildFromFilePath(f)
	if err != nil {
		return fmt.Errorf("Create torrent file for %s failed: %v", f, err)
	}
	var announceList [][]string
	announceList = append(announceList, s.trackers)
	mi := metainfo.MetaInfo{
		AnnounceList: announceList,
	}
	mi.SetDefaults()
	mi.InfoBytes, err = bencode.Marshal(&info)
	if err != nil {
		return fmt.Errorf("Create torrent file for %s failed: %v", f, err)
	}

	tf := s.storage.GetTorrentFilePath(id)
	err = s.storage.CreateWithMetaInfo(tf, &mi)
	if err != nil {
		return fmt.Errorf("CreateWithMetaInfo for %s failed: %v", tf, err)
	}

	log.Infof("Create torrent file %s success", tf)
	return nil
}

func (s *Seeder) addTorrent(id string, tt *torrent.Torrent) {
	s.Lock()
	defer s.Unlock()
	s.idInfos[id] = tt
}
