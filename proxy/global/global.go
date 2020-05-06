package global

import (
	"regexp"
	"sync"

	log "github.com/sirupsen/logrus"
)

type CommandParam struct {
	Urlfilter string

	Version  bool
	Verbose  bool
	Help     bool
	Port     uint
	Registry string //https://xxx.xx.x:port or http://xxx.xx.x:port
	DownRule string

	CertFile                   string
	KeyFile                    string
	P2PClientRootDir           string
	P2PClientSeeders           string
	P2PClientTrackers          string
	P2PClientDownloadRateLimit string
	P2PClientUploadRateLimit   string
	P2PClientCacheLimitSize    string
	P2PClientDownloadTimeout   int
}

var (
	G_UseHttps bool

	G_CommandLine CommandParam

	G_RegProto string

	G_RegDomain string

	G_DFPattern = make(map[string]*regexp.Regexp)

	G_P2PClientSeeders []string

	G_P2PClientTrackers []string

	rwMutex sync.RWMutex
)

func UpdateDFPattern(reg string) {
	if reg == "" {
		return
	}
	rwMutex.Lock()
	defer rwMutex.Unlock()
	if compiledReg, err := regexp.Compile(reg); err == nil {
		G_DFPattern[reg] = compiledReg
	} else {
		log.Warnf("pattern:%s is invalid", reg)
	}
}

func CopyDfPattern() []string {
	rwMutex.RLock()
	defer rwMutex.RUnlock()
	copiedPattern := make([]string, 0, len(G_DFPattern))
	for _, value := range G_DFPattern {
		copiedPattern = append(copiedPattern, value.String())
	}
	return copiedPattern
}

func MatchDfPattern(location string) bool {
	useGetter := false
	rwMutex.RLock()
	defer rwMutex.RUnlock()
	for key, regex := range G_DFPattern {
		if regex.MatchString(location) {
			useGetter = true
			break
		}
		log.Debugf("location:%s not match reg:%s", location, key)
	}
	return useGetter
}
