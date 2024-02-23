// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"context"
	"fmt"
	"sync"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"

	"github.com/hashicorp/terraform/internal/lang"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackconfig"
	"github.com/hashicorp/terraform/internal/stacks/stackconfig/stackconfigtypes"
	"github.com/hashicorp/terraform/internal/stacks/stackconfig/typeexpr"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type EvalPhase rune

//go:generate go run golang.org/x/tools/cmd/stringer -type EvalPhase

const (
	NoPhase       EvalPhase = 0
	ValidatePhase EvalPhase = 'V'
	PlanPhase     EvalPhase = 'P'
	ApplyPhase    EvalPhase = 'A'

	// InspectPhase is a special phase that is used only to inspect the
	// current dynamic situation, without any intention of changing it.
	// This mode allows evaluation against some existing state (possibly
	// empty) but cannot plan to make changes nor apply previously-created
	// plans.
	InspectPhase EvalPhase = 'I'
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
	return evalContextForTraversals(ctx, expr.Variables(), phase, scope)
}

// EvalContextForBody produces an HCL expression context for decoding the
// given [hcl.Body] into a value using the given [hcldec.Spec].
func EvalContextForBody(ctx context.Context, body hcl.Body, spec hcldec.Spec, phase EvalPhase, scope ExpressionScope) (*hcl.EvalContext, tfdiags.Diagnostics) {
	if body == nil {
		panic("EvalContextForBody with nil body")
	}
	if spec == nil {
		panic("EvalContextForBody with nil spec")
	}
	return evalContextForTraversals(ctx, hcldec.Variables(body, spec), phase, scope)
}

func evalContextForTraversals(ctx context.Context, traversals []hcl.Traversal, phase EvalPhase, scope ExpressionScope) (*hcl.EvalContext, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
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
	providerVals := make(map[string]map[string]cty.Value)
	eachVals := make(map[string]cty.Value)
	countVals := make(map[string]cty.Value)
	var selfVal cty.Value
	var testOnlyGlobals map[string]cty.Value // allocated only when needed (see below)

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
			if _, exists := providerVals[addr.ProviderLocalName]; !exists {
				providerVals[addr.ProviderLocalName] = make(map[string]cty.Value)
			}
			providerVals[addr.ProviderLocalName][addr.Name] = val
		case stackaddrs.ContextualRef:
			switch addr {
			case stackaddrs.EachKey:
				eachVals["key"] = val
			case stackaddrs.EachValue:
				eachVals["value"] = val
			case stackaddrs.CountIndex:
				countVals["index"] = val
			case stackaddrs.Self:
				selfVal = val
			default:
				// The above should be exhaustive for all values of this enumeration
				panic(fmt.Sprintf("unsupported ContextualRef %#v", addr))
			}
		case stackaddrs.TestOnlyGlobal:
			// These are available only to some select unit tests in this
			// package, and are not exposed as a real language feature to
			// end-users.
			if testOnlyGlobals == nil {
				testOnlyGlobals = make(map[string]cty.Value)
			}
			testOnlyGlobals[addr.Name] = val
		default:
			// The above should cover all possible referenceable address types.
			panic(fmt.Sprintf("don't know how to place %T in expression scope", addr))
		}
	}

	providerValVals := make(map[string]cty.Value, len(providerVals))
	for k, v := range providerVals {
		providerValVals[k] = cty.ObjectVal(v)
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
			"provider":  cty.ObjectVal(providerValVals),
		},
		Functions: fakeScope.Functions(),
	}
	if len(eachVals) != 0 {
		hclCtx.Variables["each"] = cty.ObjectVal(eachVals)
	}
	if len(countVals) != 0 {
		hclCtx.Variables["count"] = cty.ObjectVal(countVals)
	}
	if selfVal != cty.NilVal {
		hclCtx.Variables["self"] = selfVal
	}
	if testOnlyGlobals != nil {
		hclCtx.Variables["_test_only_global"] = cty.ObjectVal(testOnlyGlobals)
	}

	return hclCtx, diags
}

