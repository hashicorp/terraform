// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/collections"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/lang"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/promising"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackconfig"
	"github.com/hashicorp/terraform/internal/stacks/stackconfig/typeexpr"
	"github.com/hashicorp/terraform/internal/stacks/stackplan"
	"github.com/hashicorp/terraform/internal/stacks/stackstate"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Stack represents an instance of a [StackConfig] after it's had its
// repetition arguments (if any) evaluated to determine how many instances
// it has.
type Stack struct {
	addr stackaddrs.StackInstance

	main *Main

	// The remaining fields memoize other objects we might create in response
	// to method calls. Must lock "mu" before interacting with them.
	mu sync.Mutex
	// childStacks is a cache of all of the child stacks that have been
	// requested, but this could include non-existant child stacks requested
	// through ChildStackUnchecked, so should not be used directly by
	// anything that needs to return only child stacks that actually exist;
	// use ChildStackChecked if you need to be sure it's actually configured.
	childStacks    map[stackaddrs.StackInstanceStep]*Stack
	inputVariables map[stackaddrs.InputVariable]*InputVariable
	localValues    map[stackaddrs.LocalValue]*LocalValue
	stackCalls     map[stackaddrs.StackCall]*StackCall
	outputValues   map[stackaddrs.OutputValue]*OutputValue
	components     map[stackaddrs.Component]*Component
	providers      map[stackaddrs.ProviderConfigRef]*Provider
}

var (
	_ ExpressionScope = (*Stack)(nil)
	_ Plannable       = (*Stack)(nil)
)

func newStack(main *Main, addr stackaddrs.StackInstance) *Stack {
	return &Stack{
		addr: addr,
		main: main,
	}
}

func (s *Stack) Addr() stackaddrs.StackInstance {
	return s.addr
}

func (s *Stack) IsRoot() bool {
	return s.addr.IsRoot()
}

// ParentStack returns the object representing this stack's caller, or nil
// if this is already the root stack.
func (s *Stack) ParentStack(ctx context.Context) *Stack {
	if s.IsRoot() {
		return nil
	}
	parentAddr := s.Addr().Parent()
	// Unchecked because if our own address is valid then the parent
	// address must also be valid.
	return s.main.StackUnchecked(ctx, parentAddr)
}

