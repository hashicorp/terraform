// Copyright IBM Corp. 2014, 2026
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

// ResourceInstanceData is an optional extension of [Data] for implementations
// which can find the value of a single resource instance without building the
// value of the entire containing resource. It's used to resolve "self"
// references, which always refer to a particular resource instance.
type ResourceInstanceData interface {
	// GetResourceInstance returns the value of the given resource instance,
	// which must be equal to the element GetResource would produce for that
	// instance, or the whole value for a resource with no instance key.
	//
	// The result must not depend on any other instance of the resource.
	// Instances aren't ordered relative to one another, so another instance
	// may not have been processed yet, and any problem GetResource would
	// report for it doesn't affect this one.
	//
	// If the value can't be determined from this instance alone, or this
	// instance has a problem GetResource would report, the second return
	// value is false and the caller must use GetResource instead.
	GetResourceInstance(addrs.ResourceInstance) (cty.Value, bool)
}
