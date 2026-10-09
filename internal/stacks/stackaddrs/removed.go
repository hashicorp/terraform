// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package stackaddrs

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type RemovedFrom struct {
	Stack []StackRemovedFrom

	// Component to be removed. Optional, if not set then the whole stack
	// should be removed.
	Component *ComponentRemovedFrom
}

func (rf RemovedFrom) TargetStack() Stack {
	stack := make(Stack, 0, len(rf.Stack))
	for _, step := range rf.Stack {
		stack = append(stack, StackStep{Name: step.Name})
	}
	return stack
}

func (rf RemovedFrom) TargetConfigComponent() ConfigComponent {
	if rf.Component == nil {
		panic("should call TargetStack() when no component was specified")
	}
	return ConfigComponent{
		Stack: rf.TargetStack(),
		Item: Component{
			rf.Component.Name,
		},
	}
}

func (rf RemovedFrom) Variables() []hcl.Traversal {
	var traversals []hcl.Traversal
	for _, step := range rf.Stack {
		if step.Index != nil {
			traversals = append(traversals, step.Index.Variables()...)
		}
	}
	if rf.Component != nil && rf.Component.Index != nil {
		traversals = append(traversals, rf.Component.Index.Variables()...)
	}
	return traversals
}

func (rf RemovedFrom) TargetStackInstance(ctx *hcl.EvalContext, parent StackInstance) (StackInstance, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	var stackInstance StackInstance
	for _, stack := range rf.Stack {
		step, moreDiags := stack.StackInstanceStep(ctx)
		diags = diags.Append(moreDiags)

		stackInstance = append(stackInstance, step)
	}
	return append(parent, stackInstance...), diags
}

func (rf RemovedFrom) TargetAbsComponentInstance(ctx *hcl.EvalContext, parent StackInstance) (AbsComponentInstance, tfdiags.Diagnostics) {
	if rf.Component == nil {
		panic("should call TargetStackInstance() when no component was specified")
	}
	var diags tfdiags.Diagnostics
	stackInstance, moreDiags := rf.TargetStackInstance(ctx, parent)
	diags = diags.Append(moreDiags)
	componentInstance, moreDiags := rf.Component.ComponentInstance(ctx)
	diags = diags.Append(moreDiags)

	return AbsComponentInstance{Stack: stackInstance, Item: componentInstance}, diags
}

type StackRemovedFrom struct {
	Name  string
	Index hcl.Expression
}

func (rf StackRemovedFrom) StackStep() StackStep {
	return StackStep{Name: rf.Name}
}

func (rf StackRemovedFrom) StackInstanceStep(ctx *hcl.EvalContext) (StackInstanceStep, tfdiags.Diagnostics) {
	key, diags := exprAsKey(rf.Index, ctx)
	return StackInstanceStep{
		Name: rf.Name,
		Key:  key,
	}, diags
}

type ComponentRemovedFrom struct {
	Name  string
	Index hcl.Expression
}

func (rf ComponentRemovedFrom) Component() Component {
	return Component{
		Name: rf.Name,
	}
}

func (rf ComponentRemovedFrom) ComponentInstance(ctx *hcl.EvalContext) (ComponentInstance, tfdiags.Diagnostics) {
	key, diags := exprAsKey(rf.Index, ctx)
	return ComponentInstance{
		Component: Component{
			Name: rf.Name,
		},
		Key: key,
	}, diags
}

