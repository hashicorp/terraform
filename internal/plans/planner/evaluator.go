package planner

import (
	"fmt"
	"sync"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/didyoumean"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/lang"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

type evaluator struct {
	planner *planner

	modules   map[addrs.UniqueKey]*evaluationDataModule
	modulesMu sync.Mutex
}

func (e *evaluator) DataForModuleInstance(addr addrs.ModuleInstance) *evaluationDataModule {
	e.modulesMu.Lock()
	key := addr.UniqueKey()
	if e.modules[key] == nil {
		e.modules[key] = &evaluationDataModule{
			evaluator:  e,
			planner:    e.planner,
			moduleAddr: addr,
		}
	}
	ret := e.modules[key]
	e.modulesMu.Unlock()
	return ret
}

// evaluationDataModule is an implementation of lang.Data that returns data
// for a particular module instance.
type evaluationDataModule struct {
	evaluator *evaluator
	planner   *planner

	moduleAddr addrs.ModuleInstance
}

var _ lang.Data = (*evaluationDataModule)(nil)

func (ed *evaluationDataModule) ForObjectInstance(repData instances.RepetitionData) *evaluationDataInstance {
	return &evaluationDataInstance{
		evaluationDataModule: ed,
		repData:              repData,
	}
}

func (d *evaluationDataModule) GetCountAttr(addr addrs.CountAttr, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	switch addr.Name {

	case "index":
		// There's no index available in an evaluationDataModule.
		// (should be using evaluationDataIndex for instance-sensitive contexts)
		d.planner.AddDiagnostics(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  `Reference to "count" in non-counted context`,
			Detail:   `The "count" object can only be used in "module", "resource", and "data" blocks, and only when the "count" argument is set.`,
			Subject:  rng.ToHCL().Ptr(),
		})
		return cty.UnknownVal(cty.Number), nil

	default:
		d.planner.AddDiagnostics(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  `Invalid "count" attribute`,
			Detail:   fmt.Sprintf(`The "count" object does not have an attribute named %q. The only supported attribute is count.index, which is the index of each instance of a resource block that has the "count" argument set.`, addr.Name),
			Subject:  rng.ToHCL().Ptr(),
		})
		return cty.DynamicVal, nil
	}
}

func (d *evaluationDataModule) GetForEachAttr(addr addrs.ForEachAttr, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	switch addr.Name {

	case "key", "value":
		// There's no key or value available in an evaluationDataModule.
		// (should be using evaluationDataIndex for instance-sensitive contexts)
		d.planner.AddDiagnostics(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  `Reference to "each" in context without for_each`,
			Detail:   `The "each" object can be used only in "module" or "resource" blocks, and only when the "for_each" argument is set.`,
			Subject:  rng.ToHCL().Ptr(),
		})
		switch addr.Name {
		case "key":
			return cty.UnknownVal(cty.String), nil
		default:
			return cty.DynamicVal, nil
		}
	default:
		d.planner.AddDiagnostics(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  `Invalid "each" attribute`,
			Detail:   fmt.Sprintf(`The "each" object does not have an attribute named %q. The supported attributes are each.key and each.value, the current key and value pair of the "for_each" attribute set.`, addr.Name),
			Subject:  rng.ToHCL().Ptr(),
		})
		return cty.DynamicVal, nil
	}
}

func (d *evaluationDataModule) GetInputVariable(addr addrs.InputVariable, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	v := d.planner.InputVariable(addr.Absolute(d.moduleAddr))

	if !v.IsDeclared() {
		moduleConfig := v.ModuleInstance().Call().InConfig().ContentConfig()
		var suggestions []string
		for k := range moduleConfig.Variables {
			suggestions = append(suggestions, k)
		}
		suggestion := didyoumean.NameSuggestion(addr.Name, suggestions)
		if suggestion != "" {
			suggestion = fmt.Sprintf(" Did you mean %q?", suggestion)
		} else {
			suggestion = fmt.Sprintf(" This variable can be declared with a variable %q {} block.", addr.Name)
		}

		d.planner.AddDiagnostics(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  `Reference to undeclared input variable`,
			Detail:   fmt.Sprintf(`An input variable with the name %q has not been declared.%s`, addr.Name, suggestion),
			Subject:  rng.ToHCL().Ptr(),
		})
		return cty.DynamicVal, nil
	}

	return v.Value(), nil
}

