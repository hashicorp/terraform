// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package tfdiags

import (
	"github.com/hashicorp/terraform/internal/lang/format"
)

// These functions have been moved to the format package to allow for imports
// which would otherwise cause cycles.

// FormatCtyPath is a helper function to produce a user-friendly string
// representation of a cty.Path. The result uses a syntax similar to the
// HCL expression language in the hope of it being familiar to users.
var FormatCtyPath = format.CtyPath

// FormatError is a helper function to produce a user-friendly string
// representation of certain special error types that we might want to
// include in diagnostic messages.
var FormatError = format.ErrorDiag

// FormatErrorPrefixed is like FormatError except that it presents any path
// information after the given prefix string, which is assumed to contain
// an HCL syntax representation of the value that errors are relative to.
var FormatErrorPrefixed = format.ErrorDiagPrefixed
