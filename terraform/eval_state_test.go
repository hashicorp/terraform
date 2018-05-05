package terraform

import (
	"sync"
	"testing"

	"github.com/hashicorp/terraform/addrs"
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
		"ReadStateDeposed gets deposed instance": {
			Resources: map[string]*ResourceState{
				"aws_instance.bar": &ResourceState{
					Deposed: []*InstanceState{
						&InstanceState{ID: "i-abc123"},
					},
				},
			},
			Node: &EvalReadStateDeposed{
				Name:   "aws_instance.bar",
				Output: &output,
				Index:  0,
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
		ctx.PathPath = addrs.RootModuleInstance

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

func TestEvalWriteState(t *testing.T) {
	state := &State{}
	ctx := new(MockEvalContext)
	ctx.StateState = state
	ctx.StateLock = new(sync.RWMutex)
	ctx.PathPath = addrs.RootModuleInstance

	is := &InstanceState{ID: "i-abc123"}
	node := &EvalWriteState{
		Name:         "restype.resname",
		ResourceType: "restype",
		State:        &is,
	}
	_, err := node.Eval(ctx)
	if err != nil {
		t.Fatalf("Got err: %#v", err)
	}

	checkStateString(t, state, `
restype.resname:
  ID = i-abc123
	`)
}

func TestEvalWriteStateDeposed(t *testing.T) {
	state := &State{}
	ctx := new(MockEvalContext)
	ctx.StateState = state
	ctx.StateLock = new(sync.RWMutex)
	ctx.PathPath = addrs.RootModuleInstance

	is := &InstanceState{ID: "i-abc123"}
	node := &EvalWriteStateDeposed{
		Name:         "restype.resname",
		ResourceType: "restype",
		State:        &is,
		Index:        -1,
	}
	_, err := node.Eval(ctx)
	if err != nil {
		t.Fatalf("Got err: %#v", err)
	}

	checkStateString(t, state, `
restype.resname: (1 deposed)
  ID = <not created>
  Deposed ID 1 = i-abc123
	`)
}
