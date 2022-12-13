package testconfigs

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hashicorp/terraform/internal/getproviders"
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

// ProviderRequirements collects all of the declared provider dependencies
// across all scenarios in the suite and returns them in the form expected
// by Terraform's provider plugin installer.
//
// NOTE: This doesn't include requirements for any modules that the test
// steps might refer to. If a caller needs to know the full set of requirements
// to execute the test scenarios then it will need to also load the
// configuration rooted at each designated module and incorporate their
// own provider requirements.
func (s *Suite) ProviderRequirements() getproviders.Requirements {
	ret := make(getproviders.Requirements)
	for _, scenario := range s.Scenarios {
		for _, reqt := range scenario.ProviderReqs.RequiredProviders {
			addr := reqt.Type

			// The model of version constraints in the configs package is still
			// the old one using a different upstream module to represent versions,
			// so we'll need to shim that out here for now. The two parsers
			// don't exactly agree in practice so this might produce new errors.
			// TODO: Use the new parser throughout all provider version work so
			// we can get the better error messages it produces in more situations.
			constraints, err := getproviders.ParseVersionConstraints(reqt.Requirement.Required.String())
			if err != nil {
				// Since this is just a prototype we'll just ignore errors here
				// for now and assume that we'll make the configs package
				// use the new constraints model before we finalize this for real.
				continue
			}

			ret[addr] = append(ret[addr], constraints...)
		}
	}
	return ret
}
