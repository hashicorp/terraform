// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

// Spacer is an interface that can be implemented by any view that needs to log
// empty lines to output, to space out messages. This is only used in human-readable output
// of commands that produce multiple logs during a long-running or multi-step operation.
//
// A clear example of this is the `terraform init` command, which uses empty lines to space
// out messages related to distinct steps like backend initialisation, provider download etc.
type Spacer interface {
	// Spacer logs an empty line to output
	// It should be a no-op for JSON views.
	Spacer()
}
