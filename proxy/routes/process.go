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
