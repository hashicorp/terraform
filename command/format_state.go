package command

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/colorstring"
)

// FormatStateOpts are the options for formatting a state.
type FormatStateOpts struct {
	// State is the state to format. This is required.
	State *terraform.State

	// Color is the colorizer. This is optional.
	Color *colorstring.Colorize

	// ModuleDepth is the depth of the modules to expand. By default this
	// is zero which will not expand modules at all.
	ModuleDepth int
}

// FormatState takes a state and returns a string
func FormatState(opts *FormatStateOpts) string {
	if opts.Color == nil {
		panic("colorize not given")
	}

	s := opts.State
	if len(s.Modules) == 0 {
		return "The state file is empty. No resources are represented."
	}

	var buf bytes.Buffer
	buf.WriteString("[reset]")

	// Format all the modules
	for _, m := range s.Modules {
		if len(m.Path)-1 <= opts.ModuleDepth || opts.ModuleDepth == -1 {
			formatStateModuleExpand(&buf, m, opts)
		} else {
			formatStateModuleSingle(&buf, m, opts)
		}
	}

	// Write the outputs for the root module
	m := s.RootModule()
	if len(m.Outputs) > 0 {
		buf.WriteString("\nOutputs:\n\n")

		// Sort the outputs
		ks := make([]string, 0, len(m.Outputs))
		for k, _ := range m.Outputs {
			ks = append(ks, k)
		}
		sort.Strings(ks)

		// Output each output k/v pair
		for _, k := range ks {
			v := m.Outputs[k]
			buf.WriteString(fmt.Sprintf("%s = %s\n", k, v))
		}
	}

	return opts.Color.Color(strings.TrimSpace(buf.String()))
}

func formatStateModuleExpand(
	buf *bytes.Buffer, m *terraform.ModuleState, opts *FormatStateOpts) {
	var moduleName string
	if !m.IsRoot() {
		moduleName = fmt.Sprintf("module.%s", strings.Join(m.Path[1:], "."))
	}

	// First get the names of all the resources so we can show them
	// in alphabetical order.
	names := make([]string, 0, len(m.Resources))
	for name, _ := range m.Resources {
		names = append(names, name)
	}
	sort.Strings(names)

	// Go through each resource and begin building up the output.
	for _, k := range names {
		name := k
		if moduleName != "" {
			name = moduleName + "." + name
		}

		rs := m.Resources[k]
		is := rs.Primary
		var id string
		if is != nil {
			id = is.ID
		}
		if id == "" {
			id = "<not created>"
		}

		taintStr := ""
		if len(rs.Tainted) > 0 {
			taintStr = " (tainted)"
		}

		buf.WriteString(fmt.Sprintf("%s:%s\n", name, taintStr))
		buf.WriteString(fmt.Sprintf("  id = %s\n", id))

		if is != nil {
			// Sort the attributes
			attrKeys := make([]string, 0, len(is.Attributes))
			for ak, _ := range is.Attributes {
				// Skip the id attribute since we just show the id directly
				if ak == "id" {
					continue
				}

				attrKeys = append(attrKeys, ak)
			}
			sort.Strings(attrKeys)

			// Output each attribute
			for _, ak := range attrKeys {
				av := is.Attributes[ak]
				buf.WriteString(fmt.Sprintf("  %s = %s\n", ak, av))
			}
		}
	}

	buf.WriteString("[reset]\n")
}

func formatStateModuleSingle(
	buf *bytes.Buffer, m *terraform.ModuleState, opts *FormatStateOpts) {
	// Header with the module name
	buf.WriteString(fmt.Sprintf("module.%s\n", strings.Join(m.Path[1:], ".")))

	// Now just write how many resources are in here.
	buf.WriteString(fmt.Sprintf("  %d resource(s)\n", len(m.Resources)))
}
