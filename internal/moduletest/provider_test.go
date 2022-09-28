package moduletest

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/zclconf/go-cty-debug/ctydebug"
	"github.com/zclconf/go-cty/cty"
)

func TestProvider(t *testing.T) {

	assertionConfig := cty.ObjectVal(map[string]cty.Value{
		"component": cty.StringVal("spline_reticulator"),
		"equal": cty.MapVal(map[string]cty.Value{
			"match": cty.ObjectVal(map[string]cty.Value{
				"description": cty.StringVal("this should match"),
				"got":         cty.StringVal("a"),
				"want":        cty.StringVal("a"),
			}),
			"unmatch": cty.ObjectVal(map[string]cty.Value{
				"description": cty.StringVal("this should not match"),
				"got":         cty.StringVal("a"),
				"want":        cty.StringVal("b"),
			}),
		}),
		"check": cty.MapVal(map[string]cty.Value{
			"pass": cty.ObjectVal(map[string]cty.Value{
				"description": cty.StringVal("this should pass"),
				"condition":   cty.True,
			}),
			"fail": cty.ObjectVal(map[string]cty.Value{
				"description": cty.StringVal("this should fail"),
				"condition":   cty.False,
			}),
		}),
	})

	// The provider code expects to receive an object that was decoded from
	// HCL using the schema, so to make sure we're testing a more realistic
	// situation here we'll require the config to conform to the schema. If
	// this fails, it's a bug in the configuration definition above rather
	// than in the provider itself.
	for _, err := range assertionConfig.Type().TestConformance(testAssertionsSchema.Block.ImpliedType()) {
		t.Error(err)
	}

	p := NewProvider()

	configureResp := p.ConfigureProvider(providers.ConfigureProviderRequest{
		Config: cty.EmptyObjectVal,
	})
	if got, want := len(configureResp.Diagnostics), 1; got != want {
		t.Fatalf("got %d Configure diagnostics, but want %d", got, want)
	}
	if got, want := configureResp.Diagnostics[0].Description().Summary, "The test provider is experimental"; got != want {
		t.Fatalf("wrong diagnostic message\ngot:  %s\nwant: %s", got, want)
	}

	validateResp := p.ValidateResourceConfig(providers.ValidateResourceConfigRequest{
		TypeName: "test_assertions",
		Config:   assertionConfig,
	})
	if got, want := len(validateResp.Diagnostics), 0; got != want {
		t.Fatalf("got %d ValidateResourceTypeConfig diagnostics, but want %d", got, want)
	}

	planResp := p.PlanResourceChange(providers.PlanResourceChangeRequest{
		TypeName:         "test_assertions",
		Config:           assertionConfig,
		PriorState:       cty.NullVal(assertionConfig.Type()),
		ProposedNewState: assertionConfig,
	})
	if got, want := len(planResp.Diagnostics), 0; got != want {
		t.Fatalf("got %d PlanResourceChange diagnostics, but want %d", got, want)
	}
	planned := planResp.PlannedState
	if got, want := planned, assertionConfig; !want.RawEquals(got) {
		t.Fatalf("wrong planned new value\n%s", ctydebug.DiffValues(want, got))
	}

	gotComponents := p.TestResults()
	wantComponents := map[string]*Component{
		"spline_reticulator": {
			Assertions: map[string]*Assertion{
				"pass": {
					Outcome:     Pending,
					Description: "this should pass",
				},
				"fail": {
					Outcome:     Pending,
					Description: "this should fail",
				},
				"match": {
					Outcome:     Pending,
					Description: "this should match",
				},
				"unmatch": {
					Outcome:     Pending,
					Description: "this should not match",
				},
			},
		},
	}
	if diff := cmp.Diff(wantComponents, gotComponents); diff != "" {
		t.Fatalf("wrong test results after planning\n%s", diff)
	}

	applyResp := p.ApplyResourceChange(providers.ApplyResourceChangeRequest{
		TypeName:     "test_assertions",
		Config:       assertionConfig,
		PriorState:   cty.NullVal(assertionConfig.Type()),
		PlannedState: planned,
	})
	if got, want := len(applyResp.Diagnostics), 0; got != want {
		t.Fatalf("got %d ApplyResourceChange diagnostics, but want %d", got, want)
	}
	final := applyResp.NewState
	if got, want := final, assertionConfig; !want.RawEquals(got) {
		t.Fatalf("wrong new value\n%s", ctydebug.DiffValues(want, got))
	}

	gotComponents = p.TestResults()
	wantComponents = map[string]*Component{
		"spline_reticulator": {
			Assertions: map[string]*Assertion{
				"pass": {
					Outcome:     Passed,
					Description: "this should pass",
					Message:     "condition passed",
				},
				"fail": {
					Outcome:     Failed,
					Description: "this should fail",
					Message:     "condition failed",
				},
				"match": {
					Outcome:     Passed,
					Description: "this should match",
					Message:     "correct value\n    got: \"a\"\n",
				},
				"unmatch": {
					Outcome:     Failed,
					Description: "this should not match",
					Message:     "wrong value\n    got:  \"a\"\n    want: \"b\"\n",
				},
			},
		},
	}
	if diff := cmp.Diff(wantComponents, gotComponents); diff != "" {
		t.Fatalf("wrong test results after applying\n%s", diff)
	}

}
