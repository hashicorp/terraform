// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package addrs

// Self is the address of the special object "self" that behaves as an alias
// for a containing object currently in scope.
const Self SelfType = 0

// Caller is a pseudo-alias for self, representing the object triggering an
// action. An action is called from within a resource, but we use "caller"
// because the configuration is written outside of the resource context.
const Caller SelfType = 1

type SelfType int

func (s SelfType) referenceableSigil() {
}

func (s SelfType) String() string {
	switch s {
	case 0:
		return "self"
	case 1:
		return "caller"
	default:
		panic("invalid self type")
	}
}

func (s SelfType) UniqueKey() UniqueKey {
	return s
}

func (s SelfType) uniqueKeySigil() {}
