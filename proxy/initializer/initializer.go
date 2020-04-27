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
	log "github.com/sirupsen/logrus"
	"os"
	"regexp"
	"strings"

	"github.com/duyanghao/eagle/proxy/constant"
	. "github.com/duyanghao/eagle/proxy/global"
	. "github.com/duyanghao/eagle/proxy/muxconf"
)

func init() {
	log.Info("init...")

	// init command line param
	initParam()

	//http handler mapper
	InitMux()

	log.Info("init finish")
}

func initParam() {
	flag.StringVar(&G_CommandLine.Urlfilter, "urlfilter", "Signature&Expires&OSSAccessKeyId", "filter specified url fields")
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
	flag.StringVar(&G_CommandLine.P2PClientDownloadRateLimit, "drl", "50M", "net speed limit for bt download,format:xxxM/K")
	flag.StringVar(&G_CommandLine.P2PClientUploadRateLimit, "url", "50M", "net speed limit for bt upload,format:xxxM/K")
	flag.StringVar(&G_CommandLine.P2PClientCacheLimitSize, "limitsize", "100G", "cache size limit for p2p client,format:xxxT/G")
	flag.IntVar(&G_CommandLine.P2PClientDownloadTimeout, "timeout", 30, "p2pclient download timeout(s)")

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

	if isMatch, _ := regexp.MatchString("^[[:digit:]]+[MK]$", G_CommandLine.P2PClientDownloadRateLimit); !isMatch {
		os.Exit(constant.CODE_EXIT_RATE_LIMIT_INVALID)
	}

	if isMatch, _ := regexp.MatchString("^[[:digit:]]+[MK]$", G_CommandLine.P2PClientUploadRateLimit); !isMatch {
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