// ParseRemovedFrom parses the "from" attribute of a "removed" block in a
// configuration and returns the address of the configuration object being
// removed.
//
// In addition to the address, this function also returns a traversal that
// represents the unparsed index within the from expression. Users can
// optionally specify a specific index of a component to target.
func ParseRemovedFrom(expr hcl.Expression) (RemovedFrom, tfdiags.Diagnostics) {
	// we always return the same diagnostic from this function when we
	// error, so we'll encapsulate it here.
	diag := &hcl.Diagnostic{
		Severity: hcl.DiagError,
		Summary:  "Invalid 'from' attribute",
		Detail:   "The 'from' attribute must designate a component or stack that has been removed, in the form of an address such as `component.component_name` or `stack.stack_name`.",
		Subject:  expr.Range().Ptr(),
	}

	var diags tfdiags.Diagnostics

	// The instance keys can be expressions, which are evaluated later with
	// the context of the removed block.
	steps, moreDiags := addrs.ParseAddressExpr(expr)
	diags = diags.Append(moreDiags)
	if moreDiags.HasErrors() {
		return RemovedFrom{}, diags
	}

	removedFrom := RemovedFrom{}
	for len(steps) > 0 {
		// Each part of the address is either "stack" or "component", followed
		// by a name and an optional instance key.
		if len(steps) < 2 {
			return RemovedFrom{}, diags.Append(diag)
		}
		kind, ok := addressExprStepName(steps[0])
		if !ok {
			return RemovedFrom{}, diags.Append(diag)
		}
		name, ok := steps[1].Step.(hcl.TraverseAttr)
		if !ok {
			return RemovedFrom{}, diags.Append(diag)
		}
		steps = steps[2:]

		var index hcl.Expression
		if len(steps) > 0 {
			if steps[0].Key != nil {
				index = steps[0].Key
				steps = steps[1:]
			} else if idx, ok := steps[0].Step.(hcl.TraverseIndex); ok {
				index = hcl.StaticExpr(idx.Key, idx.SrcRange)
				steps = steps[1:]
			}
		}

		switch kind {
		case "component":
			// A component must be the last part of the address.
			if len(steps) > 0 {
				return RemovedFrom{}, diags.Append(diag)
			}
			removedFrom.Component = &ComponentRemovedFrom{
				Name:  name.Name,
				Index: index,
			}
			return removedFrom, diags
		case "stack":
			removedFrom.Stack = append(removedFrom.Stack, StackRemovedFrom{
				Name:  name.Name,
				Index: index,
			})
		default:
			return RemovedFrom{}, diags.Append(diag)
		}
	}

	// if we fall out, then we're just targeting a stack directly instead of a
	// component in a stack
	return removedFrom, diags
}

// addressExprStepName returns the name in the given step of an address, if it
// isn't an instance key.
func addressExprStepName(step addrs.AddressExprStep) (string, bool) {
	switch step := step.Step.(type) {
	case hcl.TraverseRoot:
		return step.Name, true
	case hcl.TraverseAttr:
		return step.Name, true
	default:
		return "", false
	}
}

func exprAsKey(expr hcl.Expression, ctx *hcl.EvalContext) (addrs.InstanceKey, tfdiags.Diagnostics) {
	if expr == nil {
		return addrs.NoKey, nil
	}
	var diags tfdiags.Diagnostics

	value, moreDiags := expr.Value(ctx)
	diags = diags.Append(moreDiags)
	if moreDiags.HasErrors() {
		return addrs.WildcardKey, diags
	}

	if value.IsNull() {
		return addrs.WildcardKey, diags.Append(&hcl.Diagnostic{
			Severity:    hcl.DiagError,
			Summary:     "Invalid `from` attribute",
			Detail:      "The `from` attribute has an invalid index: cannot be null.",
			Subject:     expr.Range().Ptr(),
			Expression:  expr,
			EvalContext: ctx,
		})
	}

	if !value.IsKnown() {
		switch value.Type() {
		case cty.String, cty.Number:
			// this is potentially the right type, so we'll allow this
			return addrs.WildcardKey, diags
		case cty.DynamicPseudoType:
			// not ideal, but we can't confirm this for sure so we'll allow it
			return addrs.WildcardKey, diags
		default:
			// bad, this isn't the right type even if we don't know what the
			// value actually will be in the end
			return addrs.WildcardKey, diags.Append(&hcl.Diagnostic{
				Severity:    hcl.DiagError,
				Summary:     "Invalid `from` attribute",
				Detail:      "The `from` attribute has an invalid index: either a string or integer is required.",
				Subject:     expr.Range().Ptr(),
				Expression:  expr,
				EvalContext: ctx,
			})
		}
	}

	key, err := addrs.ParseInstanceKey(value)
	if err != nil {
		return addrs.WildcardKey, diags.Append(&hcl.Diagnostic{
			Severity:    hcl.DiagError,
			Summary:     "Invalid `from` attribute",
			Detail:      fmt.Sprintf("The `from` attribute has an invalid index: %s.", err),
			Subject:     expr.Range().Ptr(),
			Expression:  expr,
			EvalContext: ctx,
		})
	}

	return key, diags
}
