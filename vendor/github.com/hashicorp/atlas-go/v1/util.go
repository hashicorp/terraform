package atlas

import (
	"fmt"
	"strings"
)

// ParseSlug parses a slug of the format (x/y) into the x and y components. It
// accepts a string of the format "x/y" ("user/name" for example). If an empty
// string is given, an error is returned. If the given string is not a valid
// slug format, an error is returned.
func ParseSlug(slug string) (string, string, error) {
	if slug == "" {
		return "", "", fmt.Errorf("missing slug")
	}

	parts := strings.Split(slug, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("malformed slug %q", slug)
	}
	return parts[0], parts[1], nil
}
