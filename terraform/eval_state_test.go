package terraform

import (
	"sync"
	"testing"
)

func TestEvalRequireState(t *testing.T) {
	ctx := new(MockEvalContext)

	cases := []struct {
		State *InstanceState
		Exit  bool
	}{
		{
			nil,
			true,
		},
		{
			&InstanceState{},
			true,
		},
		{
			&InstanceState{ID: "foo"},
			false,
		},
	}

	var exitVal EvalEarlyExitError
	for _, tc := range cases {
		node := &EvalRequireState{State: &tc.State}
		_, err := node.Eval(ctx)
		if tc.Exit {
			if err != exitVal {
				t.Fatalf("should've exited: %#v", tc.State)
			}

			continue
		}
		if !tc.Exit && err != nil {
			t.Fatalf("shouldn't exit: %#v", tc.State)
		}
		if err != nil {
			t.Fatalf("err: %s", err)
		}
	}
}

func TestEvalUpdateStateHook(t *testing.T) {
	mockHook := new(MockHook)

	ctx := new(MockEvalContext)
	ctx.HookHook = mockHook
	ctx.StateState = &State{Serial: 42}
	ctx.StateLock = new(sync.RWMutex)

	node := &EvalUpdateStateHook{}
	if _, err := node.Eval(ctx); err != nil {
		t.Fatalf("err: %s", err)
	}

	if !mockHook.PostStateUpdateCalled {
		t.Fatal("should call PostStateUpdate")
	}
	if mockHook.PostStateUpdateState.Serial != 42 {
		t.Fatalf("bad: %#v", mockHook.PostStateUpdateState)
	}
}

func TestEvalReadState(t *testing.T) {
	var output *InstanceState
	cases := map[string]struct {
		Resources          map[string]*ResourceState
		Node               EvalNode
		ExpectedInstanceId string
	}{
		"ReadState gets primary instance state": {
			Resources: map[string]*ResourceState{
				"aws_instance.bar": &ResourceState{
					Primary: &InstanceState{
						ID: "i-abc123",
					},
				},
			},
			Node: &EvalReadState{
				Name:   "aws_instance.bar",
				Output: &output,
			},
			ExpectedInstanceId: "i-abc123",
		},
		"ReadStateTainted gets tainted instance": {
			Resources: map[string]*ResourceState{
				"aws_instance.bar": &ResourceState{
					Tainted: []*InstanceState{
						&InstanceState{ID: "i-abc123"},
					},
				},
			},
			Node: &EvalReadStateTainted{
				Name:         "aws_instance.bar",
				Output:       &output,
				TaintedIndex: 0,
			},
			ExpectedInstanceId: "i-abc123",
		},
		"ReadStateDeposed gets deposed instance": {
			Resources: map[string]*ResourceState{
				"aws_instance.bar": &ResourceState{
					Deposed: &InstanceState{ID: "i-abc123"},
				},
			},
			Node: &EvalReadStateDeposed{
				Name:   "aws_instance.bar",
				Output: &output,
			},
			ExpectedInstanceId: "i-abc123",
		},
	}

	for k, c := range cases {
		ctx := new(MockEvalContext)
		ctx.StateState = &State{
			Modules: []*ModuleState{
				&ModuleState{
					Path:      rootModulePath,
					Resources: c.Resources,
				},
			},
		}
		ctx.StateLock = new(sync.RWMutex)
		ctx.PathPath = rootModulePath

		result, err := c.Node.Eval(ctx)
		if err != nil {
			t.Fatalf("[%s] Got err: %#v", k, err)
		}

		expected := c.ExpectedInstanceId
		if !(result != nil && result.(*InstanceState).ID == expected) {
			t.Fatalf("[%s] Expected return with ID %#v, got: %#v", k, expected, result)
		}

		if !(output != nil && output.ID == expected) {
			t.Fatalf("[%s] Expected output with ID %#v, got: %#v", k, expected, output)
		}

		output = nil
	}
}
