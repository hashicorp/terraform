package command

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/terraform/configs/configupgrade"
	"github.com/hashicorp/terraform/terraform"
	"github.com/hashicorp/terraform/tfdiags"
)

// ZeroTwelveUpgradeCommand is a Command implementation that can upgrade
// the configuration files for a module from pre-0.11 syntax to new 0.12
// idiom, while also flagging any suspicious constructs that will require
// human review.
type ZeroTwelveUpgradeCommand struct {
	Meta
}

func (c *ZeroTwelveUpgradeCommand) Run(args []string) int {
	args, err := c.Meta.process(args, true)
	if err != nil {
		return 1
	}

	var skipConfirm, force bool

	flags := c.Meta.extendedFlagSet("0.12upgrade")
	flags.BoolVar(&skipConfirm, "yes", false, "skip confirmation prompt")
	flags.BoolVar(&force, "force", false, "override duplicate upgrade heuristic")
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
			"The command 0.12upgrade expects only a single argument, giving the directory containing the module to upgrade.",
		))
		c.showDiagnostics(diags)
		return 1
	}

	dir = c.normalizePath(dir)

	sources, err := configupgrade.LoadModule(dir)
	if err != nil {
		if os.IsNotExist(err) {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Module directory not found",
				fmt.Sprintf("The given directory %s does not exist.", dir),
			))
		} else {
			diags = diags.Append(err)
		}
		c.showDiagnostics(diags)
		return 1
	}

	if len(sources) == 0 {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Not a module directory",
			fmt.Sprintf("The given directory %s does not contain any Terraform configuration files.", dir),
		))
		c.showDiagnostics(diags)
		return 1
	}

	// The config loader doesn't naturally populate our sources
	// map, so we'll do it manually so our diagnostics can have
	// source code snippets inside them.
	// This is weird, but this whole upgrade codepath is pretty
	// weird and temporary, so we'll accept it.
	if loader, err := c.initConfigLoader(); err == nil {
		parser := loader.Parser()
		for name, src := range sources {
			parser.ForceFileSource(filepath.Join(dir, name), src)
		}
	}

	if !force {
		// We'll check first if this directory already looks upgraded, so we
		// don't waste the user's time dealing with an interactive prompt
		// immediately followed by an error.
		if already, rng := sources.MaybeAlreadyUpgraded(); already {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Module already upgraded",
				Detail:   fmt.Sprintf("The module in directory %s has a version constraint that suggests it has already been upgraded for v0.12. If this is incorrect, either remove this constraint or override this heuristic with the -force argument. Upgrading a module that was already upgraded may change the meaning of that module.", dir),
				Subject:  rng.ToHCL().Ptr(),
			})
			c.showDiagnostics(diags)
			return 1
		}
	}

	if !skipConfirm {
		c.Ui.Output(fmt.Sprintf(`
This command will rewrite the configuration files in the given directory so
that they use the new syntax features from Terraform v0.12, and will identify
any constructs that may need to be adjusted for correct operation with
Terraform v0.12.

We recommend using this command in a clean version control work tree, so that
you can easily see the proposed changes as a diff against the latest commit.
If you have uncommited changes already present, we recommend aborting this
command and dealing with them before running this command again.
`))

		query := "Would you like to upgrade the module in the current directory?"
		if dir != "." {
			query = fmt.Sprintf("Would you like to upgrade the module in %s?", dir)
		}
		v, err := c.UIInput().Input(&terraform.InputOpts{
			Id:          "approve",
			Query:       query,
			Description: `Only 'yes' will be accepted to confirm.`,
		})
		if err != nil {
			diags = diags.Append(err)
			c.showDiagnostics(diags)
			return 1
		}
		if v != "yes" {
			c.Ui.Info("Upgrade cancelled.")
			return 0
		}

		c.Ui.Output(`-----------------------------------------------------------------------------`)
	}

	upgrader := &configupgrade.Upgrader{
		Providers:    c.providerResolver(),
		Provisioners: c.provisionerFactories(),
	}
	newSources, upgradeDiags := upgrader.Upgrade(sources)
	diags = diags.Append(upgradeDiags)
	if upgradeDiags.HasErrors() {
		c.showDiagnostics(diags)
		return 2
	}

	// Now we'll write the contents of newSources into the filesystem.
	for name, src := range newSources {
		fn := filepath.Join(dir, name)
		if src == nil {
			// indicates a file to be deleted
			err := os.Remove(fn)
			if err != nil {
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Failed to remove file",
					fmt.Sprintf("The file %s must be renamed as part of the upgrade process, but the old file could not be deleted: %s.", fn, err),
				))
			}
			continue
		}

		err := ioutil.WriteFile(fn, src, 0644)
		if err != nil {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Failed to write file",
				fmt.Sprintf("The file %s must be updated or created as part of the upgrade process, but there was an error while writing: %s.", fn, err),
			))
		}
	}

	c.showDiagnostics(diags)
	if diags.HasErrors() {
		return 2
	}

	if !skipConfirm {
		if len(diags) != 0 {
			c.Ui.Output(`-----------------------------------------------------------------------------`)
		}
		c.Ui.Output(c.Colorize().Color(`
[bold][green]Upgrade complete![reset]

The configuration files were upgraded successfully. Use your version control
system to review the proposed changes, make any necessary adjustments, and
then commit.
`))
		if len(diags) != 0 {
			// We checked for errors above, so these must be warnings.
			c.Ui.Output(`Some warnings were generated during the upgrade, as shown above. These
indicate situations where Terraform could not decide on an appropriate course
of action without further human input.

Where possible, these have also been marked with TF-UPGRADE-TODO comments to
mark the locations where a decision must be made. After reviewing and adjusting
these, manually remove the TF-UPGRADE-TODO comment before continuing.
`)
		}

	}
	return 0
}

func (c *ZeroTwelveUpgradeCommand) Help() string {
	helpText := `
Usage: terraform 0.12upgrade [module-dir]

  Rewrites the .tf files for a single module that was written for a Terraform
  version prior to v0.12 so that it uses new syntax features from v0.12
  and later.

  Also rewrites constructs that behave differently after v0.12, and flags any
  suspicious constructs that require human review,

  By default, 0.12upgrade rewrites the files in the current working directory.
  However, a path to a different directory can be provided. The command will
  prompt for confirmation interactively unless the -yes option is given.

Options:

  -yes        Skip the initial introduction messages and interactive
              confirmation. This can be used to run this command in
              batch from a script.

  -force      Override the heuristic that attempts to detect if a
              configuration is already written for v0.12 or later.
              Some of the transformations made by this command are
              not idempotent, so re-running against the same module
              may change the meanings expressions in the module.
`
	return strings.TrimSpace(helpText)
}

func (c *ZeroTwelveUpgradeCommand) Synopsis() string {
	return "Rewrites pre-0.12 module source code for v0.12"
}
