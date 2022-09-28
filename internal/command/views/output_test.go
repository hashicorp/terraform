package views

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/terminal"
	"github.com/zclconf/go-cty/cty"
)

// Test various single output values for human-readable UI. Note that since
// OutputHuman defers to repl.FormatValue to render a single value, most of the
// test coverage should be in that package.
func TestOutputHuman_single(t *testing.T) {
	testCases := map[string]struct {
		value   cty.Value
		want    string
		wantErr bool
	}{
		"string": {
			value: cty.StringVal("hello"),
			want:  "\"hello\"\n",
		},
		"list of maps": {
			value: cty.ListVal([]cty.Value{
				cty.MapVal(map[string]cty.Value{
					"key":  cty.StringVal("value"),
					"key2": cty.StringVal("value2"),
				}),
				cty.MapVal(map[string]cty.Value{
					"key": cty.StringVal("value"),
				}),
			}),
			want: `tolist([
  tomap({
    "key" = "value"
    "key2" = "value2"
  }),
  tomap({
    "key" = "value"
  }),
])
`,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			streams, done := terminal.StreamsForTesting(t)
			v := NewOutput(arguments.ViewHuman, NewView(streams))

			outputs := map[string]*states.OutputValue{
				"foo": {Value: tc.value},
			}
			diags := v.Output("foo", outputs)

			if diags.HasErrors() {
				if !tc.wantErr {
					t.Fatalf("unexpected diagnostics: %s", diags)
				}
			} else if tc.wantErr {
				t.Fatalf("succeeded, but want error")
			}

			if got, want := done(t).Stdout(), tc.want; got != want {
				t.Errorf("wrong result\ngot:  %q\nwant: %q", got, want)
			}
		})
	}
}

// Sensitive output values are rendered to the console intentionally when
// requesting a single output.
func TestOutput_sensitive(t *testing.T) {
	testCases := map[string]arguments.ViewType{
		"human": arguments.ViewHuman,
		"json":  arguments.ViewJSON,
		"raw":   arguments.ViewRaw,
	}
	for name, vt := range testCases {
		t.Run(name, func(t *testing.T) {
			streams, done := terminal.StreamsForTesting(t)
			v := NewOutput(vt, NewView(streams))

			outputs := map[string]*states.OutputValue{
				"foo": {
					Value:     cty.StringVal("secret"),
					Sensitive: true,
				},
			}
			diags := v.Output("foo", outputs)

			if diags.HasErrors() {
				t.Fatalf("unexpected diagnostics: %s", diags)
			}

			// Test for substring match here because we don't care about exact
			// output format in this test, just the presence of the sensitive
			// value.
			if got, want := done(t).Stdout(), "secret"; !strings.Contains(got, want) {
				t.Errorf("wrong result\ngot:  %q\nwant: %q", got, want)
			}
		})
	}
}

// Showing all outputs is supported by human and JSON output format.
func TestOutput_all(t *testing.T) {
	outputs := map[string]*states.OutputValue{
		"foo": {
			Value:     cty.StringVal("secret"),
			Sensitive: true,
		},
		"bar": {
			Value: cty.ListVal([]cty.Value{cty.True, cty.False, cty.True}),
		},
		"baz": {
			Value: cty.ObjectVal(map[string]cty.Value{
				"boop": cty.NumberIntVal(5),
				"beep": cty.StringVal("true"),
			}),
		},
	}

	testCases := map[string]struct {
		vt   arguments.ViewType
		want string
	}{
		"human": {
			arguments.ViewHuman,
			`bar = tolist([
  true,
  false,
  true,
])
baz = {
  "beep" = "true"
  "boop" = 5
}
foo = <sensitive>
`,
		},
		"json": {
			arguments.ViewJSON,
			`{
  "bar": {
    "sensitive": false,
    "type": [
      "list",
      "bool"
    ],
    "value": [
      true,
      false,
      true
    ]
  },
  "baz": {
    "sensitive": false,
    "type": [
      "object",
      {
        "beep": "string",
        "boop": "number"
      }
    ],
    "value": {
      "beep": "true",
      "boop": 5
    }
  },
  "foo": {
    "sensitive": true,
    "type": "string",
    "value": "secret"
  }
}
`,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			streams, done := terminal.StreamsForTesting(t)
			v := NewOutput(tc.vt, NewView(streams))
			diags := v.Output("", outputs)

			if diags.HasErrors() {
				t.Fatalf("unexpected diagnostics: %s", diags)
			}

			if got := done(t).Stdout(); got != tc.want {
				t.Errorf("wrong result\ngot:  %q\nwant: %q", got, tc.want)
			}
		})
	}
}

