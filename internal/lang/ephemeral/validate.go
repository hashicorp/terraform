// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package ephemeral

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

func ValidateWriteOnlyAttributes(newVal cty.Value, schema *configschema.Block, provider addrs.AbsProviderConfig, addr addrs.AbsResourceInstance) (diags tfdiags.Diagnostics) {
	if writeOnlyPaths := NonNullWriteOnlyPaths(newVal, schema, nil); len(writeOnlyPaths) != 0 {
		for _, p := range writeOnlyPaths {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Write-only attribute set",
				fmt.Sprintf(
					"Provider %q returned a value for the write-only attribute \"%s%s\". Write-only attributes cannot be read back from the provider. This is a bug in the provider, which should be reported in the provider's own issue tracker.",
					provider.String(), addr.String(), tfdiags.FormatCtyPath(p),
				),
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
