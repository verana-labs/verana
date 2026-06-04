// Package validation provides shared, compile-once validators for the data
// formats used across Verana modules (DID and Subresource Integrity digests).
//
// Each module previously carried its own copy of these validators with subtly
// different regexes, so the same string could be accepted by one module and
// rejected by another. This package is the single source of truth: the regexes
// are compiled once at init and reused on every call.
package validation

import "regexp"

// didRe enforces W3C DID Core syntax: `did:` <method-name> `:` <method-specific-id>,
// followed by an optional DID-URL fragment (`#...`) and/or query (`?...`).
//   - method-name is lowercase letters/digits only (per the spec).
//   - method-specific-id allows alphanumerics, `.` `_` `-` `:` `/` and `%`
//     (percent-encoding, e.g. did:webvh).
var didRe = regexp.MustCompile(`^did:[a-z0-9]+:[A-Za-z0-9._:/%\-]+(#[^\s]*)?(\?[^\s]*)?$`)

// IsValidDID reports whether s is a syntactically valid DID (or DID URL).
func IsValidDID(s string) bool { return didRe.MatchString(s) }

// digestSRIRe enforces W3C Subresource Integrity digest syntax: one of the
// sha256/sha384/sha512 algorithms, a hyphen, then a base64 body with up to two
// `=` padding characters.
var digestSRIRe = regexp.MustCompile(`^(sha256|sha384|sha512)-[A-Za-z0-9+/]+={0,2}$`)

// IsValidDigestSRI reports whether s is a valid SRI digest string,
// e.g. "sha256-<base64>". An empty string is not valid; callers that treat the
// digest as optional must guard the empty case themselves.
func IsValidDigestSRI(s string) bool { return digestSRIRe.MatchString(s) }
