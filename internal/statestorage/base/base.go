// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package base

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

type Base struct {
	Schema *configschema.Block
}

func (b Base) ConfigSchema() *configschema.Block {
	return b.Schema
}

// PrepareConfig coerces the given value to the storage's schema if possible,
// and emits deprecation warnings if any deprecated arguments have values
// assigned to them.
func (b Base) PrepareConfig(configVal cty.Value) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	if configVal.IsNull() {
		// We expect the storage configuration to be an object, so if it's
		// null for some reason (e.g. because of an interrupt), we'll turn
		// it into an empty object so that we can still coerce it
		configVal = cty.EmptyObjectVal
	}

	schema := b.Schema

	v, err := schema.CoerceValue(configVal)
	if err != nil {
		var path cty.Path
		if err, ok := err.(cty.PathError); ok {
			path = err.Path
		}
		diags = diags.Append(tfdiags.AttributeValue(
			tfdiags.Error,
			"Invalid state storage configuration",
			fmt.Sprintf("The state storage configuration is incorrect: %s.", tfdiags.FormatError(err)),
			path,
		))
		return cty.DynamicVal, diags
	}

	cty.Walk(v, func(path cty.Path, v cty.Value) (bool, error) {
		if v.IsNull() {
			// Null values for deprecated arguments do not generate deprecation
			// warnings, because that represents the argument not being set.
			return false, nil
		}

		// If this path refers to a schema attribute then it might be
		// deprecated, in which case we need to return a warning.
		attr := schema.AttributeByPath(path)
		if attr == nil {
			return true, nil
		}
		if attr.Deprecated {
			// The configschema model only has a boolean flag for whether the
			// argument is deprecated or not, so this warning message is
			// generic. Storages that want to return a custom message should
			// leave this flag unset and instead implement a check inside
			// their Configure method that returns a warning diagnostic.
			diags = diags.Append(tfdiags.AttributeValue(
				tfdiags.Warning,
				"Deprecated provider argument",
				fmt.Sprintf("The argument %s is deprecated. Refer to the state storage documentation for more information.", tfdiags.FormatCtyPath(path)),
				path,
			))
		}

		return false, nil
	})

	return v, diags
}
