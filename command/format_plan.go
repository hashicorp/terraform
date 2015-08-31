package command

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/colorstring"
)

// FormatPlanOpts are the options for formatting a plan.
type FormatPlanOpts struct {
	// Plan is the plan to format. This is required.
	Plan *terraform.Plan

	// Color is the colorizer. This is optional.
	Color *colorstring.Colorize

	// ModuleDepth is the depth of the modules to expand. By default this
	// is zero which will not expand modules at all.
	ModuleDepth int
}

// FormatPlan takes a plan and returns a
func FormatPlan(opts *FormatPlanOpts) string {
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
	buf *bytes.Buffer, m *terraform.ModuleDiff, opts *FormatPlanOpts) {
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

		if moduleName != "" {
			name = moduleName + "." + name
		}

		// Determine the color for the text (green for adding, yellow
		// for change, red for delete), and symbol, and output the
		// resource header.
		color := "yellow"
		symbol := "~"
		switch rdiff.ChangeType() {
		case terraform.DiffDestroyCreate:
			color = "green"
			symbol = "-/+"
		case terraform.DiffCreate:
			color = "green"
			symbol = "+"
		case terraform.DiffDestroy:
			color = "red"
			symbol = "-"
		}

		buf.WriteString(opts.Color.Color(fmt.Sprintf(
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
				newResource = opts.Color.Color(" [red](forces new resource)")
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
		buf.WriteString(opts.Color.Color("[reset]\n"))
	}
}

// formatPlanModuleSingle will output the given module and all of its
// resources.
func formatPlanModuleSingle(
	buf *bytes.Buffer, m *terraform.ModuleDiff, opts *FormatPlanOpts) {
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
