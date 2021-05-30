package hcl2shim

import (
	"fmt"

	hcl2 "github.com/hashicorp/hcl/v2"
)

// SingleAttrBody is a weird implementation of hcl2.Body that acts as if
// it has a single attribute whose value is the given expression.
//
// This is used to shim Resource.RawCount and Output.RawConfig to behave
// more like they do in the old HCL loader.
type SingleAttrBody struct {
	Name string
	Expr hcl2.Expression
}

var _ hcl2.Body = SingleAttrBody{}

func (b SingleAttrBody) Content(schema *hcl2.BodySchema) (*hcl2.BodyContent, hcl2.Diagnostics) {
	content, all, diags := b.content(schema)
	if !all {
		// This should never happen because this body implementation should only
		// be used by code that is aware that it's using a single-attr body.
		diags = append(diags, &hcl2.Diagnostic{
			Severity: hcl2.DiagError,
			Summary:  "Invalid attribute",
			Detail:   fmt.Sprintf("The correct attribute name is %q.", b.Name),
			Subject:  b.Expr.Range().Ptr(),
		})
	}
	return content, diags
}

func (b SingleAttrBody) PartialContent(schema *hcl2.BodySchema) (*hcl2.BodyContent, hcl2.Body, hcl2.Diagnostics) {
	content, all, diags := b.content(schema)
	var remain hcl2.Body
	if all {
		// If the request matched the one attribute we represent, then the
		// remaining body is empty.
		remain = hcl2.EmptyBody()
	} else {
		remain = b
	}
	return content, remain, diags
}

func (b SingleAttrBody) content(schema *hcl2.BodySchema) (*hcl2.BodyContent, bool, hcl2.Diagnostics) {
	ret := &hcl2.BodyContent{}
	all := false
	var diags hcl2.Diagnostics

	for _, attrS := range schema.Attributes {
		if attrS.Name == b.Name {
			attrs, _ := b.JustAttributes()
			ret.Attributes = attrs
			all = true
		} else if attrS.Required {
			diags = append(diags, &hcl2.Diagnostic{
				Severity: hcl2.DiagError,
				Summary:  "Missing attribute",
				Detail:   fmt.Sprintf("The attribute %q is required.", attrS.Name),
				Subject:  b.Expr.Range().Ptr(),
			})
		}
	}

	return ret, all, diags
}

func (b SingleAttrBody) JustAttributes() (hcl2.Attributes, hcl2.Diagnostics) {
	return hcl2.Attributes{
		b.Name: {
			Expr:      b.Expr,
			Name:      b.Name,
			NameRange: b.Expr.Range(),
			Range:     b.Expr.Range(),
		},
	}, nil
}

func (b SingleAttrBody) MissingItemRange() hcl2.Range {
	return b.Expr.Range()
}
