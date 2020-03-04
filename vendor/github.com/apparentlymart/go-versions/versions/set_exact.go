package versions

import (
	"bytes"
	"fmt"
)

type setExact map[Version]struct{}

func (s setExact) Has(v Version) bool {
	_, has := s[v]
	return has
}

func (s setExact) AllRequested() Set {
	// We just return the receiver verbatim here, because everything in it
	// is explicitly requested.
	return Set{setI: s}
}

func (s setExact) GoString() string {
	if len(s) == 0 {
		// Degenerate case; caller should use None instead
		return "versions.Set{setExact{}}"
	}

	if len(s) == 1 {
		var first Version
		for v := range s {
			first = v
			break
		}
		return fmt.Sprintf("versions.Only(%#v)", first)
	}

	var buf bytes.Buffer
	fmt.Fprint(&buf, "versions.Selection(")
	versions := s.listVersions()
	versions.Sort()
	for i, version := range versions {
		if i == 0 {
			fmt.Fprint(&buf, version.GoString())
		} else {
			fmt.Fprintf(&buf, ", %#v", version)
		}
	}
	fmt.Fprint(&buf, ")")
	return buf.String()
}

// Only returns a version set containing only the given version.
//
// This function is guaranteed to produce a finite set.
func Only(v Version) Set {
	return Set{
		setI: setExact{v: struct{}{}},
	}
}

// Selection returns a version set containing only the versions given
// as arguments.
//
// This function is guaranteed to produce a finite set.
func Selection(vs ...Version) Set {
	if len(vs) == 0 {
		return None
	}
	ret := make(setExact)
	for _, v := range vs {
		ret[v] = struct{}{}
	}
	return Set{setI: ret}
}

// Exactly returns true if and only if the receiving set is finite and
// contains only a single version that is the same as the version given.
func (s Set) Exactly(v Version) bool {
	if !s.IsFinite() {
		return false
	}
	l := s.List()
	if len(l) != 1 {
		return false
	}
	return v.Same(l[0])
}

var _ setFinite = setExact(nil)

func (s setExact) isFinite() bool {
	return true
}

func (s setExact) listVersions() List {
	if len(s) == 0 {
		return nil
	}
	ret := make(List, 0, len(s))
	for v := range s {
		ret = append(ret, v)
	}
	return ret
}
