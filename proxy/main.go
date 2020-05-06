package main

import (
	"fmt"
	"net/http"
	"runtime"

	"github.com/sirupsen/logrus"

	. "github.com/duyanghao/eagle/proxy/global"
	"github.com/duyanghao/eagle/proxy/handler"
	_ "github.com/duyanghao/eagle/proxy/initializer"
)

func main() {

	runtime.GOMAXPROCS(4)

	logrus.Infof("start proxy param: %+v", G_CommandLine)

	logrus.Infof("launch proxy on port: %d", G_CommandLine.Port)

	// start p2pClient
	var err error

	if err = handler.Run(); err != nil {
		logrus.Fatal("start p2pClient failure: %v", err)
	}
	logrus.Infof("start p2pClient successfully ...")

	if G_UseHttps {
		err = http.ListenAndServeTLS(fmt.Sprintf(":%d", G_CommandLine.Port), G_CommandLine.CertFile, G_CommandLine.KeyFile, nil)
	} else {
		err = http.ListenAndServe(fmt.Sprintf(":%d", G_CommandLine.Port), nil)

	}

	if err != nil {
		logrus.Fatal(err)
	}
}
