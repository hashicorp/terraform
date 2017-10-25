package testharness

import (
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/terraform/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

func TestLoadSpecDir(t *testing.T) {
	spec, diags := LoadSpecDir("test-fixtures/basic-specs")
	if len(diags) != 0 {
		t.Errorf("unexpected diagnostics")
		for _, diag := range diags {
			t.Logf("- %s", diag.Description())
		}
	}

	{
		want := map[string]*Scenario{
			"first": &Scenario{
				Name: "first",
				Variables: map[string]cty.Value{
					"foo": cty.StringVal("bar"),
				},

				DefRange: tfdiags.SourceRange{
					Filename: "test-fixtures/basic-specs/a.tfspec",
					Start:    tfdiags.SourcePos{Line: 2, Column: 1, Byte: -1},
					End:      tfdiags.SourcePos{Line: 2, Column: 1, Byte: -1},
				},
			},
			"second": &Scenario{
				Name:      "second",
				Variables: map[string]cty.Value{},

				DefRange: tfdiags.SourceRange{
					Filename: "test-fixtures/basic-specs/a.tfspec",
					Start:    tfdiags.SourcePos{Line: 8, Column: 1, Byte: -1},
					End:      tfdiags.SourcePos{Line: 8, Column: 1, Byte: -1},
				},
			},
			"third": &Scenario{
				Name:      "third",
				Variables: map[string]cty.Value{},

				DefRange: tfdiags.SourceRange{
					Filename: "test-fixtures/basic-specs/b.tfspec",
					Start:    tfdiags.SourcePos{Line: 2, Column: 1, Byte: -1},
					End:      tfdiags.SourcePos{Line: 2, Column: 1, Byte: -1},
				},
			},
		}
		got := spec.scenarios

		if !reflect.DeepEqual(got, want) {
			t.Errorf("wrong result\ngot: %swant %s", spew.Sdump(got), spew.Sdump(want))
		}
	}
}
