package ratelimiter

import "regexp"

// ValidateRateLimiter the rate limiter
func ValidateRateLimiter(rateLimiter string) bool {
	if rateLimiter == "" {
		return false
	}
	if isMatch, _ := regexp.MatchString("^[[:digit:]]+[MKGT]$", rateLimiter); !isMatch {
		return false
	}
	return true
}
