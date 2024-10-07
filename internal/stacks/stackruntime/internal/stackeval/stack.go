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
	removed        map[stackaddrs.Component]*Removed
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

func (s *Stack) Removeds(ctx context.Context) map[stackaddrs.Component]*Removed {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.removed != nil {
		return s.removed
	}

	decls := s.ConfigDeclarations(ctx)
	ret := make(map[stackaddrs.Component]*Removed, len(decls.Removed))
	for _, r := range decls.Removed {
		absAddr := stackaddrs.AbsComponent{
			Stack: s.Addr(),
			Item:  r.FromComponent,
		}
		ret[absAddr.Item] = newRemoved(s.main, absAddr)
	}
	s.removed = ret
	return ret
}

func (s *Stack) Removed(ctx context.Context, addr stackaddrs.Component) *Removed {
	return s.Removeds(ctx)[addr]
}

// ApplyableComponents returns the combination of removed blocks and declared
// components for a given component address.
func (s *Stack) ApplyableComponents(ctx context.Context, addr stackaddrs.Component) (*Component, *Removed) {
	return s.Component(ctx, addr), s.Removed(ctx, addr)
}

// KnownComponentInstances returns a set of the component instances that belong
// to the given component from the current state or plan.
func (s *Stack) KnownComponentInstances(component stackaddrs.Component, phase EvalPhase) collections.Set[stackaddrs.ComponentInstance] {
	switch phase {
	case PlanPhase:
		return s.main.PlanPrevState().ComponentInstances(stackaddrs.AbsComponent{
			Stack: s.Addr(),
			Item:  component,
		})
	case ApplyPhase:
		return s.main.PlanBeingApplied().ComponentInstances(stackaddrs.AbsComponent{
			Stack: s.Addr(),
			Item:  component,
		})
	default:
		// We're not executing with an existing state in the other phases, so
		// we have no known instances.
		return collections.NewSet[stackaddrs.ComponentInstance]()
	}
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

func (s *Stack) OutputValue(ctx context.Context, addr stackaddrs.OutputValue) *OutputValue {
	return s.OutputValues(ctx)[addr]
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
func (s *Stack) ExternalFunctions(ctx context.Context) (lang.ExternalFuncs, tfdiags.Diagnostics) {
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

	var diags tfdiags.Diagnostics

	// We want to check that all of the components we have in state are
	// targeted by something (either a component or a removed block) in
	// the configuration.
	//
	// The root stack analysis is the best place to do this. We must do this
	// during the plan (and not during the analysis) because we may have
	// for-each attributes that need to be expanded before we can determine
	// if a component is targeted.

	var changes []stackplan.PlannedChange
	for inst := range s.main.PlanPrevState().AllComponentInstances().All() {

		// We track here whether this component instance has any associated
		// resources. If this component is empty, and not referenced in the
		// configuration, then we won't return an error. Instead, we'll just
		// mark this as to-be deleted. There could have been some error
		// marking the state previously, but whatever it is we can just fix
		// this so why bother the user with it.
		empty := s.main.PlanPrevState().ComponentInstanceResourceInstanceObjects(inst).Len() == 0

		stack := s.main.Stack(ctx, inst.Stack, PlanPhase)
		if stack == nil {
			if empty {
				changes = append(changes, &stackplan.PlannedChangeComponentInstanceRemoved{
					Addr: inst,
				})
				continue
			}

			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Unclaimed component instance",
				fmt.Sprintf("The component instance %s is not claimed by any component or removed block in the configuration. Make sure it is instantiated by a component block, or targeted for removal by a removed block.", inst.String()),
			))
			continue
		}

		component, removed := stack.ApplyableComponents(ctx, inst.Item.Component)
		if component != nil {
			insts, unknown := component.Instances(ctx, PlanPhase)
			if unknown {
				// We can't determine if the component is targeted or not. This
				// is okay, as any changes to this component will be deferred
				// anyway and a follow up plan will then detect the missing
				// component if it exists.
				continue
			}

			if _, exists := insts[inst.Item.Key]; exists {
				// This component is targeted by a component block, so we won't
				// add an error.
				continue
			}
		}

		if removed != nil {
			insts, unknown, _ := removed.Instances(ctx, PlanPhase)
			if unknown {
				// We can't determine if the component is targeted or not. This
				// is okay, as any changes to this component will be deferred
				// anyway and a follow up plan will then detect the missing
				// component if it exists.
				continue
			}

			if _, exists := insts[inst.Item.Key]; exists {
				// This component is targeted by a removed block, so we won't
				// add an error.
				continue
			}
		}

		// Otherwise, we have a component that is not targeted by anything in
		// the configuration.

		if empty {
			// It's empty, so we can just remove it.
			changes = append(changes, &stackplan.PlannedChangeComponentInstanceRemoved{
				Addr: inst,
			})
			continue
		}

		// Otherwise, it's an error.
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Unclaimed component instance",
			fmt.Sprintf("The component instance %s is not claimed by any component or removed block in the configuration. Make sure it is instantiated by a component block, or targeted for removal by a removed block.", inst.String()),
		))
	}

	// Finally, we'll look at the input and output values we have in state
	// and any that do not appear in the configuration we'll mark as deleted.

	for addr, value := range s.main.PlanPrevState().RootOutputValues() {
		if s.OutputValue(ctx, addr) != nil {
			// Then this output value is in the configuration, and will be
			// processed independently.
			continue
		}

		// Otherwise, it's been removed from the configuration.
		changes = append(changes, &stackplan.PlannedChangeOutputValue{
			Addr:   addr,
			Action: plans.Delete,
			Before: value,
			After:  cty.NullVal(cty.DynamicPseudoType),
		})
	}

	// Finally, we'll look at the input variables we have in state and delete
	// any that don't appear in the configuration any more.

	for addr, variable := range s.main.PlanPrevState().RootInputVariables() {
		if s.InputVariable(ctx, addr) != nil {
			// Then this input variable is in the configuration, and will
			// be processed independently.
			continue
		}

		// Otherwise, we'll add a delete notification for this root input
		// variable.
		changes = append(changes, &stackplan.PlannedChangeRootInputValue{
			Addr:   addr,
			Action: plans.Delete,
			Before: variable,
			After:  cty.NullVal(cty.DynamicPseudoType),
		})
	}

	return changes, diags
}

// CheckApply implements Applyable.
func (s *Stack) CheckApply(ctx context.Context) ([]stackstate.AppliedChange, tfdiags.Diagnostics) {
	if !s.IsRoot() {
		return nil, nil
	}

	var diags tfdiags.Diagnostics
	var changes []stackstate.AppliedChange

	// We're also just going to quickly emit any cleanup . These remaining
	// values are basically just everything that have been in the configuration
	// in the past but is no longer and so needs to be removed from the state.

	for value := range s.main.PlanBeingApplied().DeletedOutputValues.All() {
		changes = append(changes, &stackstate.AppliedChangeOutputValue{
			Addr:  value,
			Value: cty.NilVal,
		})
	}

	for value := range s.main.PlanBeingApplied().DeletedInputVariables.All() {
		changes = append(changes, &stackstate.AppliedChangeInputVariable{
			Addr:  value,
			Value: cty.NilVal,
		})
	}

	for value := range s.main.PlanBeingApplied().DeletedComponents.All() {
		changes = append(changes, &stackstate.AppliedChangeComponentInstanceRemoved{
			ComponentAddr: stackaddrs.AbsComponent{
				Stack: value.Stack,
				Item:  value.Item.Component,
			},
			ComponentInstanceAddr: value,
		})
	}

	return changes, diags
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
