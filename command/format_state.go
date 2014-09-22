package command

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/colorstring"
)

// FormatState takes a state and returns a string
func FormatState(s *terraform.State, c *colorstring.Colorize) string {
	if c == nil {
		panic("colorize not given")
	}

	if len(s.Modules) == 0 {
		return "The state file is empty. No resources are represented."
	}

	var buf bytes.Buffer
	for _, m := range s.Modules {
		formatStateModule(&buf, m, c)
	}

	return c.Color(strings.TrimSpace(buf.String()))
}

func formatStateModule(buf *bytes.Buffer, m *terraform.ModuleState, c *colorstring.Colorize) {
	buf.WriteString("[reset]")

	// First get the names of all the resources so we can show them
	// in alphabetical order.
	names := make([]string, 0, len(m.Resources))
	for name, _ := range m.Resources {
		names = append(names, name)
	}
	sort.Strings(names)

	// Go through each resource and begin building up the output.
	for _, k := range names {
		rs := m.Resources[k]
		is := rs.Primary
		id := is.ID
		if id == "" {
			id = "<not created>"
		}

		taintStr := ""
		if len(rs.Tainted) > 0 {
			taintStr = " (tainted)"
		}

		buf.WriteString(fmt.Sprintf("%s:%s\n", k, taintStr))
		buf.WriteString(fmt.Sprintf("  id = %s\n", id))

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
}
