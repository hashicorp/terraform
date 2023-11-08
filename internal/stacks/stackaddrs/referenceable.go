// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackaddrs

// Referenceable is a type set containing all address types that can be
// the target of an expression-based reference within a particular stack.
type Referenceable interface {
	referenceableSigil()
	String() string
}

var _ Referenceable = Component{}
var _ Referenceable = StackCall{}
var _ Referenceable = InputVariable{}
var _ Referenceable = LocalValue{}
var _ Referenceable = ProviderConfigRef{}
var _ Referenceable = TestOnlyGlobal{}
var _ Referenceable = ContextualRef(0)

type ContextualRef rune

const EachValue = ContextualRef('v')
const EachKey = ContextualRef('k')
const CountIndex = ContextualRef('i')
const Self = ContextualRef('s')

// String implements Referenceable.
func (e ContextualRef) String() string {
	switch e {
	case EachKey:
		return "each.key"
	case EachValue:
		return "each.value"
	case CountIndex:
		return "count.index"
	case Self:
		return "self"
	default:
		// The four constants in this package are the only valid values of this type
		panic("invalid ContextualRef instance")
	}
}

// referenceableSigil implements Referenceable.
func (e ContextualRef) referenceableSigil() {}
