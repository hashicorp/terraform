package format

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/colorstring"
)

// PlanOpts are the options for formatting a plan.
type PlanOpts struct {
	// Plan is the plan to format. This is required.
	Plan *terraform.Plan

	// Color is the colorizer. This is optional.
	Color *colorstring.Colorize

	// ModuleDepth is the depth of the modules to expand. By default this
	// is zero which will not expand modules at all.
	ModuleDepth int
}

// Plan takes a plan and returns a
func Plan(opts *PlanOpts) string {
	p := opts.Plan
	if p.Diff == nil || p.Diff.Empty() {
		return "This plan does nothing."
	}

	if opts.Color == nil {
		opts.Color = &colorstring.Colorize{
			Colors: colorstring.DefaultColors,
			Reset:  false,
		}
	}

	buf := new(bytes.Buffer)
	for _, m := range p.Diff.Modules {
		if len(m.Path)-1 <= opts.ModuleDepth || opts.ModuleDepth == -1 {
			formatPlanModuleExpand(buf, m, opts)
		} else {
			formatPlanModuleSingle(buf, m, opts)
		}
	}

	return strings.TrimSpace(buf.String())
}

// formatPlanModuleExpand will output the given module and all of its
// resources.
func formatPlanModuleExpand(
	buf *bytes.Buffer, m *terraform.ModuleDiff, opts *PlanOpts) {
	// Ignore empty diffs
	if m.Empty() {
		return
	}

	var moduleName string
	if !m.IsRoot() {
		moduleName = fmt.Sprintf("module.%s", strings.Join(m.Path[1:], "."))
	}

	// We want to output the resources in sorted order to make things
	// easier to scan through, so get all the resource names and sort them.
	names := make([]string, 0, len(m.Resources))
	for name, _ := range m.Resources {
		names = append(names, name)
	}
	sort.Strings(names)

	// Go through each sorted name and start building the output
	for _, name := range names {
		rdiff := m.Resources[name]
		if rdiff.Empty() {
			continue
		}

		dataSource := strings.HasPrefix(name, "data.")

		if moduleName != "" {
			name = moduleName + "." + name
		}

		// Determine the color for the text (green for adding, yellow
		// for change, red for delete), and symbol, and output the
		// resource header.
		color := "yellow"
		symbol := "~"
		oldValues := true
		switch rdiff.ChangeType() {
		case terraform.DiffDestroyCreate:
			color = "green"
			symbol = "-/+"
		case terraform.DiffCreate:
			color = "green"
			symbol = "+"
			oldValues = false

			// If we're "creating" a data resource then we'll present it
			// to the user as a "read" operation, so it's clear that this
			// operation won't change anything outside of the Terraform state.
			// Unfortunately by the time we get here we only have the name
			// to work with, so we need to cheat and exploit knowledge of the
			// naming scheme for data resources.
			if dataSource {
				symbol = "<="
				color = "cyan"
			}
		case terraform.DiffDestroy:
			color = "red"
			symbol = "-"
		}

		var extraAttr []string
		if rdiff.DestroyTainted {
			extraAttr = append(extraAttr, "tainted")
		}
		if rdiff.DestroyDeposed {
			extraAttr = append(extraAttr, "deposed")
		}
		var extraStr string
		if len(extraAttr) > 0 {
			extraStr = fmt.Sprintf(" (%s)", strings.Join(extraAttr, ", "))
		}

		buf.WriteString(opts.Color.Color(fmt.Sprintf(
			"[%s]%s %s%s\n",
			color, symbol, name, extraStr)))

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
			if v == "" && attrDiff.NewComputed {
				v = "<computed>"
			}

			if attrDiff.Sensitive {
				v = "<sensitive>"
			}

			updateMsg := ""
			if attrDiff.RequiresNew && rdiff.Destroy {
				updateMsg = opts.Color.Color(" [red](forces new resource)")
			} else if attrDiff.Sensitive && oldValues {
				updateMsg = opts.Color.Color(" [yellow](attribute changed)")
			}

			if oldValues {
				var u string
				if attrDiff.Sensitive {
					u = "<sensitive>"
				} else {
					u = attrDiff.Old
				}
				buf.WriteString(fmt.Sprintf(
					"    %s:%s %#v => %#v%s\n",
					attrK,
					strings.Repeat(" ", keyLen-len(attrK)),
					u,
					v,
					updateMsg))
			} else {
				buf.WriteString(fmt.Sprintf(
					"    %s:%s %#v%s\n",
					attrK,
					strings.Repeat(" ", keyLen-len(attrK)),
					v,
					updateMsg))
			}
		}

		// Write the reset color so we don't overload the user's terminal
		buf.WriteString(opts.Color.Color("[reset]\n"))
	}
}

// formatPlanModuleSingle will output the given module and all of its
// resources.
func formatPlanModuleSingle(
	buf *bytes.Buffer, m *terraform.ModuleDiff, opts *PlanOpts) {
	// Ignore empty diffs
	if m.Empty() {
		return
	}

	moduleName := fmt.Sprintf("module.%s", strings.Join(m.Path[1:], "."))

	// Determine the color for the text (green for adding, yellow
	// for change, red for delete), and symbol, and output the
	// resource header.
	color := "yellow"
	symbol := "~"
	switch m.ChangeType() {
	case terraform.DiffCreate:
		color = "green"
		symbol = "+"
	case terraform.DiffDestroy:
		color = "red"
		symbol = "-"
	}

	buf.WriteString(opts.Color.Color(fmt.Sprintf(
		"[%s]%s %s\n",
		color, symbol, moduleName)))
	buf.WriteString(fmt.Sprintf(
		"    %d resource(s)",
		len(m.Resources)))
	buf.WriteString(opts.Color.Color("[reset]\n"))
}
