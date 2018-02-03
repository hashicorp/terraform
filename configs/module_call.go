package configs

import (
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hcl/hclsyntax"
)

// ModuleCall represents a "module" block in a module or file.
type ModuleCall struct {
	Name string

	SourceAddr      string
	SourceAddrRange hcl.Range

	Config hcl.Body

	Version VersionConstraint

	Count   hcl.Expression
	ForEach hcl.Expression

	DependsOn []hcl.Traversal

	DeclRange hcl.Range
}

func decodeModuleBlock(block *hcl.Block) (*ModuleCall, hcl.Diagnostics) {
	mc := &ModuleCall{
		Name:      block.Labels[0],
		DeclRange: block.DefRange,
	}

	content, remain, diags := block.Body.PartialContent(moduleBlockSchema)
	mc.Config = remain

	if !hclsyntax.ValidIdentifier(mc.Name) {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid module instance name",
			Detail:   badIdentifierDetail,
			Subject:  &block.LabelRanges[0],
		})
	}

	if attr, exists := content.Attributes["source"]; exists {
		valDiags := gohcl.DecodeExpression(attr.Expr, nil, &mc.SourceAddr)
		diags = append(diags, valDiags...)
		mc.SourceAddrRange = attr.Expr.Range()
	}

	if attr, exists := content.Attributes["version"]; exists {
		var versionDiags hcl.Diagnostics
		mc.Version, versionDiags = decodeVersionConstraint(attr)
		diags = append(diags, versionDiags...)
	}

	if attr, exists := content.Attributes["count"]; exists {
		mc.Count = attr.Expr
	}

	if attr, exists := content.Attributes["for_each"]; exists {
		mc.ForEach = attr.Expr
	}

	if attr, exists := content.Attributes["depends_on"]; exists {
		deps, depsDiags := decodeDependsOn(attr)
		diags = append(diags, depsDiags...)
		mc.DependsOn = append(mc.DependsOn, deps...)
	}

	return mc, diags
}

var moduleBlockSchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		{
			Name:     "source",
			Required: true,
		},
		{
			Name: "version",
		},
		{
			Name: "count",
		},
		{
			Name: "for_each",
		},
		{
			Name: "depends_on",
		},
	},
}
