package addrs

// LocalValue is the address of a local value.
type LocalValue struct {
	referenceable
	Name string
}
