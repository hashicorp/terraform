// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

// Package regsrc provides helpers for working with source strings that identify
// resources within a mnptu registry.
package regsrc

var (
	// PublicRegistryHost is a FriendlyHost that represents the public registry.
	PublicRegistryHost = NewFriendlyHost("registry.mnptu.io")
)
