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
	clientCfg *ClientCfg `yaml:"clientCfg,omitempty"`
	proxyCfg  *ProxyCfg  `yaml:"proxyCfg,omitempty"`
}

// validate the configuration
func (c *Config) validate() error {
	if c.clientCfg.RootDirectory == "" || len(c.clientCfg.Trackers) == 0 || len(c.clientCfg.Seeders) == 0 ||
		c.clientCfg.LimitSize == "" || c.clientCfg.Port <= 0 {
		return fmt.Errorf("Invalid eagle client configurations, please check ...")
	}
	if !ratelimiter.ValidateRateLimiter(c.clientCfg.DownloadRateLimit) ||
		!ratelimiter.ValidateRateLimiter(c.clientCfg.UploadRateLimit) ||
		!ratelimiter.ValidateRateLimiter(c.clientCfg.LimitSize) {
		return fmt.Errorf("Invalid ratelimiter format, please check ...")
	}
	if c.proxyCfg.Port <= 0 {
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
