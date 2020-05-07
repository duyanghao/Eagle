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
package cmd

import (
	"fmt"
	"github.com/duyanghao/eagle/pkg/utils/ratelimiter"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type ClientCfg struct {
	RootDirectory     string   `yaml:"rootDirectory,omitempty"`
	Trackers          []string `yaml:"trackers,omitempty"`
	Seeders           []string `yaml:"seeders,omitempty"`
	LimitSize         string   `yaml:"limitSize,omitempty"`
	DownloadRateLimit string   `yaml:"downloadRateLimit,omitempty"`
	UploadRateLimit   string   `yaml:"uploadRateLimit,omitempty"`
	DownloadTimeout   int      `yaml:"downloadTimeout,omitempty"`
	Port              int      `yaml:"port,omitempty"`
}

type ProxyCfg struct {
	Port     int      `yaml:"port,omitempty"`
	Verbose  bool     `yaml:"verbose,omitempty"`
	Rules    []string `yaml:"rules,omitempty"`
	CertFile string   `yaml:"certFile,omitempty"`
	KeyFile  string   `yaml:"keyFile,omitempty"`
}

type Config struct {
	ClientCfg *ClientCfg `yaml:"clientCfg,omitempty"`
	ProxyCfg  *ProxyCfg  `yaml:"proxyCfg,omitempty"`
}

// validate the configuration
func (c *Config) validate() error {
	if c.ClientCfg.RootDirectory == "" || len(c.ClientCfg.Trackers) == 0 || len(c.ClientCfg.Seeders) == 0 ||
		c.ClientCfg.LimitSize == "" || c.ClientCfg.Port <= 0 {
		return fmt.Errorf("Invalid eagle client configurations, please check ...")
	}
	if !ratelimiter.ValidateRateLimiter(c.ClientCfg.DownloadRateLimit) ||
		!ratelimiter.ValidateRateLimiter(c.ClientCfg.UploadRateLimit) ||
		!ratelimiter.ValidateRateLimiter(c.ClientCfg.LimitSize) {
		return fmt.Errorf("Invalid ratelimiter format, please check ...")
	}
	if c.ProxyCfg.Port <= 0 {
		return fmt.Errorf("Invalid proxy configurations, please check ...")
	}
	// TODO: other configuration validate ...
	return nil
}

// LoadConfig parses configuration file and returns
// an initialized Settings object and an error object if any. For instance if it
// cannot find the configuration file it will set the returned error appropriately.
func LoadConfig(path string) (*Config, error) {
	c := &Config{}
	contents, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("Failed to read configuration file: %s,error: %s", path, err)
	}
	if err = yaml.Unmarshal(contents, c); err != nil {
		return nil, fmt.Errorf("Failed to parse configuration,error: %s", err)
	}
	if err = c.validate(); err != nil {
		return nil, fmt.Errorf("Invalid configuration,error: %s", err)
	}
	return c, nil
}
