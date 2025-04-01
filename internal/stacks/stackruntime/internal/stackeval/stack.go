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
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackconfig/typeexpr"
	"github.com/hashicorp/terraform/internal/stacks/stackplan"
	"github.com/hashicorp/terraform/internal/stacks/stackstate"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Stack represents an instance of a [StackConfig] after it's had its
// repetition arguments (if any) evaluated to determine how many instances
// it has.
type Stack struct {
	addr   stackaddrs.StackInstance
	parent *Stack
	config *StackConfig
	main   *Main

	// The remaining fields memoize other objects we might create in response
	// to method calls. Must lock "mu" before interacting with them.
	mu             sync.Mutex
	inputVariables map[stackaddrs.InputVariable]*InputVariable
	localValues    map[stackaddrs.LocalValue]*LocalValue
	stackCalls     map[stackaddrs.StackCall]*StackCall
	outputValues   map[stackaddrs.OutputValue]*OutputValue
	components     map[stackaddrs.Component]*Component
	removed        map[stackaddrs.Component][]*RemovedComponent
	providers      map[stackaddrs.ProviderConfigRef]*Provider
}

var (
	_ ExpressionScope = (*Stack)(nil)
	_ Plannable       = (*Stack)(nil)
	_ Applyable       = (*Stack)(nil)
)

func newStack(main *Main, addr stackaddrs.StackInstance, parent *Stack, config *StackConfig) *Stack {
	return &Stack{
		parent: parent,
		config: config,
		addr:   addr,
		main:   main,
	}
}

// ChildStack returns the child stack at the given address.
func (s *Stack) ChildStack(ctx context.Context, addr stackaddrs.StackInstanceStep, phase EvalPhase) *Stack {
	calls := s.EmbeddedStackCalls()
	callAddr := stackaddrs.StackCall{Name: addr.Name}
	call := calls[callAddr]

	instances, unknown := call.Instances(ctx, phase)
	if unknown {
		return call.UnknownInstance(ctx, phase).Stack(ctx, phase)
	}

	if instance, exists := instances[addr.Key]; exists {
		return instance.Stack(ctx, phase)
	}
	return nil
}

// InputVariables returns a map of all of the input variables declared within
// this stack's configuration.
func (s *Stack) InputVariables() map[stackaddrs.InputVariable]*InputVariable {
	s.mu.Lock()
	defer s.mu.Unlock()

	// We intentionally save a non-nil map below even if it's empty so that
	// we can unambiguously recognize whether we've loaded this or not.
	if s.inputVariables != nil {
		return s.inputVariables
	}

	decls := s.config.config.Stack
	ret := make(map[stackaddrs.InputVariable]*InputVariable, len(decls.InputVariables))
	for _, c := range decls.InputVariables {
		absAddr := stackaddrs.AbsInputVariable{
			Stack: s.addr,
			Item:  stackaddrs.InputVariable{Name: c.Name},
		}
		ret[absAddr.Item] = newInputVariable(s.main, absAddr, s, s.config.InputVariable(absAddr.Item))
	}
	s.inputVariables = ret
	return ret
}

func (s *Stack) InputVariable(addr stackaddrs.InputVariable) *InputVariable {
	return s.InputVariables()[addr]
}

// LocalValues returns a map of all of the input variables declared within
// this stack's configuration.
func (s *Stack) LocalValues() map[stackaddrs.LocalValue]*LocalValue {
	s.mu.Lock()
	defer s.mu.Unlock()

	// We intentionally save a non-nil map below even if it's empty so that
	// we can unambiguously recognize whether we've loaded this or not.
	if s.localValues != nil {
		return s.localValues
	}

	decls := s.config.config.Stack
	ret := make(map[stackaddrs.LocalValue]*LocalValue, len(decls.LocalValues))
	for _, c := range decls.LocalValues {
		absAddr := stackaddrs.AbsLocalValue{
			Stack: s.addr,
			Item:  stackaddrs.LocalValue{Name: c.Name},
		}
		ret[absAddr.Item] = newLocalValue(s.main, absAddr, s, s.config.LocalValue(absAddr.Item))
	}
	s.localValues = ret
	return ret
}

// LocalValue returns the [LocalValue] specified by address
func (s *Stack) LocalValue(addr stackaddrs.LocalValue) *LocalValue {
	return s.LocalValues()[addr]
}

