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

// EphemeralModule is the address of an "ephemeral_module" block within a stack config.
type EphemeralModule struct {
	Name string
}

func (EphemeralModule) referenceableSigil()   {}
func (EphemeralModule) inStackConfigSigil()   {}
func (EphemeralModule) inStackInstanceSigil() {}

func (c EphemeralModule) String() string {
	return "ephemeral_module." + c.Name
}

func (c EphemeralModule) UniqueKey() collections.UniqueKey[EphemeralModule] {
	return c
}

// An EphemeralModule is its own [collections.UniqueKey].
func (EphemeralModule) IsUniqueKey(EphemeralModule) {}

// ConfigEphemeralModule places a [EphemeralModule] in the context of a particular [Stack].
type ConfigEphemeralModule = InStackConfig[EphemeralModule]

// AbsEphemeralModule places a [EphemeralModule] in the context of a particular [StackInstance].
type AbsEphemeralModule = InStackInstance[EphemeralModule]

func AbsEphemeralModuleToInstance(ist AbsEphemeralModule, ik addrs.InstanceKey) AbsEphemeralModuleInstance {
	return AbsEphemeralModuleInstance{
		Stack: ist.Stack,
		Item: EphemeralModuleInstance{
			EphemeralModule: ist.Item,
			Key:                ik,
		},
	}
}

// EphemeralModuleInstance is the address of a dynamic instance of an ephemeral module.
type EphemeralModuleInstance struct {
	EphemeralModule EphemeralModule
	Key                addrs.InstanceKey
}

func (EphemeralModuleInstance) inStackConfigSigil()   {}
func (EphemeralModuleInstance) inStackInstanceSigil() {}

func (c EphemeralModuleInstance) String() string {
	if c.Key == nil {
		return c.EphemeralModule.String()
	}
	return c.EphemeralModule.String() + c.Key.String()
}

func (c EphemeralModuleInstance) UniqueKey() collections.UniqueKey[EphemeralModuleInstance] {
	return c
}

// A EphemeralModuleInstance is its own [collections.UniqueKey].
func (EphemeralModuleInstance) IsUniqueKey(EphemeralModuleInstance) {}

// ConfigEphemeralModuleInstance places a [EphemeralModuleInstance] in the context of a
// particular [Stack].
type ConfigEphemeralModuleInstance = InStackConfig[EphemeralModuleInstance]

// AbsEphemeralModuleInstance places a [EphemeralModuleInstance] in the context of a
// particular [StackInstance].
type AbsEphemeralModuleInstance = InStackInstance[EphemeralModuleInstance]

func ConfigEphemeralModuleForAbsInstance(instAddr AbsEphemeralModuleInstance) ConfigEphemeralModule {
	configInst := ConfigForAbs(instAddr) // a ConfigEphemeralModuleInstance
	return ConfigEphemeralModule{
		Stack: configInst.Stack,
		Item: EphemeralModule{
			Name: configInst.Item.EphemeralModule.Name,
		},
	}
}

func ParseAbsEphemeralModuleInstance(traversal hcl.Traversal) (AbsEphemeralModuleInstance, tfdiags.Diagnostics) {
	inst, remain, diags := ParseAbsEphemeralModuleInstanceOnly(traversal)
	if diags.HasErrors() {
		return AbsEphemeralModuleInstance{}, diags
	}

	if len(remain) > 0 {
		// Then we have some remaining traversal steps that weren't consumed
		// by the ephemeral module instance address itself, which is an error when the
		// caller is using this function.
		rng := remain.SourceRange()
		// if "remain" is empty then the source range would be zero length,
		// and so we'll use the original traversal instead.
		if len(remain) == 0 {
			rng = traversal.SourceRange()
		}
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid ephemeral module instance address",
			Detail:   "The ephemeral module instance address must include the keyword \"ephemeral_module\" followed by an ephemeral module name.",
			Subject:  &rng,
		})
		return AbsEphemeralModuleInstance{}, diags
	}

	return inst, diags
}

