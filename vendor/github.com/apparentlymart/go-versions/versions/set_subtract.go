package versions

import "fmt"

type setSubtract struct {
	from setI
	sub  setI
}

func (s setSubtract) Has(v Version) bool {
	return s.from.Has(v) && !s.sub.Has(v)
}

func (s setSubtract) AllRequested() Set {
	// Our set requests anything that is requested by "from", unless it'd
	// be excluded by "sub". Notice that the whole of "sub" is used, rather
	// than just the requested parts, because requesting is a positive
	// action only.
	return Set{setI: s.from}.AllRequested().Subtract(Set{setI: s.sub})
}

func (s setSubtract) GoString() string {
	return fmt.Sprintf("(%#v).Subtract(%#v)", s.from, s.sub)
}

// Subtract returns a new set that has all of the versions from the receiver
// except for any versions in the other given set.
//
// If the receiver is finite then the returned set is also finite.
func (s Set) Subtract(other Set) Set {
	if other == None || s == None {
		return s
	}
	if other == All {
		return None
	}
	return Set{
		setI: setSubtract{
			from: s.setI,
			sub:  other.setI,
		},
	}
}

var _ setFinite = setSubtract{}

func (s setSubtract) isFinite() bool {
	// subtract is finite if its "from" is finite
	return isFinite(s.from)
}

func (s setSubtract) listVersions() List {
	ret := s.from.(setFinite).listVersions()
	ret = ret.Filter(Set{setI: s.sub})
	return ret
}
