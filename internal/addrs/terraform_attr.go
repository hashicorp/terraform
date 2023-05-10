// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package addrs

// TerraformAttr is the address of an attribute of the "terraform" object in
// the interpolation scope, like "terraform.workspace".
type TerraformAttr struct {
	referenceable
	Name string
}

func (ta TerraformAttr) String() string {
	return "terraform." + ta.Name
}

func (ta TerraformAttr) UniqueKey() UniqueKey {
	return ta // A TerraformAttr is its own UniqueKey
}

func (ta TerraformAttr) uniqueKeySigil() {}
