package testconfigs

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Suite represents an entire test suite. This is the top-level configuration
// type and represents the full set of test scenarios for a particular
// module.
type Suite struct {
	ModuleDir string
	Scenarios map[string]*Scenario
}

func LoadSuiteForModule(moduleDir string) (*Suite, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	ret := &Suite{
		ModuleDir: filepath.Clean(moduleDir),
		Scenarios: make(map[string]*Scenario),
	}

	scenariosDir := filepath.Join(moduleDir, "test-scenarios")
	scenariosFiles, err := os.ReadDir(scenariosDir)
	if os.IsNotExist(err) {
		// It's fine for the scenarios directory to not exist at all. That
		// just means that this module doesn't have any test scenarios.
		return ret, diags
	}
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Cannot read test scenarios directory",
			fmt.Sprintf("Error reading the test scenarios directory: %s.", err),
		))
		return ret, diags
	}

	for _, entry := range scenariosFiles {
		filename := filepath.Join(scenariosDir, entry.Name())
		scenario, moreDiags := LoadScenarioFile(filename)
		diags = diags.Append(moreDiags)
		if diags.HasErrors() {
			continue
		}
		for _, step := range scenario.Steps {
			if step.ModuleDir == "" {
				// If the module argument isn't set explicitly then it
				// defaults to testing the module whose test suite this is.
				step.ModuleDir = moduleDir
			}
		}
		ret.Scenarios[scenario.Name] = scenario
	}

	return ret, diags
}
