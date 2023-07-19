package stackplan

import (
	"time"

	"github.com/hashicorp/terraform/internal/addrs"
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
	ResourceInstanceChangedOutside addrs.Map[addrs.AbsResourceInstance, *plans.ResourceInstanceChangeSrc]

	// ResourceInstancePlanned describes the changes that Terraform is proposing
	// to make to try to converge the real system state with the desired state
	// as described by the configuration.
	ResourceInstancePlanned addrs.Map[addrs.AbsResourceInstance, *plans.ResourceInstanceChangeSrc]

	// TODO: Something for deferred resource instance changes, once we have
	// such a concept.

	// PlanTimestamp is the time Terraform Core recorded as the single "plan
	// timestamp", which is used only for the result of the "plantimestamp"
	// function during apply and must not be used for any other purpose.
	PlanTimestamp time.Time
}
