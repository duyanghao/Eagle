package handler

import (
	"net/http"
	"net/http/httputil"
	"net/url"

	log "github.com/sirupsen/logrus"

	. "github.com/duyanghao/eagle/proxy/global"
	"github.com/duyanghao/eagle/proxy/util"
)

func Process(w http.ResponseWriter, r *http.Request) {

	if r.URL.Host == "" {
		r.URL.Host = r.Host
		if r.URL.Host == "" {
			r.URL.Host = r.Header.Get("Host")
		}
		if r.URL.Host == "" {
			log.Errorf("url host is empty")
		}
	}
	r.Host = r.URL.Host
	r.Header.Set("Host", r.Host)
	if r.URL.Scheme == "" {
		if G_UseHttps {
			r.URL.Scheme = "https"
		} else {
			r.URL.Scheme = "http"
		}

	}
	log.Debugf("pre access:%s", r.URL.String())

	targetUrl := new(url.URL)
	*targetUrl = *r.URL
	targetUrl.Path = ""
	targetUrl.RawQuery = ""

	hostIp := util.ExtractHost(r.URL.Host)
	switch hostIp {
	case "127.0.0.1", "localhost":
		if len(G_CommandLine.Registry) > 0 {
			targetUrl.Host = G_RegDomain
			targetUrl.Scheme = G_RegProto
		} else {
			log.Warnf("registry not config but url host is %s", hostIp)
		}

	}

	log.Debugf("post access:%s", targetUrl.String())

	reverseProxy := httputil.NewSingleHostReverseProxy(targetUrl)

	reverseProxy.Transport = proxyRoundTripper

	reverseProxy.ServeHTTP(w, r)
}
