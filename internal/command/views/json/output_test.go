package json

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/zclconf/go-cty/cty"
)

func TestOutputsFromMap(t *testing.T) {
	got, diags := OutputsFromMap(map[string]*states.OutputValue{
		// Normal non-sensitive output
		"boop": {
			Value: cty.NumberIntVal(1234),
		},
		// Sensitive string output
		"beep": {
			Value:     cty.StringVal("horse-battery").Mark(marks.Sensitive),
			Sensitive: true,
		},
		// Sensitive object output which is marked at the leaf
		"blorp": {
			Value: cty.ObjectVal(map[string]cty.Value{
				"a": cty.ObjectVal(map[string]cty.Value{
					"b": cty.ObjectVal(map[string]cty.Value{
						"c": cty.StringVal("oh, hi").Mark(marks.Sensitive),
					}),
				}),
			}),
			Sensitive: true,
		},
		// Null value
		"honk": {
			Value: cty.NullVal(cty.Map(cty.Bool)),
		},
	})
	if len(diags) > 0 {
		t.Fatal(diags.Err())
	}

	want := Outputs{
		"boop": {
			Sensitive: false,
			Type:      json.RawMessage(`"number"`),
			Value:     json.RawMessage(`1234`),
		},
		"beep": {
			Sensitive: true,
			Type:      json.RawMessage(`"string"`),
			Value:     json.RawMessage(`"horse-battery"`),
		},
		"blorp": {
			Sensitive: true,
			Type:      json.RawMessage(`["object",{"a":["object",{"b":["object",{"c":"string"}]}]}]`),
			Value:     json.RawMessage(`{"a":{"b":{"c":"oh, hi"}}}`),
		},
		"honk": {
			Sensitive: false,
			Type:      json.RawMessage(`["map","bool"]`),
			Value:     json.RawMessage(`null`),
		},
	}

	if !cmp.Equal(want, got) {
		t.Fatalf("unexpected result\n%s", cmp.Diff(want, got))
	}
}

func TestOutputs_String(t *testing.T) {
	outputs := Outputs{
		"boop": {
			Sensitive: false,
			Type:      json.RawMessage(`"number"`),
			Value:     json.RawMessage(`1234`),
		},
		"beep": {
			Sensitive: true,
			Type:      json.RawMessage(`"string"`),
			Value:     json.RawMessage(`"horse-battery"`),
		},
	}
	if got, want := outputs.String(), "Outputs: 2"; got != want {
		t.Fatalf("unexpected value\n got: %q\nwant: %q", got, want)
	}
}
