// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package backendbase

import (
	"fmt"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Base is a partial implementation of [backend.Backend] that can be embedded
// into another implementer to handle most of the configuration schema
// wrangling.
//
// Specifically it implements the ConfigSchema and PrepareConfig methods.
// Implementers embedding this base type must still implement all of the other
// Backend methods.
type Base struct {
	// Schema is the schema for the backend configuration.
	//
	// This shares the same configuration schema model as used in provider
	// schemas for resource types, etc, but only a subset of the model is
	// actually meaningful for backends. In particular, it doesn't make sense
	// to define "computed" attributes because objects conforming to the
	// schema are only use for input based on the configuration, and can't
	// export any data for use elsewhere in the configuration.
	Schema *configschema.Block

	// SDKLikeDefaults is an optional addition for backends that were built
	// to rely heavily on the legacy SDK's support for default values, both
	// hard-coded and taken from environment variables.
	//
	// If this is non-empty then any key specified here must match a
	// primitive-typed toplevel attribute in Schema, and PrepareConfig will
	// arrange for the default values to be inserted before it returns.
	//
	// As a special case, if the value in the configuration is unset (null),
	// none of the environment variables are non-empty, and the fallback
	// value is empty, then the attribute value will be left as null in the
	// object returned by PrepareConfig. In all other situations an attribute
	// specified here is definitely not null.
	SDKLikeDefaults SDKLikeDefaults
}

// ConfigSchema returns the configuration schema for the backend.
func (b Base) ConfigSchema() *configschema.Block {
	return b.Schema
}

// PrepareConfig coerces the given value to the backend's schema if possible,
// and emits deprecation warnings if any deprecated arguments have values
// assigned to them.
func (b Base) PrepareConfig(configVal cty.Value) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	if configVal.IsNull() {
		// We expect the backend configuration to be an object, so if it's
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
			"Invalid backend configuration",
			fmt.Sprintf("The backend configuration is incorrect: %s.", tfdiags.FormatError(err)),
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
			// generic. Backends that want to return a custom message should
			// leave this flag unset and instead implement a check inside
			// their Configure method that returns a warning diagnostic.
			diags = diags.Append(tfdiags.AttributeValue(
				tfdiags.Warning,
				"Deprecated provider argument",
				fmt.Sprintf("The argument %s is deprecated. Refer to the backend documentation for more information.", tfdiags.FormatCtyPath(path)),
				path,
			))
		}

		return false, nil
	})

	if len(b.SDKLikeDefaults) != 0 {
		var err error
		v, err = b.SDKLikeDefaults.ApplyTo(v)
		if err != nil {
			diags = diags.Append(err)
		}
	}

	return v, diags
}
