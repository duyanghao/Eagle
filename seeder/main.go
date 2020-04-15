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
	argTrackers    string
	argPort        int
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
	flag.IntVar(&argPort, "port", 50017, "The port seeder listens to")
	flag.StringVar(&argRootDataDir, "rootdir", "/data/", "The root directory of seeder")
	flag.StringVar(&argTrackers, "trackers", "", "The tracker list of seeder")
	trackers := strings.Split(argTrackers, ",")
	seeder = bt.NewSeeder(argRootDataDir, trackers, nil)
	err := seeder.Run()
	if err != nil {
		log.Fatal(err)
	}
	muxconf.InitMux(seeder)
}