// ChildStackUnchecked returns an object representing a child of this stack, or
// nil if the "name" part of the step doesn't correspond to a declared
// embedded stack call.
//
// This method does not check whether the "key" part of the given step matches
// one of the stack call's declared instance keys, so it should be used only
// with known-good instance keys or the resulting object will fail
// unpredictably. If you aren't sure whether the key you have is correct
// then consider [Stack.ChildStackChecked], but note that it's more expensive
// because it must block for the "for_each" expression to be fully resolved.
func (s *Stack) ChildStackUnchecked(ctx context.Context, addr stackaddrs.StackInstanceStep) *Stack {
	calls := s.EmbeddedStackCalls(ctx)
	callAddr := stackaddrs.StackCall{Name: addr.Name}
	if _, exists := calls[callAddr]; !exists {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.childStacks[addr]; !exists {
		childAddr := s.Addr().Child(addr.Name, addr.Key)
		if s.childStacks == nil {
			s.childStacks = make(map[stackaddrs.StackInstanceStep]*Stack)
		}
		s.childStacks[addr] = newStack(s.main, childAddr)
	}

	return s.childStacks[addr]
}

// ChildStackChecked returns an object representing a child of this stack,
// or nil if the step doesn't correspond to a declared instance of the
// corresponding embedded stack call.
//
// This is a more correct but also more expensive variant of
// [Stack.ChildStackUnchecked]. To perform the check it must force the relevant
// child stack call to evaluate its for_each expression, if any, which might
// in turn block on the resolution of other objects. The need to evaluate
// for_each also means that callers must designate which evaluation phase
// they intend to work in, since the set of instances might be known in one
// phase but not another.
//
// If the stack call's for_each expression isn't yet ready to resolve such that
// we cannot know which child instances are declared then ChildStackChecked
// will optimistically assume that the child exists, but operations on that
// optimistic result are likely to return unknown values or other indeterminate
// results themselves.
func (s *Stack) ChildStackChecked(ctx context.Context, addr stackaddrs.StackInstanceStep, phase EvalPhase) *Stack {
	candidate := s.ChildStackUnchecked(ctx, addr)
	if candidate == nil {
		// Don't even need to check the instance key, then.
		return nil
	}

	// If the "unchecked" function succeeded then we know the embedded stack
	// call is present but we don't know if it has the given instance key.
	calls := s.EmbeddedStackCalls(ctx)
	callAddr := stackaddrs.StackCall{Name: addr.Name}
	call := calls[callAddr]

	instances, unknown := call.Instances(ctx, phase)
	if unknown {
		return candidate
	}

	if instances == nil {
		return nil
	}
	if _, exists := instances[addr.Key]; !exists {
		return nil
	}
	return candidate
}

// StackConfig returns the [StackConfig] corresponding to this Stack, which
// represents the static configuration as opposed to the dynamic instance
// of that configuration.
//
// For embedded stacks called using for_each, multiple dynamic Stack instances
// can potentially share the same [StackConfig]. The root stack config always
// has exactly one instance.
func (s *Stack) StackConfig(ctx context.Context) *StackConfig {
	configAddr := s.addr.ConfigAddr()
	return s.main.StackConfig(ctx, configAddr)
}

// ConfigDeclarations returns a pointer to the [stackconfig.Declarations]
// object describing the configured declarations from this stack's
// configuration files.
func (s *Stack) ConfigDeclarations(ctx context.Context) *stackconfig.Declarations {
	// The declarations really belong to the static StackConfig, since
	// all instances of a particular stack configuration share the same
	// source code.ResolveExpressionReference
	return s.StackConfig(ctx).ConfigDeclarations(ctx)
}

// InputVariables returns a map of all of the input variables declared within
// this stack's configuration.
func (s *Stack) InputVariables(ctx context.Context) map[stackaddrs.InputVariable]*InputVariable {
	s.mu.Lock()
	defer s.mu.Unlock()

	// We intentionally save a non-nil map below even if it's empty so that
	// we can unambiguously recognize whether we've loaded this or not.
	if s.inputVariables != nil {
		return s.inputVariables
	}

	decls := s.ConfigDeclarations(ctx)
	ret := make(map[stackaddrs.InputVariable]*InputVariable, len(decls.InputVariables))
	for _, c := range decls.InputVariables {
		absAddr := stackaddrs.AbsInputVariable{
			Stack: s.Addr(),
			Item:  stackaddrs.InputVariable{Name: c.Name},
		}
		ret[absAddr.Item] = newInputVariable(s.main, absAddr)
	}
	s.inputVariables = ret
	return ret
}

func (s *Stack) InputVariable(ctx context.Context, addr stackaddrs.InputVariable) *InputVariable {
	return s.InputVariables(ctx)[addr]
}

// LocalValues returns a map of all of the input variables declared within
// this stack's configuration.
func (s *Stack) LocalValues(ctx context.Context) map[stackaddrs.LocalValue]*LocalValue {
	s.mu.Lock()
	defer s.mu.Unlock()

	// We intentionally save a non-nil map below even if it's empty so that
	// we can unambiguously recognize whether we've loaded this or not.
	if s.localValues != nil {
		return s.localValues
	}

	decls := s.ConfigDeclarations(ctx)
	ret := make(map[stackaddrs.LocalValue]*LocalValue, len(decls.LocalValues))
	for _, c := range decls.LocalValues {
		absAddr := stackaddrs.AbsLocalValue{
			Stack: s.Addr(),
			Item:  stackaddrs.LocalValue{Name: c.Name},
		}
		ret[absAddr.Item] = newLocalValue(s.main, absAddr)
	}
	s.localValues = ret
	return ret
}

// LocalValue returns the [LocalValue] specified by address
func (s *Stack) LocalValue(ctx context.Context, addr stackaddrs.LocalValue) *LocalValue {
	return s.LocalValues(ctx)[addr]
}

// InputsType returns an object type that the object representing the caller's
// values for this stack's input variables must conform to.
func (s *Stack) InputsType(ctx context.Context) (cty.Type, *typeexpr.Defaults) {
	vars := s.InputVariables(ctx)
	atys := make(map[string]cty.Type, len(vars))
	defs := &typeexpr.Defaults{
		DefaultValues: make(map[string]cty.Value),
		Children:      map[string]*typeexpr.Defaults{},
	}
	var opts []string
	for vAddr, v := range vars {
		cfg := v.Config(ctx)
		atys[vAddr.Name] = cfg.TypeConstraint()
		if def := cfg.DefaultValue(ctx); def != cty.NilVal {
			defs.DefaultValues[vAddr.Name] = def
			opts = append(opts, vAddr.Name)
		}
		if childDefs := cfg.NestedDefaults(); childDefs != nil {
			defs.Children[vAddr.Name] = childDefs
		}
	}
	retTy := cty.ObjectWithOptionalAttrs(atys, opts)
	defs.Type = retTy
	return retTy, defs
}

// EmbeddedStackCalls returns a map of all of the embedded stack calls declared
// within this stack's configuration.
func (s *Stack) EmbeddedStackCalls(ctx context.Context) map[stackaddrs.StackCall]*StackCall {
	s.mu.Lock()
	defer s.mu.Unlock()

	// We intentionally save a non-nil map below even if it's empty so that
	// we can unambiguously recognize whether we've loaded this or not.
	if s.stackCalls != nil {
		return s.stackCalls
	}

	decls := s.ConfigDeclarations(ctx)
	ret := make(map[stackaddrs.StackCall]*StackCall, len(decls.EmbeddedStacks))
	for _, c := range decls.EmbeddedStacks {
		absAddr := stackaddrs.AbsStackCall{
			Stack: s.Addr(),
			Item:  stackaddrs.StackCall{Name: c.Name},
		}
		ret[absAddr.Item] = newStackCall(s.main, absAddr)
	}
	s.stackCalls = ret
	return ret
}

func (s *Stack) EmbeddedStackCall(ctx context.Context, addr stackaddrs.StackCall) *StackCall {
	return s.EmbeddedStackCalls(ctx)[addr]
}

func (s *Stack) Components(ctx context.Context) map[stackaddrs.Component]*Component {
	s.mu.Lock()
	defer s.mu.Unlock()

	// We intentionally save a non-nil map below even if it's empty so that
	// we can unambiguously recognize whether we've loaded this or not.
	if s.components != nil {
		return s.components
	}

	decls := s.ConfigDeclarations(ctx)
	ret := make(map[stackaddrs.Component]*Component, len(decls.Components))
	for _, c := range decls.Components {
		absAddr := stackaddrs.AbsComponent{
			Stack: s.Addr(),
			Item:  stackaddrs.Component{Name: c.Name},
		}
		ret[absAddr.Item] = newComponent(s.main, absAddr)
	}
	s.components = ret
	return ret
}

func (s *Stack) Component(ctx context.Context, addr stackaddrs.Component) *Component {
	return s.Components(ctx)[addr]
}

func (s *Stack) ProviderByLocalAddr(ctx context.Context, localAddr stackaddrs.ProviderConfigRef) *Provider {
	s.mu.Lock()
	defer s.mu.Unlock()

	if existing, ok := s.providers[localAddr]; ok {
		return existing
	}
	if s.providers == nil {
		s.providers = make(map[stackaddrs.ProviderConfigRef]*Provider)
	}

	decls := s.ConfigDeclarations(ctx)

	sourceAddr, ok := decls.RequiredProviders.ProviderForLocalName(localAddr.ProviderLocalName)
	if !ok {
		return nil
	}
	configAddr := stackaddrs.AbsProviderConfig{
		Stack: s.Addr(),
		Item: stackaddrs.ProviderConfig{
			Provider: sourceAddr,
			Name:     localAddr.Name,
		},
	}

	// FIXME: stackconfig borrows a type from "addrs" rather than the one
	// in "stackaddrs", for no good reason other than implementation order.
	// We should eventually heal this and use stackaddrs.ProviderConfigRef
	// in the stackconfig API too.
	k := addrs.LocalProviderConfig{
		LocalName: localAddr.ProviderLocalName,
		Alias:     localAddr.Name,
	}
	decl, ok := decls.ProviderConfigs[k]
	if !ok {
		return nil
	}

	provider := newProvider(s.main, configAddr, decl)
	s.providers[localAddr] = provider
	return provider
}

func (s *Stack) Provider(ctx context.Context, addr stackaddrs.ProviderConfig) *Provider {
	decls := s.ConfigDeclarations(ctx)

	localName, ok := decls.RequiredProviders.LocalNameForProvider(addr.Provider)
	if !ok {
		return nil
	}
	return s.ProviderByLocalAddr(ctx, stackaddrs.ProviderConfigRef{
		ProviderLocalName: localName,
		Name:              addr.Name,
	})
}

func (s *Stack) Providers(ctx context.Context) map[stackaddrs.ProviderConfigRef]*Provider {
	decls := s.ConfigDeclarations(ctx)
	if len(decls.ProviderConfigs) == 0 {
		return nil
	}
	ret := make(map[stackaddrs.ProviderConfigRef]*Provider, len(decls.ProviderConfigs))
	// package stackconfig is using the addrs package for provider configuration
	// addresses instead of stackaddrs, because it was written before we had
	// stackaddrs, so we need to do some address adaptation for now.
	// FIXME: Rationalize this so that stackconfig uses the stackaddrs types.
	for weirdAddr := range decls.ProviderConfigs {
		addr := stackaddrs.ProviderConfigRef{
			ProviderLocalName: weirdAddr.LocalName,
			Name:              weirdAddr.Alias,
		}
		ret[addr] = s.ProviderByLocalAddr(ctx, addr)
		// FIXME: The above doesn't deal with the case where the provider
		// block refers to an undeclared provider local name. What should
		// we do in that case? Maybe it doesn't matter if package stackconfig
		// validates that during configuration loading anyway.
	}
	return ret
}

// OutputValues returns a map of all of the output values declared within
// this stack's configuration.
func (s *Stack) OutputValues(ctx context.Context) map[stackaddrs.OutputValue]*OutputValue {
	s.mu.Lock()
	defer s.mu.Unlock()

	// We intentionally save a non-nil map below even if it's empty so that
	// we can unambiguously recognize whether we've loaded this or not.
	if s.outputValues != nil {
		return s.outputValues
	}

	decls := s.ConfigDeclarations(ctx)
	ret := make(map[stackaddrs.OutputValue]*OutputValue, len(decls.OutputValues))
	for _, c := range decls.OutputValues {
		absAddr := stackaddrs.AbsOutputValue{
			Stack: s.Addr(),
			Item:  stackaddrs.OutputValue{Name: c.Name},
		}
		ret[absAddr.Item] = newOutputValue(s.main, absAddr)
	}
	s.outputValues = ret
	return ret
}

func (s *Stack) ResultValue(ctx context.Context, phase EvalPhase) cty.Value {
	ovs := s.OutputValues(ctx)
	elems := make(map[string]cty.Value, len(ovs))
	for addr, ov := range ovs {
		elems[addr.Name] = ov.ResultValue(ctx, phase)
	}
	return cty.ObjectVal(elems)
}

// ResolveExpressionReference implements ExpressionScope, providing the
// global scope for evaluation within an already-instanciated stack during the
// plan and apply phases.
func (s *Stack) ResolveExpressionReference(ctx context.Context, ref stackaddrs.Reference) (Referenceable, tfdiags.Diagnostics) {
	return s.resolveExpressionReference(ctx, ref, nil, instances.RepetitionData{})
}

// ExternalFunctions implements ExpressionScope.
func (s *Stack) ExternalFunctions(ctx context.Context) (lang.ExternalFuncs, func(), tfdiags.Diagnostics) {
	return s.main.ProviderFunctions(ctx, s.StackConfig(ctx))
}

// PlanTimestamp implements ExpressionScope, providing the timestamp at which
// the current plan is being run.
func (s *Stack) PlanTimestamp() time.Time {
	return s.main.PlanTimestamp()
}

// resolveExpressionReference is a shared implementation of [ExpressionScope]
// used for this stack's scope and all of the nested scopes of declarations
// in the same stack, since they tend to differ only in what "self" means
// and what each.key, each.value, or count.index are set to (if anything).
func (s *Stack) resolveExpressionReference(
	ctx context.Context,
	ref stackaddrs.Reference,
	selfAddr stackaddrs.Referenceable,
	repetition instances.RepetitionData,
) (Referenceable, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	// "Test-only globals" is a special affordance we have only when running
	// unit tests in this package. The function called in this branch will
	// return an error itself if we're not running in a suitable test situation.
	if addr, ok := ref.Target.(stackaddrs.TestOnlyGlobal); ok {
		return s.main.resolveTestOnlyGlobalReference(ctx, addr, ref.SourceRange)
	}

	// TODO: Most of the below would benefit from "Did you mean..." suggestions
	// when something is missing but there's a similarly-named object nearby.

	// See also a very similar function in stack_config.go. Both are returning
	// similar referenceable objects but the context is different. For example,
	// in this function we return an instanced Component, while in the other
	// function we return a static ComponentConfig.
	//
	// Some of the returned types are the same across both functions, but most
	// are different in terms of static vs dynamic types.
	switch addr := ref.Target.(type) {
	case stackaddrs.InputVariable:
		ret := s.InputVariable(ctx, addr)
		if ret == nil {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Reference to undeclared input variable",
				Detail:   fmt.Sprintf("There is no variable %q block declared in this stack.", addr.Name),
				Subject:  ref.SourceRange.ToHCL().Ptr(),
			})
			return nil, diags
		}
		return ret, diags
	case stackaddrs.LocalValue:
		ret := s.LocalValue(ctx, addr)
		if ret == nil {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Reference to undeclared local value",
				Detail:   fmt.Sprintf("There is no local %q declared in this stack.", addr.Name),
				Subject:  ref.SourceRange.ToHCL().Ptr(),
			})
			return nil, diags
		}
		return ret, diags
	case stackaddrs.Component:
		ret := s.Component(ctx, addr)
		if ret == nil {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Reference to undeclared component",
				Detail:   fmt.Sprintf("There is no component %q block declared in this stack.", addr.Name),
				Subject:  ref.SourceRange.ToHCL().Ptr(),
			})
			return nil, diags
		}
		return ret, diags
	case stackaddrs.StackCall:
		ret := s.EmbeddedStackCall(ctx, addr)
		if ret == nil {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Reference to undeclared embedded stack",
				Detail:   fmt.Sprintf("There is no stack %q block declared in this stack.", addr.Name),
				Subject:  ref.SourceRange.ToHCL().Ptr(),
			})
			return nil, diags
		}
		return ret, diags
	case stackaddrs.ProviderConfigRef:
		ret := s.ProviderByLocalAddr(ctx, addr)
		if ret == nil {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Reference to undeclared provider configuration",
				Detail:   fmt.Sprintf("There is no provider %q %q block declared in this stack.", addr.ProviderLocalName, addr.Name),
				Subject:  ref.SourceRange.ToHCL().Ptr(),
			})
			return nil, diags
		}
		return ret, diags
	case stackaddrs.ContextualRef:
		switch addr {
		case stackaddrs.EachKey:
			if repetition.EachKey == cty.NilVal {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid 'each' reference",
					Detail:   "The special symbol 'each' is not defined in this location. This symbol is valid only inside multi-instance blocks that use the 'for_each' argument.",
					Subject:  ref.SourceRange.ToHCL().Ptr(),
				})
				return nil, diags
			}
			return JustValue{repetition.EachKey}, diags
		case stackaddrs.EachValue:
			if repetition.EachValue == cty.NilVal {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid 'each' reference",
					Detail:   "The special symbol 'each' is not defined in this location. This symbol is valid only inside multi-instance blocks that use the 'for_each' argument.",
					Subject:  ref.SourceRange.ToHCL().Ptr(),
				})
				return nil, diags
			}
			return JustValue{repetition.EachValue}, diags
		case stackaddrs.CountIndex:
			if repetition.CountIndex == cty.NilVal {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid 'count' reference",
					Detail:   "The special symbol 'count' is not defined in this location. This symbol is valid only inside multi-instance blocks that use the 'count' argument.",
					Subject:  ref.SourceRange.ToHCL().Ptr(),
				})
				return nil, diags
			}
			return JustValue{repetition.CountIndex}, diags
		case stackaddrs.Self:
			if selfAddr != nil {
				// We'll just pretend the reference was to whatever "self"
				// is referring to, then.
				ref.Target = selfAddr
				return s.resolveExpressionReference(ctx, ref, nil, repetition)
			} else {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid 'self' reference",
					Detail:   "The special symbol 'self' is not defined in this location.",
					Context:  ref.SourceRange.ToHCL().Ptr(),
				})
				return nil, diags
			}
		case stackaddrs.TerraformApplying:
			return JustValue{cty.BoolVal(s.main.Applying()).Mark(marks.Ephemeral)}, diags
		default:
			// The above should be exhaustive for all defined values of this type.
			panic(fmt.Sprintf("unsupported ContextualRef %#v", addr))
		}

	default:
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid reference",
			Detail:   fmt.Sprintf("The object %s is not in scope at this location.", addr.String()),
			Subject:  ref.SourceRange.ToHCL().Ptr(),
		})
		return nil, diags
	}
}

