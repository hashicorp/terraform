// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackaddrs

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type RemovedFrom struct {
	Stack     []StackRemovedFrom
	Component ComponentRemovedFrom
}

func (rf RemovedFrom) ConfigComponent() ConfigComponent {
	return ConfigComponent{
		Stack: func() Stack {
			stack := make(Stack, 0, len(rf.Stack))
			for _, step := range rf.Stack {
				stack = append(stack, StackStep{Name: step.Name})
			}
			return stack
		}(),
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
	if rf.Component.Index != nil {
		traversals = append(traversals, rf.Component.Index.Variables()...)
	}
	return traversals
}

func (rf RemovedFrom) AbsComponentInstance(ctx *hcl.EvalContext, parent StackInstance) (AbsComponentInstance, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	var stackInstance StackInstance
	for _, stack := range rf.Stack {
		step, moreDiags := stack.StackInstanceStep(ctx)
		diags = diags.Append(moreDiags)

		stackInstance = append(stackInstance, step)
	}

	componentInstance, moreDiags := rf.Component.ComponentInstance(ctx)
	diags = diags.Append(moreDiags)

	return AbsComponentInstance{Stack: append(parent, stackInstance...), Item: componentInstance}, diags
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
	var diags tfdiags.Diagnostics

	removedFrom := RemovedFrom{}

	current, moreDiags := exprToComponentTraversal(expr)
	diags = diags.Append(moreDiags)
	if moreDiags.HasErrors() {
		return RemovedFrom{}, diags
	}

	for current != nil {

		// we're going to parse the traversal in sets of 2-3 depending on
		// the indices, so we'll check that now.
		nextTraversal := current.Current

		for len(nextTraversal) > 0 {
			var currentTraversal hcl.Traversal
			var indexExpr hcl.Expression

			switch {
			case len(nextTraversal) < 2:
				// this is simply an error, we always need at least 2 values
				// for either stack.name or component.name.
				return RemovedFrom{}, diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid 'from' attribute",
					Detail:   "The 'from' attribute must designate a component that has been removed, in the form `component.component_name` or `component.component_name[\"key\"].",
					Subject:  expr.Range().Ptr(),
				})
			case len(nextTraversal) == 2:
				indexExpr = current.Index
				currentTraversal = nextTraversal
				nextTraversal = nil
			case len(nextTraversal) == 3:
				if current.Index != nil {
					// this is an error, the last traversal should be taking
					// its index from the outer value if it exists, and to be
					// exactly three means something is invalid somewhere.
					return RemovedFrom{}, diags.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Invalid 'from' attribute",
						Detail:   "The 'from' attribute must designate a component that has been removed, in the form `component.component_name` or `component.component_name[\"key\"].",
						Subject:  expr.Range().Ptr(),
					})
				}

				index, ok := nextTraversal[2].(hcl.TraverseIndex)
				if !ok {
					// This is an error, with exactly 3 we don't have another
					// traversal to go to after this so the last entry must
					// be the index.
					return RemovedFrom{}, diags.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Invalid 'from' attribute",
						Detail:   "The 'from' attribute must designate a component that has been removed, in the form `component.component_name` or `component.component_name[\"key\"].",
						Subject:  expr.Range().Ptr(),
					})
				}

				currentTraversal = nextTraversal
				nextTraversal = nil
				indexExpr = hcl.StaticExpr(index.Key, index.SrcRange)

			default: // len(nextTraversal) > 3
				if index, ok := nextTraversal[2].(hcl.TraverseIndex); ok {
					currentTraversal = nextTraversal[:3]
					nextTraversal = nextTraversal[3:]
					indexExpr = hcl.StaticExpr(index.Key, index.SrcRange)
					break
				}
				currentTraversal = nextTraversal[:2]
				nextTraversal = nextTraversal[2:]
			}

			var name string

			switch root := currentTraversal[0].(type) {
			case hcl.TraverseRoot:
				name = root.Name
			case hcl.TraverseAttr:
				name = root.Name
			default:
				return RemovedFrom{}, diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid 'from' attribute",
					Detail:   "The 'from' attribute must designate a component that has been removed, in the form `component.component_name` or `component.component_name[\"key\"].",
					Subject:  expr.Range().Ptr(),
				})
			}

			switch name {
			case "component":
				name, ok := currentTraversal[1].(hcl.TraverseAttr)
				if !ok {
					return RemovedFrom{}, diags.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Invalid 'from' attribute",
						Detail:   "The 'from' attribute must designate a component that has been removed, in the form `component.component_name` or `component.component_name[\"key\"].",
						Subject:  expr.Range().Ptr(),
					})
				}

				if len(nextTraversal) > 0 || current.Rest != nil {
					return RemovedFrom{}, diags.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Invalid 'from' attribute",
						Detail:   "The 'from' attribute must designate a component that has been removed, in the form `component.component_name` or `component.component_name[\"key\"].",
						Subject:  expr.Range().Ptr(),
					})
				}

				removedFrom.Component = ComponentRemovedFrom{
					Name:  name.Name,
					Index: indexExpr,
				}
				return removedFrom, diags
			case "stack":
				name, ok := currentTraversal[1].(hcl.TraverseAttr)
				if !ok {
					return RemovedFrom{}, diags.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Invalid 'from' attribute",
						Detail:   "The 'from' attribute must designate a component that has been removed, in the form `component.component_name` or `component.component_name[\"key\"].",
						Subject:  expr.Range().Ptr(),
					})
				}

				removedFrom.Stack = append(removedFrom.Stack, StackRemovedFrom{
					Name:  name.Name,
					Index: indexExpr,
				})

			default:
				return RemovedFrom{}, diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid 'from' attribute",
					Detail:   "The 'from' attribute must designate a component that has been removed, in the form `component.component_name` or `component.component_name[\"key\"].",
					Subject:  expr.Range().Ptr(),
				})
			}
		}

		current = current.Rest
	}

	return RemovedFrom{}, diags.Append(&hcl.Diagnostic{
		Severity: hcl.DiagError,
		Summary:  "Invalid 'from' attribute",
		Detail:   "The 'from' attribute must designate a component that has been removed, in the form `component.component_name` or `component.component_name[\"key\"].",
		Subject:  expr.Range().Ptr(),
	})
}

