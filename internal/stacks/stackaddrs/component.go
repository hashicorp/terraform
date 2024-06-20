// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackaddrs

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/collections"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Component is the address of a "component" block within a stack config.
type Component struct {
	Name string
}

func (Component) referenceableSigil()   {}
func (Component) inStackConfigSigil()   {}
func (Component) inStackInstanceSigil() {}

func (c Component) String() string {
	return "component." + c.Name
}

func (c Component) UniqueKey() collections.UniqueKey[Component] {
	return c
}

// A Component is its own [collections.UniqueKey].
func (Component) IsUniqueKey(Component) {}

// ConfigComponent places a [Component] in the context of a particular [Stack].
type ConfigComponent = InStackConfig[Component]

// AbsComponent places a [Component] in the context of a particular [StackInstance].
type AbsComponent = InStackInstance[Component]

// ComponentInstance is the address of a dynamic instance of a component.
type ComponentInstance struct {
	Component Component
	Key       addrs.InstanceKey
}

func (ComponentInstance) inStackConfigSigil()   {}
func (ComponentInstance) inStackInstanceSigil() {}

func (c ComponentInstance) String() string {
	if c.Key == nil {
		return c.Component.String()
	}
	return c.Component.String() + c.Key.String()
}

func (c ComponentInstance) UniqueKey() collections.UniqueKey[ComponentInstance] {
	return c
}

// A ComponentInstance is its own [collections.UniqueKey].
func (ComponentInstance) IsUniqueKey(ComponentInstance) {}

// ConfigComponentInstance places a [ComponentInstance] in the context of a
// particular [Stack].
type ConfigComponentInstance = InStackConfig[ComponentInstance]

// AbsComponentInstance places a [ComponentInstance] in the context of a
// particular [StackInstance].
type AbsComponentInstance = InStackInstance[ComponentInstance]

func ConfigComponentForAbsInstance(instAddr AbsComponentInstance) ConfigComponent {
	configInst := ConfigForAbs(instAddr) // a ConfigComponentInstance
	return ConfigComponent{
		Stack: configInst.Stack,
		Item: Component{
			Name: configInst.Item.Component.Name,
		},
	}
}

func ParseAbsComponentInstance(traversal hcl.Traversal) (AbsComponentInstance, tfdiags.Diagnostics) {
	inst, remain, diags := parseAbsComponentInstance(traversal)
	if diags.HasErrors() {
		return AbsComponentInstance{}, diags
	}

	if len(remain) > 0 {
		// Then we have some remaining traversal steps that weren't consumed
		// by the component instance address itself, which is an error when the
		// caller is using this function.
		rng := remain.SourceRange()
		// if "remain" is empty then the source range would be zero length,
		// and so we'll use the original traversal instead.
		if len(remain) == 0 {
			rng = traversal.SourceRange()
		}
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid component instance address",
			Detail:   "The component instance address must include the keyword \"component\" followed by a component name.",
			Subject:  &rng,
		})
		return AbsComponentInstance{}, diags
	}

	return inst, diags
}

func ParseAbsComponentInstanceStr(s string) (AbsComponentInstance, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	traversal, hclDiags := hclsyntax.ParseTraversalAbs([]byte(s), "", hcl.InitialPos)
	diags = diags.Append(hclDiags)
	if diags.HasErrors() {
		return AbsComponentInstance{}, diags
	}

	ret, moreDiags := ParseAbsComponentInstance(traversal)
	diags = diags.Append(moreDiags)
	return ret, diags
}

func ParsePartialComponentInstanceStr(s string) (AbsComponentInstance, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	traversal, hclDiags := hclsyntax.ParseTraversalPartial([]byte(s), "", hcl.InitialPos)
	diags = diags.Append(hclDiags)
	if diags.HasErrors() {
		return AbsComponentInstance{}, diags
	}

	ret, moreDiags := ParseAbsComponentInstance(traversal)
	diags = diags.Append(moreDiags)
	return ret, diags
}

func parseAbsComponentInstance(traversal hcl.Traversal) (AbsComponentInstance, hcl.Traversal, tfdiags.Diagnostics) {
	if traversal.IsRelative() {
		// This is always a caller bug: caller must only pass absolute
		// traversals in here.
		panic("parseAbsComponentInstance with relative traversal")
	}

	stackInst, remain, diags := parseInStackInstancePrefix(traversal)
	if diags.HasErrors() {
		return AbsComponentInstance{}, remain, diags
	}

	// "remain" should now be the keyword "component" followed by a valid
	// component name, optionally followed by an instance key.
	const diagSummary = "Invalid component instance address"

	if kwStep, ok := remain[0].(hcl.TraverseAttr); !ok || kwStep.Name != "component" {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  diagSummary,
			Detail:   "The component instance address must include the keyword \"component\" followed by a component name.",
			Subject:  remain[0].SourceRange().Ptr(),
		})
		return AbsComponentInstance{}, remain, diags
	}
	remain = remain[1:]

	nameStep, ok := remain[0].(hcl.TraverseAttr)
	if !ok {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  diagSummary,
			Detail:   "The component instance address must include the keyword \"component\" followed by a component name.",
			Subject:  remain[1].SourceRange().Ptr(),
		})
		return AbsComponentInstance{}, remain, diags
	}
	remain = remain[1:]
	componentAddr := ComponentInstance{
		Component: Component{Name: nameStep.Name},
	}

	if len(remain) > 0 {
		switch instStep := remain[0].(type) {
		case hcl.TraverseIndex:
			var err error
			componentAddr.Key, err = addrs.ParseInstanceKey(instStep.Key)
			if err != nil {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  diagSummary,
					Detail:   fmt.Sprintf("Invalid instance key: %s.", err),
					Subject:  instStep.SourceRange().Ptr(),
				})
				return AbsComponentInstance{}, remain, diags
			}

			remain = remain[1:]
		case hcl.TraverseSplat:
			componentAddr.Key = addrs.WildcardKey
			remain = remain[1:]
		}
	}

	return AbsComponentInstance{
		Stack: stackInst,
		Item:  componentAddr,
	}, remain, diags
}