// PlanChanges implements Plannable for the root stack by emitting a planned
// change for each output value.
//
// It does nothing for a non-root stack, because embedded stacks themselves are
// just inert containers; the plan walk driver must also explore everything
// nested inside the stack and plan those separately.
func (s *Stack) PlanChanges(ctx context.Context) ([]stackplan.PlannedChange, tfdiags.Diagnostics) {
	if !s.IsRoot() {
		// Nothing to do for non-root stacks
		return nil, nil
	}

	// For a root stack we'll return a PlannedChange for each of the output
	// values, so the caller can see how these would change if this plan is
	// applied.
	resultVal := s.ResultValue(ctx, PlanPhase)
	if !resultVal.Type().IsObjectType() || resultVal.IsNull() || !resultVal.IsKnown() {
		// None of these situations should be possible if Stack.ResultValue is
		// correctly implemented.
		panic(fmt.Sprintf("invalid result from Stack.ResultValue: %#v", resultVal))
	}

	var changes []stackplan.PlannedChange
	for it := resultVal.ElementIterator(); it.Next(); {
		k, v := it.Element()
		outputAddr := stackaddrs.OutputValue{Name: k.AsString()}

		// TODO: For now we just assume that all values are being created.
		// Once we have a prior state we should compare with that to
		// produce accurate change actions. Also, once outputs are stored in
		// state, we should update the definition of Applyable for a stack to
		// reflect updates to outputs making a stack "applyable".

		v, markses := v.UnmarkDeepWithPaths()
		sensitivePaths, otherMarkses := marks.PathsWithMark(markses, marks.Sensitive)
		if len(otherMarkses) != 0 {
			// Any other marks should've been dealt with by our caller before
			// getting here, since we only know how to preserve the sensitive
			// marking.
			var diags tfdiags.Diagnostics
			diags = diags.Append(fmt.Errorf(
				"%s%s: unhandled value marks %#v (this is a bug in Terraform)",
				outputAddr,
				tfdiags.FormatCtyPath(otherMarkses[0].Path),
				otherMarkses[0].Marks,
			))
			return nil, diags
		}
		dv, err := plans.NewDynamicValue(v, v.Type())
		if err != nil {
			// Should not be possible since we generated the value internally;
			// suggests that there's a bug elsewhere in this package.
			panic(fmt.Sprintf("%s has unencodable value: %s", outputAddr, err))
		}
		oldDV, err := plans.NewDynamicValue(cty.NullVal(cty.DynamicPseudoType), cty.DynamicPseudoType)
		if err != nil {
			// Should _definitely_ not be possible since the value is written
			// directly above and should always be encodable.
			panic(fmt.Sprintf("unencodable value: %s", err))
		}

		changes = append(changes, &stackplan.PlannedChangeOutputValue{
			Addr:   outputAddr,
			Action: plans.Create,

			OldValue:               oldDV,
			OldValueSensitivePaths: nil,

			NewValue:               dv,
			NewValueSensitivePaths: sensitivePaths,
		})
	}
	return changes, nil
}

