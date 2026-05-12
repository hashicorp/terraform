// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import "net/url"

// Workspace represents the command-line arguments common between all workspace subcommands.
//
// Subcommands that accept additional arguments should have a specific struct that embeds this struct.
type Workspace struct {
	// ViewType specifies which output format to use
	ViewType ViewType
}

// ValidWorkspaceName returns true is this name is valid to use as a workspace name.
// Since most named states are accessed via a filesystem path or URL, check if
// escaping the name would be required.
func ValidWorkspaceName(name string) bool {
	if name == "" {
		return false
	}
	return name == url.PathEscape(name)
}

const EnvInvalidName = `
The workspace name %q is not allowed. The name must contain only URL safe
characters, contain no path separators, and not be an empty string.
`
