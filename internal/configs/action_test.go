// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package configs

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcltest"
	"github.com/zclconf/go-cty/cty"
)

func TestDecodeActionBlock(t *testing.T) {
	fooResourceExpr := hcltest.MockExprTraversalSrc("resource_type.foo")
	barResourceExpr := hcltest.MockExprTraversalSrc("resource_type.bar")
	fooAndBarExpr := hcltest.MockExprList([]hcl.Expression{fooResourceExpr, barResourceExpr})
	moduleResourceExpr := hcltest.MockExprTraversalSrc("module.foo.resource_type.bar")
	fooDataSourceExpr := hcltest.MockExprTraversalSrc("data.example.foo")

	tests := map[string]struct {
		input       *hcl.Block
		want        *Action
		expectDiags []string
	}{
		"one linked resource": {
			&hcl.Block{
				Type:   "action",
				Labels: []string{"an_action", "foo"},
				Body: hcltest.MockBody(&hcl.BodyContent{
					Attributes: hcl.Attributes{
						"linked_resource": {
							Name: "linked_resource",
							Expr: fooResourceExpr,
						},
					},
				}),
				DefRange:    blockRange,
				LabelRanges: []hcl.Range{{}},
			},
			&Action{
				Type:            "an_action",
				Name:            "foo",
				LinkedResources: []hcl.Traversal{mustAbsTraversalForExpr(fooResourceExpr)},
				DeclRange:       blockRange,
			},
			nil,
		},
		"multiple linked resources": {
			&hcl.Block{
				Type:   "action",
				Labels: []string{"an_action", "foo"},
				Body: hcltest.MockBody(&hcl.BodyContent{
					Attributes: hcl.Attributes{
						"linked_resources": {
							Name: "linked_resources",
							Expr: fooAndBarExpr,
						},
					},
				}),
				DefRange:    blockRange,
				LabelRanges: []hcl.Range{{}},
			},
			&Action{
				Type:            "an_action",
				Name:            "foo",
				LinkedResources: []hcl.Traversal{mustAbsTraversalForExpr(fooResourceExpr), mustAbsTraversalForExpr(barResourceExpr)},
				DeclRange:       blockRange,
			},
			nil,
		},
		"invalid linked resources (module ref)": { // for now! This test will change when we support cross-module actions
			&hcl.Block{
				Type:   "action",
				Labels: []string{"an_action", "foo"},
				Body: hcltest.MockBody(&hcl.BodyContent{
					Attributes: hcl.Attributes{
						"linked_resources": {
							Name: "linked_resources",
							Expr: hcltest.MockExprList([]hcl.Expression{moduleResourceExpr}),
						},
					},
				}),
				DefRange:    blockRange,
				LabelRanges: []hcl.Range{{}},
			},
			&Action{
				Type:            "an_action",
				Name:            "foo",
				LinkedResources: []hcl.Traversal{},
				DeclRange:       blockRange,
			},
			[]string{`:0,0-0: Invalid "linked_resources"; "linked_resources" must only refer to managed resources in the current module.`},
		},
		"invalid linked resource (datasource ref)": {
			&hcl.Block{
				Type:   "action",
				Labels: []string{"an_action", "foo"},
				Body: hcltest.MockBody(&hcl.BodyContent{
					Attributes: hcl.Attributes{
						"linked_resource": {
							Name: "linked_resource",
							Expr: fooDataSourceExpr,
						},
					},
				}),
				DefRange:    blockRange,
				LabelRanges: []hcl.Range{{}},
			},
			&Action{
				Type:            "an_action",
				Name:            "foo",
				LinkedResources: nil,
				DeclRange:       blockRange,
			},
			[]string{`:0,0-0: Invalid "linked_resource"; "linked_resource" must only refer to a managed resource in the current module.`},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got, diags := decodeActionBlock(test.input)
			assertExactDiagnostics(t, diags, test.expectDiags)
			assertResultDeepEqual(t, got, test.want)
		})
	}
}

