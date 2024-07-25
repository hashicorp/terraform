// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package instances

import (
	"fmt"
	"sort"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
)

// expansion is an internal interface used to represent the different
// ways expansion can operate depending on how repetition is configured for
// an object.
type expansion interface {
	// instanceKeys determines the full set of instance keys for whatever
	// object this expansion is associated with, or indicates that we
	// can't yet know what they are (using keysUnknown=true).
	//
	// if keysUnknown is true then the "keys" result is likely to be incomplete,
	// meaning that there might be other instance keys that we've not yet
	// calculated, but we know that they will be of type keyType.
	//
	// If len(keys) == 0 when keysUnknown is true then we don't yet know _any_
	// instance keys for this object. keyType might be [addrs.UnknownKeyType]
	// if we don't even have enough information to predict what type of
	// keys we will be using for this object.
	instanceKeys() (keyType addrs.InstanceKeyType, keys []addrs.InstanceKey, keysUnknown bool)
	repetitionData(addrs.InstanceKey) RepetitionData
}

// expansionSingle is the expansion corresponding to no repetition arguments
// at all, producing a single object with no key.
//
// expansionSingleVal is the only valid value of this type.
type expansionSingle uintptr

var singleKeys = []addrs.InstanceKey{addrs.NoKey}
var expansionSingleVal expansionSingle

func (e expansionSingle) instanceKeys() (addrs.InstanceKeyType, []addrs.InstanceKey, bool) {
	return addrs.NoKeyType, singleKeys, false
}

func (e expansionSingle) repetitionData(key addrs.InstanceKey) RepetitionData {
	if key != addrs.NoKey {
		panic("cannot use instance key with non-repeating object")
	}
	return RepetitionData{}
}

// expansionCount is the expansion corresponding to the "count" argument.
type expansionCount int

func (e expansionCount) instanceKeys() (addrs.InstanceKeyType, []addrs.InstanceKey, bool) {
	ret := make([]addrs.InstanceKey, int(e))
	for i := range ret {
		ret[i] = addrs.IntKey(i)
	}
	return addrs.IntKeyType, ret, false
}

func (e expansionCount) repetitionData(key addrs.InstanceKey) RepetitionData {
	i := int(key.(addrs.IntKey))
	if i < 0 || i >= int(e) {
		panic(fmt.Sprintf("instance key %d out of range for count %d", i, e))
	}
	return RepetitionData{
		CountIndex: cty.NumberIntVal(int64(i)),
	}
}

// expansionForEach is the expansion corresponding to the "for_each" argument.
type expansionForEach map[string]cty.Value

func (e expansionForEach) instanceKeys() (addrs.InstanceKeyType, []addrs.InstanceKey, bool) {
	ret := make([]addrs.InstanceKey, 0, len(e))
	for k := range e {
		ret = append(ret, addrs.StringKey(k))
	}
	sort.Slice(ret, func(i, j int) bool {
		return ret[i].(addrs.StringKey) < ret[j].(addrs.StringKey)
	})
	return addrs.StringKeyType, ret, false
}

func (e expansionForEach) repetitionData(key addrs.InstanceKey) RepetitionData {
	k := string(key.(addrs.StringKey))
	v, ok := e[k]
	if !ok {
		panic(fmt.Sprintf("instance key %q does not match any instance", k))
	}
	return RepetitionData{
		EachKey:   cty.StringVal(k),
		EachValue: v,
	}
}

// expansionDeferred is a special expansion which represents that we don't
// yet have enough information to calculate the expansion of a particular
// object.
//
// [expansionDeferredIntKey] and [expansionDeferredStringKey] are the only
// valid values of this type.
type expansionDeferred rune

const expansionDeferredIntKey = expansionDeferred(addrs.IntKeyType)
const expansionDeferredStringKey = expansionDeferred(addrs.StringKeyType)

func (e expansionDeferred) instanceKeys() (addrs.InstanceKeyType, []addrs.InstanceKey, bool) {
	return addrs.InstanceKeyType(e), nil, true
}

func (e expansionDeferred) repetitionData(key addrs.InstanceKey) RepetitionData {
	panic("no known instances for object with deferred expansion")
}

func expansionIsDeferred(exp expansion) bool {
	_, ret := exp.(expansionDeferred)
	return ret
}
