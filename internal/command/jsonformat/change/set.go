package change

import (
	"bytes"
	"fmt"

	"github.com/hashicorp/terraform/internal/command/format"
	"github.com/hashicorp/terraform/internal/plans"
)

func Set(elements []Change) Renderer {
	return &setRenderer{
		elements: elements,
	}
}

type setRenderer struct {
	NoWarningsRenderer

	elements []Change
}

func (renderer setRenderer) Render(change Change, indent int, opts RenderOpts) string {
	if len(renderer.elements) == 0 {
		return fmt.Sprintf("[]%s%s", change.nullSuffix(opts.overrideNullSuffix), change.forcesReplacement())
	}

	elementOpts := opts.Clone()
	elementOpts.overrideNullSuffix = true

	unchangedElements := 0

	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("[%s\n", change.forcesReplacement()))
	for _, element := range renderer.elements {
		if element.action == plans.NoOp && !opts.showUnchangedChildren {
			unchangedElements++
			continue
		}

		for _, warning := range element.Warnings(indent + 1) {
			buf.WriteString(fmt.Sprintf("%s%s\n", change.indent(indent+1), warning))
		}
		buf.WriteString(fmt.Sprintf("%s%s %s,\n", change.indent(indent+1), format.DiffActionSymbol(element.action), element.Render(indent+1, elementOpts)))
	}

	if unchangedElements > 0 {
		buf.WriteString(fmt.Sprintf("%s%s %s\n", change.indent(indent+1), format.DiffActionSymbol(plans.NoOp), change.unchanged("element", unchangedElements)))
	}

	buf.WriteString(fmt.Sprintf("%s%s ]%s", change.indent(indent), format.DiffActionSymbol(plans.NoOp), change.nullSuffix(opts.overrideNullSuffix)))
	return buf.String()
}
