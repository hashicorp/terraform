// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package ephemeral

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

// ValidateWriteOnlyAttributes identifies all instances of write-only paths that contain non-null values
// and returns a diagnostic for each instance
func ValidateWriteOnlyAttributes(summary string, detail func(cty.Path) string, newVal cty.Value, schema *configschema.Block) (diags tfdiags.Diagnostics) {
	writeOnlyPaths, err := nonNullWriteOnlyPaths(newVal, schema, nil)
	if err != nil {
		return diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			summary,
			fmt.Sprintf("Error validating write-only attributes: %s.", err),
		))
	}
	if len(writeOnlyPaths) != 0 {
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

// nonNullWriteOnlyPaths returns a list of paths to attributes that are write-only
// and non-null in the given value.
func nonNullWriteOnlyPaths(val cty.Value, schema *configschema.Block, p cty.Path) ([]cty.Path, error) {
	if schema == nil {
		panic("nonNullWriteOnlyPaths called wih nil schema")
	}
	var paths []cty.Path

	for _, path := range schema.WriteOnlyPaths(val, nil) {
		// Note that path.Apply will fail if the path traverses a set, but ephemeral
		// values won't work in a set anyway, and they are prohibited by the
		// plugin framework.
		v, err := path.Apply(val)
		if err != nil {
			return nil, err
		}
		if !v.IsNull() {
			paths = append(paths, path)
		}

	}

	return paths, nil
}
