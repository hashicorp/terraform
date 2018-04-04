package addrs

// PathAttr is the address of an attribute of the "path" object in
// the interpolation scope, like "path.module".
type PathAttr struct {
	referenceable
	Name string
}

func (pa PathAttr) String() string {
	return "path." + pa.Name
}
