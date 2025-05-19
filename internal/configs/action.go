// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package configs

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/hashicorp/terraform/internal/addrs"
)

// Action represents an "action" block inside a configuration
type Action struct {
	Name      string
	Type      string
	Config    hcl.Body
	Count     hcl.Expression
	ForEach   hcl.Expression
	DependsOn []hcl.Traversal

	ProviderConfigRef *ProviderConfigRef
	Provider          addrs.Provider

	DeclRange hcl.Range
	TypeRange hcl.Range
}

// ActionTrigger represents a configured "action_trigger" inside the lifecycle
// block of a managed resource.
type ActionTrigger struct {
	Condition hcl.Expression
	Events    []ActionTriggerEvent
	Actions   []ActionRef

	DeclRange hcl.Range
}

// ActionTriggerEvent is an enum for valid values for events for action
// triggers.
type ActionTriggerEvent int

//go:generate go tool golang.org/x/tools/cmd/stringer -type ActionTriggerEvent

const (
	BeforeCreate ActionTriggerEvent = iota
	AfterCreate
	BeforeUpdate
	AfterUpdate
	BeforeDestroy
	AfterDestroy
)

// ActionRef represents a reference to a configured Action
// copypasta of providerconfigref; not sure what's needed.
type ActionRef struct {
	Traversal hcl.Traversal
	Range     hcl.Range
}

func decodeActionTriggerBlock(block *hcl.Block) (*ActionTrigger, hcl.Diagnostics) {
	var diags hcl.Diagnostics
	a := &ActionTrigger{
		Events:    []ActionTriggerEvent{},
		Actions:   []ActionRef{},
		Condition: nil,
	}

	content, bodyDiags := block.Body.Content(actionTriggerSchema)
	diags = append(diags, bodyDiags...)

	if attr, exists := content.Attributes["condition"]; exists {
		a.Condition = attr.Expr
	}

	// this is parsing events like expressions, so it's angry when there's quotes
	// needs to parse strings:
	// Quoted references are deprecated; In this context, references are expected literally rather than in quotes.
	if attr, exists := content.Attributes["events"]; exists {
		exprs, ediags := hcl.ExprList(attr.Expr)
		diags = append(diags, ediags...)

		events := []ActionTriggerEvent{}

		for _, expr := range exprs {
			var event ActionTriggerEvent
			switch hcl.ExprAsKeyword(expr) {
			case "before_create":
				event = BeforeCreate
			case "after_create":
				event = AfterCreate
			case "before_update":
				event = BeforeUpdate
			case "after_update":
				event = AfterUpdate
			case "before_destroy":
				event = BeforeDestroy
			case "after_destroy":
				event = AfterDestroy
			default:
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  fmt.Sprintf("Invalid \"event\" value %s", hcl.ExprAsKeyword(expr)),
					Detail:   "The \"event\" argument supports the following values: before_create, after_create, before_update, after_update, before_destroy, after_destroy.",
					Subject:  expr.Range().Ptr(),
				})
			}
			events = append(events, event)
		}
		a.Events = events
	}

	if attr, exists := content.Attributes["actions"]; exists {
		exprs, ediags := hcl.ExprList(attr.Expr)
		diags = append(diags, ediags...)
		actions := []ActionRef{}
		for _, expr := range exprs {
			traversal, travDiags := hcl.AbsTraversalForExpr(expr)
			diags = append(diags, travDiags...)
			// verify that the traversal is an action
			if traversal.RootName() != "action" {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid actions argument inside action_triggers",
					Detail:   "action_triggers.actions accepts a list of one or more actions",
					Subject:  block.DefRange.Ptr(),
				})
			}
			if len(traversal) != 0 {
				actionRef := ActionRef{
					Traversal: traversal,
					Range:     expr.Range(),
				}
				actions = append(actions, actionRef)
			}
		}
		a.Actions = actions
	}

	if len(a.Actions) == 0 {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "No actions specified",
			Detail:   "At least one action must be specified for an action_trigger.",
			Subject:  block.DefRange.Ptr(),
		})
	}

	if len(a.Events) == 0 {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "No events specified",
			Detail:   "At least one event must be specified for an action_trigger.",
			Subject:  block.DefRange.Ptr(),
		})
	}
	return a, diags
}

func decodeActionBlock(block *hcl.Block) (*Action, hcl.Diagnostics) {
	var diags hcl.Diagnostics
	a := &Action{
		Type:      block.Labels[0],
		Name:      block.Labels[1],
		DeclRange: block.DefRange,
		TypeRange: block.LabelRanges[0],
	}

	if !hclsyntax.ValidIdentifier(a.Type) {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid action type name",
			Detail:   badIdentifierDetail,
			Subject:  &block.LabelRanges[0],
		})
	}
	if !hclsyntax.ValidIdentifier(a.Name) {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid action name",
			Detail:   badIdentifierDetail,
			Subject:  &block.LabelRanges[1],
		})
	}

	content, remain, moreDiags := block.Body.PartialContent(actionBlockSchema)
	diags = append(diags, moreDiags...)
	a.Config = remain

	if attr, exists := content.Attributes["count"]; exists {
		a.Count = attr.Expr
	}

	if attr, exists := content.Attributes["for_each"]; exists {
		a.ForEach = attr.Expr
		// Cannot have count and for_each on the same action block
		if a.Count != nil {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  `Invalid combination of "count" and "for_each"`,
				Detail:   `The "count" and "for_each" meta-arguments are mutually-exclusive, only one should be used.`,
				Subject:  &attr.NameRange,
			})
		}
	}

	if attr, exists := content.Attributes["provider"]; exists {
		var providerDiags hcl.Diagnostics
		a.ProviderConfigRef, providerDiags = decodeProviderConfigRef(attr.Expr, "provider")
		diags = append(diags, providerDiags...)
	}

	if attr, exists := content.Attributes["depends_on"]; exists {
		deps, depsDiags := DecodeDependsOn(attr)
		diags = append(diags, depsDiags...)
		a.DependsOn = append(a.DependsOn, deps...)
	}

	return a, diags
}

// actionBlockSchema is the schema for an action type within terraform.
var actionBlockSchema = &hcl.BodySchema{
	Attributes: commonResourceAttributes,
}

var actionTriggerSchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		{
			Name:     "events",
			Required: true,
		},
		{
			Name:     "condition",
			Required: false,
		},
		{
			Name:     "actions",
			Required: true,
		},
	},
}

func (a *Action) moduleUniqueKey() string {
	return a.Addr().String()
}

// Addr returns a resource address for the receiver that is relative to the
// resource's containing module.
func (a *Action) Addr() addrs.Action {
	return addrs.Action{
		Type: a.Type,
		Name: a.Name,
	}
}

// ProviderConfigAddr returns the address for the provider configuration that
// should be used for this action. This function returns a default provider
// config addr if an explicit "provider" argument was not provided.
func (a *Action) ProviderConfigAddr() addrs.LocalProviderConfig {
	if a.ProviderConfigRef == nil {
		// If no specific "provider" argument is given, we want to look up the
		// provider config where the local name matches the implied provider
		// from the resource type. This may be different from the resource's
		// provider type.
		return addrs.LocalProviderConfig{
			LocalName: a.Addr().ImpliedProvider(),
		}
	}

	return addrs.LocalProviderConfig{
		LocalName: a.ProviderConfigRef.Name,
		Alias:     a.ProviderConfigRef.Alias,
	}
}
