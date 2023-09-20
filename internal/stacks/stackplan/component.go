package stackplan

import (
	"time"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/states"
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
	//
	// FIXME: This modelling is incorrect, because it doesn't handle the fact that
	// a resource instance change might actually be for a deposed object
	// rather than the current object.
	ResourceInstancePlanned addrs.Map[addrs.AbsResourceInstance, *plans.ResourceInstanceChangeSrc]

	// TODO: Something for deferred resource instance changes, once we have
	// such a concept.

	// PlanTimestamp is the time Terraform Core recorded as the single "plan
	// timestamp", which is used only for the result of the "plantimestamp"
	// function during apply and must not be used for any other purpose.
	PlanTimestamp time.Time
}

// ForModulesRuntime translates the component instance plan into the form
// expected by the modules runtime, which is what would ultimately be used
// to apply the plan.
//
// The stack component planning model preserves only the most crucial details
// of a component plan produced by the modules runtime, and so the result
// will not exactly match the [plans.Plan] that the component plan was produced
// from, but should be complete enough to successfully apply the plan.
//
// Conversion with this method should always succeed if the given previous
// run state is truly the one that the plan was created from. If this method
// returns an error then that suggests that the recieving plan is inconsistent
// with the given previous run state, which should not happen if the caller
// is using Terraform Core correctly.
func (c *Component) ForModulesRuntime(prevRunState *states.State) (*plans.Plan, error) {
	changes := plans.NewChanges()
	priorState := prevRunState.DeepCopy()
	plan := &plans.Plan{
		Changes:      changes,
		Timestamp:    c.PlanTimestamp,
		PrevRunState: prevRunState,
		PriorState:   priorState,
	}

	sc := changes.SyncWrapper()
	for _, elem := range c.ResourceInstancePlanned.Elems {
		changeSrc := elem.Value
		sc.AppendResourceInstanceChange(changeSrc)
	}

	// FIXME: For ResourceInstanceChangedOutside we actually need to modify
	// priorState, since that will mimick what the modules runtime would've
	// done itself during its own refresh and plan process during the
	// planning phase. But we can't do that here because we'd need providers
	// to help us convert from msgpack to JSON. So we'll just ignore the
	// "changed outside" stuff for now and figure out what to do with this
	// problem later.

	return plan, nil
}
