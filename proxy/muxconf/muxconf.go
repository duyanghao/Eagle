package muxconf

import (
	"net/http"

	"github.com/duyanghao/eagle/proxy/handler"
)

func InitMux() {
	router := map[string]func(http.ResponseWriter, *http.Request){
		"/":       handler.Process,
		"/debug/": handler.DebugInfo,
	}

	for key, value := range router {
		http.HandleFunc(key, value)
	}
}
