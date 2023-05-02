// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package configs

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

// SynthBody produces a synthetic hcl.Body that behaves as if it had attributes
// corresponding to the elements given in the values map.
//
// This is useful in situations where, for example, values provided on the
// command line can override values given in configuration, using MergeBodies.
//
// The given filename is used in case any diagnostics are returned. Since
// the created body is synthetic, it is likely that this will not be a "real"
// filename. For example, if from a command line argument it could be
// a representation of that argument's name, such as "-var=...".
func SynthBody(filename string, values map[string]cty.Value) hcl.Body {
	return synthBody{
		Filename: filename,
		Values:   values,
	}
}

type synthBody struct {
	Filename string
	Values   map[string]cty.Value
}

func (b synthBody) Content(schema *hcl.BodySchema) (*hcl.BodyContent, hcl.Diagnostics) {
	content, remain, diags := b.PartialContent(schema)
	remainS := remain.(synthBody)
	for name := range remainS.Values {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Unsupported attribute",
			Detail:   fmt.Sprintf("An attribute named %q is not expected here.", name),
			Subject:  b.synthRange().Ptr(),
		})
	}
	return content, diags
}

func (b synthBody) PartialContent(schema *hcl.BodySchema) (*hcl.BodyContent, hcl.Body, hcl.Diagnostics) {
	var diags hcl.Diagnostics
	content := &hcl.BodyContent{
		Attributes:       make(hcl.Attributes),
		MissingItemRange: b.synthRange(),
	}

	remainValues := make(map[string]cty.Value)
	for attrName, val := range b.Values {
		remainValues[attrName] = val
	}

	for _, attrS := range schema.Attributes {
		delete(remainValues, attrS.Name)
		val, defined := b.Values[attrS.Name]
		if !defined {
			if attrS.Required {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Missing required attribute",
					Detail:   fmt.Sprintf("The attribute %q is required, but no definition was found.", attrS.Name),
					Subject:  b.synthRange().Ptr(),
				})
			}
			continue
		}
		content.Attributes[attrS.Name] = b.synthAttribute(attrS.Name, val)
	}

	// We just ignore blocks altogether, because this body type never has
	// nested blocks.

	remain := synthBody{
		Filename: b.Filename,
		Values:   remainValues,
	}

	return content, remain, diags
}

func (b synthBody) JustAttributes() (hcl.Attributes, hcl.Diagnostics) {
	ret := make(hcl.Attributes)
	for name, val := range b.Values {
		ret[name] = b.synthAttribute(name, val)
	}
	return ret, nil
}

func (b synthBody) MissingItemRange() hcl.Range {
	return b.synthRange()
}

func (b synthBody) synthAttribute(name string, val cty.Value) *hcl.Attribute {
	rng := b.synthRange()
	return &hcl.Attribute{
		Name: name,
		Expr: &hclsyntax.LiteralValueExpr{
			Val:      val,
			SrcRange: rng,
		},
		NameRange: rng,
		Range:     rng,
	}
}

func (b synthBody) synthRange() hcl.Range {
	return hcl.Range{
		Filename: b.Filename,
		Start:    hcl.Pos{Line: 1, Column: 1},
		End:      hcl.Pos{Line: 1, Column: 1},
	}
}
