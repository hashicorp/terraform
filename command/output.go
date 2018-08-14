package command

import (
	"bytes"
	"flag"
	"fmt"
	"sort"
	"strings"

	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/tfdiags"
)

// OutputCommand is a Command implementation that reads an output
// from a Terraform state and prints it.
type OutputCommand struct {
	Meta
}

func (c *OutputCommand) Run(args []string) int {
	args, err := c.Meta.process(args, false)
	if err != nil {
		return 1
	}

	var module string
	var jsonOutput bool
	cmdFlags := flag.NewFlagSet("output", flag.ContinueOnError)
	cmdFlags.BoolVar(&jsonOutput, "json", false, "json")
	cmdFlags.StringVar(&c.Meta.statePath, "state", DefaultStateFilename, "path")
	cmdFlags.StringVar(&module, "module", "", "module")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
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

	name := ""
	if len(args) > 0 {
		name = args[0]
	}

	var diags tfdiags.Diagnostics

	// Load the backend
	b, backendDiags := c.Backend(nil)
	diags = diags.Append(backendDiags)
	if backendDiags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}

	env := c.Workspace()

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

	moduleAddr, addrDiags := addrs.ParseModuleInstanceStr(module)
	diags = diags.Append(addrDiags)
	if addrDiags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}

	state := stateStore.State()
	mod := state.Module(moduleAddr)
	if mod == nil {
		c.Ui.Error(fmt.Sprintf(
			"The module %s could not be found. There is nothing to output.",
			module))
		return 1
	}

	// TODO: We need to do an eval walk here to make sure all of the output
	// values recorded in the state are up-to-date.
	c.Ui.Error("output command not yet updated to do eval walk")
	return 1

	if !jsonOutput && (state.Empty() || len(mod.OutputValues) == 0) {
		c.Ui.Error(
			"The state file either has no outputs defined, or all the defined\n" +
				"outputs are empty. Please define an output in your configuration\n" +
				"with the `output` keyword and run `terraform refresh` for it to\n" +
				"become available. If you are using interpolation, please verify\n" +
				"the interpolated value is not empty. You can use the \n" +
				"`terraform console` command to assist.")
		return 1
	}

	if name == "" {
		if jsonOutput {
			vals := make(map[string]cty.Value, len(mod.OutputValues))
			for n, os := range mod.OutputValues {
				vals[n] = os.Value
			}
			valsObj := cty.ObjectVal(vals)
			jsonOutputs, err := ctyjson.Marshal(valsObj, valsObj.Type())
			if err != nil {
				return 1
			}

			c.Ui.Output(string(jsonOutputs))
			return 0
		} else {
			c.Ui.Output(outputsAsString(state, moduleAddr, nil, false))
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

	if jsonOutput {
		jsonOutput, err := ctyjson.Marshal(v, v.Type())
		if err != nil {
			return 1
		}

		c.Ui.Output(string(jsonOutput))
	} else {
		c.Ui.Error("TODO: update output command to use the same value renderer as the console")
		return 1
	}

	return 0
}

func formatNestedList(indent string, outputList []interface{}) string {
	outputBuf := new(bytes.Buffer)
	outputBuf.WriteString(fmt.Sprintf("%s[", indent))

	lastIdx := len(outputList) - 1

	for i, value := range outputList {
		outputBuf.WriteString(fmt.Sprintf("\n%s%s%s", indent, "    ", value))
		if i != lastIdx {
			outputBuf.WriteString(",")
		}
	}

	outputBuf.WriteString(fmt.Sprintf("\n%s]", indent))
	return strings.TrimPrefix(outputBuf.String(), "\n")
}

func formatListOutput(indent, outputName string, outputList []interface{}) string {
	keyIndent := ""

	outputBuf := new(bytes.Buffer)

	if outputName != "" {
		outputBuf.WriteString(fmt.Sprintf("%s%s = [", indent, outputName))
		keyIndent = "    "
	}

	lastIdx := len(outputList) - 1

	for i, value := range outputList {
		switch typedValue := value.(type) {
		case string:
			outputBuf.WriteString(fmt.Sprintf("\n%s%s%s", indent, keyIndent, value))
		case []interface{}:
			outputBuf.WriteString(fmt.Sprintf("\n%s%s", indent,
				formatNestedList(indent+keyIndent, typedValue)))
		case map[string]interface{}:
			outputBuf.WriteString(fmt.Sprintf("\n%s%s", indent,
				formatNestedMap(indent+keyIndent, typedValue)))
		}

		if lastIdx != i {
			outputBuf.WriteString(",")
		}
	}

	if outputName != "" {
		if len(outputList) > 0 {
			outputBuf.WriteString(fmt.Sprintf("\n%s]", indent))
		} else {
			outputBuf.WriteString("]")
		}
	}

	return strings.TrimPrefix(outputBuf.String(), "\n")
}

func formatNestedMap(indent string, outputMap map[string]interface{}) string {
	ks := make([]string, 0, len(outputMap))
	for k, _ := range outputMap {
		ks = append(ks, k)
	}
	sort.Strings(ks)

	outputBuf := new(bytes.Buffer)
	outputBuf.WriteString(fmt.Sprintf("%s{", indent))

	lastIdx := len(outputMap) - 1
	for i, k := range ks {
		v := outputMap[k]
		outputBuf.WriteString(fmt.Sprintf("\n%s%s = %v", indent+"    ", k, v))

		if lastIdx != i {
			outputBuf.WriteString(",")
		}
	}

	outputBuf.WriteString(fmt.Sprintf("\n%s}", indent))

	return strings.TrimPrefix(outputBuf.String(), "\n")
}

func formatMapOutput(indent, outputName string, outputMap map[string]interface{}) string {
	ks := make([]string, 0, len(outputMap))
	for k, _ := range outputMap {
		ks = append(ks, k)
	}
	sort.Strings(ks)

	keyIndent := ""

	outputBuf := new(bytes.Buffer)
	if outputName != "" {
		outputBuf.WriteString(fmt.Sprintf("%s%s = {", indent, outputName))
		keyIndent = "  "
	}

	for _, k := range ks {
		v := outputMap[k]
		outputBuf.WriteString(fmt.Sprintf("\n%s%s%s = %v", indent, keyIndent, k, v))
	}

	if outputName != "" {
		if len(outputMap) > 0 {
			outputBuf.WriteString(fmt.Sprintf("\n%s}", indent))
		} else {
			outputBuf.WriteString("}")
		}
	}

	return strings.TrimPrefix(outputBuf.String(), "\n")
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

  -module=name     If specified, returns the outputs for a
                   specific module

  -json            If specified, machine readable output will be
                   printed in JSON format

`
	return strings.TrimSpace(helpText)
}

func (c *OutputCommand) Synopsis() string {
	return "Read an output from a state file"
}
