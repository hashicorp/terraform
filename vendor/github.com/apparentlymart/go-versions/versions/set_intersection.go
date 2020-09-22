package versions

import (
	"bytes"
	"fmt"
)

type setIntersection []setI

func (s setIntersection) Has(v Version) bool {
	if len(s) == 0 {
		// Weird to have an intersection with no elements, but we'll
		// allow it and return something sensible.
		return false
	}
	for _, ss := range s {
		if !ss.Has(v) {
			return false
		}
	}
	return true
}

func (s setIntersection) AllRequested() Set {
	// The requested set for an intersection is the union of all of its
	// members requested sets intersection the receiver. Therefore we'll
	// borrow the same logic from setUnion's implementation here but
	// then wrap it up in a setIntersection before we return.

	asUnion := setUnion(s)
	ar := asUnion.AllRequested()
	si := make(setIntersection, len(s)+1)
	si[0] = ar.setI
	copy(si[1:], s)
	return Set{setI: si}
}

func (s setIntersection) GoString() string {
	var buf bytes.Buffer
	fmt.Fprint(&buf, "versions.Intersection(")
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

// Intersection creates a new set that contains the versions that all of the
// given sets have in common.
//
// The result is finite if any of the given sets are finite.
func Intersection(sets ...Set) Set {
	if len(sets) == 0 {
		return None
	}

	r := make(setIntersection, 0, len(sets))
	for _, set := range sets {
		if set == All {
			continue
		}
		if set == None {
			return None
		}
		if su, ok := set.setI.(setIntersection); ok {
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

// Intersection returns a new set that contains all of the versions that
// the receiver and the given sets have in common.
//
// The result is a finite set if the receiver or any of the given sets are
// finite.
func (s Set) Intersection(others ...Set) Set {
	r := make(setIntersection, 1, len(others)+1)
	r[0] = s.setI
	for _, ss := range others {
		if ss == All {
			continue
		}
		if ss == None {
			return None
		}
		if su, ok := ss.setI.(setIntersection); ok {
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

var _ setFinite = setIntersection{}

func (s setIntersection) isFinite() bool {
	// intersection is finite if any of its members are, or if it is empty
	if len(s) == 0 {
		return true
	}
	for _, ss := range s {
		if isFinite(ss) {
			return true
		}
	}
	return false
}

func (s setIntersection) listVersions() List {
	var ret List
	for _, ss := range s {
		if isFinite(ss) {
			ret = append(ret, ss.(setFinite).listVersions()...)
		}
	}
	ret.Filter(Set{setI: s})
	return ret
}
