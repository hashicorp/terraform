package applying

import (
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/plans"
)

// Hooks is an interface type that a caller can implement to receive
// notifications of ongoing progress during an apply operation.
//
// Implementers of this interface should embed NoOpHooks in order to
// automatically obtain no-op implementations of all of the current hook
// methods and any new hook methods added in the future.
//
// The methods of this type are notifications, and so the implementations
// of these methods should not attempt to influence the apply process and
// specifically MUST NOT mutate any values they are passed by pointer, even
// if the Go type system does not prevent it.
type Hooks interface {
	// BeginResourceInstanceAction and EndResourceInstanceAction delimit
	// a single action against a particular resource instance.
	//
	// Both are passed the address of the instance, the action type, and the
	// old and new values (both of which might be null in the cty sense).
	// EndResourceInstanceAction additionally receives the outcome of the
	// action.
	BeginResourceInstanceAction(addr addrs.AbsResourceInstance, action plans.Action, old, new cty.Value)
	EndResourceInstanceAction(addr addrs.AbsResourceInstance, action plans.Action, old, new cty.Value, outcome Outcome)
}

// NoOpHooks is an implementation of Hooks where the hook methods do nothing.
// Embed this in your implementation of Hooks in order to avoid being broken
// by future extensions to interface Hooks.
type NoOpHooks struct{}

var _ Hooks = NoOpHooks{} // must implement Hooks

func (h NoOpHooks) BeginResourceInstanceAction(addr addrs.AbsResourceInstance, action plans.Action, old, new cty.Value) {
}

func (h NoOpHooks) EndResourceInstanceAction(addr addrs.AbsResourceInstance, action plans.Action, old, new cty.Value, outcome Outcome) {
}

// defaultHooks is used as a placeholder Hooks when one isn't provided to
// function Apply, just so that downstream code does not need to repeatedly
// check whether the hooks object is nil.
var defaultHooks = NoOpHooks{}

// Outcome represents the outcome of an action in notifications to
// implementations of interface Hooks.
type Outcome int

const (
	// Failure indicates that the action did not completely succeed. Some
	// of the action's side-effects may be visible, however.
	Failure Outcome = 0

	// Success indicates that the action was completely successful.
	Success Outcome = 1

	// Cancelled indicates that the action was cancelled, which means that
	// it's uncertain whether all of its side-effects were applied or not.
	Cancelled Outcome = 2
)
