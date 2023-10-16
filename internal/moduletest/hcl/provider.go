// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package hcl

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/lang"
)

var _ hcl.Body = (*ProviderConfig)(nil)

// ProviderConfig is an implementation of an hcl.Block that evaluates the
// attributes within the block using the provided config and variables before
// returning them.
//
// This is used by configs.Provider objects that are defined within the test
// framework, so they should only use variables available to the test framework
// but are instead initialised within the Terraform graph so we have to delay
// evaluation of their attributes until the schemas are retrieved.
type ProviderConfig struct {
	Original hcl.Body

	ConfigVariables    map[string]*configs.Variable
	AvailableVariables map[string]backend.UnparsedVariableValue
}

func (p *ProviderConfig) Content(schema *hcl.BodySchema) (*hcl.BodyContent, hcl.Diagnostics) {
	content, diags := p.Original.Content(schema)
	attrs, attrDiags := p.transformAttributes(content.Attributes)
	diags = append(diags, attrDiags...)

	return &hcl.BodyContent{
		Attributes:       attrs,
		Blocks:           p.transformBlocks(content.Blocks),
		MissingItemRange: content.MissingItemRange,
	}, diags
}

func (p *ProviderConfig) PartialContent(schema *hcl.BodySchema) (*hcl.BodyContent, hcl.Body, hcl.Diagnostics) {
	content, rest, diags := p.Original.PartialContent(schema)
	attrs, attrDiags := p.transformAttributes(content.Attributes)
	diags = append(diags, attrDiags...)

	return &hcl.BodyContent{
		Attributes:       attrs,
		Blocks:           p.transformBlocks(content.Blocks),
		MissingItemRange: content.MissingItemRange,
	}, &ProviderConfig{rest, p.ConfigVariables, p.AvailableVariables}, diags
}

func (p *ProviderConfig) JustAttributes() (hcl.Attributes, hcl.Diagnostics) {
	originals, diags := p.Original.JustAttributes()
	attrs, moreDiags := p.transformAttributes(originals)
	return attrs, append(diags, moreDiags...)
}

func (p *ProviderConfig) MissingItemRange() hcl.Range {
	return p.Original.MissingItemRange()
}

func (p *ProviderConfig) transformAttributes(originals hcl.Attributes) (hcl.Attributes, hcl.Diagnostics) {
	var diags hcl.Diagnostics

	relevantVariables := make(map[string]cty.Value)
	var exprs []hcl.Expression

	for _, original := range originals {
		exprs = append(exprs, original.Expr)

		// We revalidate this later, so we actually only care about the
		// references we can extract.
		refs, _ := lang.ReferencesInExpr(addrs.ParseRefFromTestingScope, original.Expr)
		for _, ref := range refs {
			if addr, ok := ref.Subject.(addrs.InputVariable); ok {
				if variable, exists := p.AvailableVariables[addr.Name]; exists {

					parsingMode := configs.VariableParseHCL
					if config, exists := p.ConfigVariables[addr.Name]; exists {
						parsingMode = config.ParsingMode
					}

					value, valueDiags := variable.ParseVariableValue(parsingMode)
					diags = append(diags, valueDiags.ToHCL()...)
					if value != nil {
						relevantVariables[addr.Name] = value.Value
					}
				}
			}
		}
	}

	ctx, ctxDiags := EvalContext(exprs, relevantVariables, nil)
	diags = append(diags, ctxDiags.ToHCL()...)
	if ctxDiags.HasErrors() {
		return originals, diags
	}

	attrs := make(hcl.Attributes, len(originals))
	for name, attr := range originals {
		value, valueDiags := attr.Expr.Value(ctx)
		diags = append(diags, valueDiags...)
		if valueDiags.HasErrors() {
			attrs[name] = attr
		} else {
			attrs[name] = &hcl.Attribute{
				Name:      name,
				Expr:      hcl.StaticExpr(value, attr.Expr.Range()),
				Range:     attr.Range,
				NameRange: attr.NameRange,
			}
		}
	}
	return attrs, diags
}

func (p *ProviderConfig) transformBlocks(originals hcl.Blocks) hcl.Blocks {
	blocks := make(hcl.Blocks, len(originals))
	for name, block := range originals {
		blocks[name] = &hcl.Block{
			Type:        block.Type,
			Labels:      block.Labels,
			Body:        &ProviderConfig{block.Body, p.ConfigVariables, p.AvailableVariables},
			DefRange:    block.DefRange,
			TypeRange:   block.TypeRange,
			LabelRanges: block.LabelRanges,
		}
	}
	return blocks
}
