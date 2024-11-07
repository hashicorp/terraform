// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackconfig

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Declarations represent the various items that can be declared in a stack
// configuration.
//
// This just represents the fields that both [Stack] and [File] have in common,
// so we can share some code between the two.
type Declarations struct {
	// EmbeddedStacks are calls to other stack configurations that should
	// be treated as a part of the overall desired state produced from this
	// stack. These are declared with "stack" blocks in the stack language.
	EmbeddedStacks map[string]*EmbeddedStack

	// Components are calls to trees of Terraform modules that represent the
	// real infrastructure described by a stack.
	Components map[string]*Component

	// InputVariables, LocalValues, and OutputValues together represent all
	// of the "named values" in the stack configuration, which are just glue
	// to pass values between scopes or to factor out common expressions for
	// reuse in multiple locations.
	InputVariables map[string]*InputVariable
	LocalValues    map[string]*LocalValue
	OutputValues   map[string]*OutputValue

	// RequiredProviders represents the single required_providers block
	// that's allowed in any stack, declaring which providers this stack
	// depends on and which versions of those providers it is compatible with.
	RequiredProviders *ProviderRequirements

	// ProviderConfigs are the provider configurations declared in this
	// particular stack configuration. Other stack configurations in the
	// overall tree might have their own provider configurations.
	ProviderConfigs map[addrs.LocalProviderConfig]*ProviderConfig

	// Removed are the list of components that have been removed from the
	// configuration.
	Removed map[string]*Removed
}

func makeDeclarations() Declarations {
	return Declarations{
		EmbeddedStacks:  make(map[string]*EmbeddedStack),
		Components:      make(map[string]*Component),
		InputVariables:  make(map[string]*InputVariable),
		LocalValues:     make(map[string]*LocalValue),
		OutputValues:    make(map[string]*OutputValue),
		ProviderConfigs: make(map[addrs.LocalProviderConfig]*ProviderConfig),
		Removed:         make(map[string]*Removed),
	}
}

func (d *Declarations) addComponent(decl *Component) tfdiags.Diagnostics {
	if decl == nil {
		return nil
	}
	var diags tfdiags.Diagnostics

	name := decl.Name
	if existing, exists := d.Components[name]; exists {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Duplicate component declaration",
			Detail: fmt.Sprintf(
				"An component named %q was already declared at %s.",
				name, existing.DeclRange.ToHCL(),
			),
			Subject: decl.DeclRange.ToHCL().Ptr(),
		})
		return diags
	}

	if removed, exists := d.Removed[name]; exists && removed.FromIndex == nil {
		// If a component has been removed, we should not also find it in the
		// configuration.
		//
		// If the removed block has an index, then it's possible that only a
		// specific instance was removed and not the whole thing. This is okay
		// at this point, and will be validated more later. See the addRemoved
		// method for more information.
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Component exists for removed block",
			Detail: fmt.Sprintf(
				"A removed block for component %q was declared without an index, but a component block with the same name was declared at %s.\n\nA removed block without an index indicates that the component and all instances were removed from the configuration, and this is not the case.",
				name, decl.DeclRange.ToHCL(),
			),
			Subject: removed.DeclRange.ToHCL().Ptr(),
		})
		return diags
	}

	d.Components[name] = decl
	return diags
}

func (d *Declarations) addEmbeddedStack(decl *EmbeddedStack) tfdiags.Diagnostics {
	if decl == nil {
		return nil
	}
	var diags tfdiags.Diagnostics

	name := decl.Name
	if existing, exists := d.EmbeddedStacks[name]; exists {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Duplicate embedded stack call",
			Detail: fmt.Sprintf(
				"An embedded stack call named %q was already declared at %s.",
				name, existing.DeclRange.ToHCL(),
			),
			Subject: decl.DeclRange.ToHCL().Ptr(),
		})
		return diags
	}

	d.EmbeddedStacks[name] = decl
	return diags
}

func (d *Declarations) addInputVariable(decl *InputVariable) tfdiags.Diagnostics {
	if decl == nil {
		return nil
	}
	var diags tfdiags.Diagnostics

	name := decl.Name
	if existing, exists := d.InputVariables[name]; exists {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Duplicate input variable declaration",
			Detail: fmt.Sprintf(
				"An input variable named %q was already declared at %s.",
				name, existing.DeclRange.ToHCL(),
			),
			Subject: decl.DeclRange.ToHCL().Ptr(),
		})
		return diags
	}

	d.InputVariables[name] = decl
	return diags
}

