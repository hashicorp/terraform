package addrs

// Set represents a set of addresses of types that implement UniqueKeyer.
type Set map[UniqueKey]UniqueKeyer

func (s Set) Has(addr UniqueKeyer) bool {
	_, exists := s[addr.UniqueKey()]
	return exists
}

func (s Set) Add(addr UniqueKeyer) {
	s[addr.UniqueKey()] = addr
}

func (s Set) Remove(addr UniqueKeyer) {
	delete(s, addr.UniqueKey())
}

func (s Set) Union(other Set) Set {
	ret := make(Set)
	for k, addr := range s {
		ret[k] = addr
	}
	for k, addr := range other {
		ret[k] = addr
	}
	return ret
}

func (s Set) Intersection(other Set) Set {
	ret := make(Set)
	for k, addr := range s {
		if _, exists := other[k]; exists {
			ret[k] = addr
		}
	}
	for k, addr := range other {
		if _, exists := s[k]; exists {
			ret[k] = addr
		}
	}
	return ret
}
