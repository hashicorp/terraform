package command

import (
	"fmt"
	"os"

	"github.com/mitchellh/cli"

	"github.com/hashicorp/terraform/tfdiags"
)

// ProvidersMirrorCommand is a Command implementation that creates a local
// of the providers required by the current configuration.
type ProvidersMirrorCommand struct {
	Meta
}

func (c *ProvidersMirrorCommand) Help() string {
	return providersMirrorCommandHelp
}

func (c *ProvidersMirrorCommand) Synopsis() string {
	return "Creates a local mirror of the providers used in the configuration"
}

func (c *ProvidersMirrorCommand) Run(args []string) int {
	args, err := c.Meta.process(args, false)
	if err != nil {
		return 1
	}

	cmdFlags := c.Meta.defaultFlagSet("providers mirror")
	if err := cmdFlags.Parse(args); err != nil {
		c.Ui.Error(fmt.Sprintf("Error parsing command-line flags: %s\n", err.Error()))
		return 1
	}

	args = cmdFlags.Args()
	if len(args) != 1 {
		return cli.RunResultHelp
	}

	var diags tfdiags.Diagnostics

	if c.ProviderSource == nil {
		// This should not happen in normal use, but it might arise in unit
		// tests if the test doesn't add a provider source, in which case
		// we'll panic explicitly here to make it clearer what's going on.
		// If you see a panic here, then the embedded Meta value inside the
		// command struct has not been populated correctly. If the panic is
		// in a unit test then you may need to provide a mock
		// getproviders.Source, or a real one directed at a fake registry/mirror.
		panic("providers mirror without a provider source")
	}

	targetDir := args[0]
	if info, err := os.Stat(targetDir); err != nil || !info.IsDir() {
		const summary = "Invalid target directory"
		switch {
		case err == nil:
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				summary,
				fmt.Sprintf("The target %q already exists and is not a directory.", targetDir),
			))
		case os.IsNotExist(err):
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				summary,
				fmt.Sprintf("The target directory %q does not exist.", targetDir),
			))
		default:
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				summary,
				fmt.Sprintf("Cannot use %q as target directory: %s.", targetDir, err),
			))
		}
		c.showDiagnostics(diags)
		return 1
	}

	cfg, moreDiags := c.loadConfigEarly(".")
	diags = diags.Append(moreDiags)
	if moreDiags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}

	deps, moreDiags := cfg.ProviderDependencies()
	diags = diags.Append(moreDiags)
	if moreDiags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}

	// Unlike other commands that install plugins for immediate use with the
	// current workspace, this command intentionally considers only the
	// configuration to ensure that it can be run in situations where it would
	// be inconvenient to initialize a backend, such as if we're preparing
	// a mirror that is intended to then be transported onto the system that
	// has the necessary backend connectivity/credentials but might not itself
	// be able to reach origin registries.
	reqs := deps.AllPluginRequirements()

	fmt.Printf("reqs %#v\n", reqs)

	c.showDiagnostics(diags)
	if diags.HasErrors() {
		return 1
	}
	return 0
}

const providersMirrorCommandHelp = `
Usage: terraform providers mirror <directory>

  Creates a local mirror of the providers used in the current configuration
  in the given directory.

  The target directory can then be used either directly as a filesystem mirror
  or as the document root for an HTTP server providing a network mirror, via
  the provider_installation settings in the CLI configuration file.
`
