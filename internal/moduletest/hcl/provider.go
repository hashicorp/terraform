// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package hcl

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/lang/langrefs"
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
//
// We don't parse the attributes until they are requested, so we can only use
// unparsed values and hcl.Expressions within the struct itself.
type ProviderConfig struct {
	Original hcl.Body

	VariableCache       *VariableCache
	AvailableRunOutputs map[addrs.Run]cty.Value
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
	}, &ProviderConfig{rest, p.VariableCache, p.AvailableRunOutputs}, diags
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

	availableVariables := make(map[string]cty.Value)

	exprs := make(map[string]hcl.Expression, len(originals))
	for _, original := range originals {
		exprs[original.Name] = original.Expr

		// We also need to parse the variables we're going to use, so we extract
		// the references from this expression now and see if they reference any
		// input variables. If we find an input variable, we'll copy it into
		// our availableVariables local.
		refs, _ := langrefs.ReferencesInExpr(addrs.ParseRefFromTestingScope, original.Expr)
		for _, ref := range refs {
			if addr, ok := ref.Subject.(addrs.InputVariable); ok {
				value, valueDiags := p.VariableCache.GetFileVariable(addr.Name)
				diags = append(diags, valueDiags.ToHCL()...)
				if value != nil {
					availableVariables[addr.Name] = value.Value
					continue
				}

				// If the variable wasn't a file variable, it might be a global.
				value, valueDiags = p.VariableCache.GetGlobalVariable(addr.Name)
				diags = append(diags, valueDiags.ToHCL()...)
				if value != nil {
					availableVariables[addr.Name] = value.Value
					continue
				}
			}
		}
	}

	ctx, ctxDiags := EvalContext(TargetProvider, exprs, availableVariables, p.AvailableRunOutputs)
	diags = append(diags, ctxDiags.ToHCL()...)
	if ctxDiags.HasErrors() {
		return nil, diags
	}

	attrs := make(hcl.Attributes, len(originals))
	for name, attr := range originals {
		value, valueDiags := attr.Expr.Value(ctx)
		diags = append(diags, valueDiags...)
		if valueDiags.HasErrors() {
			continue
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
			Body:        &ProviderConfig{block.Body, p.VariableCache, p.AvailableRunOutputs},
			DefRange:    block.DefRange,
			TypeRange:   block.TypeRange,
			LabelRanges: block.LabelRanges,
		}
	}
	return blocks
}
