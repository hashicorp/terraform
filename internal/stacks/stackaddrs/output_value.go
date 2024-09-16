// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackaddrs

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/hashicorp/terraform/internal/collections"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type OutputValue struct {
	Name string
}

func (OutputValue) inStackConfigSigil()   {}
func (OutputValue) inStackInstanceSigil() {}

func (v OutputValue) String() string {
	return "output." + v.Name
}

func (v OutputValue) UniqueKey() collections.UniqueKey[OutputValue] {
	return v
}

// An OutputValue is its own [collections.UniqueKey].
func (OutputValue) IsUniqueKey(OutputValue) {}

// ConfigOutputValue places an [OutputValue] in the context of a particular [Stack].
type ConfigOutputValue = InStackConfig[OutputValue]

// AbsOutputValue places an [OutputValue] in the context of a particular [StackInstance].
type AbsOutputValue = InStackInstance[OutputValue]

func ParseAbsOutputValueStr(s string) (AbsOutputValue, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	traversal, hclDiags := hclsyntax.ParseTraversalAbs([]byte(s), "", hcl.InitialPos)
	diags = diags.Append(hclDiags)
	if diags.HasErrors() {
		return AbsOutputValue{}, diags
	}

	ret, moreDiags := ParseAbsOutputValue(traversal)
	return ret, diags.Append(moreDiags)
}

func ParseAbsOutputValue(traversal hcl.Traversal) (AbsOutputValue, tfdiags.Diagnostics) {
	if traversal.IsRelative() {
		// This is always a caller bug: caller must only pass absolute
		// traversals in here.
		panic("ParseAbsOutputValue with relative traversal")
	}

	stackInst, remain, diags := parseInStackInstancePrefix(traversal)
	if diags.HasErrors() {
		return AbsOutputValue{}, diags
	}

	if len(remain) != 2 {
		// it must be output.name, no more and no less.
		return AbsOutputValue{}, diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid output address",
			Detail:   "The output address must be the keyword \"output\" followed by an output name.",
			Subject:  traversal.SourceRange().Ptr(),
		})
	}

	if kwStep, ok := remain[0].(hcl.TraverseAttr); !ok || kwStep.Name != "output" {
		return AbsOutputValue{}, diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid output address",
			Detail:   "The output address must be the keyword \"output\" followed by an output name.",
			Subject:  remain[0].SourceRange().Ptr(),
		})
	}

	nameStep, ok := remain[1].(hcl.TraverseAttr)
	if !ok {
		return AbsOutputValue{}, diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid output address",
			Detail:   "The output address must be the keyword \"output\" followed by an output name.",
			Subject:  remain[1].SourceRange().Ptr(),
		})
	}

	return AbsOutputValue{
		Stack: stackInst,
		Item: OutputValue{
			Name: nameStep.Name,
		},
	}, diags
}
