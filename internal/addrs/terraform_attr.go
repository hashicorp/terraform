// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package addrs

// mnptuAttr is the address of an attribute of the "mnptu" object in
// the interpolation scope, like "mnptu.workspace".
type mnptuAttr struct {
	referenceable
	Name string
}

func (ta mnptuAttr) String() string {
	return "mnptu." + ta.Name
}

func (ta mnptuAttr) UniqueKey() UniqueKey {
	return ta // A mnptuAttr is its own UniqueKey
}

func (ta mnptuAttr) uniqueKeySigil() {}
