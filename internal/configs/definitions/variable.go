// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package definitions

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/ext/typeexpr"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
)

// Variable represents a "variable" block in a module or file.
type Variable struct {
	Name        string
	Description string
	Default     cty.Value

	// Type is the concrete type of the variable value.
	Type cty.Type
	// ConstraintType is used for decoding and type conversions, and may
	// contain nested ObjectWithOptionalAttr types.
	ConstraintType cty.Type
	TypeDefaults   *typeexpr.Defaults

	ParsingMode VariableParsingMode
	Validations []*CheckRule
	Sensitive   bool
	Ephemeral   bool

	DescriptionSet bool
	SensitiveSet   bool
	EphemeralSet   bool

	// Nullable indicates that null is a valid value for this variable. Setting
	// Nullable to false means that the module can expect this variable to
	// never be null.
	Nullable    bool
	NullableSet bool

	DeclRange hcl.Range
}

// Addr returns the address of the variable.
func (v *Variable) Addr() addrs.InputVariable {
	return addrs.InputVariable{Name: v.Name}
}

// Required returns true if this variable is required to be set by the caller,
// or false if there is a default value that will be used when it isn't set.
func (v *Variable) Required() bool {
	return v.Default == cty.NilVal
}