// JSON output format supports empty outputs by rendering an empty object
// without diagnostics.
func TestOutputJSON_empty(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	v := NewOutput(arguments.ViewJSON, NewView(streams))

	diags := v.Output("", map[string]*states.OutputValue{})

	if diags.HasErrors() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}

	if got, want := done(t).Stdout(), "{}\n"; got != want {
		t.Errorf("wrong result\ngot:  %q\nwant: %q", got, want)
	}
}

// Human and raw formats render a warning if there are no outputs.
func TestOutput_emptyWarning(t *testing.T) {
	testCases := map[string]arguments.ViewType{
		"human": arguments.ViewHuman,
		"raw":   arguments.ViewRaw,
	}

	for name, vt := range testCases {
		t.Run(name, func(t *testing.T) {
			streams, done := terminal.StreamsForTesting(t)
			v := NewOutput(vt, NewView(streams))

			diags := v.Output("", map[string]*states.OutputValue{})

			if got, want := done(t).Stdout(), ""; got != want {
				t.Errorf("wrong result\ngot:  %q\nwant: %q", got, want)
			}

			if len(diags) != 1 {
				t.Fatalf("expected 1 diagnostic, got %d", len(diags))
			}

			if diags.HasErrors() {
				t.Fatalf("unexpected error diagnostics: %s", diags)
			}

			if got, want := diags[0].Description().Summary, "No outputs found"; got != want {
				t.Errorf("unexpected diagnostics: %s", diags)
			}
		})
	}
}

// Raw output is a simple unquoted output format designed for shell scripts,
// which relies on the cty.AsString() implementation. This test covers
// formatting for supported value types.
func TestOutputRaw(t *testing.T) {
	values := map[string]cty.Value{
		"str":      cty.StringVal("bar"),
		"multistr": cty.StringVal("bar\nbaz"),
		"num":      cty.NumberIntVal(2),
		"bool":     cty.True,
		"obj":      cty.EmptyObjectVal,
		"null":     cty.NullVal(cty.String),
		"unknown":  cty.UnknownVal(cty.String),
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
		"unknown":  {WantErr: true},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			streams, done := terminal.StreamsForTesting(t)
			v := NewOutput(arguments.ViewRaw, NewView(streams))

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

// Raw cannot render all outputs.
func TestOutputRaw_all(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	v := NewOutput(arguments.ViewRaw, NewView(streams))

	outputs := map[string]*states.OutputValue{
		"foo": {Value: cty.StringVal("secret")},
		"bar": {Value: cty.True},
	}
	diags := v.Output("", outputs)

	if got, want := done(t).Stdout(), ""; got != want {
		t.Errorf("wrong result\ngot:  %q\nwant: %q", got, want)
	}

	if !diags.HasErrors() {
		t.Fatalf("expected diagnostics, got %s", diags)
	}

	if got, want := diags.Err().Error(), "Raw output format is only supported for single outputs"; got != want {
		t.Errorf("unexpected diagnostics: %s", diags)
	}
}

// All outputs render an error if a specific output is requested which is
// missing from the map of outputs.
func TestOutput_missing(t *testing.T) {
	testCases := map[string]arguments.ViewType{
		"human": arguments.ViewHuman,
		"json":  arguments.ViewJSON,
		"raw":   arguments.ViewRaw,
	}

	for name, vt := range testCases {
		t.Run(name, func(t *testing.T) {
			streams, done := terminal.StreamsForTesting(t)
			v := NewOutput(vt, NewView(streams))

			diags := v.Output("foo", map[string]*states.OutputValue{
				"bar": {Value: cty.StringVal("boop")},
			})

			if len(diags) != 1 {
				t.Fatalf("expected 1 diagnostic, got %d", len(diags))
			}

			if !diags.HasErrors() {
				t.Fatalf("expected error diagnostics, got %s", diags)
			}

			if got, want := diags[0].Description().Summary, `Output "foo" not found`; got != want {
				t.Errorf("unexpected diagnostics: %s", diags)
			}

			if got, want := done(t).Stdout(), ""; got != want {
				t.Errorf("wrong result\ngot:  %q\nwant: %q", got, want)
			}
		})
	}
}
