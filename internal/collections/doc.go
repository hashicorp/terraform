// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

// Package collections contains some helper types representing collections
// of values that could not normally be represented using Go's built-in
// collection types, typically because of the need to use key types that
// are not directly comparable.
//
// There have been some discussions about introducing similar functionality
// into the Go standard library. Should that happen in future then we should
// consider removing the types from this package and adapting callers to use
// the standard library equivalents instead, since there is nothing
// Terraform-specific about the implementations in this package.
package collections
