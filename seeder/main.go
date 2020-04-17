package main

import (
	"flag"
	"fmt"
	"github.com/duyanghao/eagle/seeder/bt"
	"github.com/duyanghao/eagle/seeder/muxconf"
	log "github.com/sirupsen/logrus"
	"net/http"
	"strings"
)

var (
	argRootDataDir string
	argOrigin      string
	argTrackers    string
	argPort        int
	argVerbose     bool
	seeder         *bt.Seeder
)

func main() {
	log.Infof("launch seeder on port: %d", argPort)

	// start seeder
	err := http.ListenAndServe(fmt.Sprintf(":%d", argPort), nil)
	if err != nil {
		log.Fatal(err)
	}
}

func init() {
	flag.IntVar(&argPort, "port", 65005, "The port seeder listens to")
	flag.StringVar(&argRootDataDir, "rootdir", "/data/", "The root directory of seeder")
	flag.StringVar(&argOrigin, "origin", "", "The data origin of seeder")
	flag.StringVar(&argTrackers, "trackers", "", "The tracker list of seeder")
	flag.BoolVar(&argVerbose, "verbose", false, "verbose")
	flag.Parse()
	if argVerbose {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}
	trackers := strings.Split(argTrackers, ",")
	seeder = bt.NewSeeder(argRootDataDir, argOrigin, trackers, nil)
	err := seeder.Run()
	if err != nil {
		log.Fatal(err)
	}
	muxconf.InitMux(seeder)
}
