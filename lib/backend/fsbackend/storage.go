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
package fsbackend

import (
	"errors"
	"github.com/duyanghao/eagle/lib/backend"
	"gopkg.in/yaml.v2"
	"os"
)

const _fs = "fs"

func init() {
	backend.Register(_fs, &factory{})
}

type factory struct{}

func (f *factory) Create(
	confRaw interface{}, authConfRaw interface{}) (backend.Storage, error) {

	confBytes, err := yaml.Marshal(confRaw)
	if err != nil {
		return nil, errors.New("marshal fs config")
	}
	var config Config
	if err := yaml.Unmarshal(confBytes, &config); err != nil {
		return nil, errors.New("unmarshal fs config")
	}
	storage, err := NewStorage(config)
	if err != nil {
		return nil, err
	}

	// Create data and torrent directory
	if err := os.MkdirAll(storage.GetDataDir(), 0700); err != nil && !os.IsExist(err) {
		return nil, err
	}
	if err := os.MkdirAll(storage.GetTorrentDir(), 0700); err != nil && !os.IsExist(err) {
		return nil, err
	}
	return storage, nil
}

// Client implements a backend.Storage for FileSystem.
type Storage struct {
	config Config
}

// Option allows setting optional Client parameters.
type Option func(storage *Storage)

// WithS3 configures a Storage with a custom config implementation.
func WithS3(config Config) Option {
	return func(storage *Storage) { storage.config = config }
}

// NewStorage creates a new Storage for file system.
func NewStorage(
	config Config, opts ...Option) (*Storage, error) {

	storage := &Storage{config}
	for _, opt := range opts {
		opt(storage)
	}
	return storage, nil
}
