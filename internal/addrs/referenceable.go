// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package addrs

// Referenceable is an interface implemented by all address types that can
// appear as references in configuration language expressions.
type Referenceable interface {
	// Everything referenceable is also referenceable from tests.
	ReferenceableFromTests

	// All implementations of this interface must be covered by the type switch
	// in lang.Scope.buildEvalContext.
	referenceableSigil()

	// All Referenceable address types must have unique keys.
	UniqueKeyer

	// String produces a string representation of the address that could be
	// parsed as a HCL traversal and passed to ParseRef to produce an identical
	// result.
	String() string
}

type referenceable struct {
	// Everything referenceable is also referencable from any tests.
	referenceableFromTests
}

func (r referenceable) referenceableSigil() {
}

type ReferenceableFromTests interface {
	referenceableFromTestsSigil()

	UniqueKeyer

	String() string
}

type referenceableFromTests struct {
}

func (r referenceableFromTests) referenceableFromTestsSigil() {
}
