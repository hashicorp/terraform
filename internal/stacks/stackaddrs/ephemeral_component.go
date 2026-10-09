// Copyright IBM Corp. 2014, 2026
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

// EphemeralComponent is the address of an "ephemeral_component" block within a stack config.
type EphemeralComponent struct {
	Name string
}

func (EphemeralComponent) referenceableSigil()   {}
func (EphemeralComponent) inStackConfigSigil()   {}
func (EphemeralComponent) inStackInstanceSigil() {}

func (c EphemeralComponent) String() string {
	return "ephemeral_component." + c.Name
}

func (c EphemeralComponent) UniqueKey() collections.UniqueKey[EphemeralComponent] {
	return c
}

// An EphemeralComponent is its own [collections.UniqueKey].
func (EphemeralComponent) IsUniqueKey(EphemeralComponent) {}

// ConfigEphemeralComponent places a [EphemeralComponent] in the context of a particular [Stack].
type ConfigEphemeralComponent = InStackConfig[EphemeralComponent]

// AbsEphemeralComponent places a [EphemeralComponent] in the context of a particular [StackInstance].
type AbsEphemeralComponent = InStackInstance[EphemeralComponent]

func AbsEphemeralComponentToInstance(ist AbsEphemeralComponent, ik addrs.InstanceKey) AbsEphemeralComponentInstance {
	return AbsEphemeralComponentInstance{
		Stack: ist.Stack,
		Item: EphemeralComponentInstance{
			EphemeralComponent: ist.Item,
			Key:                ik,
		},
	}
}

// EphemeralComponentInstance is the address of a dynamic instance of an ephemeral component.
type EphemeralComponentInstance struct {
	EphemeralComponent EphemeralComponent
	Key                addrs.InstanceKey
}

func (EphemeralComponentInstance) inStackConfigSigil()   {}
func (EphemeralComponentInstance) inStackInstanceSigil() {}

func (c EphemeralComponentInstance) String() string {
	if c.Key == nil {
		return c.EphemeralComponent.String()
	}
	return c.EphemeralComponent.String() + c.Key.String()
}

func (c EphemeralComponentInstance) UniqueKey() collections.UniqueKey[EphemeralComponentInstance] {
	return c
}

// A EphemeralComponentInstance is its own [collections.UniqueKey].
func (EphemeralComponentInstance) IsUniqueKey(EphemeralComponentInstance) {}

// ConfigEphemeralComponentInstance places a [EphemeralComponentInstance] in the context of a
// particular [Stack].
type ConfigEphemeralComponentInstance = InStackConfig[EphemeralComponentInstance]

// AbsEphemeralComponentInstance places a [EphemeralComponentInstance] in the context of a
// particular [StackInstance].
type AbsEphemeralComponentInstance = InStackInstance[EphemeralComponentInstance]

func ConfigEphemeralComponentForAbsInstance(instAddr AbsEphemeralComponentInstance) ConfigEphemeralComponent {
	configInst := ConfigForAbs(instAddr) // a ConfigEphemeralComponentInstance
	return ConfigEphemeralComponent{
		Stack: configInst.Stack,
		Item: EphemeralComponent{
			Name: configInst.Item.EphemeralComponent.Name,
		},
	}
}

func ParseAbsEphemeralComponentInstance(traversal hcl.Traversal) (AbsEphemeralComponentInstance, tfdiags.Diagnostics) {
	inst, remain, diags := ParseAbsEphemeralComponentInstanceOnly(traversal)
	if diags.HasErrors() {
		return AbsEphemeralComponentInstance{}, diags
	}

	if len(remain) > 0 {
		// Then we have some remaining traversal steps that weren't consumed
		// by the ephemeral component instance address itself, which is an error when the
		// caller is using this function.
		rng := remain.SourceRange()
		// if "remain" is empty then the source range would be zero length,
		// and so we'll use the original traversal instead.
		if len(remain) == 0 {
			rng = traversal.SourceRange()
		}
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid ephemeral component instance address",
			Detail:   "The ephemeral component instance address must include the keyword \"ephemeral_component\" followed by an ephemeral component name.",
			Subject:  &rng,
		})
		return AbsEphemeralComponentInstance{}, diags
	}

	return inst, diags
}

