package configs

import (
	"testing"
)

func TestResourceForEachExperiment(t *testing.T) {
	parser := NewParser(nil)
	_, diags := parser.LoadConfigDir("testdata/experiments/resource-for_each-with-experiment")
	if diags.HasErrors() {
		t.Errorf("got errors; want none")
		for _, diag := range diags {
			t.Logf("- %s", diag)
		}
	}
	if len(diags) != 1 {
		t.Fatalf("got %d diagnostic(s); want 1", len(diags))
	}
	if got, want := diags[0].Summary, `Experimental feature "resource_for_each" is active`; got != want {
		t.Errorf("Wrong warning message\ngot:  %s\nwant: %s", got, want)
	}
}
