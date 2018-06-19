package plans

import (
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/states"
)

// Changes describes various actions that Terraform will attempt to take if
// the corresponding plan is applied.
//
// A Changes object can be rendered into a visual diff (by the caller, using
// code in another package) for display to the user.
type Changes struct {
	Resources   []*ResourceInstanceChange
	RootOutputs map[string]*OutputChange
}

// ResourceInstanceChange describes a change to a particular resource instance
// object.
type ResourceInstanceChange struct {
	// Addr is the absolute address of the resource instance that the change
	// will apply to.
	Addr addrs.AbsResourceInstance

	// DeposedKey is the identifier for a deposed object associated with the
	// given instance, or states.NotDeposed if this change applies to the
	// current object.
	//
	// A Replace change for a resource with create_before_destroy set will
	// create a new DeposedKey temporarily during replacement. In that case,
	// DeposedKey in the plan is always states.NotDeposed, representing that
	// the current object is being replaced with the deposed.
	DeposedKey states.DeposedKey

	// Change is an embedded description of the change.
	Change
}

// OutputChange describes a change to an output value.
type OutputChange struct {
	// Change is an embedded description of the change.
	//
	// For output value changes, the type constraint for the DynamicValue
	// instances is always cty.DynamicPseudoType.
	Change

	// Sensitive, if true, indicates that either the old or new value in the
	// change is sensitive and so a rendered version of the plan in the UI
	// should elide the actual values while still indicating the action of the
	// change.
	Sensitive bool
}

// Change describes a single change with a given action.
type Change struct {
	// Action defines what kind of change is being made.
	Action Action

	// Interpretation of Before and After depend on Action:
	//
	//     NoOp     Before and After are the same, unchanged value
	//     Create   Before is nil, and After is the expected value after create.
	//     Read     Before is any prior value (nil if no prior), and After is the
	//              value that was or will be read.
	//     Update   Before is the value prior to update, and After is the expected
	//              value after update.
	//     Replace  As with Update.
	//     Delete   Before is the value prior to delete, and After is always nil.
	//
	// Unknown values may appear anywhere within the Before and After values,
	// either as the values themselves or as nested elements within known
	// collections/structures.
	//
	// A plan contains only raw (not yet decoded) values. The caller must use
	// schema information obtained out-of-band to decode dynamic values before
	// they can be used.
	Before, After DynamicValue
}
