package command

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/terraform"
)

// parseOutputName extracts the name and index from the remaining arguments in
// the output command.
func parseOutputNameIndex(args []string) (string, string, error) {
	if len(args) > 2 {
		return "", "", fmt.Errorf(
			"This command expects exactly one argument with the name\n" +
				"of an output variable or no arguments to show all outputs.\n")
	}

	name := ""
	if len(args) > 0 {
		name = args[0]
	}

	index := ""

	if len(args) > 1 {
		index = args[1]
	}

	return name, index, nil
}

// moduleFromState returns a module from a Terraform state.
func moduleFromState(state *terraform.State, module string) (*terraform.ModuleState, error) {
	if module == "" {
		module = "root"
	} else {
		module = "root." + module
	}

	// Get the proper module we want to get outputs for
	modPath := strings.Split(module, ".")
	mod := state.ModuleByPath(modPath)

	if mod == nil {
		return nil, fmt.Errorf("The module %s could not be found. There is nothing to output.", module)
	}

	if state.Empty() || len(mod.Outputs) == 0 {
		return nil, fmt.Errorf(
			"The state file has no outputs defined. Define an output\n" +
				"in your configuration with the `output` directive and re-run\n" +
				"`terraform apply` for it to become available.")
	}

	return mod, nil
}

// singleOutputAsString looks for a single output in a module path and outputs
// as a string.
func singleOutputAsString(mod *terraform.ModuleState, name, index string) (string, error) {
	v, ok := mod.Outputs[name]
	if !ok {
		return "", fmt.Errorf(
			"The output variable requested could not be found in the state.\n" +
				"If you recently added this to your configuration, be\n" +
				"sure to run `terraform apply`, since the state won't be updated\n" +
				"with new output variables until that command is run.")
	}

	var s string
	switch output := v.Value.(type) {
	case string:
		s = output
	case []interface{}:
		if index == "" {
			s = formatListOutput("", "", output)
			break
		}

		indexInt, err := strconv.Atoi(index)
		if err != nil {
			return "", fmt.Errorf(
				"The index %q requested is not valid for the list output\n"+
					"%q - indices must be numeric, and in the range 0-%d", index, name,
				len(output)-1)
		}

		if indexInt < 0 || indexInt >= len(output) {
			return "", fmt.Errorf(
				"The index %d requested is not valid for the list output\n"+
					"%q - indices must be in the range 0-%d", indexInt, name,
				len(output)-1)
		}

		s = fmt.Sprintf("%s", output[indexInt])
	case map[string]interface{}:
		if index == "" {
			s = formatMapOutput("", "", output)
		}

		if value, ok := output[index]; ok {
			s = fmt.Sprintf("%s", value)
		} else {
			return "", fmt.Errorf("")
		}
	default:
		panic(fmt.Errorf("Unknown output type: %T", output))
	}
	return s, nil
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
		if len(outputList) > 0 {
			outputBuf.WriteString(fmt.Sprintf("\n%s]", indent))
		} else {
			outputBuf.WriteString("]")
		}
	}

	return strings.TrimPrefix(outputBuf.String(), "\n")
}

func formatMapOutput(indent, outputName string, outputMap map[string]interface{}) string {
	ks := make([]string, 0, len(outputMap))
	for k := range outputMap {
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

// allOutputsAsString returns all outputs, pretty formatted, for a given
// module path.
func allOutputsAsString(mod *terraform.ModuleState, schema []*config.Output, includeHeader bool) string {
	outputs := mod.Outputs
	outputBuf := new(bytes.Buffer)
	if len(outputs) > 0 {
		schemaMap := make(map[string]*config.Output)
		if schema != nil {
			for _, s := range schema {
				schemaMap[s.Name] = s
			}
		}

		if includeHeader {
			outputBuf.WriteString("[reset][bold][green]\nOutputs:\n\n")
		}

		// Output the outputs in alphabetical order
		keyLen := 0
		ks := make([]string, 0, len(outputs))
		for key := range outputs {
			ks = append(ks, key)
			if len(key) > keyLen {
				keyLen = len(key)
			}
		}
		sort.Strings(ks)

		for _, k := range ks {
			schema, ok := schemaMap[k]
			if ok && schema.Sensitive {
				outputBuf.WriteString(fmt.Sprintf("%s = <sensitive>\n", k))
				continue
			}

			v := outputs[k]
			switch typedV := v.Value.(type) {
			case string:
				outputBuf.WriteString(fmt.Sprintf("%s = %s\n", k, typedV))
			case []interface{}:
				outputBuf.WriteString(formatListOutput("", k, typedV))
				outputBuf.WriteString("\n")
			case map[string]interface{}:
				outputBuf.WriteString(formatMapOutput("", k, typedV))
				outputBuf.WriteString("\n")
			}
		}
	}

	return strings.TrimSpace(outputBuf.String())
}
