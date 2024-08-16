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
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/lang"
	"github.com/hashicorp/terraform/internal/promising"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackconfig"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// StackConfig represents a stack as represented in the configuration: either the
// root stack or one of the embedded stacks before it's been expanded into
// individual instances.
//
// After instance expansion we use [StackInstance] to represent each of the
// individual instances.
type StackConfig struct {
	addr stackaddrs.Stack

	config *stackconfig.ConfigNode

	main *Main

	// The remaining fields are where we memoize related objects that we've
	// constructed and returned. Must lock "mu" before interacting with these.
	mu             sync.Mutex
	children       map[stackaddrs.StackStep]*StackConfig
	inputVariables map[stackaddrs.InputVariable]*InputVariableConfig
	localValues    map[stackaddrs.LocalValue]*LocalValueConfig
	outputValues   map[stackaddrs.OutputValue]*OutputValueConfig
	stackCalls     map[stackaddrs.StackCall]*StackCallConfig
	components     map[stackaddrs.Component]*ComponentConfig
	providers      map[stackaddrs.ProviderConfig]*ProviderConfig
}

var (
	_ ExpressionScope      = (*StackConfig)(nil)
	_ namedPromiseReporter = (*StackConfig)(nil)
)

func newStackConfig(main *Main, addr stackaddrs.Stack, config *stackconfig.ConfigNode) *StackConfig {
	return &StackConfig{
		addr:   addr,
		config: config,
		main:   main,

		children:       make(map[stackaddrs.StackStep]*StackConfig, len(config.Children)),
		inputVariables: make(map[stackaddrs.InputVariable]*InputVariableConfig, len(config.Stack.Declarations.InputVariables)),
		localValues:    make(map[stackaddrs.LocalValue]*LocalValueConfig, len(config.Stack.Declarations.LocalValues)),
		outputValues:   make(map[stackaddrs.OutputValue]*OutputValueConfig, len(config.Stack.Declarations.OutputValues)),
		stackCalls:     make(map[stackaddrs.StackCall]*StackCallConfig, len(config.Stack.Declarations.EmbeddedStacks)),
		components:     make(map[stackaddrs.Component]*ComponentConfig, len(config.Stack.Declarations.Components)),
		providers:      make(map[stackaddrs.ProviderConfig]*ProviderConfig, len(config.Stack.Declarations.ProviderConfigs)),
	}
}

func (s *StackConfig) Addr() stackaddrs.Stack {
	return s.addr
}

func (s *StackConfig) IsRoot() bool {
	return s.addr.IsRoot()
}

// ParentAddr returns the address of the containing stack, or panics if called
// on the root stack (since it has no parent).
func (s *StackConfig) ParentAddr() stackaddrs.Stack {
	return s.addr.Parent()
}

// ConfigDeclarations returns a pointer to the [stackconfig.Declarations]
// object describing the configured declarations from this stack config's
// configuration files.
func (s *StackConfig) ConfigDeclarations(ctx context.Context) *stackconfig.Declarations {
	return &s.config.Stack.Declarations
}

// ParentConfig returns the [StackConfig] object representing the configuration
// of the containing stack, or nil if the receiver is the root stack in the
// tree.
func (s *StackConfig) ParentConfig(ctx context.Context) *StackConfig {
	if s.IsRoot() {
		return nil
	}
	// Each StackConfig is only responsible for looking after its direct
	// children, so to navigate upwards we need to start back at the
	// root and work our way through the tree again.
	return s.main.StackConfig(ctx, s.ParentAddr())
}

// ChildConfig returns a [StackConfig] representing the embedded stack matching
// the given address step, or nil if there is no such stack.
func (s *StackConfig) ChildConfig(ctx context.Context, step stackaddrs.StackStep) *StackConfig {
	s.mu.Lock()
	defer s.mu.Unlock()

	ret, ok := s.children[step]
	if !ok {
		childNode, ok := s.config.Children[step.Name]
		if !ok {
			return nil
		}
		childAddr := s.Addr().Child(step.Name)
		s.children[step] = newStackConfig(s.main, childAddr, childNode)
		ret = s.children[step]
	}
	return ret
}

