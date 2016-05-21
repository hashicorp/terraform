package command

import (
	"bytes"
	"flag"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// OutputCommand is a Command implementation that reads an output
// from a Terraform state and prints it.
type OutputCommand struct {
	Meta
}

func (c *OutputCommand) Run(args []string) int {
	args = c.Meta.process(args, false)

	var module string
	cmdFlags := flag.NewFlagSet("output", flag.ContinueOnError)
	cmdFlags.StringVar(&c.Meta.statePath, "state", DefaultStateFilename, "path")
	cmdFlags.StringVar(&module, "module", "", "module")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }

	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	args = cmdFlags.Args()
	if len(args) > 2 {
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

	index := ""
	if len(args) > 1 {
		index = args[1]
	}

	stateStore, err := c.Meta.State()
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error reading state: %s", err))
		return 1
	}

	if module == "" {
		module = "root"
	} else {
		module = "root." + module
	}

	// Get the proper module we want to get outputs for
	modPath := strings.Split(module, ".")

	state := stateStore.State()
	mod := state.ModuleByPath(modPath)

	if mod == nil {
		c.Ui.Error(fmt.Sprintf(
			"The module %s could not be found. There is nothing to output.",
			module))
		return 1
	}

	if state.Empty() || len(mod.Outputs) == 0 {
		c.Ui.Error(fmt.Sprintf(
			"The state file has no outputs defined. Define an output\n" +
				"in your configuration with the `output` directive and re-run\n" +
				"`terraform apply` for it to become available."))
		return 1
	}

	if name == "" {
		c.Ui.Output(outputsAsString(state, nil, false))
		return 0
	}

	v, ok := mod.Outputs[name]
	if !ok {
		c.Ui.Error(fmt.Sprintf(
			"The output variable requested could not be found in the state\n" +
				"file. If you recently added this to your configuration, be\n" +
				"sure to run `terraform apply`, since the state won't be updated\n" +
				"with new output variables until that command is run."))
		return 1
	}

	switch output := v.Value.(type) {
	case string:
		c.Ui.Output(output)
		return 0
	case []interface{}:
		if index == "" {
			c.Ui.Output(formatListOutput("", "", output))
			break
		}

		indexInt, err := strconv.Atoi(index)
		if err != nil {
			c.Ui.Error(fmt.Sprintf(
				"The index %q requested is not valid for the list output\n"+
					"%q - indices must be numeric, and in the range 0-%d", index, name,
				len(output)-1))
			break
		}

		if indexInt < 0 || indexInt >= len(output) {
			c.Ui.Error(fmt.Sprintf(
				"The index %d requested is not valid for the list output\n"+
					"%q - indices must be in the range 0-%d", indexInt, name,
				len(output)-1))
			break
		}

		c.Ui.Output(fmt.Sprintf("%s", output[indexInt]))
		return 0
	case map[string]interface{}:
		if index == "" {
			c.Ui.Output(formatMapOutput("", "", output))
			break
		}

		if value, ok := output[index]; ok {
			c.Ui.Output(fmt.Sprintf("%s", value))
			return 0
		} else {
			return 1
		}
	default:
		c.Ui.Error(fmt.Sprintf("Unknown output type: %T", v.Type))
		return 1
	}

	return 0
}

func formatListOutput(indent, outputName string, outputList []interface{}) string {
	keyIndent := ""

	outputBuf := new(bytes.Buffer)
	if outputName != "" {
		outputBuf.WriteString(fmt.Sprintf("%s%s = [", indent, outputName))
		keyIndent = "  "
	}

	for _, value := range outputList {
		outputBuf.WriteString(fmt.Sprintf("\n%s%s%s", indent, keyIndent, value))
	}

	if outputName != "" {
		outputBuf.WriteString(fmt.Sprintf("\n%s]", indent))
	}

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
		outputBuf.WriteString(fmt.Sprintf("\n%s}", indent))
	}

	return strings.TrimPrefix(outputBuf.String(), "\n")
}

func (c *OutputCommand) Help() string {
	helpText := `
Usage: terraform output [options] [NAME]

  Reads an output variable from a Terraform state file and prints
  the value.  If NAME is not specified, all outputs are printed.

Options:

  -state=path      Path to the state file to read. Defaults to
                   "terraform.tfstate".

  -no-color        If specified, output won't contain any color.

  -module=name     If specified, returns the outputs for a
                   specific module

`
	return strings.TrimSpace(helpText)
}

func (c *OutputCommand) Synopsis() string {
	return "Read an output from a state file"
}
