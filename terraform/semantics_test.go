package terraform

import (
	"testing"

	"github.com/zclconf/go-cty/cty"
)

func TestSMCUserVariables(t *testing.T) {
	c := testModule(t, "smc-uservars")

	// No variables set
	diags := checkInputVariables(c.Module.Variables, nil)
	if !diags.HasErrors() {
		t.Fatal("check succeeded, but want errors")
	}

	// Required variables set, optional variables unset
	// This is still an error at this layer, since it's the caller's
	// responsibility to have already merged in any default values.
	diags = checkInputVariables(c.Module.Variables, InputValues{
		"foo": &InputValue{
			Value:      cty.StringVal("bar"),
			SourceType: ValueFromCLIArg,
		},
	})
	if !diags.HasErrors() {
		t.Fatal("check succeeded, but want errors")
	}

	// All variables set
	diags = checkInputVariables(c.Module.Variables, InputValues{
		"foo": &InputValue{
			Value:      cty.StringVal("bar"),
			SourceType: ValueFromCLIArg,
		},
		"bar": &InputValue{
			Value:      cty.StringVal("baz"),
			SourceType: ValueFromCLIArg,
		},
		"map": &InputValue{
			Value:      cty.StringVal("baz"), // okay because config has no type constraint
			SourceType: ValueFromCLIArg,
		},
	})
	if diags.HasErrors() {
		//t.Fatal("check succeeded, but want errors")
		t.Fatalf("unexpected errors: %s", diags.Err())
	}
}