func EvalComponentInputVariables(ctx context.Context, wantTy cty.Type, defs *typeexpr.Defaults, decl *stackconfig.Component, phase EvalPhase, scope ExpressionScope) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	v := cty.EmptyObjectVal
	expr := decl.Inputs
	rng := decl.DeclRange
	var hclCtx *hcl.EvalContext
	if expr != nil {
		result, moreDiags := EvalExprAndEvalContext(ctx, expr, phase, scope)
		diags = diags.Append(moreDiags)
		if moreDiags.HasErrors() {
			return cty.DynamicVal, diags
		}
		expr = result.Expression
		hclCtx = result.EvalContext
		v = result.Value
		rng = tfdiags.SourceRangeFromHCL(result.Expression.Range())
	}

	if defs != nil {
		v = defs.Apply(v)
	}
	v, err := convert.Convert(v, wantTy)
	if err != nil {
		// A conversion failure here could either be caused by an author-provided
		// expression that's invalid or by the author omitting the argument
		// altogether when there's at least one required attribute, so we'll
		// return slightly different messages in each case.
		if expr != nil {
			diags = diags.Append(&hcl.Diagnostic{
				Severity:    hcl.DiagError,
				Summary:     "Invalid inputs for component",
				Detail:      fmt.Sprintf("Invalid input variable definition object: %s.", tfdiags.FormatError(err)),
				Subject:     rng.ToHCL().Ptr(),
				Expression:  expr,
				EvalContext: hclCtx,
			})
		} else {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Missing required inputs for component",
				Detail:   fmt.Sprintf("Must provide \"inputs\" argument to define the component's input variables: %s.", tfdiags.FormatError(err)),
				Subject:  rng.ToHCL().Ptr(),
			})
		}
		return cty.DynamicVal, diags
	}

	for _, path := range stackconfigtypes.ProviderInstancePathsInValue(v) {
		err := path.NewErrorf("cannot send provider configuration reference to Terraform module input variable")
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid inputs for component",
			Detail: fmt.Sprintf(
				"Invalid input variable definition object: %s.\n\nUse the separate \"providers\" argument to specify the provider configurations to use for this component's root module.",
				tfdiags.FormatError(err),
			),
			Subject:     rng.ToHCL().Ptr(),
			Expression:  expr,
			EvalContext: hclCtx,
		})
	}

	return v, diags
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

// EvalBody evaluates the expressions in the given body using hcldec with
// the given schema, returning the resulting value.
func EvalBody(ctx context.Context, body hcl.Body, spec hcldec.Spec, phase EvalPhase, scope ExpressionScope) (cty.Value, tfdiags.Diagnostics) {
	hclCtx, diags := EvalContextForBody(ctx, body, spec, phase, scope)
	if hclCtx == nil {
		return cty.NilVal, diags
	}
	val, hclDiags := hcldec.Decode(body, spec, hclCtx)
	diags = diags.Append(hclDiags)
	if val == cty.NilVal {
		val = cty.DynamicVal // just so the caller can assume the result is always a value
	}
	return val, diags
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

// Each calls the given reporting callback for all of the values the
// receiver is currently tracking.
//
// Each blocks calls to the For method throughout its execution, so callback
// functions must not interact with the receiver to avoid a deadlock.
func (pep *perEvalPhase[T]) Each(report func(EvalPhase, *T)) {
	pep.mu.Lock()
	for phase, val := range pep.vals {
		report(phase, val)
	}
	pep.mu.Unlock()
}

// JustValue is a special implementation of [Referenceable] used in special
// situations where an [ExpressionScope] needs to just return a specific
// value directly, rather athn indirect through some other referencable object
// for dynamic value resolution.
type JustValue struct {
	v cty.Value
}

var _ Referenceable = JustValue{}

// ExprReferenceValue implements Referenceable.
func (jv JustValue) ExprReferenceValue(ctx context.Context, phase EvalPhase) cty.Value {
	return jv.v
}