func TestDecodeActionTriggerBlock(t *testing.T) {
	conditionExpr := hcltest.MockExprLiteral(cty.True)
	eventsListExpr := hcltest.MockExprList([]hcl.Expression{hcltest.MockExprTraversalSrc("after_create"), hcltest.MockExprTraversalSrc("after_update")})

	fooActionExpr := hcltest.MockExprTraversalSrc("action.action_type.foo")
	barActionExpr := hcltest.MockExprTraversalSrc("action.action_type.bar")
	fooAndBarExpr := hcltest.MockExprList([]hcl.Expression{fooActionExpr, barActionExpr})

	// bad inputs!
	moduleActionExpr := hcltest.MockExprTraversalSrc("module.foo.action.action_type.bar")
	fooDataSourceExpr := hcltest.MockExprTraversalSrc("data.example.foo")

	tests := map[string]struct {
		input       *hcl.Block
		want        *ActionTrigger
		expectDiags []string
	}{
		"simple example": {
			&hcl.Block{
				Type: "action_trigger",
				Body: hcltest.MockBody(&hcl.BodyContent{
					Attributes: hcltest.MockAttrs(map[string]hcl.Expression{
						"condition": conditionExpr,
						"events":    eventsListExpr,
						"actions":   fooAndBarExpr,
					}),
				}),
			},
			&ActionTrigger{
				Condition: conditionExpr,
				Events:    []ActionTriggerEvent{AfterCreate, AfterUpdate},
				Actions: []ActionRef{
					{
						mustAbsTraversalForExpr(fooActionExpr),
						fooActionExpr.Range(),
					},
					{
						mustAbsTraversalForExpr(barActionExpr),
						barActionExpr.Range(),
					},
				},
			},
			nil,
		},
		"error - referencing actions in other modules": {
			&hcl.Block{
				Type: "action_trigger",
				Body: hcltest.MockBody(&hcl.BodyContent{
					Attributes: hcltest.MockAttrs(map[string]hcl.Expression{
						"condition": conditionExpr,
						"events":    eventsListExpr,
						"actions":   hcltest.MockExprList([]hcl.Expression{moduleActionExpr}),
					}),
				}),
			},
			&ActionTrigger{
				Condition: conditionExpr,
				Events:    []ActionTriggerEvent{AfterCreate, AfterUpdate},
				Actions:   []ActionRef{},
			},
			[]string{
				"MockExprTraversal:0,0-33: Invalid actions argument inside action_triggers; action_triggers.actions accepts a list of one or more actions, which must be in the current module.",
				":0,0-0: No actions specified; At least one action must be specified for an action_trigger.",
			},
		},
		"error - action is not an action": {
			&hcl.Block{
				Type: "action_trigger",
				Body: hcltest.MockBody(&hcl.BodyContent{
					Attributes: hcltest.MockAttrs(map[string]hcl.Expression{
						"condition": conditionExpr,
						"events":    eventsListExpr,
						"actions":   hcltest.MockExprList([]hcl.Expression{fooDataSourceExpr}),
					}),
				}),
			},
			&ActionTrigger{
				Condition: conditionExpr,
				Events:    []ActionTriggerEvent{AfterCreate, AfterUpdate},
				Actions:   []ActionRef{},
			},
			[]string{
				"MockExprTraversal:0,0-16: Invalid actions argument inside action_triggers; action_triggers.actions accepts a list of one or more actions, which must be in the current module.",
				":0,0-0: No actions specified; At least one action must be specified for an action_trigger.",
			},
		},
		"error - invalid event": {
			&hcl.Block{
				Type: "action_trigger",
				Body: hcltest.MockBody(&hcl.BodyContent{
					Attributes: hcltest.MockAttrs(map[string]hcl.Expression{
						"condition": conditionExpr,
						"events":    hcltest.MockExprList([]hcl.Expression{hcltest.MockExprTraversalSrc("not_an_event")}),
						"actions":   hcltest.MockExprList([]hcl.Expression{fooActionExpr}),
					}),
				}),
			},
			&ActionTrigger{
				Condition: conditionExpr,
				Events:    []ActionTriggerEvent{},
				Actions: []ActionRef{
					{
						mustAbsTraversalForExpr(fooActionExpr),
						fooActionExpr.Range(),
					},
				},
			},
			[]string{
				"MockExprTraversal:0,0-12: Invalid \"event\" value not_an_event; The \"event\" argument supports the following values: before_create, after_create, before_update, after_update, before_destroy, after_destroy.",
				":0,0-0: No events specified; At least one event must be specified for an action_trigger.",
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got, diags := decodeActionTriggerBlock(test.input)
			assertExactDiagnostics(t, diags, test.expectDiags)
			assertResultDeepEqual(t, got, test.want)
		})
	}
}

func mustAbsTraversalForExpr(expr hcl.Expression) hcl.Traversal {
	trav, diags := hcl.AbsTraversalForExpr(expr)
	if diags.HasErrors() {
		panic(diags.Errs())
	}
	return trav
}
