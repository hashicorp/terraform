package command

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"
	ctyjson "github.com/zclconf/go-cty/cty/json"

	"github.com/hashicorp/terraform/repl"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/tfdiags"
)

// OutputCommand is a Command implementation that reads an output
// from a Terraform state and prints it.
type OutputCommand struct {
	Meta

	// Unit tests may set rawPrint to capture the output from the -raw
	// option, which would normally go to stdout directly.
	rawPrint func(string)
}

func (c *OutputCommand) Run(args []string) int {
	args = c.Meta.process(args)
	var statePath string
	var jsonOutput, rawOutput bool
	cmdFlags := c.Meta.defaultFlagSet("output")
	cmdFlags.BoolVar(&jsonOutput, "json", false, "json")
	cmdFlags.BoolVar(&rawOutput, "raw", false, "raw")
	cmdFlags.StringVar(&statePath, "state", "", "path")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		c.Ui.Error(fmt.Sprintf("Error parsing command-line flags: %s\n", err.Error()))
		return 1
	}

	args = cmdFlags.Args()
	if len(args) > 1 {
		c.Ui.Error(
			"The output command expects exactly one argument with the name\n" +
				"of an output variable or no arguments to show all outputs.\n")
		cmdFlags.Usage()
		return 1
	}

	if jsonOutput && rawOutput {
		c.Ui.Error("The -raw and -json options are mutually-exclusive.\n")
		cmdFlags.Usage()
		return 1
	}

	if rawOutput && len(args) == 0 {
		c.Ui.Error("You must give the name of a single output value when using the -raw option.\n")
		cmdFlags.Usage()
		return 1
	}

	name := ""
	if len(args) > 0 {
		name = args[0]
	}

	if statePath != "" {
		c.Meta.statePath = statePath
	}

	var diags tfdiags.Diagnostics

	// Load the backend
	b, backendDiags := c.Backend(nil)
	diags = diags.Append(backendDiags)
	if backendDiags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}

	// This is a read-only command
	c.ignoreRemoteBackendVersionConflict(b)

	env, err := c.Workspace()
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error selecting workspace: %s", err))
		return 1
	}

	// Get the state
	stateStore, err := b.StateMgr(env)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to load state: %s", err))
		return 1
	}

	if err := stateStore.RefreshState(); err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to load state: %s", err))
		return 1
	}

	state := stateStore.State()
	if state == nil {
		state = states.NewState()
	}

	mod := state.RootModule()

	if !jsonOutput && (state.Empty() || len(mod.OutputValues) == 0) {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Warning,
			"No outputs found",
			"The state file either has no outputs defined, or all the defined "+
				"outputs are empty. Please define an output in your configuration "+
				"with the `output` keyword and run `terraform refresh` for it to "+
				"become available. If you are using interpolation, please verify "+
				"the interpolated value is not empty. You can use the "+
				"`terraform console` command to assist.",
		))
		c.showDiagnostics(diags)
		return 0
	}

	if name == "" {
		if jsonOutput {
			// Due to a historical accident, the switch from state version 2 to
			// 3 caused our JSON output here to be the full metadata about the
			// outputs rather than just the output values themselves as we'd
			// show in the single value case. We must now maintain that behavior
			// for compatibility, so this is an emulation of the JSON
			// serialization of outputs used in state format version 3.
			type OutputMeta struct {
				Sensitive bool            `json:"sensitive"`
				Type      json.RawMessage `json:"type"`
				Value     json.RawMessage `json:"value"`
			}
			outputs := map[string]OutputMeta{}

			for n, os := range mod.OutputValues {
				jsonVal, err := ctyjson.Marshal(os.Value, os.Value.Type())
				if err != nil {
					diags = diags.Append(err)
					c.showDiagnostics(diags)
					return 1
				}
				jsonType, err := ctyjson.MarshalType(os.Value.Type())
				if err != nil {
					diags = diags.Append(err)
					c.showDiagnostics(diags)
					return 1
				}
				outputs[n] = OutputMeta{
					Sensitive: os.Sensitive,
					Type:      json.RawMessage(jsonType),
					Value:     json.RawMessage(jsonVal),
				}
			}

			jsonOutputs, err := json.MarshalIndent(outputs, "", "  ")
			if err != nil {
				diags = diags.Append(err)
				c.showDiagnostics(diags)
				return 1
			}
			c.Ui.Output(string(jsonOutputs))
			return 0
		} else {
			c.Ui.Output(outputsAsString(state, false))
			return 0
		}
	}

	os, ok := mod.OutputValues[name]
	if !ok {
		c.Ui.Error(fmt.Sprintf(
			"The output variable requested could not be found in the state\n" +
				"file. If you recently added this to your configuration, be\n" +
				"sure to run `terraform apply`, since the state won't be updated\n" +
				"with new output variables until that command is run."))
		return 1
	}
	v := os.Value

	switch {
	case jsonOutput:
		jsonOutput, err := ctyjson.Marshal(v, v.Type())
		if err != nil {
			return 1
		}

		c.Ui.Output(string(jsonOutput))
	case rawOutput:
		strV, err := convert.Convert(v, cty.String)
		if err != nil {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Unsupported value for raw output",
				fmt.Sprintf(
					"The -raw option only supports strings, numbers, and boolean values, but output value %q is %s.\n\nUse the -json option for machine-readable representations of output values that have complex types.",
					name, v.Type().FriendlyName(),
				),
			))
			c.showDiagnostics(diags)
			return 1
		}
		if strV.IsNull() {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Unsupported value for raw output",
				fmt.Sprintf(
					"The value for output value %q is null, so -raw mode cannot print it.",
					name,
				),
			))
			c.showDiagnostics(diags)
			return 1
		}
		if !strV.IsKnown() {
			// Since we're working with values from the state it would be very
			// odd to end up in here, but we'll handle it anyway to avoid a
			// panic in case our rules somehow change in future.
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Unsupported value for raw output",
				fmt.Sprintf(
					"The value for output value %q won't be known until after a successful terraform apply, so -raw mode cannot print it.",
					name,
				),
			))
			c.showDiagnostics(diags)
			return 1
		}
		// If we get out here then we should have a valid string to print.
		// We're writing it directly to the output here so that a shell caller
		// will get exactly the value and no extra whitespace.
		str := strV.AsString()
		if c.rawPrint != nil {
			c.rawPrint(str)
		} else {
			fmt.Print(str)
		}
	default:
		result := repl.FormatValue(v, 0)
		c.Ui.Output(result)
	}

	return 0
}

func (c *OutputCommand) Help() string {
	helpText := `
Usage: terraform output [options] [NAME]

  Reads an output variable from a Terraform state file and prints
  the value. With no additional arguments, output will display all
  the outputs for the root module.  If NAME is not specified, all
  outputs are printed.

Options:

  -state=path      Path to the state file to read. Defaults to
                   "terraform.tfstate".

  -no-color        If specified, output won't contain any color.

  -json            If specified, machine readable output will be
                   printed in JSON format.

  -raw             For value types that can be automatically
                   converted to a string, will print the raw
                   string directly, rather than a human-oriented
                   representation of the value.
`
	return strings.TrimSpace(helpText)
}

func (c *OutputCommand) Synopsis() string {
	return "Show output values from your root module"
}
