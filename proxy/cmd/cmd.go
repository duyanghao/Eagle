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
	"github.com/duyanghao/eagle/eagleclient"
	"github.com/duyanghao/eagle/proxy/routes"
	"net/http"

	log "github.com/sirupsen/logrus"

	"flag"
	"github.com/duyanghao/eagle/pkg/utils/ratelimiter"
	"github.com/duyanghao/eagle/proxy/transport"
	"time"
)

// Flags defines seeder CLI flags.
type Flags struct {
	ConfigFile string
}

// ParseFlags parses origin CLI flags.
func ParseFlags() *Flags {
	var flags Flags
	flag.StringVar(
		&flags.ConfigFile, "config", "", "configuration file path")
	flag.Parse()
	return &flags
}

func Run(flags *Flags) {
	// create config
	log.Infof("Start to load config %s ...", flags.ConfigFile)
	config, err := LoadConfig(flags.ConfigFile)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	log.Infof("Load config %s successfully", flags.ConfigFile)

	// set log level
	if config.proxyCfg.Verbose {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}

	// start eagleClient
	log.Infof("Start eagleClient ...")
	c := &eagleclient.Config{
		EnableUpload:      true,
		EnableSeeding:     true,
		IncomingPort:      config.clientCfg.Port,
		DownloadTimeout:   time.Duration(config.clientCfg.DownloadTimeout),
		UploadRateLimit:   ratelimiter.RateConvert(config.clientCfg.UploadRateLimit),
		DownloadRateLimit: ratelimiter.RateConvert(config.clientCfg.DownloadRateLimit),
		CacheLimitSize:    ratelimiter.RateConvert(config.clientCfg.LimitSize),
	}
	eagleClient := eagleclient.NewBtEngine(config.clientCfg.RootDirectory, config.clientCfg.Trackers, config.clientCfg.Seeders, c)
	proxyRoundTripper := transport.NewProxyRoundTripper(eagleClient, config.proxyCfg.Rules)
	err = proxyRoundTripper.P2PClient.Run()
	if err != nil {
		log.Fatal("Start eagleClient failure: %v", err)
	}
	log.Infof("Start eagleClient successfully ...")

	// init routes
	routes.InitMux()

	// start proxy
	log.Infof("Launch proxy on port: %d", config.proxyCfg.Port)
	if config.proxyCfg.CertFile != "" && config.proxyCfg.KeyFile != "" {
		err = http.ListenAndServeTLS(fmt.Sprintf(":%d", config.proxyCfg.Port), config.proxyCfg.CertFile, config.proxyCfg.KeyFile, nil)
	} else {
		err = http.ListenAndServe(fmt.Sprintf(":%d", config.proxyCfg.Port), nil)
	}
	if err != nil {
		log.Fatal(err)
	}
}
