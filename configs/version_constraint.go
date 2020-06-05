package configs

import (
	"fmt"

	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"
)

// VersionConstraint represents a version constraint on some resource
// (e.g. Terraform Core, a provider, a module, ...) that carries with it
// a source range so that a helpful diagnostic can be printed in the event
// that a particular constraint does not match.
type VersionConstraint struct {
	Required  version.Constraints
	DeclRange hcl.Range
}

func decodeVersionConstraint(attr *hcl.Attribute) (VersionConstraint, hcl.Diagnostics) {
	ret := VersionConstraint{
		DeclRange: attr.Range,
	}

	val, diags := attr.Expr.Value(nil)
	if diags.HasErrors() {
		return ret, diags
	}
	var err error
	val, err = convert.Convert(val, cty.String)
	if err != nil {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid version constraint",
			Detail:   fmt.Sprintf("A string value is required for %s.", attr.Name),
			Subject:  attr.Expr.Range().Ptr(),
		})
		return ret, diags
	}

	if val.IsNull() {
		// A null version constraint is strange, but we'll just treat it
		// like an empty constraint set.
		return ret, diags
	}

	if !val.IsWhollyKnown() {
		// If there is a syntax error, HCL sets the value of the given attribute
		// to cty.DynamicVal. A diagnostic for the syntax error will already
		// bubble up, so we will move forward gracefully here.
		return ret, diags
	}

	constraintStr := val.AsString()
	constraints, err := version.NewConstraint(constraintStr)
	if err != nil {
		// NewConstraint doesn't return user-friendly errors, so we'll just
		// ignore the provided error and produce our own generic one.
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid version constraint",
			Detail:   "This string does not use correct version constraint syntax.", // Not very actionable :(
			Subject:  attr.Expr.Range().Ptr(),
		})
		return ret, diags
	}

	ret.Required = constraints
	return ret, diags
}
