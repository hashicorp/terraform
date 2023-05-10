// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

// Package replacefile is a small helper package focused directly at the
// problem of atomically "renaming" one file over another one.
//
// On Unix systems this is the standard behavior of the rename function, but
// the equivalent operation on Windows requires some specific operation flags
// which this package encapsulates.
//
// This package uses conditional compilation to select a different
// implementation for Windows vs. all other platforms. It may therefore
// require further fiddling in future if Terraform is ported to another
// OS that is neither Unix-like nor Windows.
package replacefile
