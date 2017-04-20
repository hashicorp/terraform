package utils

import "regexp"

// some general string function helpers
var (
	reRemoveJSON = regexp.MustCompile("(.*)({.*}).*")
	reGetJSON    = regexp.MustCompile("(.*)({.*}).*")
)

// StringRemoveJSON - remove a json string from regular strings
func StringRemoveJSON(s string) string {
	a := reRemoveJSON.FindStringSubmatch(s)
	if len(a) > 2 {
		return StringRemoveJSON(a[1]) // keep trying to remove json till there is no more left
	}
	return s
}

// StringGetJSON - just get the JSON from the string
//                 should only find the first json
func StringGetJSON(s string) string {
	a := reGetJSON.FindStringSubmatch(s)
	if len(a) > 2 {
		return a[2]
	}
	return ""
}
