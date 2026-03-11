// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/hcl/v2"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/lang"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// evaluationPlaceholderData is an implementation of lang.Data that deals
// with resolving references inside module prefixes whose full expansion
// isn't known yet, and thus returns placeholder values that represent
// only what we know to be true for all possible final module instances
// that could exist for the prefix.
type evaluationPlaceholderData struct {
	*evaluationData

	// ModulePath is the partially-expanded path through the dynamic module
	// tree to a set of possible module instances that share a common known
	// prefix.
	ModulePath addrs.PartialExpandedModule

	// CountAvailable is true if this data object is representing an evaluation
	// scope where the "count" symbol would be available.
	CountAvailable bool

	// EachAvailable is true if this data object is representing an evaluation
	// scope where the "each" symbol would be available.
	EachAvailable bool

	// Operation records the type of walk the evaluationStateData is being used
	// for.
	Operation walkOperation
}

// TODO: Historically we were inconsistent about whether static validation
// logic is implemented in Evaluator.StaticValidateReference or inline in
// methods of evaluationStateData, because the dedicated static validator
// came later.
//
// Some validation rules (and their associated error messages) have therefore
// ended up being duplicated between evaluationPlaceholderData and
// evaluationStateData. We've accepted that for now to avoid creating a bunch
// of churn in pre-existing code while adding support for partial expansion
// placeholders, but one day it would be nice to refactor this a little so
// that the division between these three units is a little clearer and so
// that all of the error checks are implemented in only one place each.

var _ lang.Data = (*evaluationPlaceholderData)(nil)

// GetCountAttr implements lang.Data.
func (d *evaluationPlaceholderData) GetCountAttr(addr addrs.CountAttr, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	switch addr.Name {

	case "index":
		if !d.CountAvailable {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  `Reference to "count" in non-counted context`,
				Detail:   `The "count" object can only be used in "module", "resource", and "data" blocks, and only when the "count" argument is set.`,
				Subject:  rng.ToHCL().Ptr(),
			})
		}
		// When we're under a partially-expanded prefix, the leaf instance
		// keys are never known because otherwise we'd be under a fully-known
		// prefix by definition. We do know it's always >= 0 and not null,
		// though.
		return cty.UnknownVal(cty.Number).Refine().
			NumberRangeLowerBound(cty.Zero, true).
			NotNull().
			NewValue(), diags

	default:
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  `Invalid "count" attribute`,
			Detail:   fmt.Sprintf(`The "count" object does not have an attribute named %q. The only supported attribute is count.index, which is the index of each instance of a resource block that has the "count" argument set.`, addr.Name),
			Subject:  rng.ToHCL().Ptr(),
		})
		return cty.DynamicVal, diags
	}
}

// GetForEachAttr implements lang.Data.
func (d *evaluationPlaceholderData) GetForEachAttr(addr addrs.ForEachAttr, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	// When we're under a partially-expanded prefix, the leaf instance
	// keys are never known because otherwise we'd be under a fully-known
	// prefix by definition. Therefore all return paths here produce unknown
	// values.

	if !d.EachAvailable {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  `Reference to "each" in context without for_each`,
			Detail:   `The "each" object can be used only in "module" or "resource" blocks, and only when the "for_each" argument is set.`,
			Subject:  rng.ToHCL().Ptr(),
		})
		return cty.UnknownVal(cty.DynamicPseudoType), diags
	}

	switch addr.Name {

	case "key":
		// each.key is always a string and is never null
		return cty.UnknownVal(cty.String).RefineNotNull(), diags
	case "value":
		return cty.DynamicVal, diags
	default:
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  `Invalid "each" attribute`,
			Detail:   fmt.Sprintf(`The "each" object does not have an attribute named %q. The supported attributes are each.key and each.value, the current key and value pair of the "for_each" attribute set.`, addr.Name),
			Subject:  rng.ToHCL().Ptr(),
		})
		return cty.DynamicVal, diags
	}
}

