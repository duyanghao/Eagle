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
package ratelimiter

import (
	"strconv"
)

func RateConvert(rateLimiter string) int64 {
	result, _ := strconv.ParseInt(rateLimiter[:len(rateLimiter)-1], 10, 64)
	switch rateLimiter[len(rateLimiter)-1:] {
	case "K":
		result *= 1024
	case "M":
		result *= 1024 * 1024
	case "G":
		result *= 1024 * 1024 * 1024
	case "T":
		result *= 1024 * 1024 * 1024 * 1024
	}
	return result
}
