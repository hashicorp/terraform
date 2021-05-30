package addrs

// ForEachAttr is the address of an attribute referencing the current "for_each" object in
// the interpolation scope, addressed using the "each" keyword, ex. "each.key" and "each.value"
type ForEachAttr struct {
	referenceable
	Name string
}

func (f ForEachAttr) String() string {
	return "each." + f.Name
}
