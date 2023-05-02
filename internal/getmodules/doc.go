// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

// Package getmodules contains the low-level functionality for fetching
// remote module packages. It's essentially just a thin wrapper around
// go-getter.
//
// This package is only for remote module source addresses, not for local
// or registry source addresses. The other address types are handled
// elsewhere.
package getmodules
