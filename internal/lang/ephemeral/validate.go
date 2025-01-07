// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package ephemeral

import (
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

// ValidateWriteOnlyAttributes identifies all instances of write-only paths that contain non-null values
// and returns a diagnostic for each instance
func ValidateWriteOnlyAttributes(summary string, detail func(cty.Path) string, newVal cty.Value, schema *configschema.Block) (diags tfdiags.Diagnostics) {
	if writeOnlyPaths := NonNullWriteOnlyPaths(newVal, schema, nil); len(writeOnlyPaths) != 0 {
		for _, p := range writeOnlyPaths {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				summary,
				detail(p),
			))
		}
	}
	return diags
}

// NonNullWriteOnlyPaths returns a list of paths to attributes that are write-only
// and non-null in the given value.
func NonNullWriteOnlyPaths(val cty.Value, schema *configschema.Block, p cty.Path) (paths []cty.Path) {
	if schema == nil {
		return paths
	}

	for name, attr := range schema.Attributes {
		attrPath := append(p, cty.GetAttrStep{Name: name})
		attrVal, _ := attrPath.Apply(val)
		if attr.WriteOnly && !attrVal.IsNull() {
			paths = append(paths, attrPath)
		}
	}

	for name, blockS := range schema.BlockTypes {
		blockPath := append(p, cty.GetAttrStep{Name: name})
		x := NonNullWriteOnlyPaths(val, &blockS.Block, blockPath)
		paths = append(paths, x...)
	}

	return paths
}
