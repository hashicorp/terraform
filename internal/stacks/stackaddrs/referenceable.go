// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackaddrs

import (
	"reflect"

	"github.com/hashicorp/terraform/internal/collections"
)

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

// ReferenceableUniqueKey returns a unique key for a dynamically-typed
// referenceable address.
//
// Since the type of the target isn't statically known, the resulting unique
// key is not comparable with unique keys of the specific static types
// that implement [Referenceable].
func ReferenceableUniqueKey(addr Referenceable) collections.UniqueKey[Referenceable] {
	// NOTE: This assumes that all distinct referenceable addresses have
	// distinct and unique string representations.
	return referenceableUniqueKey{
		ty:  reflect.TypeOf(addr),
		str: addr.String(),
	}
}

type referenceableUniqueKey struct {
	ty  reflect.Type
	str string
}

// IsUniqueKey implements collections.UniqueKey.
func (referenceableUniqueKey) IsUniqueKey(Referenceable) {}

type ContextualRef rune

const EachValue = ContextualRef('v')
const EachKey = ContextualRef('k')
const CountIndex = ContextualRef('i')
const Self = ContextualRef('s')
const TerraformApplying = ContextualRef('a')

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
	case TerraformApplying:
		return "terraform.applying"
	default:
		// The four constants in this package are the only valid values of this type
		panic("invalid ContextualRef instance")
	}
}

// referenceableSigil implements Referenceable.
func (e ContextualRef) referenceableSigil() {}

// AbsReferenceable is a [Referenceable] combined with the stack it would
// be resolved in.
//
// This type can be used only for [Referenceable] types that have stack-wide
// scope. It's not appropriate for referenceable objects with more specific
// scope, such as [ContextualRef], since describing those would require
// information about which specific block they are to be resolved within.
type AbsReferenceable struct {
	Stack StackInstance
	Item  Referenceable
}

func (r AbsReferenceable) UniqueKey() collections.UniqueKey[AbsReferenceable] {
	return absReferenceableKey{
		stackKey: r.Stack.UniqueKey(),
		itemKey:  ReferenceableUniqueKey(r.Item),
	}
}

type absReferenceableKey struct {
	stackKey collections.UniqueKey[StackInstance]
	itemKey  collections.UniqueKey[Referenceable]
}

// IsUniqueKey implements collections.UniqueKey.
func (absReferenceableKey) IsUniqueKey(AbsReferenceable) {
	panic("unimplemented")
}
