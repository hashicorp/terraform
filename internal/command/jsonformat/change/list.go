package change

import (
	"bytes"
	"fmt"

	"github.com/hashicorp/terraform/internal/command/format"
	"github.com/hashicorp/terraform/internal/plans"
)

func List(elements []Change) Renderer {
	return &listRenderer{
		displayContext: true,
		elements:       elements,
	}
}

func NestedList(elements []Change) Renderer {
	return &listRenderer{
		elements: elements,
	}
}

type listRenderer struct {
	NoWarningsRenderer

	displayContext bool
	elements       []Change
}

func (renderer listRenderer) Render(change Change, indent int, opts RenderOpts) string {
	if len(renderer.elements) == 0 {
		return fmt.Sprintf("[]%s%s", change.nullSuffix(opts.overrideNullSuffix), change.forcesReplacement())
	}

	elementOpts := opts.Clone()
	elementOpts.overrideNullSuffix = true

	unchangedElementOpts := opts.Clone()
	unchangedElementOpts.showUnchangedChildren = true

	var unchangedElements []Change
	renderNext := false

	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("[%s\n", change.forcesReplacement()))
	for _, element := range renderer.elements {
		if element.action == plans.NoOp && !renderNext && !opts.showUnchangedChildren {
			unchangedElements = append(unchangedElements, element)
			continue
		}
		renderNext = false

		if renderer.displayContext {
			if len(unchangedElements) > 1 {
				buf.WriteString(fmt.Sprintf("%s%s %s\n", change.indent(indent+1), change.emptySymbol(), change.unchanged("element", len(unchangedElements)-1)))
			}
			if len(unchangedElements) > 0 {
				lastElement := unchangedElements[len(unchangedElements)-1]
				buf.WriteString(fmt.Sprintf("%s%s %s,\n", change.indent(indent+1), lastElement.emptySymbol(), lastElement.Render(indent+1, unchangedElementOpts)))
			}
			unchangedElements = nil
			renderNext = element.action != plans.NoOp
		}

		for _, warning := range element.Warnings(indent + 1) {
			buf.WriteString(fmt.Sprintf("%s%s\n", change.indent(indent+1), warning))
		}
		if element.action == plans.NoOp {
			buf.WriteString(fmt.Sprintf("%s%s %s,\n", change.indent(indent+1), element.emptySymbol(), element.Render(indent+1, unchangedElementOpts)))
		} else {
			buf.WriteString(fmt.Sprintf("%s%s %s,\n", change.indent(indent+1), format.DiffActionSymbol(element.action), element.Render(indent+1, elementOpts)))
		}
	}

	if len(unchangedElements) > 0 {
		buf.WriteString(fmt.Sprintf("%s%s %s\n", change.indent(indent+1), change.emptySymbol(), change.unchanged("element", len(unchangedElements))))
	}

	buf.WriteString(fmt.Sprintf("%s%s ]%s", change.indent(indent), change.emptySymbol(), change.nullSuffix(opts.overrideNullSuffix)))
	return buf.String()
}
