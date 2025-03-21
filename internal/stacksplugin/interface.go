// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stacksplugin

import (
	"io"
)

// Stacks1 interface for Terraform plugin operations
type Stacks1 interface {
	// Execute runs a command with the provided arguments and returns the exit code
	Execute(args []string, stdout, stderr io.Writer) int
}
