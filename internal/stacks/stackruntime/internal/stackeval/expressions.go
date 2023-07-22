package stackeval

import (
	"context"
	"fmt"
	"sync"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/lang"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

type EvalPhase rune

//go:generate go run golang.org/x/tools/cmd/stringer -type EvalPhase

const (
	NoPhase       EvalPhase = 0
	ValidatePhase EvalPhase = 'V'
	PlanPhase     EvalPhase = 'P'
	ApplyPhase    EvalPhase = 'A'
)

// Referenceable is implemented by types that are identified by the
// implementations of [stackaddrs.Referenceable], returning the value that
// should be used to resolve a reference to that object in an expression
// elsewhere in the configuration.
type Referenceable interface {
	// ExprReferenceValue returns the value that a reference to this object
	// should resolve to during expression evaluation.
	//
	// This method cannot fail, because it's not the expression evaluator's
	// responsibility to report errors or warnings that might arise while
	// processing the target object. Instead, this method will respond to
	// internal problems by returning a suitable placeholder value, and
	// assume that diagnostics will be returned by another concurrent
	// call path.
	ExprReferenceValue(ctx context.Context, phase EvalPhase) cty.Value
}

// ExpressionScope is implemented by types that can have expressions evaluated
// within them, providing the rules for mapping between references in
// expressions to the underlying objects that will provide their values.
type ExpressionScope interface {
	// ResolveExpressionReference decides what a particular expression reference
	// means in the receiver's evaluation scope and returns the concrete object
	// that the address is referring to.
	ResolveExpressionReference(ctx context.Context, ref stackaddrs.Reference) (Referenceable, tfdiags.Diagnostics)
}

// EvalContextForExpr produces an HCL expression evaluation context for the
// given expression in the given evaluation phase within the given expression
// scope.
//
// [EvalExprAndEvalContext] is a convenient wrapper around this which also does
// the final step of evaluating the expression, returning both the value
// and the evaluation context that was used to build it.
func EvalContextForExpr(ctx context.Context, expr hcl.Expression, phase EvalPhase, scope ExpressionScope) (*hcl.EvalContext, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	traversals := expr.Variables()
	refs := make(map[stackaddrs.Referenceable]Referenceable)
	for _, traversal := range traversals {
		ref, _, moreDiags := stackaddrs.ParseReference(traversal)
		diags = diags.Append(moreDiags)
		if moreDiags.HasErrors() {
			continue
		}
		obj, moreDiags := scope.ResolveExpressionReference(ctx, ref)
		diags = diags.Append(moreDiags)
		if moreDiags.HasErrors() {
			continue
		}
		refs[ref.Target] = obj
	}
	if diags.HasErrors() {
		return nil, diags
	}

	varVals := make(map[string]cty.Value)
	localVals := make(map[string]cty.Value)
	componentVals := make(map[string]cty.Value)
	stackVals := make(map[string]cty.Value)
	// TODO: Also providerVals

	for addr, obj := range refs {
		val := obj.ExprReferenceValue(ctx, phase)
		switch addr := addr.(type) {
		case stackaddrs.InputVariable:
			varVals[addr.Name] = val
		case stackaddrs.LocalValue:
			localVals[addr.Name] = val
		case stackaddrs.Component:
			componentVals[addr.Name] = val
		case stackaddrs.StackCall:
			stackVals[addr.Name] = val
		case stackaddrs.ProviderConfigRef:
			// TODO: Implement
			panic(fmt.Sprintf("don't know how to place %T in expression scope", addr))
		default:
			// The above should cover all possible referenceable address types.
			panic(fmt.Sprintf("don't know how to place %T in expression scope", addr))
		}
	}

	// HACK: The top-level lang package bundles together the problem
	// of resolving variables with the generation of the functions table.
	// We only need the functions table here, so we're going to make a
	// pseudo-scope just to load the functions from.
	// FIXME: Separate these concerns better so that both languages can
	// use the same functions but have entirely separate implementations
	// of what data is in scope.
	fakeScope := &lang.Scope{
		Data:        nil, // not a real scope; can't actually make an evalcontext
		BaseDir:     ".",
		PureOnly:    phase != ApplyPhase,
		ConsoleMode: false,
		// TODO: PlanTimestamp
	}
	hclCtx := &hcl.EvalContext{
		Variables: map[string]cty.Value{
			"var":       cty.ObjectVal(varVals),
			"local":     cty.ObjectVal(localVals),
			"component": cty.ObjectVal(componentVals),
			"stack":     cty.ObjectVal(stackVals),
			// TODO: "provider": cty.ObjectVal(providerVals),
		},
		Functions: fakeScope.Functions(),
	}

	return hclCtx, diags
}

// EvalExprAndEvalContext evaluates the given HCL expression in the given
// expression scope and returns the resulting value, along with the HCL
// evaluation context that was used to produce it.
//
// This compact helper function is intended for the relatively-common case
// where a caller needs to perform some additional validation on the result
// of the expression which might generate additional diagnostics, and so
// the caller will need the HCL evaluation context in order to construct
// a fully-annotated diagnostic object.
func EvalExprAndEvalContext(ctx context.Context, expr hcl.Expression, phase EvalPhase, scope ExpressionScope) (ExprResultValue, tfdiags.Diagnostics) {
	hclCtx, diags := EvalContextForExpr(ctx, expr, phase, scope)
	if hclCtx == nil {
		return ExprResultValue{
			Value:       cty.NilVal,
			Expression:  expr,
			EvalContext: hclCtx,
		}, diags
	}
	val, hclDiags := expr.Value(hclCtx)
	diags = diags.Append(hclDiags)
	if val == cty.NilVal {
		val = cty.DynamicVal // just so the caller can assume the result is always a value
	}
	return ExprResultValue{
		Value:       val,
		Expression:  expr,
		EvalContext: hclCtx,
	}, diags
}

// EvalExpr evaluates the given HCL expression in the given expression scope
// and returns the resulting value.
//
// Sometimes callers also need the [hcl.EvalContext] that the expression was
// evaluated with in order to annotate later diagnostics. In that case,
// use [EvalExprAndEvalContext] instead to obtain both the resulting value
// and the evaluation context that was used to produce it.
func EvalExpr(ctx context.Context, expr hcl.Expression, phase EvalPhase, scope ExpressionScope) (cty.Value, tfdiags.Diagnostics) {
	result, diags := EvalExprAndEvalContext(ctx, expr, phase, scope)
	return result.Value, diags
}

// ExprResult bundles an arbitrary result value with the expression and
// evaluation context it was derived from, allowing the recipient to
// potentially emit additional diagnostics if the result is problematic.
//
// (HCL diagnostics related to expressions should typically carry both
// the expression and evaluation context so that we can describe the
// values that were in scope as part of our user-facing diagnostic messages.)
type ExprResult[T any] struct {
	Value T

	Expression  hcl.Expression
	EvalContext *hcl.EvalContext
}

// ExprResultValue is an alias for the common case of an expression result
// being a [cty.Value].
type ExprResultValue = ExprResult[cty.Value]

// DerivedExprResult propagates the expression evaluation context through to
// a new result that was presumably derived from the original result but
// still, from a user perspective, associated with the original expression.
func DerivedExprResult[From, To any](from ExprResult[From], newResult To) ExprResult[To] {
	return ExprResult[To]{
		Value:       newResult,
		Expression:  from.Expression,
		EvalContext: from.EvalContext,
	}
}

func (r ExprResult[T]) Diagnostic(severity tfdiags.Severity, summary string, detail string) *hcl.Diagnostic {
	return &hcl.Diagnostic{
		Severity:    severity.ToHCL(),
		Summary:     summary,
		Detail:      detail,
		Subject:     r.Expression.Range().Ptr(),
		Expression:  r.Expression,
		EvalContext: r.EvalContext,
	}
}

// perEvalPhase is a helper for segregating multiple results for the same
// conceptual operation into a separate result per evaluation phase.
// This is typically needed for any result that's derived from expression
// evaluation, since the values produced for references are constructed
// differently depending on the phase.
//
// This utility works best for types that have a ready-to-use zero value.
// If callers will need to modify the returned objects, type T should be
// a pointer type or other reference type.
type perEvalPhase[T any] struct {
	mu   sync.Mutex
	vals map[EvalPhase]*T
}

// For returns a pointer to the value belonging to the given evaluation phase,
// automatically allocating a new zero-value T if this is the first call for
// the given phase.
//
// This method is itself safe to call concurrently, but it does not constrain
// access to the returned value, and so interaction with that object may
// require additional care depending on the definition of T.
func (pep *perEvalPhase[T]) For(phase EvalPhase) *T {
	if phase == NoPhase {
		// Asking for the value for no phase at all is a nonsense.
		panic("perEvalPhase.For(NoPhase)")
	}
	pep.mu.Lock()
	if pep.vals == nil {
		pep.vals = make(map[EvalPhase]*T)
	}
	if _, exists := pep.vals[phase]; !exists {
		pep.vals[phase] = new(T)
	}
	ret := pep.vals[phase]
	pep.mu.Unlock()
	return ret
}
