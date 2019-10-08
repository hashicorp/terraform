package configs

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

// ModuleCall represents a "module" block in a module or file.
type ModuleCall struct {
	Name string

	SourceAddr      string
	SourceAddrRange hcl.Range
	SourceSet       bool

	Config hcl.Body

	Version VersionConstraint

	Count   hcl.Expression
	ForEach hcl.Expression

	Providers []PassedProviderConfig

	DependsOn []hcl.Traversal

	DeclRange hcl.Range
}

func decodeModuleBlock(block *hcl.Block, override bool) (*ModuleCall, hcl.Diagnostics) {
	mc := &ModuleCall{
		Name:      block.Labels[0],
		DeclRange: block.DefRange,
	}

	schema := moduleBlockSchema
	if override {
		schema = schemaForOverrides(schema)
	}

	content, remain, diags := block.Body.PartialContent(schema)
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
		mc.SourceSet = true
	}

	if attr, exists := content.Attributes["version"]; exists {
		var versionDiags hcl.Diagnostics
		mc.Version, versionDiags = decodeVersionConstraint(attr)
		diags = append(diags, versionDiags...)
	}

	if attr, exists := content.Attributes["count"]; exists {
		mc.Count = attr.Expr

		// We currently parse this, but don't yet do anything with it.
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Reserved argument name in module block",
			Detail:   fmt.Sprintf("The name %q is reserved for use in a future version of Terraform.", attr.Name),
			Subject:  &attr.NameRange,
		})
	}

	if attr, exists := content.Attributes["for_each"]; exists {
		mc.ForEach = attr.Expr

		// We currently parse this, but don't yet do anything with it.
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Reserved argument name in module block",
			Detail:   fmt.Sprintf("The name %q is reserved for use in a future version of Terraform.", attr.Name),
			Subject:  &attr.NameRange,
		})
	}

	if attr, exists := content.Attributes["depends_on"]; exists {
		deps, depsDiags := decodeDependsOn(attr)
		diags = append(diags, depsDiags...)
		mc.DependsOn = append(mc.DependsOn, deps...)

		// We currently parse this, but don't yet do anything with it.
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Reserved argument name in module block",
			Detail:   fmt.Sprintf("The name %q is reserved for use in a future version of Terraform.", attr.Name),
			Subject:  &attr.NameRange,
		})
	}

	if attr, exists := content.Attributes["providers"]; exists {
		seen := make(map[string]hcl.Range)
		pairs, pDiags := hcl.ExprMap(attr.Expr)
		diags = append(diags, pDiags...)
		for _, pair := range pairs {
			key, keyDiags := decodeProviderConfigRef(pair.Key, "providers")
			diags = append(diags, keyDiags...)
			value, valueDiags := decodeProviderConfigRef(pair.Value, "providers")
			diags = append(diags, valueDiags...)
			if keyDiags.HasErrors() || valueDiags.HasErrors() {
				continue
			}

			matchKey := key.String()
			if prev, exists := seen[matchKey]; exists {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Duplicate provider address",
					Detail:   fmt.Sprintf("A provider configuration was already passed to %s at %s. Each child provider configuration can be assigned only once.", matchKey, prev),
					Subject:  pair.Value.Range().Ptr(),
				})
				continue
			}

			rng := hcl.RangeBetween(pair.Key.Range(), pair.Value.Range())
			seen[matchKey] = rng
			mc.Providers = append(mc.Providers, PassedProviderConfig{
				InChild:  key,
				InParent: value,
			})
		}
	}

	// Reserved block types (all of them)
	for _, block := range content.Blocks {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Reserved block type name in module block",
			Detail:   fmt.Sprintf("The block type name %q is reserved for use by Terraform in a future version.", block.Type),
			Subject:  &block.TypeRange,
		})
	}

	return mc, diags
}

// PassedProviderConfig represents a provider config explicitly passed down to
// a child module, possibly giving it a new local address in the process.
type PassedProviderConfig struct {
	InChild  *ProviderConfigRef
	InParent *ProviderConfigRef
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
		{
			Name: "providers",
		},
	},
	Blocks: []hcl.BlockHeaderSchema{
		// These are all reserved for future use.
		{Type: "lifecycle"},
		{Type: "locals"},
		{Type: "provider", LabelNames: []string{"type"}},
	},
}
