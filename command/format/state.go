package format

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/colorstring"
)

// StateOpts are the options for formatting a state.
type StateOpts struct {
	// State is the state to format. This is required.
	State *states.State

	// Schemas are used to decode attributes. This is required.
	Schemas *terraform.Schemas

	// Color is the colorizer. This is optional.
	Color *colorstring.Colorize
}

// State takes a state and returns a string
func State(opts *StateOpts) string {
	if opts.Color == nil {
		panic("colorize not given")
	}

	if opts.Schemas == nil {
		panic("schemas not given")
	}

	s := opts.State
	if len(s.Modules) == 0 {
		return "The state file is empty. No resources are represented."
	}

	buf := bytes.NewBufferString("[reset]")
	p := blockBodyDiffPrinter{
		buf:    buf,
		color:  opts.Color,
		action: plans.NoOp,
	}

	// Format all the modules
	for _, m := range s.Modules {
		formatStateModule(p, m, opts.Schemas)
	}

	// Write the outputs for the root module
	m := s.RootModule()

	if m.OutputValues != nil {
		if len(m.OutputValues) > 0 {
			p.buf.WriteString("Outputs:\n\n")
		}

		// Sort the outputs
		ks := make([]string, 0, len(m.OutputValues))
		for k := range m.OutputValues {
			ks = append(ks, k)
		}
		sort.Strings(ks)

		// Output each output k/v pair
		for _, k := range ks {
			v := m.OutputValues[k]
			p.buf.WriteString(fmt.Sprintf("%s = ", k))
			p.writeValue(v.Value, plans.NoOp, 0)
			p.buf.WriteString("\n")
		}
	}

	trimmedOutput := strings.TrimSpace(p.buf.String())
	trimmedOutput += "[reset]"

	return opts.Color.Color(trimmedOutput)

}

func formatStateModule(p blockBodyDiffPrinter, m *states.Module, schemas *terraform.Schemas) {
	// First get the names of all the resources so we can show them
	// in alphabetical order.
	names := make([]string, 0, len(m.Resources))
	for name := range m.Resources {
		names = append(names, name)
	}
	sort.Strings(names)

	// Go through each resource and begin building up the output.
	for _, key := range names {
		for k, v := range m.Resources[key].Instances {
			// keep these in order to keep the current object first, and
			// provide deterministic output for the deposed objects
			type obj struct {
				header   string
				instance *states.ResourceInstanceObjectSrc
			}
			instances := []obj{}

			addr := m.Resources[key].Addr

			taintStr := ""
			if v.Current != nil && v.Current.Status == 'T' {
				taintStr = " (tainted)"
			}

			instances = append(instances,
				obj{fmt.Sprintf("# %s:%s\n", addr.Absolute(m.Addr).Instance(k), taintStr), v.Current})

			for dk, v := range v.Deposed {
				instances = append(instances,
					obj{fmt.Sprintf("# %s: (deposed object %s)\n", addr.Absolute(m.Addr).Instance(k), dk), v})
			}

			// Sort the instances for consistent output.
			// Starting the sort from the second index, so the current instance
			// is always first.
			sort.Slice(instances[1:], func(i, j int) bool {
				return instances[i+1].header < instances[j+1].header
			})

			for _, obj := range instances {
				header := obj.header
				instance := obj.instance
				p.buf.WriteString(header)
				if instance == nil {
					// this shouldn't happen, but there's nothing to do here so
					// don't panic below.
					continue
				}

				var schema *configschema.Block

				// TODO: Get the provider FQN when it is available from the
				// AbsoluteProviderConfig, in state.
				//
				// Check if the resource has a configured provider, otherwise
				// use the default provider.
				provider := m.Resources[key].ProviderConfig.ProviderConfig.Type
				if _, exists := schemas.Providers[provider.LegacyString()]; !exists {
					// This should never happen in normal use because we
					// should've loaded all of the schemas and checked things
					// prior to this point. We can't return errors here, but
					// since this is UI code we will try to do _something_
					// reasonable.
					p.buf.WriteString(fmt.Sprintf("# missing schema for provider %q\n\n", provider.LegacyString()))
					continue
				}

				switch addr.Mode {
				case addrs.ManagedResourceMode:
					schema, _ = schemas.ResourceTypeConfig(
						provider.LegacyString(),
						addr.Mode,
						addr.Type,
					)
					if schema == nil {
						p.buf.WriteString(fmt.Sprintf(
							"# missing schema for provider %q resource type %s\n\n", provider.LegacyString(), addr.Type))
						continue
					}

					p.buf.WriteString(fmt.Sprintf(
						"resource %q %q {",
						addr.Type,
						addr.Name,
					))
				case addrs.DataResourceMode:
					schema, _ = schemas.ResourceTypeConfig(
						provider.LegacyString(),
						addr.Mode,
						addr.Type,
					)
					if schema == nil {
						p.buf.WriteString(fmt.Sprintf(
							"# missing schema for provider %q data source %s\n\n", provider, addr.Type))
						continue
					}

					p.buf.WriteString(fmt.Sprintf(
						"data %q %q {",
						addr.Type,
						addr.Name,
					))
				default:
					// should never happen, since the above is exhaustive
					p.buf.WriteString(addr.String())
				}

				val, err := instance.Decode(schema.ImpliedType())
				if err != nil {
					fmt.Println(err.Error())
					break
				}

				path := make(cty.Path, 0, 3)
				bodyWritten := p.writeBlockBodyDiff(schema, val.Value, val.Value, 2, path)
				if bodyWritten {
					p.buf.WriteString("\n")
				}

				p.buf.WriteString("}\n\n")
			}
		}
	}
	p.buf.WriteString("\n")
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
	for k := range outputMap {
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
