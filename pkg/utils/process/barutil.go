package process

import (
	"io"

	pb "gopkg.in/cheggaaa/pb.v1"
)

func NewProgressBar(total int, w io.Writer) *pb.ProgressBar {
	bar := pb.New(total).SetUnits(pb.U_BYTES)
	bar.Output = w
	bar.SetMaxWidth(80)
	bar.ShowTimeLeft = false
	bar.ShowPercent = false
	return bar
}
