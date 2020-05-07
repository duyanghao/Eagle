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
	"github.com/anacrolix/torrent/metainfo"
	"github.com/duyanghao/eagle/lib/backend"
	"io/ioutil"
	"os"
	"path"
)

// Create creates name and returns io.Writer
func (fs *Storage) CreateWithMetaInfo(name string, info *metainfo.MetaInfo) error {
	tFile, err := os.Create(name)
	if err != nil {
		return err
	}
	defer tFile.Close()

	if err = info.Write(tFile); err != nil {
		return err
	}
	return nil
}

// Stat is useful when we need to quickly know if a blob exists (and maybe
// some basic information about it), without downloading the entire blob,
// which may be very large.
func (fs *Storage) Stat(name string) (*backend.FileInfo, error) {
	f, err := os.Lstat(name)
	if err != nil {
		return nil, err
	}
	return &backend.FileInfo{
		Name:   f.Name(),
		Length: f.Size(),
	}, nil
}

// Upload writes data to name file
func (fs *Storage) Upload(name string, data []byte) error {
	return ioutil.WriteFile(name, data, 0644)
}

// Download reads file content from name
func (fs *Storage) Download(name string) ([]byte, error) {
	content, err := ioutil.ReadFile(name)
	if err != nil {
		return nil, err
	}
	return content, nil
}

// Delete removes name file
func (fs *Storage) Delete(name string) error {
	return os.Remove(name)
}

// List lists fileEntries whose names start with prefix.
func (fs *Storage) List(prefix string) ([]*backend.FileInfo, error) {
	files, err := ioutil.ReadDir(prefix)
	if err != nil {
		return nil, err
	}
	var infos []*backend.FileInfo
	for _, f := range files {
		infos = append(infos, &backend.FileInfo{
			Name:   f.Name(),
			Length: f.Size(),
		})
	}
	return infos, nil
}

// GetFilePath returns data file path
func (fs *Storage) GetFilePath(id string) string {
	return path.Join(fs.config.RootDirectory, "data", id+".layer")
}

// GetTorrentFilePath returns torrent file path
func (fs *Storage) GetTorrentFilePath(id string) string {
	return path.Join(fs.config.RootDirectory, "torrents", id+".torrent")
}

func (fs *Storage) GetDataDir() string {
	return path.Join(fs.config.RootDirectory, "data")
}

func (fs *Storage) GetTorrentDir() string {
	return path.Join(fs.config.RootDirectory, "torrents")
}
