package validation_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana/util/validation"
)

func TestIsValidDID(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want bool
	}{
		{"simple", "did:example:123", true},
		{"web with colons", "did:web:example.com:user:alice", true},
		{"key with dot dash underscore", "did:key:zABCdef-_.", true},
		{"path slash", "did:web:example.com/abc", true},
		{"percent encoded", "did:webvh:example.com:q%20x", true},
		{"fragment", "did:example:123#key-1", true},
		{"query", "did:example:123?versionId=1", true},
		{"empty", "", false},
		{"no scheme", "not-a-did", false},
		{"missing method", "did::missing-method", false},
		{"uppercase method rejected", "did:Example:upper", false},
		{"empty method-specific-id", "did:example:", false},
		{"whitespace", "did:example:ab cd", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			require.Equal(t, c.want, validation.IsValidDID(c.in), "input: %q", c.in)
		})
	}
}

func TestIsValidDigestSRI(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want bool
	}{
		{"sha256 with padding", "sha256-abc+/=", true},
		{"sha384 double padding", "sha384-Zm9vYg==", true},
		{"sha512 single padding", "sha512-aGVsbG8=", true},
		{"sha256 no padding", "sha256-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y", true},
		{"empty", "", false},
		{"no algo prefix", "abcdef==", false},
		{"unsupported algo", "md5-deadbeef", false},
		{"wrong separator", "sha256:abc", false},
		{"bad chars", "sha256-!@#", false},
		{"empty body", "sha384-", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			require.Equal(t, c.want, validation.IsValidDigestSRI(c.in), "input: %q", c.in)
		})
	}
}
