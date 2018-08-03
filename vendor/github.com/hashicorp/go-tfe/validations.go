package tfe

import (
	"regexp"
)

// A regular expression used to validate common string ID patterns.
var reStringID = regexp.MustCompile(`^[a-zA-Z0-9\-\._]+$`)

// validString checks if the given input is present and non-empty.
func validString(v *string) bool {
	return v != nil && *v != ""
}

// validStringID checks if the given string pointer is non-nil and
// contains a typical string identifier.
func validStringID(v *string) bool {
	return v != nil && reStringID.MatchString(*v)
}
