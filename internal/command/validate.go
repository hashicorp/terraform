// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// ValidateCommand is a Command implementation that validates the terraform files
type ValidateCommand struct {
	Meta
}

func (c *ValidateCommand) Run(rawArgs []string) int {
	// Parse and apply global view arguments
	common, rawArgs := arguments.ParseView(rawArgs)
	c.View.Configure(common)

	// Parse and validate flags
	args, diags := arguments.ParseValidate(rawArgs)
	if diags.HasErrors() {
		c.View.Diagnostics(diags)
		c.View.HelpPrompt("validate")
		return 1
	}

	view := views.NewValidate(args.ViewType, c.View)

	// After this point, we must only produce JSON output if JSON mode is
	// enabled, so all errors should be accumulated into diags and we'll
	// print out a suitable result at the end, depending on the format
	// selection. All returns from this point on must be tail-calls into
	// view.Results in order to produce the expected output.

	dir, err := filepath.Abs(args.Path)
	if err != nil {
		diags = diags.Append(fmt.Errorf("unable to locate module: %s", err))
		return view.Results(diags)
	}

	// Check for user-supplied plugin path
	if c.pluginPath, err = c.loadPluginPath(); err != nil {
		diags = diags.Append(fmt.Errorf("error loading plugin path: %s", err))
		return view.Results(diags)
	}

	validateDiags := c.validate(dir, args.TestDirectory, args.NoTests)
	diags = diags.Append(validateDiags)

	// Validating with dev overrides in effect means that the result might
	// not be valid for a stable release, so we'll warn about that in case
	// the user is trying to use "terraform validate" as a sort of pre-flight
	// check before submitting a change.
	diags = diags.Append(c.providerDevOverrideRuntimeWarnings())

	return view.Results(diags)
}

func (c *ValidateCommand) validate(dir, testDir string, noTests bool) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	var cfg *configs.Config

	if noTests {
		cfg, diags = c.loadConfig(dir)
	} else {
		cfg, diags = c.loadConfigWithTests(dir, testDir)
	}
	if diags.HasErrors() {
		return diags
	}

	validate := func(cfg *configs.Config) tfdiags.Diagnostics {
		var diags tfdiags.Diagnostics

		opts, err := c.contextOpts()
		if err != nil {
			diags = diags.Append(err)
			return diags
		}

		tfCtx, ctxDiags := terraform.NewContext(opts)
		diags = diags.Append(ctxDiags)
		if ctxDiags.HasErrors() {
			return diags
		}

		return diags.Append(tfCtx.Validate(cfg, nil))
	}

	diags = diags.Append(validate(cfg))

	if noTests {
		return diags
	}

	validatedModules := make(map[string]bool)

	// We'll also do a quick validation of the Terraform test files. These live
	// outside the Terraform graph so we have to do this separately.
	for _, file := range cfg.Module.Tests {

		// The file validation only returns warnings so we'll just add them
		// without checking anything about them.
		diags = diags.Append(file.Validate(cfg))

		for _, run := range file.Runs {

			if run.Module != nil {
				// Then we can also validate the referenced modules, but we are
				// only going to do this is if they are local modules.
				//
				// Basically, local testing modules are something the user can
				// reasonably go and fix. If it's a module being downloaded from
				// the registry, the expectation is that the author of the
				// module should have ran `terraform validate` themselves.
				if _, ok := run.Module.Source.(addrs.ModuleSourceLocal); ok {

					if validated := validatedModules[run.Module.Source.String()]; !validated {

						// Since we can reference the same module twice, let's
						// not validate the same thing multiple times.

						validatedModules[run.Module.Source.String()] = true
						diags = diags.Append(validate(run.ConfigUnderTest))
					}

				}

				diags = diags.Append(run.Validate(run.ConfigUnderTest))
			} else {
				diags = diags.Append(run.Validate(cfg))
			}

		}
	}

	return diags
}

func (c *ValidateCommand) Synopsis() string {
	return "Check whether the configuration is valid"
}

func (c *ValidateCommand) Help() string {
	helpText := `
Usage: terraform [global options] validate [options]

  Validate the configuration files in a directory, referring only to the
  configuration and not accessing any remote services such as remote state,
  provider APIs, etc.

  Validate runs checks that verify whether a configuration is syntactically
  valid and internally consistent, regardless of any provided variables or
  existing state. It is thus primarily useful for general verification of
  reusable modules, including correctness of attribute names and value types.

  It is safe to run this command automatically, for example as a post-save
  check in a text editor or as a test step for a re-usable module in a CI
  system.

  Validation requires an initialized working directory with any referenced
  plugins and modules installed. To initialize a working directory for
  validation without accessing any configured remote backend, use:
      terraform init -backend=false

  To verify configuration in the context of a particular run (a particular
  target workspace, input variable values, etc), use the 'terraform plan'
  command instead, which includes an implied validation check.

Options:

  -json                 Produce output in a machine-readable JSON format, 
                        suitable for use in text editor integrations and other 
                        automated systems. Always disables color.

  -no-color             If specified, output won't contain any color.

  -no-tests             If specified, Terraform will not validate test files.

  -test-directory=path	Set the Terraform test directory, defaults to "tests".
`
	return strings.TrimSpace(helpText)
}
