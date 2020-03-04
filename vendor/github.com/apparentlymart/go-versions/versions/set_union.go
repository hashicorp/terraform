package versions

import (
	"bytes"
	"fmt"
)

type setUnion []setI

func (s setUnion) Has(v Version) bool {
	for _, ss := range s {
		if ss.Has(v) {
			return true
		}
	}
	return false
}

func (s setUnion) AllRequested() Set {
	// Since a union includes everything from its members, it includes all
	// of the requested versions from its members too.
	if len(s) == 0 {
		return None
	}
	si := make(setUnion, 0, len(s))
	for _, ss := range s {
		ar := ss.AllRequested()
		if ar == None {
			continue
		}
		si = append(si, ar.setI)
	}
	if len(si) == 1 {
		return Set{setI: si[0]}
	}
	return Set{setI: si}
}

func (s setUnion) GoString() string {
	var buf bytes.Buffer
	fmt.Fprint(&buf, "versions.Union(")
	for i, ss := range s {
		if i == 0 {
			fmt.Fprint(&buf, ss.GoString())
		} else {
			fmt.Fprintf(&buf, ", %#v", ss)
		}
	}
	fmt.Fprint(&buf, ")")
	return buf.String()
}

// Union creates a new set that contains all of the given versions.
//
// The result is finite only if the receiver and all of the other given sets
// are finite.
func Union(sets ...Set) Set {
	if len(sets) == 0 {
		return None
	}

	r := make(setUnion, 0, len(sets))
	for _, set := range sets {
		if set == None {
			continue
		}
		if su, ok := set.setI.(setUnion); ok {
			r = append(r, su...)
		} else {
			r = append(r, set.setI)
		}
	}
	if len(r) == 1 {
		return Set{setI: r[0]}
	}
	return Set{setI: r}
}

// Union returns a new set that contains all of the versions from the
// receiver and all of the versions from each of the other given sets.
//
// The result is finite only if the receiver and all of the other given sets
// are finite.
func (s Set) Union(others ...Set) Set {
	r := make(setUnion, 1, len(others)+1)
	r[0] = s.setI
	for _, ss := range others {
		if ss == None {
			continue
		}
		if su, ok := ss.setI.(setUnion); ok {
			r = append(r, su...)
		} else {
			r = append(r, ss.setI)
		}
	}
	if len(r) == 1 {
		return Set{setI: r[0]}
	}
	return Set{setI: r}
}

var _ setFinite = setUnion{}

func (s setUnion) isFinite() bool {
	// union is finite only if all of its members are finite
	for _, ss := range s {
		if !isFinite(ss) {
			return false
		}
	}
	return true
}

func (s setUnion) listVersions() List {
	var ret List
	for _, ss := range s {
		ret = append(ret, ss.(setFinite).listVersions()...)
	}
	return ret
}
