// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

// Package ephemeral contains helper functions for working with values that
// might have ephemeral parts.
//
// "Ephemeral" in this context means that a value is preserved only in memory
// for no longer than the duration of a single Terraform phase, and is not
// persisted as part of longer-lived artifacts such as state snapshots and
// saved plan files. Because ephemeral values cannot be persisted, they can
// be used only as part of the configuration of objects that are ephemeral
// themselves, such as provider configurations and provisioners.
package ephemeral
