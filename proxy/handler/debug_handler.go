package handler

import (
	"net/http"
	"net/http/pprof"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/duyanghao/eagle/proxy/constant"
)

func DebugInfo(w http.ResponseWriter, req *http.Request) {
	logrus.Debugf("access:%s", req.URL.String())

	if strings.HasPrefix(req.URL.Path, "/debug/pprof") {
		if req.URL.Path == "/debug/pprof/symbol" {
			pprof.Symbol(w, req)
		} else {
			pprof.Index(w, req)
		}
	} else if strings.HasPrefix(req.URL.Path, "/debug/version") {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "text/plain;charset=utf-8")
		w.Write([]byte(constant.VERSION))
	}

}
