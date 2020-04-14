package btclient

import (
	"fmt"
	"io"
	"time"

	"github.com/duyanghao/eagle/pkg/utils"

	pb "gopkg.in/cheggaaa/pb.v1"
)

type ProgressDownload struct {
	id     string
	output io.Writer
	bar    *pb.ProgressBar
}

func (p *ProgressDownload) waitComplete(t *Torrent) {
	writeReport := func(f string, a ...interface{}) {
		fmt.Fprintf(p.output, f, a...)
	}

	writeReport("%s: Getting torrent info\n", p.id)
	<-t.tt.GotInfo()
	writeReport("%s: Start bittorent downloading\n", p.id)

	p.bar.Start()
	for {
		total := t.tt.Info().TotalLength()
		completed := t.tt.BytesCompleted()
		if completed >= total {
			break
		}
		p.bar.Set(int(completed))
		time.Sleep(500 * time.Millisecond)
	}
	writeReport("\n")
}

func NewProgressDownload(id string, size int, output io.Writer) *ProgressDownload {
	bar := utils.NewProgressBar(size, output)
	return &ProgressDownload{
		id:     id,
		output: output,
		bar:    bar,
	}
}
