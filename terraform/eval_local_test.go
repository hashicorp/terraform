package terraform

import (
	"reflect"
	"sync"
	"testing"

	"github.com/hashicorp/terraform/config/hcl2shim"

	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hcl/hclsyntax"

	"github.com/hashicorp/terraform/addrs"

	"github.com/davecgh/go-spew/spew"
)

func TestEvalLocal_impl(t *testing.T) {
	var _ EvalNode = new(EvalLocal)
}

func TestEvalLocal(t *testing.T) {
	tests := []struct {
		Value string
		Want  interface{}
		Err   bool
	}{
		{
			"hello!",
			"hello!",
			false,
		},
		{
			"",
			"",
			false,
		},
	}

	for _, test := range tests {
		t.Run(test.Value, func(t *testing.T) {
			expr, diags := hclsyntax.ParseTemplate([]byte(test.Value), "", hcl.Pos{Line: 1, Column: 1})
			if diags.HasErrors() {
				t.Fatal(diags.Error())
			}

			n := &EvalLocal{
				Addr: addrs.LocalValue{Name: "foo"},
				Expr: expr,
			}
			ctx := &MockEvalContext{
				StateState: &State{},
				StateLock:  &sync.RWMutex{},

				EvaluateExprResult: hcl2shim.HCL2ValueFromConfigValue(test.Want),
			}

			_, err := n.Eval(ctx)
			if (err != nil) != test.Err {
				if err != nil {
					t.Errorf("unexpected error: %s", err)
				} else {
					t.Errorf("successful Eval; want error")
				}
			}

			ms := ctx.StateState.ModuleByPath(addrs.RootModuleInstance)
			gotLocals := ms.Locals
			wantLocals := map[string]interface{}{
				"foo": test.Want,
			}

			if !reflect.DeepEqual(gotLocals, wantLocals) {
				t.Errorf(
					"wrong locals after Eval\ngot:  %swant: %s",
					spew.Sdump(gotLocals), spew.Sdump(wantLocals),
				)
			}
		})
	}

}