func ParseAbsEphemeralComponentInstanceStr(s string) (AbsEphemeralComponentInstance, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	traversal, hclDiags := hclsyntax.ParseTraversalAbs([]byte(s), "", hcl.InitialPos)
	diags = diags.Append(hclDiags)
	if diags.HasErrors() {
		return AbsEphemeralComponentInstance{}, diags
	}

	ret, moreDiags := ParseAbsEphemeralComponentInstance(traversal)
	diags = diags.Append(moreDiags)
	return ret, diags
}

func ParsePartialEphemeralComponentInstanceStr(s string) (AbsEphemeralComponentInstance, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	traversal, hclDiags := hclsyntax.ParseTraversalPartial([]byte(s), "", hcl.InitialPos)
	diags = diags.Append(hclDiags)
	if diags.HasErrors() {
		return AbsEphemeralComponentInstance{}, diags
	}

	ret, moreDiags := ParseAbsEphemeralComponentInstance(traversal)
	diags = diags.Append(moreDiags)
	return ret, diags
}

func ParseAbsEphemeralComponentInstanceStrOnly(s string) (AbsEphemeralComponentInstance, hcl.Traversal, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	traversal, hclDiags := hclsyntax.ParseTraversalPartial([]byte(s), "", hcl.InitialPos)
	diags = diags.Append(hclDiags)
	if diags.HasErrors() {
		return AbsEphemeralComponentInstance{}, traversal, diags
	}

	ret, rest, moreDiags := ParseAbsEphemeralComponentInstanceOnly(traversal)
	diags = diags.Append(moreDiags)
	return ret, rest, diags
}

func ParseAbsEphemeralComponentInstanceOnly(traversal hcl.Traversal) (AbsEphemeralComponentInstance, hcl.Traversal, tfdiags.Diagnostics) {
	if traversal.IsRelative() {
		// This is always a caller bug: caller must only pass absolute
		// traversals in here.
		panic("ParseAbsEphemeralComponentInstanceOnly with relative traversal")
	}

	stackInst, remain, diags := parseInStackInstancePrefix(traversal)
	if diags.HasErrors() {
		return AbsEphemeralComponentInstance{}, remain, diags
	}

	// "remain" should now be the keyword "ephemeral_component" followed by a valid
	// ephemeral component name, optionally followed by an instance key.
	const diagSummary = "Invalid ephemeral component instance address"

	if kwStep, ok := remain[0].(hcl.TraverseAttr); !ok || kwStep.Name != "ephemeral_component" {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  diagSummary,
			Detail:   "The ephemeral component instance address must include the keyword \"ephemeral_component\" followed by an ephemeral component name.",
			Subject:  remain[0].SourceRange().Ptr(),
		})
		return AbsEphemeralComponentInstance{}, remain, diags
	}
	remain = remain[1:]

	nameStep, ok := remain[0].(hcl.TraverseAttr)
	if !ok {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  diagSummary,
			Detail:   "The ephemeral component instance address must include the keyword \"ephemeral component\" followed by an ephemeral component name.",
			Subject:  remain[1].SourceRange().Ptr(),
		})
		return AbsEphemeralComponentInstance{}, remain, diags
	}
	remain = remain[1:]
	ephComponentAddr := EphemeralComponentInstance{
		EphemeralComponent: EphemeralComponent{Name: nameStep.Name},
	}

	if len(remain) > 0 {
		switch instStep := remain[0].(type) {
		case hcl.TraverseIndex:
			var err error
			ephComponentAddr.Key, err = addrs.ParseInstanceKey(instStep.Key)
			if err != nil {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  diagSummary,
					Detail:   fmt.Sprintf("Invalid instance key: %s.", err),
					Subject:  instStep.SourceRange().Ptr(),
				})
				return AbsEphemeralComponentInstance{}, remain, diags
			}

			remain = remain[1:]
		case hcl.TraverseSplat:
			ephComponentAddr.Key = addrs.WildcardKey
			remain = remain[1:]
		}
	}

	return AbsEphemeralComponentInstance{
		Stack: stackInst,
		Item:  ephComponentAddr,
	}, remain, diags
}
