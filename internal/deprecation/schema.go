// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package deprecation

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/lang/format"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/zclconf/go-cty/cty"
)

// MarkDeprecatedValues inspects the given cty.Value according to the given
// configschema.Block schema, and marks any deprecated attributes or blocks
// found within the value with deprecation marks.
// It works based on the given cty.Value's structure matching the given schema.
func MarkDeprecatedValues(val cty.Value, schema *configschema.Block, origin string) cty.Value {
	if schema == nil {
		return val
	}
	newVal := val

	// Check if the block is deprecated
	if schema.Deprecated {
		newVal = newVal.Mark(marks.NewDeprecation("Deprecated resource used as value", origin))
	}

	if !newVal.IsKnown() {
		return newVal
	}

	// Even if the block itself is not deprecated, its attributes might be
	// deprecated as well
	if val.Type().IsObjectType() || val.Type().IsMapType() || val.Type().IsCollectionType() {
		// We ignore the error, so errors are not allowed in the transform function
		newVal, _ = cty.Transform(newVal, func(p cty.Path, v cty.Value) (cty.Value, error) {

			attr := schema.AttributeByPath(p)
			if attr != nil && attr.Deprecated {
				v = v.Mark(marks.NewDeprecation(fmt.Sprintf("Deprecated resource attribute %q used", strings.TrimPrefix(format.CtyPath(p), ".")), fmt.Sprintf("%s%s", origin, format.CtyPath(p))))
			}

			block := schema.BlockByPath(p)
			if block != nil && block.Deprecated {
				v = v.Mark(marks.NewDeprecation(fmt.Sprintf("Deprecated resource block %q used", strings.TrimPrefix(format.CtyPath(p), ".")), fmt.Sprintf("%s%s", origin, format.CtyPath(p))))
			}

			return v, nil
		})
	}

	return newVal
}
