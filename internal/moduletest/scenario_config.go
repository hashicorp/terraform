package moduletest

import (
	"fmt"
	"path/filepath"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configload"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// LoadMainConfig loads the Terraform configuration which is rooted in the
// test scenario directory.
//
// LoadMainConfig uses the given RunEnvironment to gain access to some services
// relating to the surrounding development environment outside of the particular
// test scenario.
func (s *Scenario) LoadMainConfig(env *RunEnvironment) (*configs.Config, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	configDir := s.RootModulePath()
	modulesDir := filepath.Join(configDir, ".terraform", "modules")

	// We use a separate loader from what other commands might use but we
	// use the configs.Parser we were given under the assumption that it's
	// the same parser that will be used to run all scenarios and do any other
	// config parsing that the main test command will do, so that ultimately
	// any source files we load here will be cached inside the parser for
	// possible use in diagnostics later.
	loaderConfig := &configload.Config{
		ModulesDir: modulesDir,
		Services:   env.Services,
	}
	loader, err := configload.NewLoaderWithParser(loaderConfig, env.ConfigParser)
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to create config loader for testing",
			fmt.Sprintf("Could not create a config loader for testing scenario %s: %s.", s.Addr(), err),
		))
		return nil, diags
	}
	loader.AllowLanguageExperiments(env.ExperimentsAllowed)

	const errSummary = "Invalid test scenario configuration"

	if !loader.IsConfigDir(configDir) {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			errSummary,
			fmt.Sprintf("The test scenario directory %s doesn't contain any Terraform configuration files.", configDir),
		))
		return nil, diags
	}

	config, hclDiags := loader.LoadConfig(configDir)
	diags = diags.Append(hclDiags)

	// We have various constraints on what's allowed in a test scenario
	// configuration so that we can work with them in the slightly-unusual
	// execution environment of the test harness.
	if config.Module.Backend != nil {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  errSummary,
			Detail:   "A test scenario configuration must not have a backend configuration, because the test harness controls state storage during testing.",
			Subject:  &config.Module.Backend.DeclRange,
		})
	}
	if config.Module.CloudConfig != nil {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  errSummary,
			Detail:   "A test scenario configuration must not have a Terraform Cloud configuration, because the test harness controls operaions and state storage during testing.",
			Subject:  &config.Module.CloudConfig.DeclRange,
		})
	}
	for _, v := range config.Module.Variables {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  errSummary,
			Detail:   "A test scenario configuration must not declare any input variables, because it must be self-contained and ready to use.",
			Subject:  v.DeclRange.Ptr(),
		})
	}
	if config.Module.ProviderRequirements != nil {
		for _, pr := range config.Module.ProviderRequirements.RequiredProviders {
			if len(pr.Aliases) != 0 {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  errSummary,
					Detail:   "A test scenario configuration cannot require external provider configurations, because it must be self-contained and ready to use.",
					Subject:  pr.DeclRange.Ptr(),
				})
			}
		}
	}

	return config, diags
}