func (s *StackConfig) ChildConfigs(ctx context.Context) map[stackaddrs.StackStep]*StackConfig {
	if len(s.config.Children) == 0 {
		return nil
	}
	ret := make(map[stackaddrs.StackStep]*StackConfig, len(s.config.Children))
	for n := range s.config.Children {
		stepAddr := stackaddrs.StackStep{Name: n}
		ret[stepAddr] = s.ChildConfig(ctx, stepAddr)
	}
	return ret
}

// InputVariables returns a map of the objects representing all of the
// input variables declared inside this stack configuration.
func (s *StackConfig) InputVariables(ctx context.Context) map[stackaddrs.InputVariable]*InputVariableConfig {
	if len(s.config.Stack.InputVariables) == 0 {
		return nil
	}
	ret := make(map[stackaddrs.InputVariable]*InputVariableConfig, len(s.config.Stack.InputVariables))
	for name := range s.config.Stack.InputVariables {
		addr := stackaddrs.InputVariable{Name: name}
		ret[addr] = s.InputVariable(ctx, addr)
	}
	return ret
}

// InputVariable returns an [InputVariableConfig] representing the input
// variable declared within this stack config that matches the given
// address, or nil if there is no such declaration.
func (s *StackConfig) InputVariable(ctx context.Context, addr stackaddrs.InputVariable) *InputVariableConfig {
	s.mu.Lock()
	defer s.mu.Unlock()

	ret, ok := s.inputVariables[addr]
	if !ok {
		cfg, ok := s.config.Stack.InputVariables[addr.Name]
		if !ok {
			return nil
		}
		cfgAddr := stackaddrs.Config(s.Addr(), addr)
		ret = newInputVariableConfig(s.main, cfgAddr, cfg)
		s.inputVariables[addr] = ret
	}
	return ret
}

// InputVariables returns a map of the objects representing all of the
// local values declared inside this stack configuration.
func (s *StackConfig) LocalValues(ctx context.Context) map[stackaddrs.LocalValue]*LocalValueConfig {
	if len(s.config.Stack.LocalValues) == 0 {
		return nil
	}
	ret := make(map[stackaddrs.LocalValue]*LocalValueConfig, len(s.config.Stack.LocalValues))
	for name := range s.config.Stack.LocalValues {
		addr := stackaddrs.LocalValue{Name: name}
		ret[addr] = s.LocalValue(ctx, addr)
	}
	return ret
}

// LocalValue returns an [LocalValueConfig] representing the input
// variable declared within this stack config that matches the given
// address, or nil if there is no such declaration.
func (s *StackConfig) LocalValue(_ context.Context, addr stackaddrs.LocalValue) *LocalValueConfig {
	s.mu.Lock()
	defer s.mu.Unlock()

	ret, ok := s.localValues[addr]
	if !ok {
		cfg, ok := s.config.Stack.LocalValues[addr.Name]
		if !ok {
			return nil
		}
		cfgAddr := stackaddrs.Config(s.Addr(), addr)
		ret = newLocalValueConfig(s.main, cfgAddr, cfg)
		s.localValues[addr] = ret
	}
	return ret
}

// OutputValue returns an [OutputValueConfig] representing the output
// value declared within this stack config that matches the given
// address, or nil if there is no such declaration.
func (s *StackConfig) OutputValue(ctx context.Context, addr stackaddrs.OutputValue) *OutputValueConfig {
	s.mu.Lock()
	defer s.mu.Unlock()

	ret, ok := s.outputValues[addr]
	if !ok {
		cfg, ok := s.config.Stack.OutputValues[addr.Name]
		if !ok {
			return nil
		}
		cfgAddr := stackaddrs.Config(s.Addr(), addr)
		ret = newOutputValueConfig(s.main, cfgAddr, cfg)
		s.outputValues[addr] = ret
	}
	return ret
}

// OutputValues returns a map of the objects representing all of the
// output values declared inside this stack configuration.
func (s *StackConfig) OutputValues(ctx context.Context) map[stackaddrs.OutputValue]*OutputValueConfig {
	if len(s.config.Stack.OutputValues) == 0 {
		return nil
	}
	ret := make(map[stackaddrs.OutputValue]*OutputValueConfig, len(s.config.Stack.OutputValues))
	for name := range s.config.Stack.OutputValues {
		addr := stackaddrs.OutputValue{Name: name}
		ret[addr] = s.OutputValue(ctx, addr)
	}
	return ret
}