// InputsType returns an object type that the object representing the caller's
// values for this stack's input variables must conform to.
func (s *Stack) InputsType() (cty.Type, *typeexpr.Defaults) {
	vars := s.InputVariables()
	atys := make(map[string]cty.Type, len(vars))
	defs := &typeexpr.Defaults{
		DefaultValues: make(map[string]cty.Value),
		Children:      map[string]*typeexpr.Defaults{},
	}
	var opts []string
	for vAddr, v := range vars {
		cfg := v.config
		atys[vAddr.Name] = cfg.TypeConstraint()
		if def := cfg.DefaultValue(); def != cty.NilVal {
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
func (s *Stack) EmbeddedStackCalls() map[stackaddrs.StackCall]*StackCall {
	s.mu.Lock()
	defer s.mu.Unlock()

	// We intentionally save a non-nil map below even if it's empty so that
	// we can unambiguously recognize whether we've loaded this or not.
	if s.stackCalls != nil {
		return s.stackCalls
	}

	decls := s.config.config.Stack
	ret := make(map[stackaddrs.StackCall]*StackCall, len(decls.EmbeddedStacks))
	for _, c := range decls.EmbeddedStacks {
		absAddr := stackaddrs.AbsStackCall{
			Stack: s.addr,
			Item:  stackaddrs.StackCall{Name: c.Name},
		}
		ret[absAddr.Item] = newStackCall(s.main, absAddr, s, s.config.StackCall(absAddr.Item))
	}
	s.stackCalls = ret
	return ret
}

func (s *Stack) EmbeddedStackCall(addr stackaddrs.StackCall) *StackCall {
	return s.EmbeddedStackCalls()[addr]
}

func (s *Stack) Components() map[stackaddrs.Component]*Component {
	s.mu.Lock()
	defer s.mu.Unlock()

	// We intentionally save a non-nil map below even if it's empty so that
	// we can unambiguously recognize whether we've loaded this or not.
	if s.components != nil {
		return s.components
	}

	decls := s.config.config.Stack
	ret := make(map[stackaddrs.Component]*Component, len(decls.Components))
	for _, c := range decls.Components {
		absAddr := stackaddrs.AbsComponent{
			Stack: s.addr,
			Item:  stackaddrs.Component{Name: c.Name},
		}
		ret[absAddr.Item] = newComponent(s.main, absAddr, s, s.config.Component(absAddr.Item))
	}
	s.components = ret
	return ret
}

func (s *Stack) Component(addr stackaddrs.Component) *Component {
	return s.Components()[addr]
}

func (s *Stack) RemovedComponents() map[stackaddrs.Component][]*RemovedComponent {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.removed != nil {
		return s.removed
	}

	decls := s.config.RemovedComponents()
	ret := make(map[stackaddrs.Component][]*RemovedComponent, len(decls))
	for _, blocks := range decls {
		for _, r := range blocks {
			absAddr := stackaddrs.AbsComponent{
				Stack: s.addr,
				Item:  r.config.FromComponent,
			}
			ret[absAddr.Item] = append(ret[absAddr.Item], newRemovedComponent(s.main, absAddr, s, r))
		}
	}
	s.removed = ret
	return ret
}

func (s *Stack) RemovedComponent(addr stackaddrs.Component) []*RemovedComponent {
	return s.RemovedComponents()[addr]
}

// ApplyableComponents returns the combination of removed blocks and declared
// components for a given component address.
func (s *Stack) ApplyableComponents(addr stackaddrs.Component) (*Component, []*RemovedComponent) {
	return s.Component(addr), s.RemovedComponent(addr)
}

// KnownComponentInstances returns a set of the component instances that belong
// to the given component from the current state or plan.
func (s *Stack) KnownComponentInstances(component stackaddrs.Component, phase EvalPhase) collections.Set[stackaddrs.ComponentInstance] {
	switch phase {
	case PlanPhase:
		return s.main.PlanPrevState().ComponentInstances(stackaddrs.AbsComponent{
			Stack: s.addr,
			Item:  component,
		})
	case ApplyPhase:
		return s.main.PlanBeingApplied().ComponentInstances(stackaddrs.AbsComponent{
			Stack: s.addr,
			Item:  component,
		})
	default:
		// We're not executing with an existing state in the other phases, so
		// we have no known instances.
		return collections.NewSet[stackaddrs.ComponentInstance]()
	}
}

func (s *Stack) ProviderByLocalAddr(localAddr stackaddrs.ProviderConfigRef) *Provider {
	s.mu.Lock()
	defer s.mu.Unlock()

	if existing, ok := s.providers[localAddr]; ok {
		return existing
	}
	if s.providers == nil {
		s.providers = make(map[stackaddrs.ProviderConfigRef]*Provider)
	}

	decls := s.config.config.Stack

	sourceAddr, ok := decls.RequiredProviders.ProviderForLocalName(localAddr.ProviderLocalName)
	if !ok {
		return nil
	}
	configAddr := stackaddrs.AbsProviderConfig{
		Stack: s.addr,
		Item: stackaddrs.ProviderConfig{
			Provider: sourceAddr,
			Name:     localAddr.Name,
		},
	}

	provider := newProvider(s.main, configAddr, s, s.config.ProviderByLocalAddr(localAddr))
	s.providers[localAddr] = provider
	return provider
}

func (s *Stack) Provider(addr stackaddrs.ProviderConfig) *Provider {
	decls := s.config.config.Stack

	localName, ok := decls.RequiredProviders.LocalNameForProvider(addr.Provider)
	if !ok {
		return nil
	}
	return s.ProviderByLocalAddr(stackaddrs.ProviderConfigRef{
		ProviderLocalName: localName,
		Name:              addr.Name,
	})
}

func (s *Stack) Providers() map[stackaddrs.ProviderConfigRef]*Provider {
	decls := s.config.config.Stack
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
		ret[addr] = s.ProviderByLocalAddr(addr)
		// FIXME: The above doesn't deal with the case where the provider
		// block refers to an undeclared provider local name. What should
		// we do in that case? Maybe it doesn't matter if package stackconfig
		// validates that during configuration loading anyway.
	}
	return ret
}

// OutputValues returns a map of all of the output values declared within
// this stack's configuration.
func (s *Stack) OutputValues() map[stackaddrs.OutputValue]*OutputValue {
	s.mu.Lock()
	defer s.mu.Unlock()

	// We intentionally save a non-nil map below even if it's empty so that
	// we can unambiguously recognize whether we've loaded this or not.
	if s.outputValues != nil {
		return s.outputValues
	}

	decls := s.config.config.Stack
	ret := make(map[stackaddrs.OutputValue]*OutputValue, len(decls.OutputValues))
	for _, c := range decls.OutputValues {
		absAddr := stackaddrs.AbsOutputValue{
			Stack: s.addr,
			Item:  stackaddrs.OutputValue{Name: c.Name},
		}
		ret[absAddr.Item] = newOutputValue(s.main, absAddr, s, s.config.OutputValue(absAddr.Item))
	}
	s.outputValues = ret
	return ret
}

func (s *Stack) OutputValue(addr stackaddrs.OutputValue) *OutputValue {
	return s.OutputValues()[addr]
}

func (s *Stack) ResultValue(ctx context.Context, phase EvalPhase) cty.Value {
	ovs := s.OutputValues()
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
	return s.main.ProviderFunctions(ctx, s.config)
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
		return s.main.resolveTestOnlyGlobalReference(addr, ref.SourceRange)
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
		ret := s.InputVariable(addr)
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
		ret := s.LocalValue(addr)
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
		ret := s.Component(addr)
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
		ret := s.EmbeddedStackCall(addr)
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
		ret := s.ProviderByLocalAddr(addr)
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
	var diags tfdiags.Diagnostics

	// We're going to validate that all the removed blocks in this stack resolve
	// to unique instance addresses.
	for _, blocks := range s.RemovedComponents() {
		seen := make(map[addrs.InstanceKey]*RemovedComponentInstance)
		for _, block := range blocks {
			insts, unknown, _ := block.Instances(ctx, PlanPhase)
			if unknown {
				continue
			}

			for _, inst := range insts {
				if existing, exists := seen[inst.from.Item.Key]; exists {
					diags = diags.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Invalid `from` attribute",
						Detail:   fmt.Sprintf("The `from` attribute resolved to resource instance %s, which is already claimed by another removed block at %s.", inst.from, existing.call.config.config.DeclRange.ToHCL()),
						Subject:  inst.call.config.config.DeclRange.ToHCL().Ptr(),
					})
				}
				seen[inst.from.Item.Key] = inst
			}
		}
	}

	if !s.addr.IsRoot() {
		// Nothing more to do for non-root stacks
		return nil, diags
	}

	// We want to check that all of the components we have in state are
	// targeted by something (either a component or a removed block) in
	// the configuration.
	//
	// The root stack analysis is the best place to do this. We must do this
	// during the plan (and not during the analysis) because we may have
	// for-each attributes that need to be expanded before we can determine
	// if a component is targeted.

	var changes []stackplan.PlannedChange
Instance:
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

		component, removeds := stack.ApplyableComponents(inst.Item.Component)
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

		for _, removed := range removeds {
			insts, unknown, _ := removed.Instances(ctx, PlanPhase)
			if unknown {
				// We can't determine if the component is targeted or not. This
				// is okay, as any changes to this component will be deferred
				// anyway and a follow up plan will then detect the missing
				// component if it exists.
				continue Instance
			}

			for _, i := range insts {
				// the instance key for a removed block doesn't always translate
				// directly into the instance key in the address, so we have
				// to check for the correct one.
				if i.from.Item.Key == inst.Item.Key {
					continue Instance
				}
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
		if s.OutputValue(addr) != nil {
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
		if s.InputVariable(addr) != nil {
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
func (s *Stack) CheckApply(_ context.Context) ([]stackstate.AppliedChange, tfdiags.Diagnostics) {
	if !s.addr.IsRoot() {
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
	addr := s.addr
	if addr.IsRoot() {
		return "root stack"
	}
	return addr.String()
}
