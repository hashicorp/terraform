package lang

import (
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

type dataForTests struct {
	CountAttrs        map[string]cty.Value
	ResourceInstances map[string]cty.Value
	LocalValues       map[string]cty.Value
	Modules           map[string]cty.Value
	PathAttrs         map[string]cty.Value
	TerraformAttrs    map[string]cty.Value
	InputVariables    map[string]cty.Value
}

var _ Data = &dataForTests{}

func (d *dataForTests) StaticValidateReferences(refs []*addrs.Reference, self addrs.Referenceable) tfdiags.Diagnostics {
	return nil // does nothing in this stub implementation
}

func (d *dataForTests) GetCountAttr(addr addrs.CountAttr, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	return d.CountAttrs[addr.Name], nil
}

func (d *dataForTests) GetResourceInstance(addr addrs.ResourceInstance, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	return d.ResourceInstances[addr.String()], nil
}

func (d *dataForTests) GetInputVariable(addr addrs.InputVariable, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	return d.InputVariables[addr.Name], nil
}

func (d *dataForTests) GetLocalValue(addr addrs.LocalValue, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	return d.LocalValues[addr.Name], nil
}

func (d *dataForTests) GetModuleInstance(addr addrs.ModuleCallInstance, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	return d.Modules[addr.String()], nil
}

func (d *dataForTests) GetModuleInstanceOutput(addr addrs.ModuleCallOutput, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
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
