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
package routes

import (
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/duyanghao/eagle/proxy/transport"
	log "github.com/sirupsen/logrus"
)

func Process(w http.ResponseWriter, r *http.Request) {
	log.Debugf("pre access:%s", r.URL.String())
	targetUrl := new(url.URL)
	*targetUrl = *r.URL
	targetUrl.Path = ""
	targetUrl.RawQuery = ""
	log.Debugf("post access:%s", targetUrl.String())
	reverseProxy := httputil.NewSingleHostReverseProxy(targetUrl)
	reverseProxy.Transport = transport.NewProxyRoundTripper(nil, nil)
	reverseProxy.ServeHTTP(w, r)
}
