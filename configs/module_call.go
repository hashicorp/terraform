package configs

import (
	"github.com/hashicorp/hcl2/hcl"
)

// ModuleCall represents a "module" block in a module or file.
type ModuleCall struct {
	Source      string
	SourceRange hcl.Range

	Version VersionConstraint

	Count   hcl.Expression
	ForEach hcl.Expression

	DeclRange hcl.Range
}
