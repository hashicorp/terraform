// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

// Package addrs contains types that represent "addresses", which are
// references to specific objects within a Terraform configuration or
// state.
//
// All addresses have string representations based on HCL traversal syntax
// which should be used in the user-interface, and also in-memory
// representations that can be used internally.
//
// For object types that exist within Terraform modules a pair of types is
// used. The "local" part of the address is represented by a type, and then
// an absolute path to that object in the context of its module is represented
// by a type of the same name with an "Abs" prefix added, for "absolute".
//
// All types within this package should be treated as immutable, even if this
// is not enforced by the Go compiler. It is always an implementation error
// to modify an address object in-place after it is initially constructed.
package addrs
