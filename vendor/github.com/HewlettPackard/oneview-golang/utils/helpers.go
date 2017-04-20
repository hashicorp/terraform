package utils

import "strings"

// Sanatize ...
func Sanatize(s string) string {
	if strings.LastIndex(s, "/") > 0 {
		s = strings.Trim(s, "/")
	}
	return s
}

// IsEmpty ...
// see http://golang.org/ref/spec#Assignability
func IsEmpty(s string) bool {
	if s == "" || len(strings.TrimSpace(s)) == 0 {
		return true
	}
	return false
}
