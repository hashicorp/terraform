package exprstress

import (
	"fmt"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/lang"
	"github.com/hashicorp/terraform/tfdiags"
)

var testRefValues = map[string]cty.Value{
	"string": cty.StringVal("from reference"),
	"unknown": cty.ObjectVal(map[string]cty.Value{
		"any":          cty.DynamicVal,
		"string":       cty.UnknownVal(cty.String),
		"number":       cty.UnknownVal(cty.Number),
		"bool":         cty.UnknownVal(cty.Bool),
		"empty_object": cty.UnknownVal(cty.EmptyObject),
		"object":       cty.UnknownVal(cty.Object(map[string]cty.Type{"a": cty.String})),
	}),
	"sensitive": cty.ObjectVal(map[string]cty.Value{
		"passphrase":   cty.StringVal("my voice is my passport").Mark("sensitive"),
		"zero":         cty.Zero.Mark("sensitive"),
		"true":         cty.True.Mark("sensitive"),
		"false":        cty.False.Mark("sensitive"),
		"empty_object": cty.EmptyObjectVal.Mark("sensitive"),
		"object": cty.ObjectVal(map[string]cty.Value{
			"a": cty.StringVal("reindeer flotilla"),
		}).Mark("sensitive"),
	}),
	"null": cty.ObjectVal(map[string]cty.Value{
		"unknown":      cty.NullVal(cty.DynamicPseudoType),
		"string":       cty.NullVal(cty.String),
		"number":       cty.NullVal(cty.Number),
		"bool":         cty.NullVal(cty.Bool),
		"empty_object": cty.NullVal(cty.EmptyObject),
		"object":       cty.NullVal(cty.Object(map[string]cty.Type{"a": cty.String})),
	}),
	"with_unknowns": cty.ObjectVal(map[string]cty.Value{
		"set": cty.SetVal([]cty.Value{
			cty.StringVal("a"),
			cty.UnknownVal(cty.String),
		}),
		"list": cty.ListVal([]cty.Value{
			cty.StringVal("a"),
			cty.UnknownVal(cty.String),
		}),
		"map": cty.MapVal(map[string]cty.Value{
			"a": cty.StringVal("a"),
			"b": cty.UnknownVal(cty.String),
		}),
	}),
}

// testDataImpl is an implementation of lang.Data that represents some fake objects
// we can use in our generated test cases when we're testing references to
// named values.
//
// This is not a fully-comprehensive Data implementation because our goal with
// exprstress is to test the evaluation behavior for various expression
// features, not to test the scope-building logic. We can extend this as
// needed to provide a variety of different kinds of values to include in
// expressions, but will only do that in service of testing the behaviors
// of language operators and functions.
type testDataImpl struct {
	LocalValues map[string]cty.Value
}

var testData = &testDataImpl{
	LocalValues: testRefValues,
}

var _ lang.Data = (*testDataImpl)(nil)

const errSummaryNotSupported = "Not supported in expression stress tests"

func (d *testDataImpl) StaticValidateReferences(refs []*addrs.Reference, self addrs.Referenceable) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	for _, ref := range refs {
		switch ref.Subject.(type) {
		case addrs.LocalValue:
			// OK
		default:
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  errSummaryNotSupported,
				Detail:   fmt.Sprintf("There is no %s defined for expression stress tests", ref.Subject),
				Subject:  ref.SourceRange.ToHCL().Ptr(),
			})
		}
	}
	return diags
}

func (d *testDataImpl) GetCountAttr(addr addrs.CountAttr, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	diags = diags.Append(&hcl.Diagnostic{
		Severity: hcl.DiagError,
		Summary:  errSummaryNotSupported,
		Detail:   "The 'count' object is not available for expression stress tests",
		Subject:  rng.ToHCL().Ptr(),
	})
	return cty.DynamicVal, diags
}

func (d *testDataImpl) GetForEachAttr(addr addrs.ForEachAttr, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	diags = diags.Append(&hcl.Diagnostic{
		Severity: hcl.DiagError,
		Summary:  errSummaryNotSupported,
		Detail:   "The 'each' object is not available for expression stress tests",
		Subject:  rng.ToHCL().Ptr(),
	})
	return cty.DynamicVal, diags
}

func (d *testDataImpl) GetResource(addr addrs.Resource, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	diags = diags.Append(&hcl.Diagnostic{
		Severity: hcl.DiagError,
		Summary:  errSummaryNotSupported,
		Detail:   "Resource objects are not available for expression stress tests",
		Subject:  rng.ToHCL().Ptr(),
	})
	return cty.DynamicVal, diags
}

func (d *testDataImpl) GetLocalValue(addr addrs.LocalValue, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	val, ok := d.LocalValues[addr.Name]
	if !ok {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  errSummaryNotSupported,
			Detail:   fmt.Sprintf("There is no %s defined for expression stress tests", addr),
			Subject:  rng.ToHCL().Ptr(),
		})
		return cty.DynamicVal, diags
	}

	return val, diags
}

func (d *testDataImpl) GetModule(addr addrs.ModuleCall, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	diags = diags.Append(&hcl.Diagnostic{
		Severity: hcl.DiagError,
		Summary:  errSummaryNotSupported,
		Detail:   "Module objects are not available for expression stress tests",
		Subject:  rng.ToHCL().Ptr(),
	})
	return cty.DynamicVal, diags
}

func (d *testDataImpl) GetPathAttr(addr addrs.PathAttr, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	diags = diags.Append(&hcl.Diagnostic{
		Severity: hcl.DiagError,
		Summary:  errSummaryNotSupported,
		Detail:   "The 'path' object is not available for expression stress tests",
		Subject:  rng.ToHCL().Ptr(),
	})
	return cty.DynamicVal, diags
}

func (d *testDataImpl) GetTerraformAttr(addr addrs.TerraformAttr, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	diags = diags.Append(&hcl.Diagnostic{
		Severity: hcl.DiagError,
		Summary:  errSummaryNotSupported,
		Detail:   "The 'terraform' object is not available for expression stress tests",
		Subject:  rng.ToHCL().Ptr(),
	})
	return cty.DynamicVal, diags
}

func (d *testDataImpl) GetInputVariable(addr addrs.InputVariable, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	diags = diags.Append(&hcl.Diagnostic{
		Severity: hcl.DiagError,
		Summary:  errSummaryNotSupported,
		Detail:   "Input variables are not available for expression stress tests",
		Subject:  rng.ToHCL().Ptr(),
	})
	return cty.DynamicVal, diags
}
