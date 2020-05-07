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
package transport

import (
	"crypto/tls"
	"net"
	. "net/http"
	"regexp"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/duyanghao/eagle/eagleclient"
)

type ProxyRoundTripper struct {
	Round     *Transport
	Round2    RoundTripper
	P2PClient *eagleclient.BtEngine
	Rules     []string
}

var proxyRoundTripper *ProxyRoundTripper
var once sync.Once

func NewProxyRoundTripper(eagleClient *eagleclient.BtEngine, rules []string) *ProxyRoundTripper {
	once.Do(func() {
		proxyRoundTripper = &ProxyRoundTripper{
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
			Round2:    NewFileTransport(Dir("/")),
			P2PClient: eagleClient,
			Rules:     rules,
		}
	})
	return proxyRoundTripper
}

func (prt *ProxyRoundTripper) matchRules(location string) bool {
	for _, rule := range prt.Rules {
		compiledRule, err := regexp.Compile(rule)
		if err != nil {
			log.Errorf("Proxy rule: %s format is invalid: %v", rule, err)
			return false
		}
		if compiledRule.MatchString(location) {
			return true
		}
		log.Debugf("Location: %s doesn't match rule: %s", location, rule)
	}
	return false
}

func (prt *ProxyRoundTripper) needUseP2PClient(req *Request, location string) bool {
	var compiler = regexp.MustCompile("^.+/blobs/sha256.*$")
	if req.Method == MethodGet {
		if !compiler.MatchString(req.URL.Path) {
			return false
		}
		if location != "" {
			return prt.matchRules(location)
		}
		return true
	}
	return false
}

//only process first redirect at present
//fix resource release
func (prt *ProxyRoundTripper) RoundTrip(req *Request) (*Response, error) {
	urlString := req.URL.String()
	if prt.needUseP2PClient(req, urlString) {
		log.Debugf("try to get blob: %s through p2p based image distribution system ...", urlString)
		if res, err := prt.download(req, urlString); err == nil {
			return res, err
		}
		log.Errorf("failed to get blob: %s from p2p based image distribution system, let's switch to original request ...", urlString)
	}

	req.Host = req.URL.Host
	req.Header.Set("Host", req.Host)
	res, err := prt.Round.RoundTrip(req)
	return res, err
}

func (prt *ProxyRoundTripper) download(req *Request, urlString string) (*Response, error) {
	//use P2PClient to download
	if dstPath, err := prt.P2PClient.DownloadLayer(req, urlString); err == nil {
		// defer os.Remove(dstPath)
		if fileReq, err := NewRequest("GET", "file:///"+dstPath, nil); err == nil {
			response, err := prt.Round2.RoundTrip(fileReq)
			if err == nil {
				response.Header.Set("Content-Disposition", "attachment; filename="+dstPath)
			} else {
				log.Errorf("read response from file: %s error: %v", dstPath, err)
			}
			return response, err
		} else {
			return nil, err
		}
	} else {
		log.Errorf("download fail: %v", err)
		return nil, err
	}
	return nil, nil
}
