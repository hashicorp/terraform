package addrs

type Boundary struct {
	referenceable
	Name string
}

func (b Boundary) String() string {
	return "boundary." + b.Name
}

func (b Boundary) UniqueKey() UniqueKey {
	return b
}

func (b Boundary) uniqueKeySigil() {}
