package terraform

import (
	"reflect"
	"sync"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/terraform/config"
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
			rawConfig, err := config.NewRawConfig(map[string]interface{}{
				"value": test.Value,
			})
			if err != nil {
				t.Fatal(err)
			}

			n := &EvalLocal{
				Name:  "foo",
				Value: rawConfig,
			}
			ctx := &MockEvalContext{
				StateState: &State{},
				StateLock:  &sync.RWMutex{},

				InterpolateConfigResult: testResourceConfig(t, map[string]interface{}{
					"value": test.Want,
				}),
			}

			_, err = n.Eval(ctx)
			if (err != nil) != test.Err {
				if err != nil {
					t.Errorf("unexpected error: %s", err)
				} else {
					t.Errorf("successful Eval; want error")
				}
			}

			ms := ctx.StateState.ModuleByPath([]string{})
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
