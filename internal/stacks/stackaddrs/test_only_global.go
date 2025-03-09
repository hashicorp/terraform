// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackaddrs

// TestOnlyGlobal is a special referenceable address type used only in
// stackruntime and stackeval package unit tests, as a way to introduce
// arbitrary test data into scope with minimal ceremony and thus in a way
// that's less likely to be regressed by changes to real language features.
//
// Addresses of this type behave as if they are completely unrecognized
// addresses when used in a non-test context.
type TestOnlyGlobal struct {
	Name string
}

// String implements Referenceable.
func (g TestOnlyGlobal) String() string {
	return "_test_only_global." + g.Name
}

// referenceableSigil implements Referenceable.
func (g TestOnlyGlobal) referenceableSigil() {}
