// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

// JSONOutputVersionLogger logs information at the start of a command to help systems
// consume versioned JSON output.
//
// NOTE: This is specific to SRO-style JSON output, and this interface doesn't need to
// be implemented by commands that produce no JSON output or produce _static_ JSON output.
type JSONOutputVersionLogger interface {
	// Version logs the version information for a command's JSON output.
	// This method should be a no-op for non-JSON views.
	Version()
}
