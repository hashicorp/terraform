// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"context"
	"fmt"
	"iter"
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
	addr     stackaddrs.StackInstance
	parent   *Stack
	config   *StackConfig
	main     *Main
	deferred bool
	mode     plans.Mode

	removed *Removed // contains removed logic

	// The remaining fields memoize other objects we might create in response
	// to method calls. Must lock "mu" before interacting with them.
	mu                 sync.Mutex
	inputVariables     map[stackaddrs.InputVariable]*InputVariable
	localValues        map[stackaddrs.LocalValue]*LocalValue
	stackCalls         map[stackaddrs.StackCall]*StackCall
	outputValues       map[stackaddrs.OutputValue]*OutputValue
	components         map[stackaddrs.Component]*Component
	providers          map[stackaddrs.ProviderConfigRef]*Provider
	removedInitialised bool
}

var (
	_ ExpressionScope = (*Stack)(nil)
	_ Plannable       = (*Stack)(nil)
	_ Applyable       = (*Stack)(nil)
)

func newStack(
	main *Main,
	addr stackaddrs.StackInstance,
	parent *Stack,
	config *StackConfig,
	removed *Removed,
	mode plans.Mode,
	deferred bool) *Stack {
	return &Stack{
		parent:   parent,
		config:   config,
		addr:     addr,
		deferred: deferred,
		mode:     mode,
		main:     main,
		removed:  removed,
	}
}

