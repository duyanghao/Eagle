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
	"flag"
	"fmt"
	"github.com/duyanghao/eagle/pkg/utils/ratelimiter"
	pb "github.com/duyanghao/eagle/proto/metainfo"
	"github.com/duyanghao/eagle/seeder/bt"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"net"
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
	if config.DaemonCfg.Verbose {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}
	// start seeder bt
	log.Infof("Start seeder bt on port: %d ...", config.SeederCfg.Port)
	c := &bt.Config{
		EnableUpload:    true,
		EnableSeeding:   true,
		IncomingPort:    config.SeederCfg.Port,
		DownloadTimeout: time.Duration(config.SeederCfg.DownloadTimeout),
		CacheLimitSize:  ratelimiter.RateConvert(config.SeederCfg.LimitSize),
	}
	seeder, err := bt.NewSeeder(config.SeederCfg.RootDirectory, config.SeederCfg.StorageBackend, config.SeederCfg.Origin, config.SeederCfg.Trackers, c)
	if err != nil {
		log.Fatal(err)
	}
	err = seeder.Run()
	if err != nil {
		log.Fatal(err)
	}
	log.Infof("Start seeder bt on port: %d successfully", config.SeederCfg.Port)

	// start seeder
	log.Infof("Launch seeder on port: %d", config.DaemonCfg.Port)
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", config.DaemonCfg.Port))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterMetaInfoServer(s, seeder)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
