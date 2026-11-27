// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package lang

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/lang/blocktoattr"
)

// VisitAttributes visits configured attributes with stable paths through static
// single, group, list, and map blocks. Set blocks and block types with dynamic
// declarations are skipped, including their static siblings.
//
// The body must already have passed configuration validation. Diagnostics stop
// inspection, but do not undo earlier calls to visit.
func VisitAttributes(body hcl.Body, schema *configschema.Block, visit func(cty.Path, *hcl.Attribute)) hcl.Diagnostics {
	return visitAttributes(blocktoattr.FixUpBlockAttrs(body, schema), schema, nil, visit)
}

func visitAttributes(body hcl.Body, schema *configschema.Block, path cty.Path, visit func(cty.Path, *hcl.Attribute)) hcl.Diagnostics {
	bodySchema := hcldec.ImpliedSchema(schema.DecoderSpec())
	bodySchema.Blocks = append(bodySchema.Blocks, hcl.BlockHeaderSchema{Type: "dynamic", LabelNames: []string{"type"}})
	content, _, diags := body.PartialContent(bodySchema)
	if diags.HasErrors() {
		return diags
	}

	for name, attr := range content.Attributes {
		visit(path.Copy().GetAttr(name), attr)
	}

	// Discover dynamic declarations before assigning indices to static siblings.
	dynamic := make(map[string]bool)
	for _, block := range content.Blocks {
		if block.Type == "dynamic" {
			dynamic[block.Labels[0]] = true
		}
	}

	indices := make(map[string]int)
	for _, block := range content.Blocks {
		if block.Type == "dynamic" || dynamic[block.Type] {
			continue
		}
		blk := schema.BlockTypes[block.Type]
		if blk.Nesting == configschema.NestingSet {
			continue
		}
		blockPath := path.Copy().GetAttr(block.Type)
		switch blk.Nesting {
		case configschema.NestingList:
			blockPath = blockPath.IndexInt(indices[block.Type])
			indices[block.Type]++
		case configschema.NestingMap:
			blockPath = blockPath.IndexString(block.Labels[0])
		}
		moreDiags := visitAttributes(block.Body, &blk.Block, blockPath, visit)
		diags = append(diags, moreDiags...)
		if moreDiags.HasErrors() {
			return diags
		}
	}
	return diags
}
