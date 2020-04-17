// Copyright 1999-2017 Alibaba Group.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package util

import (
	"flag"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/duyanghao/eagle/proxy/constant"
	. "github.com/duyanghao/eagle/proxy/global"
	. "github.com/duyanghao/eagle/proxy/muxconf"
	"github.com/duyanghao/eagle/proxy/util"
)

func init() {
	log.Info("init...")

	// init command line param
	initParam()

	//http handler mapper
	InitMux()

	//clean local data dir
	go cleanLocalRepo()

	log.Info("init finish")
}

func cleanLocalRepo() {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("recover cleanLocalRepo from err:%v", err)
			go cleanLocalRepo()
		}
	}()
	for {
		select {
		case <-time.After(time.Minute * 10):
			func() {
				log.Info("scan repo and clean expired files")
				filepath.Walk(path.Join(G_CommandLine.P2PClientRootDir, "data"), func(path string, info os.FileInfo, err error) error {
					if err != nil {
						log.Warnf("walk file:%s error:%v", path, err)
					} else {
						if info.Mode().IsRegular() {
							if time.Now().Unix()-reflect.ValueOf(info.Sys()).Elem().FieldByName("Atim").Field(0).Int() >= 3600 {
								if err := os.Remove(path); err == nil {
									log.Infof("remove file:%s success", path)
								} else {
									log.Warnf("remove file:%s error:%v", path, err)
								}
							}
						}
					}
					return nil
				})
			}()
		}
	}
}

func initParam() {
	flag.StringVar(&G_CommandLine.RateLimit, "ratelimit", "", "net speed limit,format:xxxM/K")
	flag.StringVar(&G_CommandLine.CallSystem, "callsystem", "com_ops_dragonfly", "caller name")
	flag.StringVar(&G_CommandLine.Urlfilter, "urlfilter", "Signature&Expires&OSSAccessKeyId", "filter specified url fields")
	flag.BoolVar(&G_CommandLine.Notbs, "notbs", true, "not try back source to download if throw exception")
	flag.BoolVar(&G_CommandLine.Version, "v", false, "version")
	flag.BoolVar(&G_CommandLine.Verbose, "verbose", false, "verbose")
	flag.BoolVar(&G_CommandLine.Help, "h", false, "help")
	flag.UintVar(&G_CommandLine.Port, "port", 65001, "daemon will listen the port")
	flag.StringVar(&G_CommandLine.Registry, "registry", "", "registry addr(https://abc.xx.x or http://abc.xx.x) and must exist if df-daemon is used to mirror mode")
	flag.StringVar(&G_CommandLine.DownRule, "rule", "", "download the url by P2P if url matches the specified pattern,format:reg1,reg2,reg3")
	flag.StringVar(&G_CommandLine.CertFile, "certpem", "", "cert.pem file path")
	flag.StringVar(&G_CommandLine.KeyFile, "keypem", "", "key.pem file path")
	flag.StringVar(&G_CommandLine.P2PClientRootDir, "rootdir", "/data/", "root directory of p2p client")
	flag.StringVar(&G_CommandLine.P2PClientTrackers, "trackers", "", "tracker list of p2p client")
	flag.StringVar(&G_CommandLine.P2PClientSeeders, "seeders", "", "seeder list of p2p client")

	flag.Parse()

	if G_CommandLine.Version {
		fmt.Print(constant.VERSION)
		os.Exit(0)
	}
	if G_CommandLine.Help {
		flag.Usage()
		os.Exit(0)
	}

	if G_CommandLine.Verbose {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}

	if len(G_CommandLine.RateLimit) == 0 {
		G_CommandLine.RateLimit = util.NetLimit()
	} else if isMatch, _ := regexp.MatchString("^[[:digit:]]+[MK]$", G_CommandLine.RateLimit); !isMatch {
		os.Exit(constant.CODE_EXIT_RATE_LIMIT_INVALID)
	}

	if G_CommandLine.Port <= 2000 || G_CommandLine.Port > 65535 {
		os.Exit(constant.CODE_EXIT_PORT_INVALID)
	}

	downRule := strings.Split(G_CommandLine.DownRule, ",")
	for _, rule := range downRule {
		UpdateDFPattern(rule)
	}

	if G_CommandLine.CertFile != "" && G_CommandLine.KeyFile != "" {
		G_UseHttps = true
	}

	if G_CommandLine.Registry != "" {
		protoAndDomain := strings.SplitN(G_CommandLine.Registry, "://", 2)
		splitedCount := len(protoAndDomain)
		G_RegProto = "http"
		G_RegDomain = protoAndDomain[splitedCount-1]
		if splitedCount == 2 {
			G_RegProto = protoAndDomain[0]
		}
	}

	G_P2PClientTrackers = strings.Split(G_CommandLine.P2PClientTrackers, ",")

	G_P2PClientSeeders = strings.Split(G_CommandLine.P2PClientSeeders, ",")
}
