package moduletest

import (
	"fmt"
	"path/filepath"

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

	if !loader.IsConfigDir(configDir) {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Invalid test scenario configuration",
			fmt.Sprintf("The test scenario directory %s doesn't contain any Terraform configuration files.", configDir),
		))
		return nil, diags
	}

	config, hclDiags := loader.LoadConfig(configDir)
	diags = diags.Append(hclDiags)

	return config, diags
}
