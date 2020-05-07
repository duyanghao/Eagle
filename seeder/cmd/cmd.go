package cmd

import (
	"flag"
	"fmt"
	pb "github.com/duyanghao/eagle/proto/metainfo"
	"github.com/duyanghao/eagle/seeder/bt"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"net"
	"strconv"
	"time"
)

// Flags defines seeder CLI flags.
type Flags struct {
	ConfigFile string
}

// ParseFlags parses origin CLI flags.
func ParseFlags() *Flags {
	var flags Flags
	flag.StringVar(
		&flags.ConfigFile, "config", "", "configuration file path")
	flag.Parse()
	return &flags
}

func Run(flags *Flags) {
	// create config
	log.Infof("Start to load config %s ...", flags.ConfigFile)
	config, err := LoadConfig(flags.ConfigFile)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	log.Infof("Load config %s successfully", flags.ConfigFile)
	// set log level
	if config.daemonCfg.Verbose {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}
	// start seeder bt
	log.Infof("Start seeder bt on port: %s ...", config.seederCfg.Port)
	c := &bt.Config{
		EnableUpload:    true,
		EnableSeeding:   true,
		IncomingPort:    config.seederCfg.Port,
		DownloadTimeout: time.Duration(config.seederCfg.DownloadTimeout),
	}
	// transform cache limit size
	limitSize := config.seederCfg.LimitSize
	switch limitSize[len(limitSize)-1:] {
	case "G":
		c.CacheLimitSize, _ = strconv.ParseInt(limitSize[:len(limitSize)-1], 10, 64)
		c.CacheLimitSize *= 1024 * 1024 * 1024
	case "T":
		c.CacheLimitSize, _ = strconv.ParseInt(limitSize[:len(limitSize)-1], 10, 64)
		c.CacheLimitSize *= 1024 * 1024 * 1024 * 1024
	}
	seeder, err := bt.NewSeeder(config.seederCfg.RootDirectory, config.seederCfg.StorageBackend, config.seederCfg.Origin, config.seederCfg.Trackers, c)
	if err != nil {
		log.Fatal(err)
	}
	err = seeder.Run()
	if err != nil {
		log.Fatal(err)
	}
	log.Infof("Start seeder bt on port: %s successfully", config.seederCfg.Port)

	// start seeder
	log.Infof("Launch seeder on port: %s", config.daemonCfg.Port)
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", config.daemonCfg.Port))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterMetaInfoServer(s, seeder)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
