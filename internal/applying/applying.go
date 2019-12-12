package applying

import (
	"context"

	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/tfdiags"
)

// Apply executes the operations described in the given plan.
//
// This is the main entry point for this package. Any mutable objects
// referenced by the given arguments must not be read or modified by the caller
// until Apply returns, or the resulting behavior is undefined.
func Apply(ctx context.Context, args Arguments) (*states.State, tfdiags.Diagnostics) {
	return apply(ctx, args.prepare())
}

// Arguments is the input to function Apply, gathering all of the arguments
// required to perform an apply operation.
type Arguments struct {
	// Plan is the plan that is to be applied.
	Plan *plans.Plan

	// Config is a representation of the configuration that the plan was
	// created from. If the given configuration does not exactly match what
	// was used to produce the plan then the result is undefined.
	Config *configs.Config

	// PriorState is a snapshot of the state as it was at the instant just
	// before the given plan was created. If the given state does not exactly
	// match what was used to produce the plan then the result is undefined.
	PriorState *states.State

	// Hooks is an optional implementation of Hooks that will receive ongoing
	// notifications of progress during the apply operation. It's intended
	// primarily for giving progress feedback in the application UI.
	Hooks Hooks

	// WorkspaceName is the value to return for terraform.workspace references
	// in the configuration.
	WorkspaceName string

	// ConcurrencyLimit defines the maximum number of operations that can be
	// in progress concurrently. If set to zero, a default of 10 is selected.
	ConcurrencyLimit int

	// Dependencies is a repository of available providers and provisioners
	// to use for actions requested in the plan.
	//
	// The caller must ensure that all of the providers and provisioners
	// required by the planned actions are included.
	Dependencies Dependencies
}

// prepare inserts default values, clones certain structures to so we can
// safely mutate them, and performs some correctness checks on the arguments.
//
// It may panic if any of the arguments are incorrect in a way that signals
// an implementation error in the caller of Apply.
func (args Arguments) prepare() Arguments {
	if args.Plan == nil {
		panic("nil Plan in Apply Arguments")
	}
	if args.PriorState == nil {
		// If this is an initial create, the caller must create and pass a
		// non-nil empty state.
		panic("nil PriorState in Apply arguments")
	}

	// We'll disconnect our state from the one held by the caller by copying
	// it. Although the contract of Apply calls for the caller not to mutate
	// or access what they pass during the Apply operation, unintentional
	// concurrent mutation of state has been a common bug elsewhere and so this
	// is some insurance against such bugs affecting callers of this package.
	args.PriorState = args.PriorState.DeepCopy()

	if args.Hooks == nil {
		args.Hooks = defaultHooks
	}

	if args.ConcurrencyLimit == 0 {
		args.ConcurrencyLimit = 10
	}

	return args
}
