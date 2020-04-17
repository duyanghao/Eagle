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
package handler

import (
	"crypto/tls"
	"net"
	. "net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/duyanghao/eagle/p2p-client/btclient"
	"github.com/duyanghao/eagle/proxy/global"
)

type ProxyRoundTripper struct {
	Round     *Transport
	Round2    RoundTripper
	P2PClient *btclient.BtEngine
}

var proxyRoundTripper = &ProxyRoundTripper{
	Round: &Transport{
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
	},
	Round2: NewFileTransport(Dir("/")),
}

var compiler = regexp.MustCompile("^.+/blobs/sha256.*$")

func Run() error {
	// construct btclient config
	c := &btclient.Config{
		EnableUpload:  true,
		EnableSeeding: true,
		IncomingPort:  50007,
	}
	// transform ratelimiter
	switch global.G_CommandLine.P2PClientUploadRateLimit[len(global.G_CommandLine.P2PClientUploadRateLimit)-1:] {
	case "M":
		c.UploadRateLimit, _ = strconv.Atoi(global.G_CommandLine.P2PClientUploadRateLimit[len(global.G_CommandLine.P2PClientUploadRateLimit)-1:])
		c.UploadRateLimit *= 1024 * 1024
	case "K":
		c.UploadRateLimit, _ = strconv.Atoi(global.G_CommandLine.P2PClientUploadRateLimit[len(global.G_CommandLine.P2PClientUploadRateLimit)-1:])
		c.UploadRateLimit *= 1024
	}
	switch global.G_CommandLine.P2PClientDownloadRateLimit[len(global.G_CommandLine.P2PClientDownloadRateLimit)-1:] {
	case "M":
		c.DownloadRateLimit, _ = strconv.Atoi(global.G_CommandLine.P2PClientDownloadRateLimit[len(global.G_CommandLine.P2PClientDownloadRateLimit)-1:])
		c.DownloadRateLimit *= 1024 * 1024
	case "K":
		c.DownloadRateLimit, _ = strconv.Atoi(global.G_CommandLine.P2PClientDownloadRateLimit[len(global.G_CommandLine.P2PClientDownloadRateLimit)-1:])
		c.DownloadRateLimit *= 1024
	}
	proxyRoundTripper.P2PClient = btclient.NewBtEngine(global.G_CommandLine.P2PClientRootDir, global.G_P2PClientTrackers, global.G_P2PClientSeeders, c)
	return proxyRoundTripper.P2PClient.Run()
}

func needUseP2PClient(req *Request, location string) bool {
	if req.Method == MethodGet {
		if !compiler.MatchString(req.URL.Path) {
			return false
		}
		if location != "" {
			return global.MatchDfPattern(location)
		}
		return true
	}
	return false
}

//only process first redirect at present
//fix resource release
func (roundTripper *ProxyRoundTripper) RoundTrip(req *Request) (*Response, error) {
	urlString := req.URL.String()
	if needUseP2PClient(req, urlString) {
		logrus.Debugf("try to get blob: %s through p2p based image distribution system ...", urlString)
		if res, err := proxyRoundTripper.download(req, urlString); err == nil {
			return res, err
		}
		logrus.Errorf("failed to get blob: %s from p2p based image distribution system, let's switch to original request ...", urlString)
	}

	req.Host = req.URL.Host
	req.Header.Set("Host", req.Host)
	res, err := roundTripper.Round.RoundTrip(req)
	return res, err
}

func (roundTripper *ProxyRoundTripper) download(req *Request, urlString string) (*Response, error) {
	//use P2PClient to download
	if dstPath, err := roundTripper.P2PClient.DownloadLayer(req, urlString); err == nil {
		// defer os.Remove(dstPath)
		if fileReq, err := NewRequest("GET", "file:///"+dstPath, nil); err == nil {
			response, err := proxyRoundTripper.Round2.RoundTrip(fileReq)
			if err == nil {
				response.Header.Set("Content-Disposition", "attachment; filename="+dstPath)
			} else {
				logrus.Errorf("read response from file:%s error:%v", dstPath, err)
			}
			return response, err
		} else {
			return nil, err
		}
	} else {
		logrus.Errorf("download fail:%v", err)
		return nil, err
	}
	return nil, nil
}
