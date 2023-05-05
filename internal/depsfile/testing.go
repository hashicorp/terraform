// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package depsfile

import (
	"github.com/google/go-cmp/cmp"
)

// ProviderLockComparer is an option for github.com/google/go-cmp/cmp that
// specifies how to compare values of type depsfile.ProviderLock.
//
// Use this, rather than crafting comparison options yourself, in case the
// comparison strategy needs to change in future due to implementation details
// of the ProviderLock type.
var ProviderLockComparer cmp.Option

func init() {
	// For now, direct comparison of the unexported fields is good enough
	// because we store everything in a normalized form. If that changes
	// later then we might need to write a custom transformer to a hidden
	// type with exported fields, so we can retain the ability for cmp to
	// still report differences deeply.
	ProviderLockComparer = cmp.AllowUnexported(ProviderLock{})
}
