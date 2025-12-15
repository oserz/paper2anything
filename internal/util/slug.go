package util

import (
	"regexp"
	"strings"
)

var rx = regexp.MustCompile(`[^a-z0-9\-]+`)

func Slugify(s string) string {
	x := strings.ToLower(s)
	x = strings.TrimSpace(x)
	x = strings.ReplaceAll(x, " ", "-")
	x = rx.ReplaceAllString(x, "-")
	x = strings.Trim(x, "-")
	if x == "" {
		return "default"
	}
	return x
}