func ParseAbsEphemeralModuleInstanceStr(s string) (AbsEphemeralModuleInstance, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	traversal, hclDiags := hclsyntax.ParseTraversalAbs([]byte(s), "", hcl.InitialPos)
	diags = diags.Append(hclDiags)
	if diags.HasErrors() {
		return AbsEphemeralModuleInstance{}, diags
	}

	ret, moreDiags := ParseAbsEphemeralModuleInstance(traversal)
	diags = diags.Append(moreDiags)
	return ret, diags
}

func ParsePartialEphemeralModuleInstanceStr(s string) (AbsEphemeralModuleInstance, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	traversal, hclDiags := hclsyntax.ParseTraversalPartial([]byte(s), "", hcl.InitialPos)
	diags = diags.Append(hclDiags)
	if diags.HasErrors() {
		return AbsEphemeralModuleInstance{}, diags
	}

	ret, moreDiags := ParseAbsEphemeralModuleInstance(traversal)
	diags = diags.Append(moreDiags)
	return ret, diags
}

func ParseAbsEphemeralModuleInstanceStrOnly(s string) (AbsEphemeralModuleInstance, hcl.Traversal, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	traversal, hclDiags := hclsyntax.ParseTraversalPartial([]byte(s), "", hcl.InitialPos)
	diags = diags.Append(hclDiags)
	if diags.HasErrors() {
		return AbsEphemeralModuleInstance{}, traversal, diags
	}

	ret, rest, moreDiags := ParseAbsEphemeralModuleInstanceOnly(traversal)
	diags = diags.Append(moreDiags)
	return ret, rest, diags
}

func ParseAbsEphemeralModuleInstanceOnly(traversal hcl.Traversal) (AbsEphemeralModuleInstance, hcl.Traversal, tfdiags.Diagnostics) {
	if traversal.IsRelative() {
		// This is always a caller bug: caller must only pass absolute
		// traversals in here.
		panic("ParseAbsEphemeralModuleInstanceOnly with relative traversal")
	}

	stackInst, remain, diags := parseInStackInstancePrefix(traversal)
	if diags.HasErrors() {
		return AbsEphemeralModuleInstance{}, remain, diags
	}

	// "remain" should now be the keyword "ephemeral_module" followed by a valid
	// ephemeral module name, optionally followed by an instance key.
	const diagSummary = "Invalid ephemeral module instance address"

	if kwStep, ok := remain[0].(hcl.TraverseAttr); !ok || kwStep.Name != "ephemeral_module" {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  diagSummary,
			Detail:   "The ephemeral module instance address must include the keyword \"ephemeral_module\" followed by an ephemeral module name.",
			Subject:  remain[0].SourceRange().Ptr(),
		})
		return AbsEphemeralModuleInstance{}, remain, diags
	}
	remain = remain[1:]

	nameStep, ok := remain[0].(hcl.TraverseAttr)
	if !ok {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  diagSummary,
			Detail:   "The ephemeral module instance address must include the keyword \"ephemeral module\" followed by an ephemeral module name.",
			Subject:  remain[1].SourceRange().Ptr(),
		})
		return AbsEphemeralModuleInstance{}, remain, diags
	}
	remain = remain[1:]
	ephModuleAddr := EphemeralModuleInstance{
		EphemeralModule: EphemeralModule{Name: nameStep.Name},
	}

	if len(remain) > 0 {
		switch instStep := remain[0].(type) {
		case hcl.TraverseIndex:
			var err error
			ephModuleAddr.Key, err = addrs.ParseInstanceKey(instStep.Key)
			if err != nil {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  diagSummary,
					Detail:   fmt.Sprintf("Invalid instance key: %s.", err),
					Subject:  instStep.SourceRange().Ptr(),
				})
				return AbsEphemeralModuleInstance{}, remain, diags
			}

			remain = remain[1:]
		case hcl.TraverseSplat:
			ephModuleAddr.Key = addrs.WildcardKey
			remain = remain[1:]
		}
	}

	return AbsEphemeralModuleInstance{
		Stack: stackInst,
		Item:  ephModuleAddr,
	}, remain, diags
}
