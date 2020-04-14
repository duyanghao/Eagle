package btclient

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

func (t *Torrent) Update() {
	t.Loaded = t.tt.Info() != nil
	if t.tt.Info() != nil {
		t.Size = t.tt.Length()
		t.Seeding = t.tt.Seeding()

		//cacluate rate
		now := time.Now()
		bytes := t.tt.BytesCompleted()
		t.Percent = percent(bytes, t.Size)
		if !t.updatedAt.IsZero() {
			dt := float32(now.Sub(t.updatedAt))
			db := float32(bytes - t.Downloaded)
			rate := db * (float32(time.Second) / dt)
			if rate >= 0 {
				t.DownloadRate = rate
			}
		}
		t.Downloaded = bytes
		t.updatedAt = now
	} else {
		t.Seeding = false
	}
}

func percent(n, total int64) float32 {
	if total == 0 {
		return float32(0)
	}
	return float32(int(float64(10000)*(float64(n)/float64(total)))) / 100
}
