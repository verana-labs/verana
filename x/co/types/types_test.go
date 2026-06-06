package types_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana/x/co/types"
)

func TestIsValidBCP47(t *testing.T) {
	cases := map[string]bool{
		"":                   false,
		"en":                 true,
		"EN":                 true,
		"en-US":              true,
		"zh-Hant-CN":         true,
		"en-":                false,
		"-en":                false,
		"toolongsubtagxxxxx": false, // first subtag > 8 chars
		"en-12345678":        true,
		"en-123456789":       false,
		"en1":                false, // first subtag must be letters only
		strings.Repeat("a", 18): false,
	}
	for in, want := range cases {
		require.Equal(t, want, types.IsValidBCP47(in), "input=%q", in)
	}
}

func TestIsValidURL(t *testing.T) {
	require.True(t, types.IsValidURL("https://example.com/path"))
	require.True(t, types.IsValidURL("http://x.y"))
	require.False(t, types.IsValidURL(""))
	require.False(t, types.IsValidURL("not a url"))
	require.False(t, types.IsValidURL("/relative/only"))
	require.False(t, types.IsValidURL("https://"))
}

// DID and DigestSRI validation now live in util/validation; see
// util/validation/validation_test.go for their table-driven tests.
