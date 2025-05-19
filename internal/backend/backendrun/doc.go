// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

// Package backendrun contains the additional types and helpers used by the
// few backends that actually run operations against Terraform configurations.
//
// Backends that only provide state storage should not use anything in this
// package.
package backendrun
