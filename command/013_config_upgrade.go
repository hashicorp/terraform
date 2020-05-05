package command

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/internal/initwd"
	"github.com/hashicorp/terraform/moduledeps"
	"github.com/hashicorp/terraform/tfdiags"
)

// ZeroThirteenUpgradeCommand upgrades configuration files for a module
// to include explicit provider source settings
type ZeroThirteenUpgradeCommand struct {
	Meta
}

func (c *ZeroThirteenUpgradeCommand) Run(args []string) int {
	args = c.Meta.process(args)
	flags := c.Meta.defaultFlagSet("0.13upgrade")
	flags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := flags.Parse(args); err != nil {
		return 1
	}

	var diags tfdiags.Diagnostics

	var dir string
	args = flags.Args()
	switch len(args) {
	case 0:
		dir = "."
	case 1:
		dir = args[0]
	default:
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Too many arguments",
			"The command 0.13upgrade expects only a single argument, giving the directory containing the module to upgrade.",
		))
		c.showDiagnostics(diags)
		return 1
	}

	// Check for user-supplied plugin path
	var err error
	if c.pluginPath, err = c.loadPluginPath(); err != nil {
		c.Ui.Error(fmt.Sprintf("Error loading plugin path: %s", err))
		return 1
	}

	dir = c.normalizePath(dir)

	// Upgrade only if some configuration is present
	empty, err := configs.IsEmptyDir(dir)
	if err != nil {
		diags = diags.Append(fmt.Errorf("Error checking configuration: %s", err))
		return 1
	}
	if empty {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Not a module directory",
			fmt.Sprintf("The given directory %s does not contain any Terraform configuration files.", dir),
		))
		c.showDiagnostics(diags)
		return 1
	}

	// Early-load the config so that we can check provider dependencies
	earlyConfig, earlyConfDiags := c.loadConfigEarly(dir)
	if earlyConfDiags.HasErrors() {
		c.Ui.Error(strings.TrimSpace("Failed to load configuration"))
		diags = diags.Append(earlyConfDiags)
		c.showDiagnostics(diags)
		return 1
	}

	{
		// Before we go further, we'll check to make sure none of the modules
		// in the configuration declare that they don't support this Terraform
		// version, so we can produce a version-related error message rather
		// than potentially-confusing downstream errors.
		versionDiags := initwd.CheckCoreVersionRequirements(earlyConfig)
		diags = diags.Append(versionDiags)
		if versionDiags.HasErrors() {
			c.showDiagnostics(diags)
			return 1
		}
	}

	// Find the provider dependencies
	configDeps, depsDiags := earlyConfig.ProviderDependencies()
	if depsDiags.HasErrors() {
		c.Ui.Error(strings.TrimSpace("Could not detect provider dependencies"))
		diags = diags.Append(depsDiags)
		c.showDiagnostics(diags)
		return 1
	}

	// Detect source for each provider
	providerSources, detectDiags := detectProviderSources(configDeps.Providers)
	if detectDiags.HasErrors() {
		c.Ui.Error(strings.TrimSpace("Unable to detect sources for providers"))
		diags = diags.Append(detectDiags)
		c.showDiagnostics(diags)
		return 1
	}

	if len(providerSources) == 0 {
		c.Ui.Output("No non-default providers found. Your configuration is ready to use!")
		return 0
	}

	// Generate the required providers configuration
	genDiags := generateRequiredProviders(providerSources, dir)
	diags = diags.Append(genDiags)

	c.showDiagnostics(diags)
	if diags.HasErrors() {
		return 2
	}

	if len(diags) != 0 {
		c.Ui.Output(`-----------------------------------------------------------------------------`)
	}
	c.Ui.Output(c.Colorize().Color(`
[bold][green]Upgrade complete![reset]

Use your version control system to review the proposed changes, make any
necessary adjustments, and then commit.
`))

	return 0
}

// For providers which need a source attribute, detect and return source
// FIXME: currently does not filter or detect sources
func detectProviderSources(providers moduledeps.Providers) (map[string]string, tfdiags.Diagnostics) {
	sources := make(map[string]string)
	for provider := range providers {
		sources[provider.Type] = provider.String()
	}
	return sources, nil
}

var providersTemplate = template.Must(template.New("providers.tf").Parse(`terraform {
  required_providers {
    {{- range $type, $source := .}}
    {{$type}} = {
      source = "{{$source}}"
    }
    {{- end}}
  }
}
`))

// Generate a file with terraform.required_providers blocks for each provider
func generateRequiredProviders(providerSources map[string]string, dir string) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	// Find unused file named "providers.tf", or fall back to e.g. "providers-1.tf"
	path := filepath.Join(dir, "providers.tf")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		for i := 1; ; i++ {
			path = filepath.Join(dir, fmt.Sprintf("providers-%d.tf", i))
			if _, err := os.Stat(path); os.IsNotExist(err) {
				break
			}
		}
	}

	f, err := os.Create(path)
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Unable to create providers file",
			fmt.Sprintf("Error when generating providers configuration at '%s': %s", path, err),
		))
		return diags
	}
	defer f.Close()

	err = providersTemplate.Execute(f, providerSources)
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Unable to create providers file",
			fmt.Sprintf("Error when generating providers configuration at '%s': %s", path, err),
		))
		return diags
	}

	return nil
}

func (c *ZeroThirteenUpgradeCommand) Help() string {
	helpText := `
Usage: terraform 0.13upgrade [module-dir]

  Generates a "providers.tf" configuration file which includes source
  configuration for every non-default provider.
`
	return strings.TrimSpace(helpText)
}

func (c *ZeroThirteenUpgradeCommand) Synopsis() string {
	return "Rewrites pre-0.13 module source code for v0.13"
}
