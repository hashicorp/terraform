package stackconfig

import (
	"github.com/apparentlymart/go-versions/versions/constraints"
	"github.com/hashicorp/go-slug/sourceaddrs"
	"github.com/hashicorp/hcl/v2"
)

// EmbeddedStack describes a call to another stack configuration whose
// declarations should be included as part of the overall stack configuration
// tree.
//
// An embedded stack exists only as a child of another stack and doesn't have
// its own independent identity outside of that calling stack.
//
// Terraform Cloud offers a related concept of "linked stacks" where the
// deployment configuration for one stack can refer to the outputs of another,
// while the other stack retains its own independent identity and lifecycle,
// but that concept only makes sense in an environment like Terraform Cloud
// where the stack outputs can be published for external consumption.
type EmbeddedStack struct {
	Name string

	SourceAddr      sourceaddrs.Source
	AllowedVersions constraints.IntersectionSpec

	ForEach hcl.Expression

	// Inputs is an expression that should produce a value that can convert
	// to an object type derived from the child stack's input variable
	// declarations, and whose attribute values will then be used to populate
	// those input variables.
	Inputs hcl.Expression
}
