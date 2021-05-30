package configs

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

// exprIsNativeQuotedString determines whether the given expression looks like
// it's a quoted string in the HCL native syntax.
//
// This should be used sparingly only for situations where our legacy HCL
// decoding would've expected a keyword or reference in quotes but our new
// decoding expects the keyword or reference to be provided directly as
// an identifier-based expression.
func exprIsNativeQuotedString(expr hcl.Expression) bool {
	_, ok := expr.(*hclsyntax.TemplateExpr)
	return ok
}

// schemaForOverrides takes a *hcl.BodySchema and produces a new one that is
// equivalent except that any required attributes are forced to not be required.
//
// This is useful for dealing with "override" config files, which are allowed
// to omit things that they don't wish to override from the main configuration.
//
// The returned schema may have some pointers in common with the given schema,
// so neither the given schema nor the returned schema should be modified after
// using this function in order to avoid confusion.
//
// Overrides are rarely used, so it's recommended to just create the override
// schema on the fly only when it's needed, rather than storing it in a global
// variable as we tend to do for a primary schema.
func schemaForOverrides(schema *hcl.BodySchema) *hcl.BodySchema {
	ret := &hcl.BodySchema{
		Attributes: make([]hcl.AttributeSchema, len(schema.Attributes)),
		Blocks:     schema.Blocks,
	}

	for i, attrS := range schema.Attributes {
		ret.Attributes[i] = attrS
		ret.Attributes[i].Required = false
	}

	return ret
}

// schemaWithDynamic takes a *hcl.BodySchema and produces a new one that
// is equivalent except that it accepts an additional block type "dynamic" with
// a single label, used to recognize usage of the HCL dynamic block extension.
func schemaWithDynamic(schema *hcl.BodySchema) *hcl.BodySchema {
	ret := &hcl.BodySchema{
		Attributes: schema.Attributes,
		Blocks:     make([]hcl.BlockHeaderSchema, len(schema.Blocks), len(schema.Blocks)+1),
	}

	copy(ret.Blocks, schema.Blocks)
	ret.Blocks = append(ret.Blocks, hcl.BlockHeaderSchema{
		Type:       "dynamic",
		LabelNames: []string{"type"},
	})

	return ret
}
