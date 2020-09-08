package terraform

import (
	"testing"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/states"
	"github.com/zclconf/go-cty/cty"
)

func TestNodeApplyableOutputExecute(t *testing.T) {
	ctx := new(MockEvalContext)
	ctx.StateState = states.NewState().SyncWrapper()

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
		node := &NodeApplyableOutput{
			Config: &configs.Output{},
			Addr:   addrs.OutputValue{Name: tc.name}.Absolute(addrs.RootModuleInstance),
		}
		ctx.EvaluateExprResult = tc.val
		t.Run(tc.name, func(t *testing.T) {
			err := node.Execute(ctx, walkApply)
			if err != nil && !tc.err {
				t.Fatal(err)
			}
		})
	}

}

func TestNodeDestroyableOutputExecute(t *testing.T) {
	outputAddr := addrs.OutputValue{Name: "foo"}.Absolute(addrs.RootModuleInstance)

	state := states.NewState()
	state.Module(addrs.RootModuleInstance).SetOutputValue("foo", cty.StringVal("bar"), false)
	state.OutputValue(outputAddr)

	ctx := &MockEvalContext{
		StateState: state.SyncWrapper(),
	}
	node := NodeDestroyableOutput{Addr: outputAddr}

	err := node.Execute(ctx, walkApply)
	if err != nil {
		t.Fatalf("Unexpected error: %s", err.Error())
	}
	if state.OutputValue(outputAddr) != nil {
		t.Fatal("Unexpected outputs in state after removal")
	}
}
