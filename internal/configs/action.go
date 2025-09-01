// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package configs

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/lang/langrefs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// There are many ways of handling plurality in error messages (linked_resource
// vs linked_resources); this is one of them.
type diagFn func(*hcl.Range) *hcl.Diagnostic

func invalidLinkedResourceDiag(subj *hcl.Range) *hcl.Diagnostic {
	return &hcl.Diagnostic{
		Severity: hcl.DiagError,
		Summary:  `Invalid linked_resource`,
		Detail:   `linked_resource must only refer to a managed resource in the current module.`,
		Subject:  subj,
	}
}

func invalidLinkedResourcesDiag(subj *hcl.Range) *hcl.Diagnostic {
	return &hcl.Diagnostic{
		Severity: hcl.DiagError,
		Summary:  `Invalid linked_resources`,
		Detail:   `linked_resources must only refer to managed resources in the current module.`,
		Subject:  subj,
	}
}

func invalidActionDiag(subj *hcl.Range) *hcl.Diagnostic {
	return &hcl.Diagnostic{
		Severity: hcl.DiagError,
		Summary:  `Invalid action argument inside action_triggers`,
		Detail:   `action_triggers.actions must only refer to actions in the current module.`,
		Subject:  subj,
	}
}

// Action represents an "action" block inside a configuration
type Action struct {
	Name    string
	Type    string
	Config  hcl.Body
	Count   hcl.Expression
	ForEach hcl.Expression
	// DependsOn []hcl.Traversal // not yet supported

	LinkedResources []hcl.Expression

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
	Actions   []ActionRef // References to actions

	DeclRange hcl.Range
}

// ActionTriggerEvent is an enum for valid values for events for action
// triggers.
type ActionTriggerEvent int

//go:generate go tool golang.org/x/tools/cmd/stringer -type ActionTriggerEvent

const (
	Unknown ActionTriggerEvent = iota
	BeforeCreate
	AfterCreate
	BeforeUpdate
	AfterUpdate
	BeforeDestroy
	AfterDestroy
	Invoke
)

// ActionRef represents a reference to a configured Action
type ActionRef struct {
	Expr  hcl.Expression
	Range hcl.Range
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
				continue
			}

			if event == BeforeDestroy || event == AfterDestroy {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid destroy event used",
					Detail:   "The destroy events (before_destroy, after_destroy) are not supported as of right now. They will be supported in a future release.",
					Subject:  expr.Range().Ptr(),
				})
				continue
			}
			events = append(events, event)
		}
		a.Events = events
	}

	if attr, exists := content.Attributes["actions"]; exists {
		actionRefs, ediags := decodeActionTriggerRef(attr.Expr)
		diags = append(diags, ediags...)
		a.Actions = actionRefs
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

	content, moreDiags := block.Body.Content(actionBlockSchema)
	diags = append(diags, moreDiags...)

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

	if attr, exists := content.Attributes["linked_resource"]; exists {
		if a.LinkedResources != nil {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  `Invalid use of "linked_resource"`,
				Detail:   `"linked_resource" and "linked_resources" are mutually exclusive, only one should be used.`,
				Subject:  &attr.NameRange,
			})
		}
		lr, lrDiags := decodeLinkedResource(attr.Expr)
		diags = append(diags, lrDiags...)
		a.LinkedResources = []hcl.Expression{lr}
	}

	if attr, exists := content.Attributes["linked_resources"]; exists {
		if a.LinkedResources != nil {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  `Invalid use of "linked_resources"`,
				Detail:   `"linked_resource" and "linked_resources" are mutually exclusive, only one should be used.`,
				Subject:  &attr.NameRange,
			})
		}

		lrs, lrDiags := decodeLinkedResources(attr.Expr)
		diags = append(diags, lrDiags...)
		a.LinkedResources = lrs
	}

	for _, block := range content.Blocks {
		switch block.Type {
		case "config":
			if a.Config != nil {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Duplicate config block",
					Detail:   "An action must contain only one nested \"config\" block.",
					Subject:  block.DefRange.Ptr(),
				})
				return nil, diags
			}
			a.Config = block.Body
		default:
			// Should not get here because the above should cover all
			// block types declared in the schema.
			panic(fmt.Sprintf("unhandled block type %q", block.Type))
		}
	}

	if attr, exists := content.Attributes["provider"]; exists {
		var providerDiags hcl.Diagnostics
		a.ProviderConfigRef, providerDiags = decodeProviderConfigRef(attr.Expr, "provider")
		diags = append(diags, providerDiags...)
	}

	// depends_on: not yet supported
	// if attr, exists := content.Attributes["depends_on"]; exists {
	// 	deps, depsDiags := DecodeDependsOn(attr)
	// 	diags = append(diags, depsDiags...)
	// 	a.DependsOn = append(a.DependsOn, deps...)
	// }

	return a, diags
}

// actionBlockSchema is the schema for an action type within terraform.
var actionBlockSchema = &hcl.BodySchema{
	Attributes: commonActionAttributes,
	Blocks: []hcl.BlockHeaderSchema{
		{Type: "config"},
	},
}

