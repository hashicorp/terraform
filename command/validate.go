package command

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/terraform"
	"github.com/hashicorp/terraform/tfdiags"
)

// ValidateCommand is a Command implementation that validates the terraform files
type ValidateCommand struct {
	Meta
}

const defaultPath = "."

func (c *ValidateCommand) Run(args []string) int {
	args, err := c.Meta.process(args, true)
	if err != nil {
		return 1
	}

	if c.Meta.variableArgs.items == nil {
		c.Meta.variableArgs = newRawFlags("-var")
	}
	varValues := c.Meta.variableArgs.Alias("-var")
	varFiles := c.Meta.variableArgs.Alias("-var-file")

	var jsonOutput bool
	cmdFlags := c.Meta.defaultFlagSet("validate")
	cmdFlags.BoolVar(&jsonOutput, "json", false, "produce JSON output")
	cmdFlags.Var(varValues, "var", "variables")
	cmdFlags.Var(varFiles, "var-file", "variable file")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	// After this point, we must only produce JSON output if JSON mode is
	// enabled, so all errors should be accumulated into diags and we'll
	// print out a suitable result at the end, depending on the format
	// selection. All returns from this point on must be tail-calls into
	// c.showResults in order to produce the expected output.
	var diags tfdiags.Diagnostics
	args = cmdFlags.Args()

	var dirPath string
	if len(args) == 1 {
		dirPath = args[0]
	} else {
		dirPath = "."
	}
	dir, err := filepath.Abs(dirPath)
	if err != nil {
		diags = diags.Append(fmt.Errorf("unable to locate module: %s", err))
		return c.showResults(diags, jsonOutput)
	}

	// Check for user-supplied plugin path
	if c.pluginPath, err = c.loadPluginPath(); err != nil {
		diags = diags.Append(fmt.Errorf("error loading plugin path: %s", err))
		return c.showResults(diags, jsonOutput)
	}

	validateDiags := c.validate(dir)
	diags = diags.Append(validateDiags)

	return c.showResults(diags, jsonOutput)
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

	opts := c.contextOpts()
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

func (c *ValidateCommand) showResults(diags tfdiags.Diagnostics, jsonOutput bool) int {
	switch {
	case jsonOutput:
		// FIXME: Eventually we'll probably want to factor this out somewhere
		// to support machine-readable outputs for other commands too, but for
		// now it's simplest to do this inline here.
		type Pos struct {
			Line   int `json:"line"`
			Column int `json:"column"`
			Byte   int `json:"byte"`
		}
		type Range struct {
			Filename string `json:"filename"`
			Start    Pos    `json:"start"`
			End      Pos    `json:"end"`
		}
		type Diagnostic struct {
			Severity string `json:"severity,omitempty"`
			Summary  string `json:"summary,omitempty"`
			Detail   string `json:"detail,omitempty"`
			Range    *Range `json:"range,omitempty"`
		}
		type Output struct {
			// We include some summary information that is actually redundant
			// with the detailed diagnostics, but avoids the need for callers
			// to re-implement our logic for deciding these.
			Valid        bool         `json:"valid"`
			ErrorCount   int          `json:"error_count"`
			WarningCount int          `json:"warning_count"`
			Diagnostics  []Diagnostic `json:"diagnostics"`
		}

		var output Output
		output.Valid = true // until proven otherwise
		for _, diag := range diags {
			var jsonDiag Diagnostic
			switch diag.Severity() {
			case tfdiags.Error:
				jsonDiag.Severity = "error"
				output.ErrorCount++
				output.Valid = false
			case tfdiags.Warning:
				jsonDiag.Severity = "warning"
				output.WarningCount++
			}

			desc := diag.Description()
			jsonDiag.Summary = desc.Summary
			jsonDiag.Detail = desc.Detail

			ranges := diag.Source()
			if ranges.Subject != nil {
				subj := ranges.Subject
				jsonDiag.Range = &Range{
					Filename: subj.Filename,
					Start: Pos{
						Line:   subj.Start.Line,
						Column: subj.Start.Column,
						Byte:   subj.Start.Byte,
					},
					End: Pos{
						Line:   subj.End.Line,
						Column: subj.End.Column,
						Byte:   subj.End.Byte,
					},
				}
			}

			output.Diagnostics = append(output.Diagnostics, jsonDiag)
		}
		if output.Diagnostics == nil {
			// Make sure this always appears as an array in our output, since
			// this is easier to consume for dynamically-typed languages.
			output.Diagnostics = []Diagnostic{}
		}

		j, err := json.MarshalIndent(&output, "", "  ")
		if err != nil {
			// Should never happen because we fully-control the input here
			panic(err)
		}
		c.Ui.Output(string(j))

	default:
		if len(diags) == 0 {
			c.Ui.Output(c.Colorize().Color("[green][bold]Success![reset] The configuration is valid.\n"))
		} else {
			c.showDiagnostics(diags)

			if !diags.HasErrors() {
				c.Ui.Output(c.Colorize().Color("[green][bold]Success![reset] The configuration is valid, but there were some validation warnings as shown above.\n"))
			}
		}
	}

	if diags.HasErrors() {
		return 1
	}
	return 0
}

func (c *ValidateCommand) Synopsis() string {
	return "Validates the Terraform files"
}

func (c *ValidateCommand) Help() string {
	helpText := `
Usage: terraform validate [options] [dir]

  Validate the configuration files in a directory, referring only to the
  configuration and not accessing any remote services such as remote state,
  provider APIs, etc.

  Validate runs checks that verify whether a configuration is
  internally-consistent, regardless of any provided variables or existing
  state. It is thus primarily useful for general verification of reusable
  modules, including correctness of attribute names and value types.

  It is safe to run this command automatically, for example as a post-save
  check in a text editor or as a test step for a re-usable module in a CI
  system.

  Validation requires an initialized working directory with any referenced
  plugins and modules installed. To initialize a working directory for
  validation without accessing any configured remote backend, use:
      terraform init -backend=false

  If dir is not specified, then the current directory will be used.

  To verify configuration in the context of a particular run (a particular
  target workspace, operation variables, etc), use the terraform plan
  subcommand instead, which includes an implied validation check.

Options:

  -json        Produce output in a machine-readable JSON format, suitable for
               use in e.g. text editor integrations.

`
	return strings.TrimSpace(helpText)
}
