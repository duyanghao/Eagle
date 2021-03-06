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
package constants

const (
	DefaultRateLimitBurst      = 4 * 1024 * 1024   // default 4Mb
	DefaultUploadRateLimit     = 100 * 1024 * 1024 // 100Mb/s
	DefaultDownloadRateLimit   = 100 * 1024 * 1024 // 100Mb/s
	DefaultMetaInfoPieceLength = 4 * 1024 * 1024   // default 4Mb
)
