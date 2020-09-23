package terraform

import (
	"sync"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
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
