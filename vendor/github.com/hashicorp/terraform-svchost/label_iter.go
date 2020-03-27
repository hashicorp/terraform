package svchost

import (
	"strings"
)

// A labelIter allows iterating over domain name labels.
//
// This type is copied from golang.org/x/net/idna, where it is used
// to segment hostnames into their separate labels for analysis. We use
// it for the same purpose here, in ForComparison.
type labelIter struct {
	orig     string
	slice    []string
	curStart int
	curEnd   int
	i        int
}

func (l *labelIter) reset() {
	l.curStart = 0
	l.curEnd = 0
	l.i = 0
}

func (l *labelIter) done() bool {
	return l.curStart >= len(l.orig)
}

func (l *labelIter) result() string {
	if l.slice != nil {
		return strings.Join(l.slice, ".")
	}
	return l.orig
}

func (l *labelIter) label() string {
	if l.slice != nil {
		return l.slice[l.i]
	}
	p := strings.IndexByte(l.orig[l.curStart:], '.')
	l.curEnd = l.curStart + p
	if p == -1 {
		l.curEnd = len(l.orig)
	}
	return l.orig[l.curStart:l.curEnd]
}

// next sets the value to the next label. It skips the last label if it is empty.
func (l *labelIter) next() {
	l.i++
	if l.slice != nil {
		if l.i >= len(l.slice) || l.i == len(l.slice)-1 && l.slice[l.i] == "" {
			l.curStart = len(l.orig)
		}
	} else {
		l.curStart = l.curEnd + 1
		if l.curStart == len(l.orig)-1 && l.orig[l.curStart] == '.' {
			l.curStart = len(l.orig)
		}
	}
}

func (l *labelIter) set(s string) {
	if l.slice == nil {
		l.slice = strings.Split(l.orig, ".")
	}
	l.slice[l.i] = s
}