// ChildStack returns the child stack at the given address.
func (s *Stack) ChildStack(ctx context.Context, addr stackaddrs.StackInstanceStep, phase EvalPhase) *Stack {
	callAddr := stackaddrs.StackCall{Name: addr.Name}

	if call := s.EmbeddedStackCalls()[callAddr]; call != nil {
		instances, unknown := call.Instances(ctx, phase)
		if unknown {
			return call.UnknownInstance(ctx, addr.Key, phase).Stack(ctx, phase)
		}

		if instance, exists := instances[addr.Key]; exists {
			return instance.Stack(ctx, phase)
		}
	}

	calls := s.Removed().stackCalls[callAddr]
	for _, call := range calls {
		absolute := append(s.addr, addr)

		instances, unknown := call.InstancesFor(ctx, absolute, phase)
		if unknown {
			return call.UnknownInstance(ctx, absolute, phase).Stack(ctx, phase)
		}
		for _, instance := range instances {
			return instance.Stack(ctx, phase)
		}
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

func (s *Stack) RemovedEmbeddedStackCall(addr stackaddrs.StackCall) []*RemovedStackCall {
	return s.Removed().stackCalls[addr]
}

func (s *Stack) KnownEmbeddedStacks(addr stackaddrs.StackCall, phase EvalPhase) iter.Seq[stackaddrs.StackInstance] {
	switch phase {
	case PlanPhase:
		return s.main.PlanPrevState().StackInstances(stackaddrs.AbsStackCall{
			Stack: s.addr,
			Item:  addr,
		})
	case ApplyPhase:
		return s.main.PlanBeingApplied().StackInstances(stackaddrs.AbsStackCall{
			Stack: s.addr,
			Item:  addr,
		})
	default:
		// We're not executing with an existing state in the other phases, so
		// we have no known instances.
		return func(yield func(stackaddrs.StackInstance) bool) {}
	}
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

func (s *Stack) Removed() *Removed {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.removedInitialised {
		return s.removed
	}

	// otherwise we're going to initialise removed.

	for addr, configs := range s.config.RemovedComponents().All() {
		blocks := make([]*RemovedComponent, 0, len(configs))
		for _, config := range configs {
			blocks = append(blocks, newRemovedComponent(s.main, addr, s, config))
		}

		s.removed.AddComponent(addr, blocks)
	}

	for addr, configs := range s.config.RemovedStackCalls().All() {
		blocks := make([]*RemovedStackCall, 0, len(configs))
		for _, config := range configs {
			blocks = append(blocks, newRemovedStackCall(s.main, addr, s, config))
		}

		s.removed.AddStackCall(addr, blocks)
	}

	s.removedInitialised = true
	return s.removed
}

func (s *Stack) RemovedComponent(addr stackaddrs.Component) []*RemovedComponent {
	return s.Removed().components[addr]
}

// ApplyableComponents returns the combination of removed blocks and declared
// components for a given component address.
func (s *Stack) ApplyableComponents(addr stackaddrs.Component) (*Component, []*RemovedComponent) {
	return s.Component(addr), s.RemovedComponent(addr)
}

// KnownComponentInstances returns a set of the component instances that belong
// to the given component from the current state or plan.
func (s *Stack) KnownComponentInstances(component stackaddrs.Component, phase EvalPhase) iter.Seq[stackaddrs.ComponentInstance] {
	switch phase {
	case PlanPhase:
		return s.main.PlanPrevState().ComponentInstances(stackaddrs.AbsComponent{
			Stack: s.addr,
			Item:  component,
		})
	case ApplyPhase:
		return s.main.PlanBeingApplied().ComponentInstanceAddresses(stackaddrs.AbsComponent{
			Stack: s.addr,
			Item:  component,
		})
	default:
		// We're not executing with an existing state in the other phases, so
		// we have no known instances.
		return func(yield func(stackaddrs.ComponentInstance) bool) {}
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
	for _, blocks := range s.Removed().components {
		seen := make(map[addrs.InstanceKey]*RemovedComponentInstance)
		for _, block := range blocks {
			insts, unknown := block.InstancesFor(ctx, s.addr, PlanPhase)
			if unknown {
				continue
			}

			for _, inst := range insts {
				if existing, exists := seen[inst.from.Item.Key]; exists {
					diags = diags.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Invalid `from` attribute",
						Detail:   fmt.Sprintf("The `from` attribute resolved to component instance %s, which is already claimed by another removed block at %s.", inst.from, existing.call.config.config.DeclRange.ToHCL()),
						Subject:  inst.call.config.config.DeclRange.ToHCL().Ptr(),
					})
					continue
				}
				seen[inst.from.Item.Key] = inst
			}
		}
	}

	for _, blocks := range s.Removed().stackCalls {
		seen := collections.NewMap[stackaddrs.StackInstance, *RemovedStackCallInstance]()
		for _, block := range blocks {
			insts, unknown := block.InstancesFor(ctx, s.addr, PlanPhase)
			if unknown {
				continue
			}

			for _, inst := range insts {
				if existing, exists := seen.GetOk(inst.from); exists {
					diags = diags.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Invalid `from` attribute",
						Detail:   fmt.Sprintf("The `from` attribute resolved to stack instance %s, which is already claimed by another removed block at %s.", inst.from, existing.call.config.config.DeclRange.ToHCL()),
						Subject:  inst.call.config.config.DeclRange.ToHCL().Ptr(),
					})
					continue
				}
				seen.Put(inst.from, inst)
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
	for inst := range s.main.PlanPrevState().AllComponentInstances() {

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

			// Normally, this is a simple error. The user has deleted an entire
			// stack without adding an equivalent removed block for the stack
			// so now the instances in that stack are all unclaimed.
			//
			// However, the user may have tried to write removed blocks that
			// target specific components within a removed stack instead of
			// just targeting the entire stack. This is invalid, for one it is
			// easier for the user if they could just remove the whole stack,
			// and for two it is very difficult for us to reconcile orphaned
			// removed components and removed embedded stacks that could be
			// floating anywhere in the configuration - instead, we'll just
			// not allow this.
			//
			// In this case, we want to change the error message to be more
			// user-friendly than the generic one, so we need to discover if
			// this has happened here, and if so, modify the error message.

			removed, _ := s.validateMissingInstanceAgainstRemovedBlocks(ctx, inst, PlanPhase)
			if removed != nil {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid removed block",
					Detail:   fmt.Sprintf("The component instance %s could not be removed. The linked removed block was not executed because the `from` attribute of the removed block targets a component or embedded stack within an orphaned embedded stack.\n\nIn order to remove an entire stack, update your removed block to target the entire removed stack itself instead of the specific elements within it.", inst.String()),
					Subject:  removed.DeclRange.ToHCL().Ptr(),
				})
				continue
			}

			// If we fall out here, then we found no relevant removed blocks
			// so we can return the generic error message!

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
			insts, unknown := removed.InstancesFor(ctx, inst.Stack, PlanPhase)
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

// validateMissingInstanceAgainstRemovedBlocks returns the removed config most
// applicable to the target address if it exists.
//
// We have an edge case where a user has written a removed block that targets
// a stacks or components within stacks that are not defined anywhere in the
// stack (either in a removed blocks or an embedded stack). We consider this to
// be an error - if you remove an entire stack from the configuration then you
// should write a removed block that targets that stack not several removed
// blocks that target things inside the removed block.
//
// The above edge case is exposed when we check that all component instances
// in state are included in the plan. This function is called with the absolute
// address of the problematic component (the target). The error we would
// normally return would say that the component isn't targeted by any component
// or removed blocks. This is misleading for the discussed edge case, as the
// user may have written a removed block that targets the component specifically
// but it is just not getting executed as it is in a stack that is also not
// in the configuration.
//
// The function aims to discover if a removed block does exist that might target
// this component. Note, that since we can have removed blocks that target
// entire stacks we do check both removed blocks and direct components on the
// assumption that a removed stack might expand to include the target component
// and we want to capture that removed stack specifically.
func (s *Stack) validateMissingInstanceAgainstRemovedBlocks(ctx context.Context, target stackaddrs.AbsComponentInstance, phase EvalPhase) (*stackconfig.Removed, *stackconfig.Component) {
	if len(target.Stack) == 0 {

		// First, we'll handle the simple case. This means we are actually
		// targeting a component that should be in the current stack, so we'll
		// just look to see if there is a removed block that targets this
		// component directly.

		components, ok := s.Removed().components[target.Item.Component]
		if ok {
			for _, component := range components {
				// we have the component, let's check the
				insts, _ := component.InstancesFor(ctx, s.addr, phase)
				if inst, ok := insts[target.Item.Key]; ok {
					return inst.call.config.config, nil
				}
			}
		}

		if component := s.Component(target.Item.Component); component != nil {
			insts, _ := component.Instances(ctx, phase)
			if inst, ok := insts[target.Item.Key]; ok {
				return nil, inst.call.config.config
			}
		}

		return nil, nil
	}

	// more complicated now, we need to look into a child stack

	next := target.Stack[0]
	rest := stackaddrs.AbsComponentInstance{
		Stack: target.Stack[1:],
		Item:  target.Item,
	}

	if child := s.ChildStack(ctx, next, phase); child != nil {
		return child.validateMissingInstanceAgainstRemovedBlocks(ctx, rest, phase)
	}

	// if we get here, then we had no child stack to check against. But, things
	// are not over yet! we also have might have orphaned removed blocks.
	// these are tracked in the Removed() struct directly, so we'll also look
	// into there. this is the actual troublesome case we're checking for so
	// we do expect to actually get here for these checks.

	if child, ok := s.Removed().children[next.Name]; ok {
		return child.validateMissingInstanceAgainstRemovedBlocks(ctx, append(s.addr, next), rest, phase)
	}

	return nil, nil
}
