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

// NewStorage creates a new Client for S3.
func NewStorage(
	config Config, opts ...Option) (*Storage, error) {

	storage := &Storage{config}
	for _, opt := range opts {
		opt(storage)
	}
	return storage, nil
}
