package testconfigs

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/configs"
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

func LoadSuiteForModule(moduleDir string, parser *configs.Parser) (*Suite, tfdiags.Diagnostics) {
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
		scenario, moreDiags := loadScenarioFile(filename, parser)
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

			rootModule, hclDiags := parser.LoadConfigDir(step.ModuleDir)
			diags = diags.Append(hclDiags)
			step.RootModule = rootModule
		}
		ret.Scenarios[scenario.Name] = scenario
	}

	// Before we return we'll catch any problems that we can detect based only
	// on the configuration source code.
	diags = diags.Append(ret.staticValidate())

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

func (s *Suite) staticValidate() tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	for _, scenario := range s.Scenarios {
		for _, step := range scenario.Steps {
			declVars := step.RootModule.Variables
			defnVars := step.VariableDefs

			for name, vc := range declVars {
				addr := vc.Addr()
				if vc.Required() {
					if _, defined := defnVars[addr]; !defined {
						diags = diags.Append(&hcl.Diagnostic{
							Severity: hcl.DiagError,
							Summary:  "No definition for required input variable",
							Detail:   fmt.Sprintf("The module requires a value for the input variable named %q.", name),
							// FIXME: This is an inexact source range, since we don't have a range for just the "variables" argument.
							Subject: vc.DeclRange.Ptr(),
						})
					}
				}
			}
			for addr, defn := range defnVars {
				name := addr.Name
				if _, declared := declVars[name]; !declared {
					diags = diags.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Value for undeclared input variable",
						Detail:   fmt.Sprintf("The module does not expect an input variable named %q.", name),
						Subject:  defn.Range().Ptr(),
					})
				}
			}

			// TODO: Also verify that the passed-in providers are sufficient
			// and match with the module's own declared or implied provider
			// configuration requirements.
		}
	}

	return diags
}
