// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

//go:generate go tool golang.org/x/tools/cmd/stringer -type=ResourceMode -output=resource_mode_string.go resource_mode.go

// ResourceMode is deprecated, use addrs.ResourceMode instead.
// It has been preserved for backwards compatibility.
type ResourceMode int

const (
	ManagedResourceMode ResourceMode = iota
	DataResourceMode
)
