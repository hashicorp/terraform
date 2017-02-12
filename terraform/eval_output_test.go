package terraform

import (
	"sync"
	"testing"
)

func TestEvalWriteMapOutput(t *testing.T) {
	ctx := new(MockEvalContext)
	ctx.StateState = NewState()
	ctx.StateLock = new(sync.RWMutex)

	cases := []struct {
		name string
		cfg  *ResourceConfig
		err  bool
	}{
		{
			// Eval should recognize a single map in a slice, and collapse it
			// into the map value
			"single-map",
			&ResourceConfig{
				Config: map[string]interface{}{
					"value": []map[string]interface{}{
						map[string]interface{}{"a": "b"},
					},
				},
			},
			false,
		},
		{
			// we can't apply a multi-valued map to a variable, so this should error
			"multi-map",
			&ResourceConfig{
				Config: map[string]interface{}{
					"value": []map[string]interface{}{
						map[string]interface{}{"a": "b"},
						map[string]interface{}{"c": "d"},
					},
				},
			},
			true,
		},
	}

	for _, tc := range cases {
		evalNode := &EvalWriteOutput{Name: tc.name}
		ctx.InterpolateConfigResult = tc.cfg
		t.Run(tc.name, func(t *testing.T) {
			_, err := evalNode.Eval(ctx)
			if err != nil && !tc.err {
				t.Fatal(err)
			}
		})
	}
}
