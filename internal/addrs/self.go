// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package addrs

// Self is the address of the special object "self" that behaves as an alias
// for a containing object currently in scope.
const Self selfT = 0

type selfT int

func (s selfT) referenceableSigil() {
}

func (s selfT) String() string {
	return "self"
}

func (s selfT) UniqueKey() UniqueKey {
	return Self // Self is its own UniqueKey
}

func (s selfT) uniqueKeySigil() {}
