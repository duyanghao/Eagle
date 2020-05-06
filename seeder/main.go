package main

import (
	"flag"
	pb "github.com/duyanghao/eagle/proto/metainfo"
	"github.com/duyanghao/eagle/seeder/bt"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"net"
	"strconv"
	"strings"
	"time"
)

var (
	argRootDataDir     string
	argOrigin          string
	argTrackers        string
	argPort            string
	argVerbose         bool
	argLimitSize       string
	argDownloadTimeout int
	seeder             *bt.Seeder
	argStorageBackend  string
)

func main() {
	log.Infof("launch seeder on port: %s", argPort)

	// start seeder
	lis, err := net.Listen("tcp", argPort)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterMetaInfoServer(s, seeder)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func init() {
	flag.StringVar(&argPort, "port", ":65005", "The port seeder listens to")
	flag.StringVar(&argRootDataDir, "rootdir", "/data/", "The root directory of seeder")
	flag.StringVar(&argOrigin, "origin", "", "The data origin of seeder")
	flag.StringVar(&argTrackers, "trackers", "", "The tracker list of seeder")
	flag.BoolVar(&argVerbose, "verbose", false, "verbose")
	flag.StringVar(&argLimitSize, "limitsize", "100G", "disk cache limit,format:xxxT/G")
	flag.IntVar(&argDownloadTimeout, "timeout", 30, "seeder download from origin timeout(s)")
	flag.StringVar(&argStorageBackend, "backend", "fs", "seeder storage backend")
	flag.Parse()
	if argVerbose {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}
	trackers := strings.Split(argTrackers, ",")
	c := &bt.Config{
		EnableUpload:    true,
		EnableSeeding:   true,
		IncomingPort:    50017,
		DownloadTimeout: time.Duration(argDownloadTimeout),
	}
	// transform ratelimiter
	switch argLimitSize[len(argLimitSize)-1:] {
	case "G":
		c.CacheLimitSize, _ = strconv.ParseInt(argLimitSize[:len(argLimitSize)-1], 10, 64)
		c.CacheLimitSize *= 1024 * 1024 * 1024
	case "T":
		c.CacheLimitSize, _ = strconv.ParseInt(argLimitSize[:len(argLimitSize)-1], 10, 64)
		c.CacheLimitSize *= 1024 * 1024 * 1024 * 1024
	}
	log.Debugf("cache limit size: %d", c.CacheLimitSize)
	seeder, err := bt.NewSeeder(argRootDataDir, argStorageBackend, argOrigin, trackers, c)
	if err != nil {
		log.Fatal(err)
	}
	err = seeder.Run()
	if err != nil {
		log.Fatal(err)
	}
}
