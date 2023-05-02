// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package configs

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
)

// LoadValuesFile reads the file at the given path and parses it as a "values
// file", which is an HCL config file whose top-level attributes are treated
// as arbitrary key.value pairs.
//
// If the file cannot be read -- for example, if it does not exist -- then
// a nil map will be returned along with error diagnostics. Callers may wish
// to disregard the returned diagnostics in this case and instead generate
// their own error message(s) with additional context.
//
// If the returned diagnostics has errors when a non-nil map is returned
// then the map may be incomplete but should be valid enough for careful
// static analysis.
//
// This method wraps LoadHCLFile, and so it inherits the syntax selection
// behaviors documented for that method.
func (p *Parser) LoadValuesFile(path string) (map[string]cty.Value, hcl.Diagnostics) {
	body, diags := p.LoadHCLFile(path)
	if body == nil {
		return nil, diags
	}

	vals := make(map[string]cty.Value)
	attrs, attrDiags := body.JustAttributes()
	diags = append(diags, attrDiags...)
	if attrs == nil {
		return vals, diags
	}

	for name, attr := range attrs {
		val, valDiags := attr.Expr.Value(nil)
		diags = append(diags, valDiags...)
		vals[name] = val
	}

	return vals, diags
}
