package configs

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"

	"github.com/hashicorp/terraform/internal/experiments"
)

func TestExperimentsConfig(t *testing.T) {
	// The experiment registrations are global, so we need to do some special
	// patching in order to get a predictable set for our tests.
	current := experiments.Experiment("current")
	concluded := experiments.Experiment("concluded")
	currentExperiments := experiments.NewSet(current)
	concludedExperiments := map[experiments.Experiment]string{
		concluded: "Reticulate your splines.",
	}
	defer experiments.OverrideForTesting(t, currentExperiments, concludedExperiments)()

	t.Run("current", func(t *testing.T) {
		parser := NewParser(nil)
		mod, diags := parser.LoadConfigDir("testdata/experiments/current")
		if got, want := len(diags), 1; got != want {
			t.Fatalf("wrong number of diagnostics %d; want %d", got, want)
		}
		got := diags[0]
		want := &hcl.Diagnostic{
			Severity: hcl.DiagWarning,
			Summary:  `Experimental feature "current" is active`,
			Detail:   "Experimental features are subject to breaking changes in future minor or patch releases, based on feedback.\n\nIf you have feedback on the design of this feature, please open a GitHub issue to discuss it.",
			Subject: &hcl.Range{
				Filename: "testdata/experiments/current/current_experiment.tf",
				Start:    hcl.Pos{Line: 2, Column: 18, Byte: 29},
				End:      hcl.Pos{Line: 2, Column: 25, Byte: 36},
			},
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("wrong warning\n%s", diff)
		}
		if got, want := len(mod.ActiveExperiments), 1; got != want {
			t.Errorf("wrong number of experiments %d; want %d", got, want)
		}
		if !mod.ActiveExperiments.Has(current) {
			t.Errorf("module does not indicate current experiment as active")
		}
	})
	t.Run("concluded", func(t *testing.T) {
		parser := NewParser(nil)
		_, diags := parser.LoadConfigDir("testdata/experiments/concluded")
		if got, want := len(diags), 1; got != want {
			t.Fatalf("wrong number of diagnostics %d; want %d", got, want)
		}
		got := diags[0]
		want := &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  `Experiment has concluded`,
			Detail:   `Experiment "concluded" is no longer available. Reticulate your splines.`,
			Subject: &hcl.Range{
				Filename: "testdata/experiments/concluded/concluded_experiment.tf",
				Start:    hcl.Pos{Line: 2, Column: 18, Byte: 29},
				End:      hcl.Pos{Line: 2, Column: 27, Byte: 38},
			},
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("wrong error\n%s", diff)
		}
	})
	t.Run("concluded", func(t *testing.T) {
		parser := NewParser(nil)
		_, diags := parser.LoadConfigDir("testdata/experiments/unknown")
		if got, want := len(diags), 1; got != want {
			t.Fatalf("wrong number of diagnostics %d; want %d", got, want)
		}
		got := diags[0]
		want := &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  `Unknown experiment keyword`,
			Detail:   `There is no current experiment with the keyword "unknown".`,
			Subject: &hcl.Range{
				Filename: "testdata/experiments/unknown/unknown_experiment.tf",
				Start:    hcl.Pos{Line: 2, Column: 18, Byte: 29},
				End:      hcl.Pos{Line: 2, Column: 25, Byte: 36},
			},
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("wrong error\n%s", diff)
		}
	})
	t.Run("invalid", func(t *testing.T) {
		parser := NewParser(nil)
		_, diags := parser.LoadConfigDir("testdata/experiments/invalid")
		if got, want := len(diags), 1; got != want {
			t.Fatalf("wrong number of diagnostics %d; want %d", got, want)
		}
		got := diags[0]
		want := &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  `Invalid expression`,
			Detail:   `A static list expression is required.`,
			Subject: &hcl.Range{
				Filename: "testdata/experiments/invalid/invalid_experiments.tf",
				Start:    hcl.Pos{Line: 2, Column: 17, Byte: 28},
				End:      hcl.Pos{Line: 2, Column: 24, Byte: 35},
			},
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("wrong error\n%s", diff)
		}
	})
}
