package moduletest

// A Suite is a set of tests run together as a single Terraform configuration.
//
// Deprecated: Will transition to ScenarioResult instead in future.
type Suite struct {
	Name       string
	Components map[string]*Component
}
