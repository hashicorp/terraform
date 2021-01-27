package views

import (
	"testing"

	"github.com/hashicorp/terraform/internal/terminal"
	"github.com/hashicorp/terraform/states"
	"github.com/zclconf/go-cty/cty"
)

func TestOutputRaw(t *testing.T) {
	values := map[string]cty.Value{
		"str":      cty.StringVal("bar"),
		"multistr": cty.StringVal("bar\nbaz"),
		"num":      cty.NumberIntVal(2),
		"bool":     cty.True,
		"obj":      cty.EmptyObjectVal,
		"null":     cty.NullVal(cty.String),
	}

	tests := map[string]struct {
		WantOutput string
		WantErr    bool
	}{
		"str":      {WantOutput: "bar"},
		"multistr": {WantOutput: "bar\nbaz"},
		"num":      {WantOutput: "2"},
		"bool":     {WantOutput: "true"},
		"obj":      {WantErr: true},
		"null":     {WantErr: true},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			streams, done := terminal.StreamsForTesting(t)
			view := NewView(streams)
			v := &OutputRaw{
				View: *view,
			}

			value := values[name]
			outputs := map[string]*states.OutputValue{
				name: {Value: value},
			}
			diags := v.Output(name, outputs)

			if diags.HasErrors() {
				if !test.WantErr {
					t.Fatalf("unexpected diagnostics: %s", diags)
				}
			} else if test.WantErr {
				t.Fatalf("succeeded, but want error")
			}

			if got, want := done(t).Stdout(), test.WantOutput; got != want {
				t.Errorf("wrong result\ngot:  %q\nwant: %q", got, want)
			}
		})
	}
}
