package json

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/plans"
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

func TestOutputsFromChanges(t *testing.T) {
	root := addrs.RootModuleInstance
	num, err := plans.NewDynamicValue(cty.NumberIntVal(1234), cty.Number)
	if err != nil {
		t.Fatalf("unexpected error creating dynamic value: %v", err)
	}
	str, err := plans.NewDynamicValue(cty.StringVal("1234"), cty.String)
	if err != nil {
		t.Fatalf("unexpected error creating dynamic value: %v", err)
	}

	got := OutputsFromChanges([]*plans.OutputChangeSrc{
		// Unchanged output "boop", value 1234
		{
			Addr: root.OutputValue("boop"),
			ChangeSrc: plans.ChangeSrc{
				Action: plans.NoOp,
				Before: num,
				After:  num,
			},
			Sensitive: false,
		},
		// New output "beep", value 1234
		{
			Addr: root.OutputValue("beep"),
			ChangeSrc: plans.ChangeSrc{
				Action: plans.Create,
				Before: nil,
				After:  num,
			},
			Sensitive: false,
		},
		// Deleted output "blorp", prior value 1234
		{
			Addr: root.OutputValue("blorp"),
			ChangeSrc: plans.ChangeSrc{
				Action: plans.Delete,
				Before: num,
				After:  nil,
			},
			Sensitive: false,
		},
		// Updated output "honk", prior value 1234, new value "1234"
		{
			Addr: root.OutputValue("honk"),
			ChangeSrc: plans.ChangeSrc{
				Action: plans.Update,
				Before: num,
				After:  str,
			},
			Sensitive: false,
		},
		// New sensitive output "secret", value "1234"
		{
			Addr: root.OutputValue("secret"),
			ChangeSrc: plans.ChangeSrc{
				Action: plans.Create,
				Before: nil,
				After:  str,
			},
			Sensitive: true,
		},
	})

	want := Outputs{
		"boop": {
			Action:    "noop",
			Sensitive: false,
		},
		"beep": {
			Action:    "create",
			Sensitive: false,
		},
		"blorp": {
			Action:    "delete",
			Sensitive: false,
		},
		"honk": {
			Action:    "update",
			Sensitive: false,
		},
		"secret": {
			Action:    "create",
			Sensitive: true,
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
