package types_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana/x/gf/types"
)

func TestIsValidBCP47(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want bool
	}{
		{"empty", "", false},
		{"too long", strings.Repeat("a", 18), false},
		{"single-letter primary", "x", true},
		{"two-letter primary lowercase", "en", true},
		{"two-letter primary uppercase", "EN", true},
		{"primary + region", "en-US", true},
		{"primary + script + region", "zh-Hans-CN", true},
		{"primary with digit (invalid for first subtag)", "1en", false},
		{"primary too long", "abcdefghi", false},
		{"empty subtag", "en--US", false},
		{"subtag with symbol", "en-US!", false},
		{"subtag too long", "en-USAAAAAAA", false}, // second subtag is 9 chars (> 8 limit)
		{"alphanum subtag ok", "en-US-001", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			require.Equal(t, c.want, types.IsValidBCP47(c.in), "input: %q", c.in)
		})
	}
}

func TestIsValidURL(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want bool
	}{
		{"empty", "", false},
		{"scheme missing", "example.com/path", false},
		{"host missing", "https://", false},
		{"https full", "https://example.com/path", true},
		{"http full", "http://example.com", true},
		{"with port", "https://example.com:8080/path", true},
		{"with query", "https://example.com/path?q=1", true},
		{"malformed", "ht!tp://bad url", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			require.Equal(t, c.want, types.IsValidURL(c.in), "input: %q", c.in)
		})
	}
}

func TestIsValidDigestSRI(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want bool
	}{
		{"empty", "", false},
		{"no algo prefix", "abcdef==", false},
		{"unsupported algo", "md5-abcdef==", false},
		{"sha256 valid", "sha256-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26", true},
		{"sha384 valid", "sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26", true},
		{"sha512 valid", "sha512-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26", true},
		{"sha384 with bad chars", "sha384-!@#$%^&*()", false},
		{"sha384 empty body", "sha384-", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			require.Equal(t, c.want, types.IsValidDigestSRI(c.in), "input: %q", c.in)
		})
	}
}
