package terraform

import (
	"sync"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/tfdiags"
)

func TestEvaluatorGetTerraformAttr(t *testing.T) {
	evaluator := &Evaluator{
		Meta: &ContextMeta{
			Env: "foo",
		},
	}
	data := &evaluationStateData{
		Evaluator: evaluator,
	}
	scope := evaluator.Scope(data, nil)

	t.Run("workspace", func(t *testing.T) {
		want := cty.StringVal("foo")
		got, diags := scope.Data.GetTerraformAttr(addrs.TerraformAttr{
			Name: "workspace",
		}, tfdiags.SourceRange{})
		if len(diags) != 0 {
			t.Errorf("unexpected diagnostics %s", spew.Sdump(diags))
		}
		if !got.RawEquals(want) {
			t.Errorf("wrong result %q; want %q", got, want)
		}
	})
}

func TestEvaluatorGetPathAttr(t *testing.T) {
	evaluator := &Evaluator{
		Meta: &ContextMeta{
			Env: "foo",
		},
		Config: &configs.Config{
			Module: &configs.Module{
				SourceDir: "bar/baz",
			},
		},
	}
	data := &evaluationStateData{
		Evaluator: evaluator,
	}
	scope := evaluator.Scope(data, nil)

	t.Run("module", func(t *testing.T) {
		want := cty.StringVal("bar/baz")
		got, diags := scope.Data.GetPathAttr(addrs.PathAttr{
			Name: "module",
		}, tfdiags.SourceRange{})
		if len(diags) != 0 {
			t.Errorf("unexpected diagnostics %s", spew.Sdump(diags))
		}
		if !got.RawEquals(want) {
			t.Errorf("wrong result %#v; want %#v", got, want)
		}
	})

	t.Run("root", func(t *testing.T) {
		want := cty.StringVal("bar/baz")
		got, diags := scope.Data.GetPathAttr(addrs.PathAttr{
			Name: "root",
		}, tfdiags.SourceRange{})
		if len(diags) != 0 {
			t.Errorf("unexpected diagnostics %s", spew.Sdump(diags))
		}
		if !got.RawEquals(want) {
			t.Errorf("wrong result %#v; want %#v", got, want)
		}
	})
}

// This particularly tests that a sensitive attribute in config
// results in a value that has a "sensitive" cty Mark
func TestEvaluatorGetInputVariable(t *testing.T) {
	evaluator := &Evaluator{
		Meta: &ContextMeta{
			Env: "foo",
		},
		Config: &configs.Config{
			Module: &configs.Module{
				Variables: map[string]*configs.Variable{
					"some_var": {
						Name:      "some_var",
						Sensitive: true,
						Default:   cty.StringVal("foo"),
					},
				},
			},
		},
		VariableValues: map[string]map[string]cty.Value{
			"": {"some_var": cty.StringVal("bar")},
		},
		VariableValuesLock: &sync.Mutex{},
	}

	data := &evaluationStateData{
		Evaluator: evaluator,
	}
	scope := evaluator.Scope(data, nil)

	want := cty.StringVal("bar").Mark("sensitive")
	got, diags := scope.Data.GetInputVariable(addrs.InputVariable{
		Name: "some_var",
	}, tfdiags.SourceRange{})

	if len(diags) != 0 {
		t.Errorf("unexpected diagnostics %s", spew.Sdump(diags))
	}
	if !got.RawEquals(want) {
		t.Errorf("wrong result %#v; want %#v", got, want)
	}
}

func TestEvaluatorGetModule(t *testing.T) {
	// Create a new evaluator with an existing state
	stateSync := states.BuildState(func(ss *states.SyncState) {
		ss.SetOutputValue(
			addrs.OutputValue{Name: "out"}.Absolute(addrs.ModuleInstance{addrs.ModuleInstanceStep{Name: "mod"}}),
			cty.StringVal("bar"),
			true,
		)
	}).SyncWrapper()
	evaluator := evaluatorForModule(stateSync, plans.NewChanges().SyncWrapper())
	data := &evaluationStateData{
		Evaluator: evaluator,
	}
	scope := evaluator.Scope(data, nil)
	want := cty.ObjectVal(map[string]cty.Value{"out": cty.StringVal("bar").Mark("sensitive")})
	got, diags := scope.Data.GetModule(addrs.ModuleCall{
		Name: "mod",
	}, tfdiags.SourceRange{})

	if len(diags) != 0 {
		t.Errorf("unexpected diagnostics %s", spew.Sdump(diags))
	}
	if !got.RawEquals(want) {
		t.Errorf("wrong result %#v; want %#v", got, want)
	}

	// Changes should override the state value
	changesSync := plans.NewChanges().SyncWrapper()
	change := &plans.OutputChange{
		Addr:      addrs.OutputValue{Name: "out"}.Absolute(addrs.ModuleInstance{addrs.ModuleInstanceStep{Name: "mod"}}),
		Sensitive: true,
		Change: plans.Change{
			After: cty.StringVal("baz"),
		},
	}
	cs, _ := change.Encode()
	changesSync.AppendOutputChange(cs)
	evaluator = evaluatorForModule(stateSync, changesSync)
	data = &evaluationStateData{
		Evaluator: evaluator,
	}
	scope = evaluator.Scope(data, nil)
	want = cty.ObjectVal(map[string]cty.Value{"out": cty.StringVal("baz").Mark("sensitive")})
	got, diags = scope.Data.GetModule(addrs.ModuleCall{
		Name: "mod",
	}, tfdiags.SourceRange{})

	if len(diags) != 0 {
		t.Errorf("unexpected diagnostics %s", spew.Sdump(diags))
	}
	if !got.RawEquals(want) {
		t.Errorf("wrong result %#v; want %#v", got, want)
	}

	// Test changes with empty state
	evaluator = evaluatorForModule(states.NewState().SyncWrapper(), changesSync)
	data = &evaluationStateData{
		Evaluator: evaluator,
	}
	scope = evaluator.Scope(data, nil)
	want = cty.ObjectVal(map[string]cty.Value{"out": cty.StringVal("baz").Mark("sensitive")})
	got, diags = scope.Data.GetModule(addrs.ModuleCall{
		Name: "mod",
	}, tfdiags.SourceRange{})

	if len(diags) != 0 {
		t.Errorf("unexpected diagnostics %s", spew.Sdump(diags))
	}
	if !got.RawEquals(want) {
		t.Errorf("wrong result %#v; want %#v", got, want)
	}
}

func evaluatorForModule(stateSync *states.SyncState, changesSync *plans.ChangesSync) *Evaluator {
	return &Evaluator{
		Meta: &ContextMeta{
			Env: "foo",
		},
		Config: &configs.Config{
			Module: &configs.Module{
				ModuleCalls: map[string]*configs.ModuleCall{
					"mod": {
						Name: "mod",
					},
				},
			},
			Children: map[string]*configs.Config{
				"mod": {
					Path: addrs.Module{"module.mod"},
					Module: &configs.Module{
						Outputs: map[string]*configs.Output{
							"out": {
								Name:      "out",
								Sensitive: true,
							},
						},
					},
				},
			},
		},
		State:   stateSync,
		Changes: changesSync,
	}
}
