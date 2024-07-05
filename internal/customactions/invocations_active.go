package customactions

import (
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/plans"
)

// ActiveInvocations returns an iterator over any invocation sequences that
// should be considered as "active" according to the given configuration,
// sequence of one-off calls, and analysis metadata.
//
// This function is a sort of "compiler" turning custom action calls into
// custom action invocations based on various trigger points declared in the
// configuration.
func ActiveInvocations(config *configs.Config, oneOffs []*OneOffCall, meta InvocationAnalysisMeta) func(yield func(*InvocationSeq) bool) {
	return func(yield func(*InvocationSeq) bool) {
		// TODO: Implement
	}
}

type OneOffCall struct {
	// Receiver is the absolute address of the object that the action is
	// being called on.
	//
	// This should be a module instance address for a module-defined custom
	// action, or a resource instance address for a provider-defined custom
	// action.
	Receiver addrs.Targetable

	// ActionType is the type of custom action to call.
	//
	// If Receiver is a resource instance address then this is assumed to
	// be an action type defined by the provider that this resource instance
	// belongs to, for the resource type of the indicated resource.
	ActionType ActionType

	// Arguments are the values to use to populate the action's arguments
	// during the call.
	Arguments map[string]cty.Value
}

// InvocationAnalysisMeta is an adapter used to obtain some specific kinds of
// information about the context where action invocations are being compiled,
// since [ActiveInvocations] only needs a small subset of the available data
// but without this indirection this package would need to depend directly
// on numerous other packages and contain logic that is more thematically
// relevant to the main modules runtime.
type InvocationAnalysisMeta interface {
	// ResourceInstanceObjectExpectedAction returns the core action that's
	// anticipated for the given resource instance object.
	//
	// This decision is made purely based on the prior state and configuration.
	// For example:
	//
	// - If there's an object declared in the configuration but
	//   not in the prior state then its expected action is [plans.Create].
	// - If an object is in the prior state but not in the configuration
	//   then its expected action is [plans.Delete].
	// - If an object is in both then its expected action is [plans.Update].
	//
	// In practice the modules runtime might select a different action
	// once full evaluation begins. For example, a [plans.Update] result
	// might be replaced by a [plans.NoOp] if the provider determines that
	// the desired state and current state are already matching.
	ResourceInstanceObjectExpectedAction(addr addrs.AbsResourceInstanceObject) plans.Action
}
