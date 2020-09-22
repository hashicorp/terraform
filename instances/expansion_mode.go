package instances

import (
	"fmt"
	"sort"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/addrs"
)

// expansion is an internal interface used to represent the different
// ways expansion can operate depending on how repetition is configured for
// an object.
type expansion interface {
	instanceKeys() []addrs.InstanceKey
	repetitionData(addrs.InstanceKey) RepetitionData
}

// expansionSingle is the expansion corresponding to no repetition arguments
// at all, producing a single object with no key.
//
// expansionSingleVal is the only valid value of this type.
type expansionSingle uintptr

var singleKeys = []addrs.InstanceKey{addrs.NoKey}
var expansionSingleVal expansionSingle

func (e expansionSingle) instanceKeys() []addrs.InstanceKey {
	return singleKeys
}

func (e expansionSingle) repetitionData(key addrs.InstanceKey) RepetitionData {
	if key != addrs.NoKey {
		panic("cannot use instance key with non-repeating object")
	}
	return RepetitionData{}
}

// expansionCount is the expansion corresponding to the "count" argument.
type expansionCount int

func (e expansionCount) instanceKeys() []addrs.InstanceKey {
	ret := make([]addrs.InstanceKey, int(e))
	for i := range ret {
		ret[i] = addrs.IntKey(i)
	}
	return ret
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

func (e expansionForEach) instanceKeys() []addrs.InstanceKey {
	ret := make([]addrs.InstanceKey, 0, len(e))
	for k := range e {
		ret = append(ret, addrs.StringKey(k))
	}
	sort.Slice(ret, func(i, j int) bool {
		return ret[i].(addrs.StringKey) < ret[j].(addrs.StringKey)
	})
	return ret
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
