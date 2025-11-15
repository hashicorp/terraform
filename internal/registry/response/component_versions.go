// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1
package response

// ComponentVersions is the response format that contains metadata about
// versions needed for terraform CLI to resolve version constraints.
type ComponentVersions struct {
	Components []*ComponentProviderVersions `json:"components"`
}

// ComponentProviderVerions is the response format for a single module instance,
// containing metadata about all versions and their dependencies.
type ComponentProviderVersions struct {
	Source   string              `json:"source"`
	Versions []*ComponentVersion `json:"version"`
}

// ComponentVersion is the output metadata for a given version needed by CLI to
// resolve candidate versions to satisfy requirements.
type ComponentVersion struct {
	Version string `json:"version"`
}
