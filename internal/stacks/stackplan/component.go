package stackplan

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/collections"
	"github.com/hashicorp/terraform/internal/plans"
)

// Component is a container for a set of changes that all belong to the same
// component instance as declared in a stack configuration.
//
// Each instance of component essentially maps to one call into the main
// Terraform language runtime to apply all of the described changes together as
// a single operation.
type Component struct {
	// ResourceInstanceChangedOutside describes changes that Terraform has
	// detected were made outside of Terraform since the last run.
	ResourceInstanceChangedOutside collections.Map[addrs.AbsResourceInstance, *plans.ResourceInstanceChange]

	// ResourceInstancePlanned describes the changes that Terraform is proposing
	// to make to try to converge the real system state with the desired state
	// as described by the configuration.
	ResourceInstancePlanned collections.Map[addrs.AbsResourceInstance, *plans.ResourceInstanceChange]

	// TODO: Something for deferred resource instance changes, once we have
	// such a concept.
}
