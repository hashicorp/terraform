package renderers

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/hashicorp/terraform/internal/command/jsonformat/computed"

	"github.com/hashicorp/terraform/internal/command/format"
	"github.com/hashicorp/terraform/internal/plans"
)

var _ computed.DiffRenderer = (*mapRenderer)(nil)

func Map(elements map[string]computed.Diff) computed.DiffRenderer {
	maximumKeyLen := 0
	for key := range elements {
		if maximumKeyLen < len(key) {
			maximumKeyLen = len(key)
		}
	}

	return &mapRenderer{
		elements:      elements,
		maximumKeyLen: maximumKeyLen,
	}
}

type mapRenderer struct {
	NoWarningsRenderer

	elements      map[string]computed.Diff
	maximumKeyLen int
}

func (renderer mapRenderer) RenderHuman(diff computed.Diff, indent int, opts computed.RenderHumanOpts) string {
	if len(renderer.elements) == 0 {
		return fmt.Sprintf("{}%s%s", nullSuffix(opts.OverrideNullSuffix, diff.Action), forcesReplacement(diff.Replace, opts.OverrideForcesReplacement))
	}

	unchangedElements := 0

	// Sort the map elements by key, so we have a deterministic ordering in
	// the output.
	var keys []string
	for key := range renderer.elements {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	elementOpts := opts.Clone()
	if diff.Action == plans.Delete {
		elementOpts.OverrideNullSuffix = true
	}

	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("{%s\n", forcesReplacement(diff.Replace, opts.OverrideForcesReplacement)))
	for _, key := range keys {
		element := renderer.elements[key]

		if element.Action == plans.NoOp && !opts.ShowUnchangedChildren {
			// Don't render NoOp operations when we are compact display.
			unchangedElements++
			continue
		}

		for _, warning := range element.WarningsHuman(indent + 1) {
			buf.WriteString(fmt.Sprintf("%s%s\n", formatIndent(indent+1), warning))
		}

		// Only show commas between elements for objects.
		comma := ""
		if _, ok := element.Renderer.(objectRenderer); ok {
			comma = ","
		}

		// When we add padding for the keys, we want the length to be an
		// additional 2 characters, as we are going to add quotation marks ("")
		// around the key when it is rendered.
		keyLenWithOffset := renderer.maximumKeyLen + 2
		buf.WriteString(fmt.Sprintf("%s%s %-*q = %s%s\n", formatIndent(indent+1), format.DiffActionSymbol(element.Action), keyLenWithOffset, key, element.RenderHuman(indent+1, elementOpts), comma))
	}

	if unchangedElements > 0 {
		buf.WriteString(fmt.Sprintf("%s%s %s\n", formatIndent(indent+1), format.DiffActionSymbol(plans.NoOp), unchanged("element", unchangedElements)))
	}

	buf.WriteString(fmt.Sprintf("%s%s }%s", formatIndent(indent), format.DiffActionSymbol(plans.NoOp), nullSuffix(opts.OverrideNullSuffix, diff.Action)))
	return buf.String()
}
