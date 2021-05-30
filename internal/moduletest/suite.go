package moduletest

// A Suite is a set of tests run together as a single Terraform configuration.
type Suite struct {
	Name       string
	Components map[string]*Component
}
