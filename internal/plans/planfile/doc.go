// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

// Package planfile deals with the file format used to serialize plans to disk
// and then deserialize them back into memory later.
//
// A plan file contains the planned changes along with the configuration and
// state snapshot that they are based on.
package planfile
