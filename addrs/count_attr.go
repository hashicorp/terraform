package addrs

// CountAttr is the address of an attribute of the "count" object in
// the interpolation scope, like "count.index".
type CountAttr struct {
	referenceable
	Name string
}
