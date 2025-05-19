// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackaddrs

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/hashicorp/terraform/internal/collections"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type InputVariable struct {
	Name string
}

func (InputVariable) referenceableSigil()   {}
func (InputVariable) inStackConfigSigil()   {}
func (InputVariable) inStackInstanceSigil() {}

func (v InputVariable) String() string {
	return "var." + v.Name
}

func (v InputVariable) UniqueKey() collections.UniqueKey[InputVariable] {
	return v
}

// An InputVariable is its own [collections.UniqueKey].
func (InputVariable) IsUniqueKey(InputVariable) {}

// ConfigInputVariable places an [InputVariable] in the context of a particular [Stack].
type ConfigInputVariable = InStackConfig[InputVariable]

// AbsInputVariable places an [InputVariable] in the context of a particular [StackInstance].
type AbsInputVariable = InStackInstance[InputVariable]

func ParseAbsInputVariableStr(s string) (AbsInputVariable, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	traversal, hclDiags := hclsyntax.ParseTraversalAbs([]byte(s), "", hcl.InitialPos)
	diags = diags.Append(hclDiags)
	if diags.HasErrors() {
		return AbsInputVariable{}, diags
	}

	ret, moreDiags := ParseAbsInputVariable(traversal)
	return ret, diags.Append(moreDiags)
}

func ParseAbsInputVariable(traversal hcl.Traversal) (AbsInputVariable, tfdiags.Diagnostics) {
	if traversal.IsRelative() {
		// This is always a caller bug: caller must only pass absolute
		// traversals in here.
		panic("ParseAbsInputVariable with relative traversal")
	}

	stackInst, remain, diags := parseInStackInstancePrefix(traversal)
	if diags.HasErrors() {
		return AbsInputVariable{}, diags
	}

	if len(remain) != 2 {
		// it must be output.name, no more and no less.
		return AbsInputVariable{}, diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid input variable address",
			Detail:   "The input variable address must be the keyword \"var\" followed by a variable name.",
			Subject:  traversal.SourceRange().Ptr(),
		})
	}

	if kwStep, ok := remain[0].(hcl.TraverseAttr); !ok || kwStep.Name != "var" {
		return AbsInputVariable{}, diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid input variable address",
			Detail:   "The input variable address must be the keyword \"var\" followed by a variable name.",
			Subject:  remain[0].SourceRange().Ptr(),
		})
	}

	nameStep, ok := remain[1].(hcl.TraverseAttr)
	if !ok {
		return AbsInputVariable{}, diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid input variable address",
			Detail:   "The input variable address must be the keyword \"var\" followed by a variable name.",
			Subject:  remain[1].SourceRange().Ptr(),
		})
	}

	return AbsInputVariable{
		Stack: stackInst,
		Item: InputVariable{
			Name: nameStep.Name,
		},
	}, diags
}
