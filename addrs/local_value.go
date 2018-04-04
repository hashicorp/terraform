package addrs

// LocalValue is the address of a local value.
type LocalValue struct {
	referenceable
	Name string
}

func (v LocalValue) String() string {
	return "local." + v.Name
}