var commonActionAttributes = []hcl.AttributeSchema{
	{
		Name: "count",
	},
	{
		Name: "for_each",
	},
	{
		Name: "provider",
	},
	{
		Name: "linked_resource",
	},
	{
		Name: "linked_resources",
	},
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

// decodeActionTriggerRef decodes and does basic validation of the Actions
// expression list inside a resource's ActionTrigger block, ensuring each only
// reference a single action. This function was largely copied from
// decodeReplaceTriggeredBy, but is much more permissive in what References are
// allowed.
func decodeActionTriggerRef(expr hcl.Expression) ([]ActionRef, hcl.Diagnostics) {
	exprs, diags := hcl.ExprList(expr)
	if diags.HasErrors() {
		return nil, diags
	}
	actionRefs := make([]ActionRef, len(exprs))

	for i, expr := range exprs {
		// Since we are manually parsing the action_trigger.Actions argument, we
		// need to specially handle json configs, in which case the values will
		// be json strings rather than hcl. To simplify parsing however we will
		// decode the individual list elements, rather than the entire
		// expression.
		var jsDiags hcl.Diagnostics
		expr, jsDiags = unwrapJSONRefExpr(expr)
		diags = diags.Extend(jsDiags)
		if diags.HasErrors() {
			continue
		}
		actionRefs[i] = ActionRef{
			Expr:  expr,
			Range: expr.Range(),
		}

		refs, refDiags := langrefs.ReferencesInExpr(addrs.ParseRef, expr)
		for _, diag := range refDiags {
			severity := hcl.DiagError
			if diag.Severity() == tfdiags.Warning {
				severity = hcl.DiagWarning
			}

			diags = append(diags, &hcl.Diagnostic{
				Severity: severity,
				Summary:  diag.Description().Summary,
				Detail:   diag.Description().Detail,
				Subject:  expr.Range().Ptr(),
			})
		}

		if refDiags.HasErrors() {
			continue
		}

		actionCount := 0
		for _, ref := range refs {
			switch ref.Subject.(type) {
			case addrs.Action, addrs.ActionInstance:
				actionCount++
			case addrs.ModuleCall, addrs.ModuleCallInstance, addrs.ModuleCallInstanceOutput:
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid reference to action outside this module",
					Detail:   "Actions can only be referenced in the module they are declared in.",
					Subject:  expr.Range().Ptr(),
				})
				continue
			case addrs.Resource, addrs.ResourceInstance:
				// definitely not an action
				diags = append(diags, invalidActionDiag(expr.Range().Ptr()))
				continue
			default:
				// we've checked what we can
			}
		}

		switch {
		case actionCount == 0:
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "No actions specified",
				Detail:   "At least one action must be specified for an action_trigger.",
				Subject:  expr.Range().Ptr(),
			})
		case actionCount > 1:
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid action expression",
				Detail:   "Multiple action references in actions expression.",
				Subject:  expr.Range().Ptr(),
			})
		}

	}

	return actionRefs, diags
}

// decodeLinkedResources decodes and does basic validation of an Action's
// LinkedResources.
func decodeLinkedResources(expr hcl.Expression) ([]hcl.Expression, hcl.Diagnostics) {
	exprs, diags := hcl.ExprList(expr)
	if diags.HasErrors() {
		return nil, diags
	}

	for i, expr := range exprs {
		// We are manually parsing config, so we need to handle json configs, in
		// which case the values will be json strings rather than hcl.
		var jsDiags hcl.Diagnostics
		expr, jsDiags = unwrapJSONRefExpr(expr)
		diags = diags.Extend(jsDiags)
		if diags.HasErrors() {
			continue
		}

		// re-assign the value in case it was modified by unwrapJSONRefExpr
		exprs[i] = expr

		_, lrDiags := decodeUnwrappedLinkedResource(expr, invalidLinkedResourcesDiag)
		diags = append(diags, lrDiags...)

	}

	return exprs, diags
}

func decodeLinkedResource(expr hcl.Expression) (hcl.Expression, hcl.Diagnostics) {
	// Handle possible json configs
	expr, diags := unwrapJSONRefExpr(expr)
	if diags.HasErrors() {
		return expr, diags
	}

	return decodeUnwrappedLinkedResource(expr, invalidLinkedResourceDiag)
}

func decodeUnwrappedLinkedResource(expr hcl.Expression, diagFunc diagFn) (hcl.Expression, hcl.Diagnostics) {
	var diags hcl.Diagnostics

	refs, refDiags := langrefs.ReferencesInExpr(addrs.ParseRef, expr)
	for _, diag := range refDiags {
		severity := hcl.DiagError
		if diag.Severity() == tfdiags.Warning {
			severity = hcl.DiagWarning
		}

		diags = append(diags, &hcl.Diagnostic{
			Severity: severity,
			Summary:  diag.Description().Summary,
			Detail:   diag.Description().Detail,
			Subject:  expr.Range().Ptr(),
		})
	}

	if refDiags.HasErrors() {
		return expr, diags
	}

	resourceCount := 0
	for _, ref := range refs {
		switch sub := ref.Subject.(type) {
		case addrs.ResourceInstance:
			if sub.Resource.Mode == addrs.ManagedResourceMode {
				diags = append(diags, diagFunc(expr.Range().Ptr()))
			} else {
				resourceCount++
			}
		case addrs.Resource:
			if sub.Mode != addrs.ManagedResourceMode {
				diags = append(diags, diagFunc(expr.Range().Ptr()))
			} else {
				resourceCount++
			}
		case addrs.ModuleCall, addrs.ModuleCallInstance, addrs.ModuleCallInstanceOutput:
			diags = append(diags, diagFunc(expr.Range().Ptr()))
		default:
			// we've checked what we can without evaluating references!
		}
	}

	switch {
	case resourceCount == 0:
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid linked_resource expression",
			Detail:   "Missing resource reference in linked_resource expression.",
			Subject:  expr.Range().Ptr(),
		})
	case resourceCount > 1:
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid linked_resource expression",
			Detail:   "Multiple resource references in linked_resource expression.",
			Subject:  expr.Range().Ptr(),
		})
	}

	return expr, diags
}
