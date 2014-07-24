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

	if len(s.Resources) == 0 {
		return "The state file is empty. No resources are represented."
	}

	buf := new(bytes.Buffer)
	buf.WriteString("[reset]")

	// First get the names of all the resources so we can show them
	// in alphabetical order.
	names := make([]string, 0, len(s.Resources))
	for name, _ := range s.Resources {
		names = append(names, name)
	}
	sort.Strings(names)

	// Go through each resource and begin building up the output.
	for _, k := range names {
		rs := s.Resources[k]
		id := rs.ID
		if id == "" {
			id = "<not created>"
		}

		taintStr := ""
		if s.Tainted != nil {
			if _, ok := s.Tainted[k]; ok {
				taintStr = " (tainted)"
			}
		}

		buf.WriteString(fmt.Sprintf("%s:%s\n", k, taintStr))
		buf.WriteString(fmt.Sprintf("  id = %s\n", id))

		// Sort the attributes
		attrKeys := make([]string, 0, len(rs.Attributes))
		for ak, _ := range rs.Attributes {
			// Skip the id attribute since we just show the id directly
			if ak == "id" {
				continue
			}

			attrKeys = append(attrKeys, ak)
		}
		sort.Strings(attrKeys)

		// Output each attribute
		for _, ak := range attrKeys {
			av := rs.Attributes[ak]
			buf.WriteString(fmt.Sprintf("  %s = %s\n", ak, av))
		}
	}

	if len(s.Outputs) > 0 {
		buf.WriteString("\nOutputs:\n\n")

		// Sort the outputs
		ks := make([]string, 0, len(s.Outputs))
		for k, _ := range s.Outputs {
			ks = append(ks, k)
		}
		sort.Strings(ks)

		// Output each output k/v pair
		for _, k := range ks {
			v := s.Outputs[k]
			buf.WriteString(fmt.Sprintf("%s = %s\n", k, v))
		}
	}

	return c.Color(strings.TrimSpace(buf.String()))
}
