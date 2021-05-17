package addrs

// TerraformAttr is the address of an attribute of the "terraform" object in
// the interpolation scope, like "terraform.workspace".
type TerraformAttr struct {
	referenceable
	Name string
}

func (ta TerraformAttr) String() string {
	return "terraform." + ta.Name
}
