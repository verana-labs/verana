package types

import (
	"net/url"
	"strings"
)

// IsValidBCP47 returns true if s looks like a valid BCP 47 language tag.
// (Mirrors x/tr/types.IsValidBCP47.)
func IsValidBCP47(s string) bool {
	if s == "" {
		return false
	}
	if len(s) > 17 {
		return false
	}
	for i, part := range strings.Split(s, "-") {
		if len(part) < 1 || len(part) > 8 {
			return false
		}
		if i == 0 {
			for _, r := range part {
				if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')) {
					return false
				}
			}
			continue
		}
		for _, r := range part {
			if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')) {
				return false
			}
		}
	}
	return true
}

func IsValidURL(s string) bool {
	u, err := url.Parse(s)
	if err != nil {
		return false
	}
	return u.Scheme != "" && u.Host != ""
}
