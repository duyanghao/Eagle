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
