// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

// VersionLogger logs information at the start of a command to help systems
// consume versioned output formats.
type VersionLogger interface {
	// Version logs the version information for a command's JSON output.
	// This method should be a no-op for non-JSON views.
	Version()
}
