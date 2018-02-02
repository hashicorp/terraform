package configs

import (
	"github.com/hashicorp/hcl2/hcl"
)

// Backend represents a "backend" block inside a "terraform" block in a module
// or file.
type Backend struct {
	Type   string
	Config hcl.Body

	DeclRange hcl.Range
}