type parsedFromExpr struct {
	Current hcl.Traversal
	Index   hcl.Expression
	Rest    *parsedFromExpr
}

// exprToComponentTraversal converts an HCL expression into a traversal that
// represents the component being targeted. We have to handle parsing this
// ourselves because removed block from arguments can contain index expressions
// which are not supported by hcl.AbsTraversalForExpr.
//
// The return values are (1) the part of the expression that can be converted
// into a traversal, (2) the index at the end of the traversal if it is an
// expression, (3) the remainder of the expression that needs to be parsed
// after (1) has been, and (4) the diagnostics.
func exprToComponentTraversal(expr hcl.Expression) (*parsedFromExpr, hcl.Diagnostics) {
	switch e := expr.(type) {
	case *hclsyntax.IndexExpr:

		current, diags := exprToComponentTraversal(e.Collection)
		if diags.HasErrors() {
			return nil, diags
		}

		for next := current; next != nil; next = next.Rest {
			if next.Rest == nil {
				next.Index = e.Key
			}
		}

		return current, diags

	case *hclsyntax.RelativeTraversalExpr:

		current, diags := exprToComponentTraversal(e.Source)
		if diags.HasErrors() {
			return nil, diags
		}

		for next := current; next != nil; next = next.Rest {
			if next.Rest == nil {
				next.Rest = &parsedFromExpr{
					Current: e.Traversal,
				}
				break
			}
		}

		return current, diags

	default:

		// For anything else, just rely on the default traversal logic.

		t, diags := hcl.AbsTraversalForExpr(expr)
		if diags.HasErrors() {
			return nil, diags
		}
		return &parsedFromExpr{
			Current: t,
			Index:   nil,
			Rest:    nil,
		}, diags

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
