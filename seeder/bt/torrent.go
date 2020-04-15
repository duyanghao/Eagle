package bt

import (
	"time"

	"github.com/anacrolix/torrent"
)

type State int

const (
	Started State = iota + 1
	Dropped
)

func (s State) String() string {
	switch s {
	case Started:
		return "Started"
	case Dropped:
		return "Dropped"
	default:
		return "Unknown"
	}
}

type Torrent struct {
	// invariant
	InfoHash string
	Name     string
	tt       *torrent.Torrent

	State        State
	Loaded       bool
	Seeding      bool
	Size         int64
	Downloaded   int64
	Percent      float32
	DownloadRate float32
	updatedAt    time.Time
}