func (d *Declarations) addLocalValue(decl *LocalValue) tfdiags.Diagnostics {
	if decl == nil {
		return nil
	}
	var diags tfdiags.Diagnostics

	name := decl.Name
	if existing, exists := d.LocalValues[name]; exists {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Duplicate local value declaration",
			Detail: fmt.Sprintf(
				"A local value named %q was already declared at %s.",
				name, existing.DeclRange.ToHCL(),
			),
			Subject: decl.DeclRange.ToHCL().Ptr(),
		})
		return diags
	}

	d.LocalValues[name] = decl
	return diags
}

func (d *Declarations) addOutputValue(decl *OutputValue) tfdiags.Diagnostics {
	if decl == nil {
		return nil
	}
	var diags tfdiags.Diagnostics

	name := decl.Name
	if existing, exists := d.OutputValues[name]; exists {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Duplicate output value declaration",
			Detail: fmt.Sprintf(
				"An output value named %q was already declared at %s.",
				name, existing.DeclRange.ToHCL(),
			),
			Subject: decl.DeclRange.ToHCL().Ptr(),
		})
		return diags
	}

	d.OutputValues[name] = decl
	return diags
}

func (d *Declarations) addRequiredProviders(decl *ProviderRequirements) tfdiags.Diagnostics {
	if decl == nil {
		return nil
	}
	var diags tfdiags.Diagnostics
	if d.RequiredProviders != nil {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Duplicate provider requirements",
			Detail: fmt.Sprintf(
				"This stack's provider requirements were already declared at %s.",
				d.RequiredProviders.DeclRange.ToHCL(),
			),
			Subject: decl.DeclRange.ToHCL().Ptr(),
		})
		return diags
	}
	d.RequiredProviders = decl
	return diags
}

func (d *Declarations) addProviderConfig(decl *ProviderConfig) tfdiags.Diagnostics {
	if decl == nil {
		return nil
	}
	var diags tfdiags.Diagnostics

	addr := decl.LocalAddr
	if existing, exists := d.ProviderConfigs[addr]; exists {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Duplicate provider configuration",
			Detail: fmt.Sprintf(
				"An configuration named %q for provider %q was already declared at %s.",
				addr.LocalName, addr.Alias, existing.DeclRange.ToHCL(),
			),
			Subject: decl.DeclRange.ToHCL().Ptr(),
		})
		return diags
	}

	d.ProviderConfigs[addr] = decl
	return diags
}

func (d *Declarations) addRemoved(decl *Removed) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	if decl == nil {
		return diags
	}
	name := decl.FromComponent.Name

	// We're going to make sure that all the removed blocks that share the same
	// FromComponent are consistent.
	if existing, exists := d.Removed[name]; exists {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Duplicate removed block",
			Detail: fmt.Sprintf(
				"A removed block for component %q was already declared at %s.",
				name, existing.DeclRange.ToHCL(),
			),
			Subject: decl.DeclRange.ToHCL().Ptr(),
		})
		return diags
	}

	if decl.FromIndex == nil {
		// If the removed block does not have an index, then we shouldn't also
		// have a component block with the same name. A removed block without
		// an index indicates that the component and all instances were removed
		// from the configuration.
		//
		// Note that a removed block with an index is allowed to coexist with a
		// component block with the same name, because it indicates that only
		// a specific instance was removed and not the whole thing. During the
		// validate and planning stages we will validate that the clashing
		// component and removed blocks are not both pointing to the same index.
		if component, exists := d.Components[name]; exists {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Component exists for removed block",
				Detail: fmt.Sprintf(
					"A removed block for component %q was declared without an index, but a component block with the same name was declared at %s.\n\nA removed block without an index indicates that the component and all instances were removed from the configuration, and this is not the case.",
					name, component.DeclRange.ToHCL(),
				),
				Subject: decl.DeclRange.ToHCL().Ptr(),
			})
			return diags
		}
	}

	d.Removed[name] = decl
	return diags
}

func (d *Declarations) merge(other *Declarations) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	for _, decl := range other.EmbeddedStacks {
		diags = diags.Append(
			d.addEmbeddedStack(decl),
		)
	}
	for _, decl := range other.Components {
		diags = diags.Append(
			d.addComponent(decl),
		)
	}
	for _, decl := range other.InputVariables {
		diags = diags.Append(
			d.addInputVariable(decl),
		)
	}
	for _, decl := range other.LocalValues {
		diags = diags.Append(
			d.addLocalValue(decl),
		)
	}
	for _, decl := range other.OutputValues {
		diags = diags.Append(
			d.addOutputValue(decl),
		)
	}
	if other.RequiredProviders != nil {
		d.addRequiredProviders(other.RequiredProviders)
	}
	for _, decl := range other.ProviderConfigs {
		diags = diags.Append(
			d.addProviderConfig(decl),
		)
	}
	for _, decl := range other.Removed {
		diags = diags.Append(
			d.addRemoved(decl),
		)
	}

	return diags
}