func (d *evaluationDataModule) GetLocalValue(addr addrs.LocalValue, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	v := d.planner.LocalValue(addr.Absolute(d.moduleAddr))

	if !v.IsDeclared() {
		moduleConfig := v.ModuleInstance().Call().InConfig().ContentConfig()
		var suggestions []string
		for k := range moduleConfig.Locals {
			suggestions = append(suggestions, k)
		}
		suggestion := didyoumean.NameSuggestion(addr.Name, suggestions)
		if suggestion != "" {
			suggestion = fmt.Sprintf(" Did you mean %q?", suggestion)
		} else {
			suggestion = " This variable can be declared inside a \"locals\" block."
		}

		d.planner.AddDiagnostics(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  `Reference to undeclared local value`,
			Detail:   fmt.Sprintf(`An local value with the name %q is not defined in this module.%s`, addr.Name, suggestion),
			Subject:  rng.ToHCL().Ptr(),
		})
		return cty.DynamicVal, nil
	}

	return v.Value(d.planner.coalescedCtx), nil
}

func (d *evaluationDataModule) GetModule(addr addrs.ModuleCall, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	// TODO: Implement
	return cty.DynamicVal, nil
}

func (d *evaluationDataModule) GetPathAttr(addr addrs.PathAttr, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	// TODO: Implement
	return cty.DynamicVal, nil
}

func (d *evaluationDataModule) GetResource(addr addrs.Resource, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	r := d.planner.Resource(addr.Absolute(d.moduleAddr))

	if !r.InConfig().IsDeclared() {
		d.planner.AddDiagnostics(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  `Reference to undeclared resource`,
			Detail:   fmt.Sprintf(`A resource %q %q has not been declared in %s`, addr.Type, addr.Name, moduleDisplayAddr(d.moduleAddr)),
			Subject:  rng.ToHCL().Ptr(),
		})
		return cty.DynamicVal, nil

	}

	return r.PlannedNewValue(d.planner.coalescedCtx), nil
}

func (d *evaluationDataModule) GetTerraformAttr(addr addrs.TerraformAttr, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	// TODO: Implement
	return cty.DynamicVal, nil
}

func (d *evaluationDataModule) StaticValidateReferences(refs []*addrs.Reference, self addrs.Referenceable) tfdiags.Diagnostics {
	// TODO: Implement
	return nil
}

// evaluationDataInstance extends evaluationDataModule with count.index,
// each.key, and each.value values for a particular instance.
//
// Unlike evaluationDataModule itself, evaluationDataInstance doesn't do any
// caching of the count and for_each symbols it additionally supports, under
// the assumption that the config for a particular resource instance will only
// be used once during planning anyway.
type evaluationDataInstance struct {
	*evaluationDataModule
	repData instances.RepetitionData
}

var _ lang.Data = (*evaluationDataInstance)(nil)

func (d *evaluationDataInstance) GetCountAttr(addr addrs.CountAttr, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	switch addr.Name {

	case "index":
		idxVal := d.repData.CountIndex
		if idxVal == cty.NilVal {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  `Reference to "count" in non-counted context`,
				Detail:   `The "count" object can only be used in "module", "resource", and "data" blocks, and only when the "count" argument is set.`,
				Subject:  rng.ToHCL().Ptr(),
			})
			return cty.UnknownVal(cty.Number), diags
		}
		return idxVal, diags

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

func (d *evaluationDataInstance) GetForEachAttr(addr addrs.ForEachAttr, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	var returnVal cty.Value
	switch addr.Name {

	case "key":
		returnVal = d.repData.EachKey
	case "value":
		returnVal = d.repData.EachValue

		if returnVal == cty.NilVal {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  `each.value cannot be used in this context`,
				Detail:   `A reference to "each.value" has been used in a context in which it unavailable, such as when the configuration no longer contains the value in its "for_each" expression. Remove this reference to each.value in your configuration to work around this error.`,
				Subject:  rng.ToHCL().Ptr(),
			})
			return cty.UnknownVal(cty.DynamicPseudoType), diags
		}
	default:
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  `Invalid "each" attribute`,
			Detail:   fmt.Sprintf(`The "each" object does not have an attribute named %q. The supported attributes are each.key and each.value, the current key and value pair of the "for_each" attribute set.`, addr.Name),
			Subject:  rng.ToHCL().Ptr(),
		})
		return cty.DynamicVal, diags
	}

	if returnVal == cty.NilVal {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  `Reference to "each" in context without for_each`,
			Detail:   `The "each" object can be used only in "module" or "resource" blocks, and only when the "for_each" argument is set.`,
			Subject:  rng.ToHCL().Ptr(),
		})
		return cty.UnknownVal(cty.DynamicPseudoType), diags
	}
	return returnVal, diags
}

// moduleDisplayAddr returns a string describing the given module instance
// address that is appropriate for returning to users in situations where the
// root module is possible. Specifically, it returns "the root module" if the
// root module instance is given, or a string representation of the module
// address otherwise.
func moduleDisplayAddr(addr addrs.ModuleInstance) string {
	switch {
	case addr.IsRoot():
		return "the root module"
	default:
		return addr.String()
	}
}
