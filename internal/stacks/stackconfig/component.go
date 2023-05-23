package stackconfig

import (
	"github.com/apparentlymart/go-versions/versions/constraints"
	"github.com/hashicorp/go-slug/sourceaddrs"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
)

// Component represents the declaration of a single component within a
// particular [Stack].
//
// Components are the most important object in a stack configuration, just as
// resources are the most important object in a Terraform module: each one
// refers to a Terraform module that describes the infrastructure that the
// component is "made of".
type Component struct {
	Name string

	SourceAddr      sourceaddrs.Source
	AllowedVersions constraints.IntersectionSpec

	ForEach hcl.Expression

	// Inputs is an expression that should produce a value that can convert
	// to an object type derived from the component's input variable
	// declarations, and whose attribute values will then be used to populate
	// those input variables.
	Inputs hcl.Expression

	// ProviderConfigs describes the mapping between the static provider
	// configuration slots declared in the component's root module and the
	// dynamic provider configuration objects in scope in the calling
	// stack configuration.
	//
	// This map deals with the slight schism between the stacks language's
	// treatment of provider configurations as regular values of a special
	// data type vs. the main Terraform language's treatment of provider
	// configurations as something special passed out of band from the
	// input variables. The overall structure and the map keys are fixed
	// statically during decoding, but the final provider configuration objects
	// are determined only at runtime by normal expression evaluation.
	//
	// The keys of this map refer to provider configuration slots inside
	// the module being called, but use the local names defined in the
	// calling stack configuration. The stacks language runtime will
	// translate the caller's local names into the callee's declared provider
	// configurations by using the stack configuration's table of local
	// provider names.
	ProviderConfigs map[addrs.LocalProviderConfig]hcl.Expression
}