func (s *Stack) RequiredComponents(ctx context.Context) collections.Set[stackaddrs.AbsComponent] {
	// The stack itself doesn't refer to anything and so cannot require
	// components. Its _call_ might, but that's handled over in
	// [StackCall.RequiredComponents].
	return collections.NewSet[stackaddrs.AbsComponent]()
}

// CheckApply implements Applyable.
func (s *Stack) CheckApply(ctx context.Context) ([]stackstate.AppliedChange, tfdiags.Diagnostics) {
	// TODO: We should emit an AppliedChange for each output value,
	// reporting its final value.
	return nil, nil
}

func (s *Stack) tracingName() string {
	addr := s.Addr()
	if addr.IsRoot() {
		return "root stack"
	}
	return addr.String()
}

// reportNamedPromises implements namedPromiseReporter.
func (s *Stack) reportNamedPromises(cb func(id promising.PromiseID, name string)) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, child := range s.childStacks {
		child.reportNamedPromises(cb)
	}
	for _, child := range s.inputVariables {
		child.reportNamedPromises(cb)
	}
	for _, child := range s.outputValues {
		child.reportNamedPromises(cb)
	}
	for _, child := range s.stackCalls {
		child.reportNamedPromises(cb)
	}
	for _, child := range s.components {
		child.reportNamedPromises(cb)
	}
	for _, child := range s.providers {
		child.reportNamedPromises(cb)
	}
}
