package cmd

import (
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type SeederCfg struct {
	RootDirectory   string   `yaml:"rootDirectory,omitempty"`
	Origin          string   `yaml:"origin,omitempty"`
	Trackers        []string `yaml:"trackers,omitempty"`
	LimitSize       string   `yaml:"limitSize,omitempty"`
	DownloadTimeout int      `yaml:"downloadTimeout,omitempty"`
	StorageBackend  string   `yaml:"storageBackend,omitempty"`
	Port            int      `yaml:"port,omitempty"`
}

type DaemonCfg struct {
	Port    int  `yaml:"port,omitempty"`
	Verbose bool `yaml:"verbose,omitempty"`
}

type Config struct {
	seederCfg *SeederCfg `yaml:"seederCfg,omitempty"`
	daemonCfg *DaemonCfg `yaml:"daemonCfg,omitempty"`
}

// validate the configuration
func (c *Config) validate() error {
	if c.seederCfg.RootDirectory == "" || c.seederCfg.Origin == "" || c.seederCfg.Port <= 0 ||
		len(c.seederCfg.Trackers) == 0 || c.seederCfg.LimitSize == "" || c.seederCfg.StorageBackend == "" {
		return fmt.Errorf("Invalid seeder configurations, please check ...")
	}
	if c.daemonCfg.Port <= 0 {
		return fmt.Errorf("Invalid daemon configurations, please check ...")
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
