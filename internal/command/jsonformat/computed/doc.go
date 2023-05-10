// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

// Package computed contains types that represent the computed diffs for
// Terraform blocks, attributes, and outputs.
//
// Each Diff struct is made up of a renderer, an action, and a boolean
// describing the diff. The renderer internally holds child diffs or concrete
// values that allow it to know how to render the diff appropriately.
package computed
