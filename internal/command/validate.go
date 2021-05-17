package command

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/views"
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

	validateDiags := c.validate(dir)
	diags = diags.Append(validateDiags)

	// Validating with dev overrides in effect means that the result might
	// not be valid for a stable release, so we'll warn about that in case
	// the user is trying to use "terraform validate" as a sort of pre-flight
	// check before submitting a change.
	diags = diags.Append(c.providerDevOverrideRuntimeWarnings())

	return view.Results(diags)
}

func (c *ValidateCommand) validate(dir string) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	cfg, cfgDiags := c.loadConfig(dir)
	diags = diags.Append(cfgDiags)

	if diags.HasErrors() {
		return diags
	}

	// "validate" is to check if the given module is valid regardless of
	// input values, current state, etc. Therefore we populate all of the
	// input values with unknown values of the expected type, allowing us
	// to perform a type check without assuming any particular values.
	varValues := make(terraform.InputValues)
	for name, variable := range cfg.Module.Variables {
		ty := variable.Type
		if ty == cty.NilType {
			// Can't predict the type at all, so we'll just mark it as
			// cty.DynamicVal (unknown value of cty.DynamicPseudoType).
			ty = cty.DynamicPseudoType
		}
		varValues[name] = &terraform.InputValue{
			Value:      cty.UnknownVal(ty),
			SourceType: terraform.ValueFromCLIArg,
		}
	}

	opts, err := c.contextOpts()
	if err != nil {
		diags = diags.Append(err)
		return diags
	}
	opts.Config = cfg
	opts.Variables = varValues

	tfCtx, ctxDiags := terraform.NewContext(opts)
	diags = diags.Append(ctxDiags)
	if ctxDiags.HasErrors() {
		return diags
	}

	validateDiags := tfCtx.Validate()
	diags = diags.Append(validateDiags)
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

  -json        Produce output in a machine-readable JSON format, suitable for
               use in text editor integrations and other automated systems.
               Always disables color.

  -no-color    If specified, output won't contain any color.
`
	return strings.TrimSpace(helpText)
}
