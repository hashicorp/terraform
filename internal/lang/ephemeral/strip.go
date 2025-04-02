// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package ephemeral

import (
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/configs/configschema"
)

// StripWriteOnlyAttributes converts all the write-only attributes in value to
// null values.
func StripWriteOnlyAttributes(value cty.Value, schema *configschema.Block) cty.Value {
	// writeOnlyTransformer never returns errors, so we don't need to detect
	// them here.
	updated, _ := cty.TransformWithTransformer(value, &writeOnlyTransformer{
		schema: schema,
	})
	return updated
}

var _ cty.Transformer = (*writeOnlyTransformer)(nil)

type writeOnlyTransformer struct {
	schema *configschema.Block
}

func (w *writeOnlyTransformer) Enter(path cty.Path, value cty.Value) (cty.Value, error) {
	attr := w.schema.AttributeByPath(path)
	if attr == nil {
		return value, nil
	}

	if attr.WriteOnly {
		value, marks := value.Unmark()
		return cty.NullVal(value.Type()).WithMarks(marks), nil
	}

	return value, nil
}

func (w *writeOnlyTransformer) Exit(_ cty.Path, value cty.Value) (cty.Value, error) {
	return value, nil // no changes
}