// GetInputVariable implements lang.Data.
func (d *evaluationPlaceholderData) GetInputVariable(addr addrs.InputVariable, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	namedVals := d.Evaluator.NamedValues
	absAddr := addrs.ObjectInPartialExpandedModule(d.ModulePath, addr)
	return namedVals.GetInputVariablePlaceholder(absAddr), nil
}

// GetLocalValue implements lang.Data.
func (d *evaluationPlaceholderData) GetLocalValue(addr addrs.LocalValue, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	namedVals := d.Evaluator.NamedValues
	absAddr := addrs.ObjectInPartialExpandedModule(d.ModulePath, addr)
	return namedVals.GetLocalValuePlaceholder(absAddr), nil
}

// GetModule implements lang.Data.
func (d *evaluationPlaceholderData) GetModule(addr addrs.ModuleCall, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	// We'll reuse the evaluator's "static evaluate" logic to check that the
	// module call being referred to is even declared in the configuration,
	// since it returns a good-quality error message for that case that
	// we don't want to have to duplicate here.
	diags := d.Evaluator.StaticValidateReference(&addrs.Reference{
		Subject:     addr,
		SourceRange: rng,
	}, d.ModulePath.Module(), nil, nil)
	if diags.HasErrors() {
		return cty.DynamicVal, diags
	}

	callerCfg := d.Evaluator.Config.Descendant(d.ModulePath.Module())
	if callerCfg == nil {
		// Strange! The above StaticValidateReference should've failed if
		// the module we're in isn't even declared. But we'll just tolerate
		// it and return a very general placeholder.
		return cty.DynamicVal, diags
	}
	callCfg := callerCfg.Module.ModuleCalls[addr.Name]
	if callCfg == nil {
		// Again strange, for the same reason as just above.
		return cty.DynamicVal, diags
	}

	// Any module call under an unexpanded prefix has an unknown set of instance
	// keys itself by definition, unless that call isn't using count or for_each
	// at all and thus we know it has exactly one "no-key" instance.
	//
	// If we don't know the instance keys then we cannot predict anything about
	// the result, because module calls with repetition appear as either
	// object or tuple types and we cannot predict those types here.
	if callCfg.Count != nil || callCfg.ForEach != nil {
		return cty.DynamicVal, diags
	}

	// If we get down here then we know we have a single-instance module, and
	// so we can return a more specific placeholder object that has all of
	// the child module's declared output values represented, which could
	// then potentially allow detecting a downstream error referring to
	// an output value that doesn't actually exist.
	calledCfg := d.Evaluator.Config.Descendant(d.ModulePath.Module().Child(addr.Name))
	if calledCfg == nil {
		// This suggests that the config wasn't constructed correctly, since
		// there should always be a child config node for any module call,
		// but that's a "package configs" problem and so we'll just tolerate
		// it here for robustness.
		return cty.DynamicVal, diags
	}

	attrs := make(map[string]cty.Value, len(calledCfg.Module.Outputs))
	for name := range calledCfg.Module.Outputs {
		// Module output values are dynamically-typed, so we cannot
		// predict anything about their results until finalized.
		attrs[name] = cty.DynamicVal
	}
	return cty.ObjectVal(attrs), diags
}

// GetOutput implements lang.Data.
func (d *evaluationPlaceholderData) GetOutput(addr addrs.OutputValue, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	namedVals := d.Evaluator.NamedValues
	absAddr := addrs.ObjectInPartialExpandedModule(d.ModulePath, addr)
	return namedVals.GetOutputValuePlaceholder(absAddr), nil

}

// GetResource implements lang.Data.
func (d *evaluationPlaceholderData) GetResource(addrs.Resource, tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	// TODO: Once we've implemented the evaluation of placeholders for
	// deferred resources during the graph walk, we should return such
	// placeholders here where possible.
	//
	// However, for resources that use count or for_each we'd not be able
	// to predict anything more than cty.DynamicVal here anyway, since
	// we don't know the instance keys, and so that improvement would only
	// really help references to single-instance resources.
	return cty.DynamicVal, nil
}
