// Copyright 2020 duyanghao
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
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
