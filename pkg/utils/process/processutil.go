package process

import (
	"fmt"
	"github.com/anacrolix/torrent"
	"io"
	"time"

	"context"
	pb "gopkg.in/cheggaaa/pb.v1"
)

type ProgressDownload struct {
	id     string
	output io.Writer
	bar    *pb.ProgressBar
}

func (p *ProgressDownload) WaitComplete(ctx context.Context, t *torrent.Torrent) {
	writeReport := func(f string, a ...interface{}) {
		fmt.Fprintf(p.output, f, a...)
	}

	writeReport("Waiting bt download %s complete ...\n", p.id)
	writeReport("Getting torrent info %s\n", p.id)
	<-t.GotInfo()
	writeReport("Start bittorent downloading %s\n", p.id)

Loop:
	for {
		select {
		case <-ctx.Done():
			writeReport("Stop bittorent downloading %s\n", p.id)
			break Loop
		default:
			total := t.Info().TotalLength()
			completed := t.BytesCompleted()
			if completed >= total {
				break Loop
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
	writeReport("Download bt %s completed\n", p.id)
}

func NewProgressDownload(id string, size int, output io.Writer) *ProgressDownload {
	bar := NewProgressBar(size, output)
	return &ProgressDownload{
		id:     id,
		output: output,
		bar:    bar,
	}
}
