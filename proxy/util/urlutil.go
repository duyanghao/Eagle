package util

import (
	"strings"
)

func ExtractHost(hostAndPort string) string {
	fields := strings.Split(strings.TrimSpace(hostAndPort), ":")
	return fields[0]
}
