// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package addrs

// ForEachAttr is the address of an attribute referencing the current "for_each" object in
// the interpolation scope, addressed using the "each" keyword, ex. "each.key" and "each.value"
type ForEachAttr struct {
	referenceable
	Name string
}

func (f ForEachAttr) String() string {
	return "each." + f.Name
}

func (f ForEachAttr) UniqueKey() UniqueKey {
	return f // A ForEachAttr is its own UniqueKey
}

func (f ForEachAttr) uniqueKeySigil() {}
