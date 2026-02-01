// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package definitions

import (
	"github.com/hashicorp/hcl/v2"
)

// CheckRule represents a configuration-defined validation rule, precondition,
// or postcondition. Blocks of this sort can appear in a few different places
// in configuration, including "validation" blocks for variables,
// and "precondition" and "postcondition" blocks for resources.
type CheckRule struct {
	// Condition is an expression that must evaluate to true if the condition
	// holds or false if it does not. If the expression produces an error then
	// that's considered to be a bug in the module defining the check.
	//
	// The available variables in a condition expression vary depending on what
	// a check is attached to. For example, validation rules attached to
	// input variables can only refer to the variable that is being validated.
	Condition hcl.Expression

	// ErrorMessage should be one or more full sentences, which should be in
	// English for consistency with the rest of the error message output but
	// can in practice be in any language. The message should describe what is
	// required for the condition to return true in a way that would make sense
	// to a caller of the module.
	//
	// The error message expression has the same variables available for
	// interpolation as the corresponding condition.
	ErrorMessage hcl.Expression

	DeclRange hcl.Range
}
