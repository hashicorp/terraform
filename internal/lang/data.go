// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package lang

import (
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Data is an interface whose implementations can provide cty.Value
// representations of objects identified by referenceable addresses from
// the addrs package.
//
// This interface will grow each time a new type of reference is added, and so
// implementations outside of the Terraform codebases are not advised.
//
// Each method returns a suitable value and optionally some diagnostics. If the
// returned diagnostics contains errors then the type of the returned value is
// used to construct an unknown value of the same type which is then used in
// place of the requested object so that type checking can still proceed. In
// cases where it's not possible to even determine a suitable result type,
// cty.DynamicVal is returned along with errors describing the problem.
type Data interface {
	StaticValidateReferences(refs []*addrs.Reference, self addrs.Referenceable, source addrs.Referenceable) tfdiags.Diagnostics

	GetCountAttr(addrs.CountAttr, tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics)
	GetForEachAttr(addrs.ForEachAttr, tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics)
	GetResource(addrs.Resource, tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics)
	GetLocalValue(addrs.LocalValue, tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics)
	GetModule(addrs.ModuleCall, tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics)
	GetPathAttr(addrs.PathAttr, tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics)
	GetTerraformAttr(addrs.TerraformAttr, tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics)
	GetInputVariable(addrs.InputVariable, tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics)
	GetOutput(addrs.OutputValue, tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics)
	GetCheckBlock(addrs.Check, tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics)
	GetRunBlock(addrs.Run, tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics)
}
