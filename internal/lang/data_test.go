// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package lang

import (
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type dataForTests struct {
	CountAttrs     map[string]cty.Value
	ForEachAttrs   map[string]cty.Value
	Resources      map[string]cty.Value
	LocalValues    map[string]cty.Value
	OutputValues   map[string]cty.Value
	Modules        map[string]cty.Value
	PathAttrs      map[string]cty.Value
	TerraformAttrs map[string]cty.Value
	InputVariables map[string]cty.Value
	CheckBlocks    map[string]cty.Value
}

var _ Data = &dataForTests{}

func (d *dataForTests) StaticValidateReferences(refs []*addrs.Reference, self addrs.Referenceable, source addrs.Referenceable) tfdiags.Diagnostics {
	return nil // does nothing in this stub implementation
}

func (d *dataForTests) GetCountAttr(addr addrs.CountAttr, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	return d.CountAttrs[addr.Name], nil
}

func (d *dataForTests) GetForEachAttr(addr addrs.ForEachAttr, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	return d.ForEachAttrs[addr.Name], nil
}

func (d *dataForTests) GetResource(addr addrs.Resource, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	return d.Resources[addr.String()], nil
}

func (d *dataForTests) GetInputVariable(addr addrs.InputVariable, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	return d.InputVariables[addr.Name], nil
}

func (d *dataForTests) GetLocalValue(addr addrs.LocalValue, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	return d.LocalValues[addr.Name], nil
}

func (d *dataForTests) GetModule(addr addrs.ModuleCall, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	return d.Modules[addr.String()], nil
}

func (d *dataForTests) GetModuleInstanceOutput(addr addrs.ModuleCallInstanceOutput, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	// This will panic if the module object does not have the requested attribute
	obj := d.Modules[addr.Call.String()]
	return obj.GetAttr(addr.Name), nil
}

func (d *dataForTests) GetPathAttr(addr addrs.PathAttr, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	return d.PathAttrs[addr.Name], nil
}

func (d *dataForTests) GetTerraformAttr(addr addrs.TerraformAttr, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	return d.TerraformAttrs[addr.Name], nil
}

func (d *dataForTests) GetOutput(addr addrs.OutputValue, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	return d.OutputValues[addr.Name], nil
}

func (d *dataForTests) GetCheckBlock(addr addrs.Check, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	return d.CheckBlocks[addr.Name], nil
}
