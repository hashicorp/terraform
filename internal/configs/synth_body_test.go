package configs

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
)

func TestSynthBodyContent(t *testing.T) {
	tests := map[string]struct {
		Values    map[string]cty.Value
		Schema    *hcl.BodySchema
		DiagCount int
	}{
		"empty": {
			Values:    map[string]cty.Value{},
			Schema:    &hcl.BodySchema{},
			DiagCount: 0,
		},
		"missing required attribute": {
			Values: map[string]cty.Value{},
			Schema: &hcl.BodySchema{
				Attributes: []hcl.AttributeSchema{
					{
						Name:     "nonexist",
						Required: true,
					},
				},
			},
			DiagCount: 1, // missing required attribute
		},
		"missing optional attribute": {
			Values: map[string]cty.Value{},
			Schema: &hcl.BodySchema{
				Attributes: []hcl.AttributeSchema{
					{
						Name: "nonexist",
					},
				},
			},
			DiagCount: 0,
		},
		"extraneous attribute": {
			Values: map[string]cty.Value{
				"foo": cty.StringVal("unwanted"),
			},
			Schema:    &hcl.BodySchema{},
			DiagCount: 1, // unsupported attribute
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			body := SynthBody("synth", test.Values)
			_, diags := body.Content(test.Schema)
			if got, want := len(diags), test.DiagCount; got != want {
				t.Errorf("wrong number of diagnostics %d; want %d", got, want)
				for _, diag := range diags {
					t.Logf("- %s", diag)
				}
			}
		})
	}
}