// ResultType returns the type of the result object that will be produced
// by this stack configuration, based on the output values declared within
// it.
func (s *StackConfig) ResultType(ctx context.Context) cty.Type {
	os := s.OutputValues(ctx)
	atys := make(map[string]cty.Type, len(os))
	for addr, o := range os {
		atys[addr.Name] = o.ValueTypeConstraint(ctx)
	}
	return cty.Object(atys)
}

// Providers returns a map of the objects representing all of the provider
// configurations declared inside this stack configuration.
func (s *StackConfig) Providers(ctx context.Context) map[stackaddrs.ProviderConfig]*ProviderConfig {
	if len(s.config.Stack.ProviderConfigs) == 0 {
		return nil
	}
	ret := make(map[stackaddrs.ProviderConfig]*ProviderConfig, len(s.config.Stack.ProviderConfigs))
	for configAddr := range s.config.Stack.ProviderConfigs {
		provider, ok := s.config.Stack.RequiredProviders.ProviderForLocalName(configAddr.LocalName)
		if !ok {
			// Then we are missing a provider declaration, this will be caught
			// elsewhere so we'll just skip it here.
			continue
		}

		addr := stackaddrs.ProviderConfig{
			Provider: provider,
			Name:     configAddr.Alias,
		}
		ret[addr] = s.Provider(ctx, addr)
	}
	return ret
}

// Provider returns a [ProviderConfig] representing the provider configuration
// block within the stack configuration that matches the given address,
// or nil if there is no such declaration.
func (s *StackConfig) Provider(ctx context.Context, addr stackaddrs.ProviderConfig) *ProviderConfig {
	s.mu.Lock()
	defer s.mu.Unlock()

	ret, ok := s.providers[addr]
	if !ok {
		localName, ok := s.config.Stack.RequiredProviders.LocalNameForProvider(addr.Provider)
		if !ok {
			return nil
		}
		// FIXME: stackconfig package currently uses addrs.LocalProviderConfig
		// instead of stackaddrs.ProviderConfigRef.
		configAddr := addrs.LocalProviderConfig{
			LocalName: localName,
			Alias:     addr.Name,
		}
		cfg, ok := s.config.Stack.ProviderConfigs[configAddr]
		if !ok {
			return nil
		}
		cfgAddr := stackaddrs.Config(s.Addr(), addr)
		ret = newProviderConfig(s.main, cfgAddr, cfg)
		s.providers[addr] = ret
	}
	return ret
}

// ProviderByLocalAddr returns a [ProviderConfig] representing the provider
// configuration block within the stack configuration that matches the given
// local address, or nil if there is no such declaration.
//
// This is equivalent to calling [Provider] just using a reference address
// instead of a config address.
func (s *StackConfig) ProviderByLocalAddr(ctx context.Context, localAddr stackaddrs.ProviderConfigRef) *ProviderConfig {
	s.mu.Lock()
	defer s.mu.Unlock()

	provider, ok := s.config.Stack.RequiredProviders.ProviderForLocalName(localAddr.ProviderLocalName)
	if !ok {
		return nil
	}

	addr := stackaddrs.ProviderConfig{
		Provider: provider,
		Name:     localAddr.Name,
	}
	ret, ok := s.providers[addr]
	if !ok {
		configAddr := addrs.LocalProviderConfig{
			LocalName: localAddr.ProviderLocalName,
			Alias:     localAddr.Name,
		}
		cfg, ok := s.config.Stack.ProviderConfigs[configAddr]
		if !ok {
			return nil
		}
		cfgAddr := stackaddrs.Config(s.Addr(), addr)
		ret = newProviderConfig(s.main, cfgAddr, cfg)
		s.providers[addr] = ret
	}
	return ret
}

// ProviderLocalName returns the local name used for the given provider
// in this particular stack configuration, based on the declarations in
// the required_providers configuration block.
//
// If the second return value is false then there is no local name declared
// for the given provider, and so the first return value is invalid.
func (s *StackConfig) ProviderLocalName(ctx context.Context, addr addrs.Provider) (string, bool) {
	return s.config.Stack.RequiredProviders.LocalNameForProvider(addr)
}

