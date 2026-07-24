// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform1

import (
	"fmt"
	"math/big"

	"github.com/zclconf/go-cty/cty"
)

// NewAttributePath constructs an [AttributePath] message object from
// a [cty.Path] value.
func NewAttributePath(from cty.Path) *AttributePath {
	ret := &AttributePath{}
	if len(from) == 0 {
		return ret
	}
	ret.Steps = make([]*AttributePath_Step, len(from))
	for i, step := range from {
		switch step := step.(type) {
		case cty.GetAttrStep:
			ret.Steps[i] = &AttributePath_Step{
				Selector: &AttributePath_Step_AttributeName{
					AttributeName: step.Name,
				},
			}
		case cty.IndexStep:
			k := step.Key
			// Although the key is cty.Value, in practice it should typically
			// be constrained only to known and non-null strings and numbers.
			// If we encounter anything else then we'll just abort and return
			// a truncated path, since the only way other values should be
			// able to appear is if we're traversing through a set, and we
			// typically avoid doing that in callers by truncating the path
			// at the same point anyway. (Note that marked values -- one of
			// our main uses for AttributePath -- cannot exist inside
			// sets anyway, so that case can't arise there.)
			if k.IsNull() || !k.IsKnown() {
				k = cty.DynamicVal // to force falling into the default case for the switch below
			}

			switch k.Type() {
			case cty.String:
				ret.Steps[i] = &AttributePath_Step{
					Selector: &AttributePath_Step_ElementKeyString{
						ElementKeyString: k.AsString(),
					},
				}
			case cty.Number:
				// We require an integer in int64 range. We might not get that
				// in the unlikely event that this is a traversal through a
				// cty.Set(cty.Number), since any number would be valid in
				// principle for that case.
				bf := k.AsBigFloat()
				idx, acc := bf.Int64()
				if acc != big.Exact {
					ret.Steps = ret.Steps[:i]
					return ret
				}
				ret.Steps[i] = &AttributePath_Step{
					Selector: &AttributePath_Step_ElementKeyInt{
						ElementKeyInt: idx,
					},
				}
			default:
				ret.Steps = ret.Steps[:i]
				return ret
			}
		default:
			// Should not get here because the above should be exhaustive for
			// all cty.PathStep implementations.
			panic(fmt.Sprintf("path has unsupported step type %T", step))
		}
	}
	return ret
}
