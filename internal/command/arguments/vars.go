// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// UnparsedVariableValue represents a variable value provided by the caller
// whose parsing must be deferred until configuration is available.
//
// This exists to allow processing of variable-setting arguments (e.g. in the
// command package) to be separated from parsing (in the backend package).
type UnparsedVariableValue interface {
	// ParseVariableValue information in the provided variable configuration
	// to parse (if necessary) and return the variable value encapsulated in
	// the receiver.
	//
	// If error diagnostics are returned, the resulting value may be invalid
	// or incomplete.
	ParseVariableValue(mode configs.VariableParsingMode) (*terraform.InputValue, tfdiags.Diagnostics)
}

// Vars describes arguments which specify non-default variable values. This
// interface is unfortunately obscure, because the order of the CLI arguments
// determines the final value of the gathered variables. In future it might be
// desirable for the arguments package to handle the gathering of variables
// directly, returning a map of variable values.
type Vars struct {
	vars     *FlagNameValueSlice
	varFiles *FlagNameValueSlice
}

func (v *Vars) All() []FlagNameValue {
	if v.vars == nil {
		return nil
	}
	return v.vars.AllItems()
}

func (v *Vars) Empty() bool {
	if v.vars == nil {
		return true
	}
	return v.vars.Empty()
}
