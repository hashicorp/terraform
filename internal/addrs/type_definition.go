// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package addrs

// TypeDefinition is the address of a type definition.
type TypeDefinition struct {
	referenceable
	Name string
}

func (v TypeDefinition) String() string {
	return "typedef." + v.Name
}

func (v TypeDefinition) UniqueKey() UniqueKey {
	return v // A TypeDefinition is its own UniqueKey
}

func (v TypeDefinition) uniqueKeySigil() {}
