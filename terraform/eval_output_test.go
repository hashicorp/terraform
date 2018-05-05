package terraform

import (
	"sync"
	"testing"

	"github.com/hashicorp/terraform/addrs"
	"github.com/zclconf/go-cty/cty"
)

func TestEvalWriteMapOutput(t *testing.T) {
	ctx := new(MockEvalContext)
	ctx.StateState = NewState()
	ctx.StateLock = new(sync.RWMutex)

	cases := []struct {
		name string
		val  cty.Value
		err  bool
	}{
		{
			// Eval should recognize a single map in a slice, and collapse it
			// into the map value
			"single-map",
			cty.MapVal(map[string]cty.Value{
				"a": cty.StringVal("b"),
			}),
			false,
		},
		{
			// we can't apply a multi-valued map to a variable, so this should error
			"multi-map",
			cty.ListVal([]cty.Value{
				cty.MapVal(map[string]cty.Value{
					"a": cty.StringVal("b"),
				}),
				cty.MapVal(map[string]cty.Value{
					"c": cty.StringVal("d"),
				}),
			}),
			true,
		},
	}

	for _, tc := range cases {
		evalNode := &EvalWriteOutput{
			Addr: addrs.OutputValue{Name: tc.name},
		}
		ctx.EvaluateExprResult = tc.val
		t.Run(tc.name, func(t *testing.T) {
			_, err := evalNode.Eval(ctx)
			if err != nil && !tc.err {
				t.Fatal(err)
			}
		})
	}
}
