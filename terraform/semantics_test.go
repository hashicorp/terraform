package terraform

import (
	"testing"

	"github.com/zclconf/go-cty/cty"
)

func TestSMCUserVariables(t *testing.T) {
	c := testModule(t, "smc-uservars")

	// Required variables not set
	diags := checkInputVariables(c.Module.Variables, nil)
	if !diags.HasErrors() {
		t.Fatal("check succeeded, but want errors")
	}

	// Required variables set, optional variables unset
	diags = checkInputVariables(c.Module.Variables, InputValues{
		"foo": &InputValue{
			Value:      cty.StringVal("bar"),
			SourceType: ValueFromCLIArg,
		},
	})
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	// Mapping complete override
	diags = checkInputVariables(c.Module.Variables, InputValues{
		"foo": &InputValue{
			Value:      cty.StringVal("bar"),
			SourceType: ValueFromCLIArg,
		},
		"map": &InputValue{
			Value:      cty.StringVal("baz"),
			SourceType: ValueFromCLIArg,
		},
	})
	if !diags.HasErrors() {
		t.Fatal("check succeeded, but want errors")
	}

}
