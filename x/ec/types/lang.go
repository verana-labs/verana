package types

import "regexp"

// IsValidBCP47 validates basic BCP 47 language tags.
// Allows: 2-3 letter primary subtag, optional additional subtags separated by hyphens.
func IsValidBCP47(tag string) bool {
	if len(tag) < 2 || len(tag) > 17 {
		return false
	}
	re := regexp.MustCompile(`^[a-zA-Z]{2,3}(-[a-zA-Z0-9]{2,8})*$`)
	return re.MatchString(tag)
}

func isValidBCP47(tag string) bool { return IsValidBCP47(tag) }
