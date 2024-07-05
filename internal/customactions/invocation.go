package customactions

import (
	"sync/atomic"

	"github.com/hashicorp/hcl/v2"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Invocation represents a single invocation of a custom action against
// a specific resource instance.
//
// An invocation is typically created from a custom action call, which is
// in turn a rule for how to call a custom action. Each call has zero or more
// invocations for each plan, depending on which action triggers are active
// during that plan.
//
// Action invocations are always against resource instances. The Terraform
// language offers the abstraction of a module defining its own custom
// action types that are implemented in terms of provider-defined action
// types, but we represent those as nested [InvocationSeq] objects rather than
// as leaf [Invocation] objects.
type Invocation struct {
	// ScopeHandle is the handle of the invocation scope where the
	// invocation's arguments should be evaluated and where the
	// results should be written.
	ScopeHandle InvocationScopeHandle

	// ResultAddr is the address of the "step" that the result of this
	// invocation should be attributed to, in the invocation scope
	// identified by [Invocation.ScopeHandle].
	ResultAddr addrs.CustomActionStep

	// Receiver is the address of the resource instance object that the custom
	// action is being invoked on. The action invocation is implemented
	// as a change to this object.
	Receiver addrs.AbsResourceInstanceObject

	// ProviderConfigAddr is the address of the provider configuration that
	// Receiver is being managed by, and thus whose provider instance should
	// handle the invocation.
	ProviderConfigAddr addrs.AbsProviderConfig

	// ActionType is the type of action to invoke. This must be one of
	// the action types declared by the provider as valid for the
	// resource type specified in Receiver.
	ActionType ActionType

	// Arguments are the HCL expressions from which the action's invocation
	// arguments should be determined. The expected argument names and type
	// constraints depend on the provider type and action type, but no
	// validation or decoding is done yet during the construction of an
	// [Invocation].
	Arguments map[string]hcl.Expression

	// CallDeclRange is the source range of the configuration block that
	// declared the call that this invocation belongs to.
	CallDeclRange tfdiags.SourceRange
}

// InvocationSeq is a sequence of invocations that must be executed sequentially
// in the specified order.
//
// Each InvocationSeq typically corresponds to something in the configuration
// that represents a trigger for action invocations. For example, a resource
// might be configured to run a sequence of actions whenever it's being
// planned for creation, in which case all of those actions would appear
// together in a single InvocationSeq.
//
// Invocations that are not grouped together into a single sequence have no
// fixed invocation order relative to one another, although the expressions
// used to populate the arguments for a given invocation might cause further
// ordering constraints that must be handled outside the scope of this
// package.
//
// A "module-defined custom action" is treated as a nested [InvocationSeq],
// since it has its own separate namespace of steps .
type InvocationSeq struct {
	Ops []InvocationOp
}

// EnterInvocationScope is an [InvocationOp] that handles the transition
// into a sequence of steps belonging to a module-defined custom action,
// which must be evaluated in a separate scope so that the nested steps
// are properly encapsulated behind the custom action's defined input
// variables and output values.
type EnterInvocationScope struct {
	// CalledScopeHandle is the handle of the scope that this operation
	// will initialize.
	CalledScopeHandle InvocationScopeHandle

	// CallerScopeHandle is a handle for the prior scope that our
	// new scope is being called from, which is the scope that
	// the expressions in [EnterInvocationScope.InputExprs] must
	// be resolved from.
	CallerScopeHandle InvocationScopeHandle

	// InputExprs are the expressions that should be used to populate
	// the new scope's local input variables, after evaluation in
	// the scope whose handle is given in
	// [EnterEvaluationScope.CallerScopeHandle].
	InputExprs map[addrs.InputVariable]hcl.Expression
}

// ExitInvocationScope is an [InvocationOp] that transitions back out of
// a subsequence of operations belonging to a module-defined custom action,
// aggregating the nested scope's results into a single step in the caller's
// scope.
type ExitInvocationScope struct {
	// CalledScopeHandle is the handle of the scope that we're exiting,
	// which is where the result expressions should be evaluated.
	//
	// This scope is no longer needed once [ExitInvocationScope] has
	// finished gathering results from it, and so it can be safely
	// discarded to free any associated execution-tracking data.
	CalledScopeHandle InvocationScopeHandle

	// CallerScopeHandle is the handle of the scope that we're returning
	// to, which is where the aggregated operation result should be recorded
	// so that later operations in the same scope can refer to it.
	CallerScopeHandle InvocationScopeHandle

	// ResultAddr is the address of the "step" that the result of the
	// nested invocation sequence should be attributed to, in the invocation
	// scope identified by [ExitInvocationScope.CallerScopeHandle].
	ResultAddr addrs.CustomActionStep

	// ResultExprs are the expressions that should be used to built the
	// aggregate result object derived from all of the operations that
	// were evaluated in the called scope.
	ResultExprs map[addrs.OutputValue]hcl.Expression
}

// InvocationOp is a type set containing the three operation types that
// can appear in an [InvocationSeq]:
//
//   - [Invocation] represents a leaf invocation of a custom action through
//     a call to a provider.
//   - [EnterInvocationScope] handles the transition into a nested scope
//     before dealing with invocations that came from a call to a
//     module-defined custom action.
//   - [ExitInvocationScope] handles the transition back out of a nested
//     scope back into its caller again, aggregating the results from
//     the nested operations into a single step in the calling scope.
//
// Over in the main modules runtime, each InvocationOp becomes a graph node
// that executes the operation. The operations that appear together in
// a single [InvocationSeq] have dependency edges between them to ensure that
// they are visited in declaration order.
type InvocationOp interface {
	invocationOpSigil()

	References() []*addrs.Reference
}

func (*Invocation) invocationOpSigil()           {}
func (*EnterInvocationScope) invocationOpSigil() {}
func (*ExitInvocationScope) invocationOpSigil()  {}

// InvocationScopeHandle is an opaque handle for a specific lexical scope
// for expressions involved in custom action invocations.
//
// Values of this type can be used as map keys for distinguishing
// tracking information in different scopes, which is needed when
// one custom action sequence calls a module-defined custom action
// that is itself internally a custom action sequence, but with its
// own namespace of inputs, outputs, and steps that's separate from
// the caller's.
//
// The zero value of this type is invalid. Only code within this
// package can create valid InvocationScopeHandle values.
type InvocationScopeHandle struct {
	id uint64
}

// We use this to make sure that each call to newInvocationScopeHandle
// returns a value that is distinct from any previous call at least
// as long as we don't overflow a uint64, which should be infeasible
// because we should run out of memory before we run out of integers
// due to each scope having at least one tracking object associated
// with it.
var latestInvocationScopeID uint64 = 0

func newInvocationScopeHandle() InvocationScopeHandle {
	// NOTE: We intentionally skip allocating zero here to make sure
	// that no result from this function is ever equal to the zero
	// value of InvocationScopeHandle.
	id := atomic.AddUint64(&latestInvocationScopeID, 1)
	return InvocationScopeHandle{
		id: id,
	}
}
