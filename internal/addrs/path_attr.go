// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package addrs

// PathAttr is the address of an attribute of the "path" object in
// the interpolation scope, like "path.module".
type PathAttr struct {
	referenceable
	Name string
}

func (pa PathAttr) String() string {
	return "path." + pa.Name
}

func (pa PathAttr) UniqueKey() UniqueKey {
	return pa // A PathAttr is its own UniqueKey
}

func (pa PathAttr) uniqueKeySigil() {}
