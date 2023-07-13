package stackeval

import (
	"context"
	"fmt"
	"sync"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/instances"
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
	stackCalls     map[stackaddrs.StackCall]*StackCallConfig
}

var _ ExpressionScope = (*StackConfig)(nil)
var _ namedPromiseReporter = (*StackConfig)(nil)

func newStackConfig(main *Main, addr stackaddrs.Stack, config *stackconfig.ConfigNode) *StackConfig {
	return &StackConfig{
		addr:   addr,
		config: config,
		main:   main,

		children:       make(map[stackaddrs.StackStep]*StackConfig, len(config.Children)),
		inputVariables: make(map[stackaddrs.InputVariable]*InputVariableConfig, len(config.Stack.Declarations.InputVariables)),
		stackCalls:     make(map[stackaddrs.StackCall]*StackCallConfig, len(config.Stack.Declarations.EmbeddedStacks)),
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

// ResolveExpressionReference implements ExpressionScope, providing the
// global scope for evaluation within an unexpanded stack during the validate
// phase.
func (s *StackConfig) ResolveExpressionReference(ctx context.Context, ref stackaddrs.Reference) (Referenceable, tfdiags.Diagnostics) {
	return s.resolveExpressionReference(ctx, ref, instances.RepetitionData{}, nil)
}

// resolveExpressionReference is the shared implementation of various
// validation-time ResolveExpressionReference methods, factoring out all
// of the common parts into one place.
func (s *StackConfig) resolveExpressionReference(ctx context.Context, ref stackaddrs.Reference, repetition instances.RepetitionData, selfAddr stackaddrs.Referenceable) (Referenceable, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	// TODO: Most of the below would benefit from "Did you mean..." suggestions
	// when something is missing but there's a similarly-named object nearby.

	switch addr := ref.Target.(type) {
	case stackaddrs.InputVariable:
		ret := s.InputVariable(ctx, addr)
		if ret == nil {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Reference to undefined input variable",
				Detail:   fmt.Sprintf("There is no variable %q block in this stack.", addr.Name),
				Subject:  ref.SourceRange.ToHCL().Ptr(),
			})
		}
		return ret, diags
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
	for _, child := range s.stackCalls {
		child.reportNamedPromises(cb)
	}
}
