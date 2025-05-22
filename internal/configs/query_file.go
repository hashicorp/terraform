// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package configs

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform/internal/addrs"
)

// QueryFile represents a single query file within a configuration directory.
//
// A query file is made up of a sequential list of List blocks, each defining a
// set of filters to apply when listning a List operation
type QueryFile struct {
	// Providers defines a set of providers that are available to the list blocks
	// within this query file.
	Providers       map[string]*Provider
	ProviderConfigs []*Provider

	Locals    []*Local
	Variables []*Variable

	// ListResources is a slice of List blocks within the query file.
	ListResources []*Resource

	VariablesDeclRange hcl.Range
}

func loadQueryFile(body hcl.Body) (*QueryFile, hcl.Diagnostics) {
	var diags hcl.Diagnostics
	file := &QueryFile{
		Providers: make(map[string]*Provider),
	}

	content, contentDiags := body.Content(queryFileSchema)
	diags = append(diags, contentDiags...)

	listBlockNames := make(map[string]hcl.Range)

	for _, block := range content.Blocks {
		switch block.Type {
		case "list":
			list, listDiags := decodeQueryListBlock(block)
			diags = append(diags, listDiags...)
			if !listDiags.HasErrors() {
				file.ListResources = append(file.ListResources, list)
			}

			if rng, exists := listBlockNames[list.Name]; exists {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Duplicate \"list\" block names",
					Detail:   fmt.Sprintf("This query file already has a list block named %s defined at %s.", list.Name, rng),
					Subject:  block.DefRange.Ptr(),
				})
				continue
			}
			listBlockNames[list.Name] = list.DeclRange
		case "provider":
			cfg, cfgDiags := decodeProviderBlock(block, false)
			diags = append(diags, cfgDiags...)
			if cfg != nil {
				file.ProviderConfigs = append(file.ProviderConfigs, cfg)
			}
		case "variable":
			cfg, cfgDiags := decodeVariableBlock(block, false)
			diags = append(diags, cfgDiags...)
			if cfg != nil {
				file.Variables = append(file.Variables, cfg)
			}
		case "locals":
			defs, defsDiags := decodeLocalsBlock(block)
			diags = append(diags, defsDiags...)
			file.Locals = append(file.Locals, defs...)
		default:
			// We don't expect any other block types in a query file.
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid block type",
				Detail:   fmt.Sprintf("This block type is not valid within a query file: %s", block.Type),
				Subject:  block.DefRange.Ptr(),
			})
		}
	}

	return file, diags
}

func decodeQueryListBlock(block *hcl.Block) (*Resource, hcl.Diagnostics) {
	var diags hcl.Diagnostics

	content, remain, contentDiags := block.Body.PartialContent(QueryListResourceBlockSchema)
	diags = append(diags, contentDiags...)

	r := Resource{
		Mode:      addrs.ListResourceMode,
		Type:      block.Labels[0],
		TypeRange: block.LabelRanges[0],
		Name:      block.Labels[1],
		DeclRange: block.DefRange,
		Config:    remain,
		List:      &ListResource{},
	}

	if attr, exists := content.Attributes["provider"]; exists {
		var providerDiags hcl.Diagnostics
		r.ProviderConfigRef, providerDiags = decodeProviderConfigRef(attr.Expr, "provider")
		diags = append(diags, providerDiags...)
	} else {
		// Must have a provider attribute.
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Missing \"provider\" attribute",
			Detail:   "You must specify a provider attribute when defining a list block.",
			Subject:  r.DeclRange.Ptr(),
		})
	}

	if !hclsyntax.ValidIdentifier(r.Name) {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid list block name",
			Detail:   badIdentifierDetail,
			Subject:  r.DeclRange.Ptr(),
		})
	}

	if attr, exists := content.Attributes["count"]; exists {
		r.Count = attr.Expr
	}

	if attr, exists := content.Attributes["for_each"]; exists {
		r.ForEach = attr.Expr
		// Cannot have count and for_each on the same resource block
		if r.Count != nil {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  `Invalid combination of "count" and "for_each"`,
				Detail:   `The "count" and "for_each" meta-arguments are mutually-exclusive.`,
				Subject:  &attr.NameRange,
			})
		}
	}

	if attr, exists := content.Attributes["include_resource"]; exists {
		r.List.IncludeResource = attr.Expr
	}

	return &r, diags
}

// QueryListResourceBlockSchema is the schema for a list resource type within
// a terraform query file.
var QueryListResourceBlockSchema = &hcl.BodySchema{
	Attributes: append(commonResourceAttributes, hcl.AttributeSchema{
		Name: "include_resource",
	}),
}

// queryFileSchema is the schema for a terraform query file. It defines the
// expected structure of the file, including the types of supported blocks and their
// attributes.
var queryFileSchema = &hcl.BodySchema{
	Blocks: []hcl.BlockHeaderSchema{
		{
			Type:       "list",
			LabelNames: []string{"type", "name"},
		},
		{
			Type:       "provider",
			LabelNames: []string{"name"},
		},
		{
			Type: "locals",
		},
		{
			Type:       "variable",
			LabelNames: []string{"name"},
		},
	},
}
