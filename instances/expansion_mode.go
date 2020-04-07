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

var singleKeys = []addrs.InstanceKey{addrs.NoKey}

type expansionValue cty.Value

func (s expansionValue) instanceKeys() []addrs.InstanceKey {
	v := cty.Value(s)
	switch {
	case v.Type() == cty.Number:
		n, _ := v.AsBigFloat().Int64()
		ret := make([]addrs.InstanceKey, int(n))
		for i := range ret {
			ret[i] = addrs.IntKey(i)
		}
		return ret

	case v.Type().IsMapType() || v.Type().IsSetType() || v.Type().IsObjectType():
		ret := make([]addrs.InstanceKey, 0, v.LengthInt())

		for it := v.ElementIterator(); it.Next(); {
			k, _ := it.Element()
			ret = append(ret, addrs.StringKey(k.AsString()))
		}

		sort.Slice(ret, func(i, j int) bool {
			return ret[i].(addrs.StringKey) < ret[j].(addrs.StringKey)
		})
		return ret

	case v == cty.NilVal:
		return singleKeys
	default:
		panic(fmt.Sprintf("unexpected expansion value: %#v", v))
	}
}

func (s expansionValue) repetitionData(key addrs.InstanceKey) RepetitionData {
	v := cty.Value(s)

	switch {
	case v.Type() == cty.Number:
		if !v.IsKnown() {
			return RepetitionData{
				CountIndex: cty.UnknownVal(cty.Number),
			}
		}

		n, _ := v.AsBigFloat().Int64()
		i := int(key.(addrs.IntKey))
		if i < 0 || i >= int(n) {
			panic(fmt.Sprintf("instance key %d out of range for count %d", i, n))
		}
		return RepetitionData{
			CountIndex: cty.NumberIntVal(int64(i)),
		}

	case v.Type().IsMapType() || v.Type().IsSetType() || v.Type().IsObjectType():
		k := string(key.(addrs.StringKey))

		if !v.IsKnown() {
			return RepetitionData{
				EachKey:   cty.UnknownVal(cty.String),
				EachValue: cty.DynamicVal,
			}
		}

		m := v.AsValueMap()
		v, ok := m[k]
		if !ok {
			panic(fmt.Sprintf("instance key %q does not match any instance", k))
		}

		return RepetitionData{
			EachKey:   cty.StringVal(k),
			EachValue: v,
		}

	case v == cty.NilVal:
		return RepetitionData{}

	default:
		panic(fmt.Sprintf("unexpected expansion value: %#v", v))
	}
}
