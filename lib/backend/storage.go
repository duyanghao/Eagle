// Copyright (c) 2016-2019 Uber Technologies, Inc.
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
package backend

import (
	"fmt"
	"github.com/anacrolix/torrent/metainfo"
)

type FileInfo struct {
	Name   string
	Length int64
}

var _factories = make(map[string]StorageFactory)

// StorageFactory creates backend client given name.
type StorageFactory interface {
	Create(config interface{}, authConfig interface{}) (Storage, error)
}

// Register registers new Factory with corresponding backend client name.
func Register(name string, factory StorageFactory) {
	_factories[name] = factory
}

// getFactory returns backend client factory given client name.
func getFactory(name string) (StorageFactory, error) {
	factory, ok := _factories[name]
	if !ok {
		return nil, fmt.Errorf("no backend client defined with name %s", name)
	}
	return factory, nil
}

func GetStorageBackend(name string, config interface{}, authConfig interface{}) (Storage, error) {
	factory, err := getFactory(name)
	if err != nil {
		return nil, fmt.Errorf("get backend storage factory: %s", err)
	}
	s, err := factory.Create(config, authConfig)
	if err != nil {
		return nil, fmt.Errorf("create backend storage: %s", err)
	}
	return s, nil
}

// Storage defines an interface for accessing blobs on a remote storage backend.
//
// Implementations of Storage must be thread-safe, since they are cached and
// used concurrently by Manager.
type Storage interface {
	// Create creates torrent with meta info
	CreateWithMetaInfo(name string, info *metainfo.MetaInfo) error

	// Stat is useful when we need to quickly know if a blob exists (and maybe
	// some basic information about it), without downloading the entire blob,
	// which may be very large.
	Stat(name string) (*FileInfo, error)

	// Upload uploads data into name.
	Upload(name string, data []byte) error

	// Download downloads name into dst. All implementations should return
	// backenderrors.ErrBlobNotFound when the blob was not found.
	Download(name string) ([]byte, error)

	// Delete removes relevant name
	Delete(name string) error

	// List lists entries whose names start with prefix.
	List(prefix string) ([]*FileInfo, error)

	// GetFilePath returns data path
	GetFilePath(id string) string

	// GetTorrentFilePath returns torrent path
	GetTorrentFilePath(id string) string

	// GetDataDir returns directory of data
	GetDataDir() string

	// GetTorrentDir returns directory of torrent
	GetTorrentDir() string
}
