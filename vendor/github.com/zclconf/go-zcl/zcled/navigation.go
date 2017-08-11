package zcled

import (
	"github.com/zclconf/go-zcl/zcl"
)

type contextStringer interface {
	ContextString(offset int) string
}

// ContextString returns a string describing the context of the given byte
// offset, if available. An empty string is returned if no such information
// is available, or otherwise the returned string is in a form that depends
// on the language used to write the referenced file.
func ContextString(file *zcl.File, offset int) string {
	if cser, ok := file.Nav.(contextStringer); ok {
		return cser.ContextString(offset)
	}
	return ""
}
