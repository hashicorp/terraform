// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package addrs

// SourceAttr is the address of an attribute of the "source" object in
// the interpolation scope, like "source.module".
type SourceAttr struct {
	referenceable
	Name string
}

func (sa SourceAttr) String() string {
	return "source." + sa.Name
}

func (sa SourceAttr) UniqueKey() UniqueKey {
	return sa // A SourceAttr is its own UniqueKey
}

func (sa SourceAttr) uniqueKeySigil() {}
