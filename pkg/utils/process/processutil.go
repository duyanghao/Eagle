package process

import (
	"fmt"
	"github.com/anacrolix/torrent"
	"io"
	"time"

	pb "gopkg.in/cheggaaa/pb.v1"
)

type ProgressDownload struct {
	id     string
	output io.Writer
	bar    *pb.ProgressBar
}

func (p *ProgressDownload) WaitComplete(t *torrent.Torrent) {
	writeReport := func(f string, a ...interface{}) {
		fmt.Fprintf(p.output, f, a...)
	}

	writeReport("%s: Getting torrent info\n", p.id)
	<-t.GotInfo()
	writeReport("%s: Start bittorent downloading\n", p.id)

	//p.bar.Start()
	for {
		total := t.Info().TotalLength()
		completed := t.BytesCompleted()
		if completed >= total {
			break
		}
		//p.bar.Set(int(completed))
		time.Sleep(10 * time.Millisecond)
	}
	writeReport("\n")
}

func NewProgressDownload(id string, size int, output io.Writer) *ProgressDownload {
	bar := NewProgressBar(size, output)
	return &ProgressDownload{
		id:     id,
		output: output,
		bar:    bar,
	}
}
