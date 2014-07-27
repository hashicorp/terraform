package command

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/colorstring"
)

// FormatPlan takes a plan and returns a
func FormatPlan(p *terraform.Plan, c *colorstring.Colorize) string {
	if p.Diff == nil || p.Diff.Empty() {
		return "This plan does nothing."
	}

	if c == nil {
		c = &colorstring.Colorize{
			Colors: colorstring.DefaultColors,
			Reset:  false,
		}
	}

	buf := new(bytes.Buffer)

	// We want to output the resources in sorted order to make things
	// easier to scan through, so get all the resource names and sort them.
	names := make([]string, 0, len(p.Diff.Resources))
	for name, _ := range p.Diff.Resources {
		names = append(names, name)
	}
	sort.Strings(names)

	// Go through each sorted name and start building the output
	for _, name := range names {
		rdiff := p.Diff.Resources[name]

		// Determine the color for the text (green for adding, yellow
		// for change, red for delete), and symbol, and output the
		// resource header.
		color := "yellow"
		symbol := "~"
		if rdiff.RequiresNew() && rdiff.Destroy {
			color = "green"
			symbol = "-/+"
		} else if rdiff.RequiresNew() {
			color = "green"
			symbol = "+"
		} else if rdiff.Destroy {
			color = "red"
			symbol = "-"
		}
		buf.WriteString(c.Color(fmt.Sprintf(
			"[%s]%s %s\n",
			color, symbol, name)))

		// Get all the attributes that are changing, and sort them. Also
		// determine the longest key so that we can align them all.
		keyLen := 0
		keys := make([]string, 0, len(rdiff.Attributes))
		for key, _ := range rdiff.Attributes {
			// Skip the ID since we do that specially
			if key == "id" {
				continue
			}

			keys = append(keys, key)
			if len(key) > keyLen {
				keyLen = len(key)
			}
		}
		sort.Strings(keys)

		// Go through and output each attribute
		for _, attrK := range keys {
			attrDiff := rdiff.Attributes[attrK]

			v := attrDiff.New
			if attrDiff.NewComputed {
				v = "<computed>"
			}

			newResource := ""
			if attrDiff.RequiresNew && rdiff.Destroy {
				newResource = " (forces new resource)"
			}

			buf.WriteString(fmt.Sprintf(
				"    %s:%s %#v => %#v%s\n",
				attrK,
				strings.Repeat(" ", keyLen-len(attrK)),
				attrDiff.Old,
				v,
				newResource))
		}

		// Write the reset color so we don't overload the user's terminal
		buf.WriteString(c.Color("[reset]\n"))
	}

	return strings.TrimSpace(buf.String())
}