// ProviderForLocalName returns the provider for the given local name in this
// particular stack configuration, based on the declarations in the
// required_providers configuration block.
//
// If the second return value is false then there is no provider declared
// for the given local name, and so the first return value is invalid.
func (s *StackConfig) ProviderForLocalName(ctx context.Context, localName string) (addrs.Provider, bool) {
	return s.config.Stack.RequiredProviders.ProviderForLocalName(localName)
}

// StackCall returns a [StackCallConfig] representing the "stack" block
// matching the given address declared within this stack config, or nil if
// there is no such declaration.
func (s *StackConfig) StackCall(ctx context.Context, addr stackaddrs.StackCall) *StackCallConfig {
	s.mu.Lock()
	defer s.mu.Unlock()

	ret, ok := s.stackCalls[addr]
	if !ok {
		cfg, ok := s.config.Stack.EmbeddedStacks[addr.Name]
		if !ok {
			return nil
		}
		cfgAddr := stackaddrs.Config(s.Addr(), addr)
		ret = newStackCallConfig(s.main, cfgAddr, cfg)
		s.stackCalls[addr] = ret
	}
	return ret
}

// StackCalls returns a map of objects representing all of the embedded stack
// calls inside this stack configuration.
func (s *StackConfig) StackCalls(ctx context.Context) map[stackaddrs.StackCall]*StackCallConfig {
	if len(s.config.Children) == 0 {
		return nil
	}
	ret := make(map[stackaddrs.StackCall]*StackCallConfig, len(s.config.Children))
	for n := range s.config.Children {
		stepAddr := stackaddrs.StackCall{Name: n}
		ret[stepAddr] = s.StackCall(ctx, stepAddr)
	}
	return ret
}

// Component returns a [ComponentConfig] representing the component call
// declared within this stack config that matches the given address, or nil if
// there is no such declaration.
func (s *StackConfig) Component(ctx context.Context, addr stackaddrs.Component) *ComponentConfig {
	s.mu.Lock()
	defer s.mu.Unlock()

	ret, ok := s.components[addr]
	if !ok {
		cfg, ok := s.config.Stack.Components[addr.Name]
		if !ok {
			return nil
		}
		cfgAddr := stackaddrs.Config(s.Addr(), addr)
		ret = newComponentConfig(s.main, cfgAddr, cfg)
		s.components[addr] = ret
	}
	return ret
}

// Components returns a map of the objects representing all of the
// component calls declared inside this stack configuration.
func (s *StackConfig) Components(ctx context.Context) map[stackaddrs.Component]*ComponentConfig {
	if len(s.config.Stack.Components) == 0 {
		return nil
	}
	ret := make(map[stackaddrs.Component]*ComponentConfig, len(s.config.Stack.Components))
	for name := range s.config.Stack.Components {
		addr := stackaddrs.Component{Name: name}
		ret[addr] = s.Component(ctx, addr)
	}
	return ret
}

// ResolveExpressionReference implements ExpressionScope, providing the
// global scope for evaluation within an unexpanded stack during the validate
// phase.
func (s *StackConfig) ResolveExpressionReference(ctx context.Context, ref stackaddrs.Reference) (Referenceable, tfdiags.Diagnostics) {
	return s.resolveExpressionReference(ctx, ref, nil, instances.RepetitionData{})
}

// resolveExpressionReference is the shared implementation of various
// validation-time ResolveExpressionReference methods, factoring out all
// of the common parts into one place.
func (s *StackConfig) resolveExpressionReference(
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
		ret := s.StackCall(ctx, addr)
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

// ExternalFunctions implements ExpressionScope.
func (s *StackConfig) ExternalFunctions(ctx context.Context) (lang.ExternalFuncs, func(), tfdiags.Diagnostics) {
	return s.main.ProviderFunctions(ctx, s)
}

// PlanTimestamp implements ExpressionScope, providing the timestamp at which
// the current plan is being run.
func (s *StackConfig) PlanTimestamp() time.Time {
	return s.main.PlanTimestamp()
}

// reportNamedPromises implements namedPromiseReporter.
func (s *StackConfig) reportNamedPromises(cb func(id promising.PromiseID, name string)) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, child := range s.children {
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
