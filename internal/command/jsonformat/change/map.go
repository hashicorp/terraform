package change

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/hashicorp/terraform/internal/command/format"
	"github.com/hashicorp/terraform/internal/plans"
)

func Map(elements map[string]Change) Renderer {
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

	elements      map[string]Change
	maximumKeyLen int
}

func (renderer mapRenderer) Render(change Change, indent int, opts RenderOpts) string {
	if len(renderer.elements) == 0 {
		return fmt.Sprintf("{}%s%s", change.nullSuffix(opts.overrideNullSuffix), change.forcesReplacement())
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
	if change.action == plans.Delete {
		elementOpts.overrideNullSuffix = true
	}

	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("{%s\n", change.forcesReplacement()))
	for _, key := range keys {
		element := renderer.elements[key]

		if element.action == plans.NoOp && !opts.showUnchangedChildren {
			// Don't render NoOp operations when we are compact display.
			unchangedElements++
			continue
		}

		for _, warning := range element.Warnings(indent + 1) {
			buf.WriteString(fmt.Sprintf("%s%s\n", change.indent(indent+1), warning))
		}

		// Only show commas between elements for objects.
		comma := ""
		if _, ok := element.renderer.(objectRenderer); ok {
			comma = ","
		}

		buf.WriteString(fmt.Sprintf("%s%s \"%s\"%-*s = %s%s\n", change.indent(indent+1), format.DiffActionSymbol(element.action), key, renderer.maximumKeyLen-len(key), "", element.Render(indent+1, elementOpts), comma))
	}

	if unchangedElements > 0 {
		buf.WriteString(fmt.Sprintf("%s%s %s\n", change.indent(indent+1), change.emptySymbol(), change.unchanged("element", unchangedElements)))
	}

	buf.WriteString(fmt.Sprintf("%s%s }%s", change.indent(indent), change.emptySymbol(), change.nullSuffix(opts.overrideNullSuffix)))
	return buf.String()
}
