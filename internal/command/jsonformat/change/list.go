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

	// renderNext tells the renderer to print out the next element in the list
	// whatever state it is in. So, even if a change is a NoOp we will still
	// print it out if the last change we processed wants us to.
	renderNext := false

	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("[%s\n", change.forcesReplacement()))
	for _, element := range renderer.elements {
		if element.action == plans.NoOp && !renderNext && !opts.showUnchangedChildren {
			unchangedElements = append(unchangedElements, element)
			continue
		}
		renderNext = false

		// If we want to display the context around this change, we want to
		// render the change immediately before this change in the list, and the
		// change immediately after in the list, even if both these changes are
		// NoOps. This will give the user reading the diff some context as to
		// where in the list these changes are being made, as order matters.
		if renderer.displayContext {
			// If our list of unchanged elements contains more than one entry
			// we'll print out a count of the number of unchanged elements that
			// we skipped. Note, this is the length of the unchanged elements
			// minus 1 as the most recent unchanged element will be printed out
			// in full.
			if len(unchangedElements) > 1 {
				buf.WriteString(fmt.Sprintf("%s%s %s\n", change.indent(indent+1), change.emptySymbol(), change.unchanged("element", len(unchangedElements)-1)))
			}
			// If our list of unchanged elements contains at least one entry,
			// we're going to print out the most recent change in full. That's
			// what happens here.
			if len(unchangedElements) > 0 {
				lastElement := unchangedElements[len(unchangedElements)-1]
				buf.WriteString(fmt.Sprintf("%s    %s,\n", change.indent(indent+1), lastElement.Render(indent+1, unchangedElementOpts)))
			}
			// We now reset the unchanged elements list, we've printed out a
			// count of all the elements we skipped so we start counting from
			// scratch again. This means that if we process a run of changed
			// elements, they won't all start printing out summaries of every
			// change that happened previously.
			unchangedElements = nil

			// As we also want to render the element immediately after any
			// changes, we make a note here to say we should render the next
			// change whatever it is. But, we only want to render the next
			// change if the current change isn't a NoOp. If the current change
			// is a NoOp then it was told to print by the last change and we
			// don't want to cascade and print all changes from now on.
			renderNext = element.action != plans.NoOp
		}

		for _, warning := range element.Warnings(indent + 1) {
			buf.WriteString(fmt.Sprintf("%s%s\n", change.indent(indent+1), warning))
		}
		if element.action == plans.NoOp {
			buf.WriteString(fmt.Sprintf("%s    %s,\n", change.indent(indent+1), element.Render(indent+1, unchangedElementOpts)))
		} else {
			buf.WriteString(fmt.Sprintf("%s%s %s,\n", change.indent(indent+1), format.DiffActionSymbol(element.action), element.Render(indent+1, elementOpts)))
		}
	}

	// If we were not displaying any context alongside our changes then the
	// unchangedElements list will contain every unchanged element, and we'll
	// print that out as we do with every other collection.
	//
	// If we were displaying context, then this will contain any unchanged
	// elements since our last change, so we should also print it out.
	if len(unchangedElements) > 0 {
		buf.WriteString(fmt.Sprintf("%s%s %s\n", change.indent(indent+1), change.emptySymbol(), change.unchanged("element", len(unchangedElements))))
	}

	buf.WriteString(fmt.Sprintf("%s%s ]%s", change.indent(indent), change.emptySymbol(), change.nullSuffix(opts.overrideNullSuffix)))
	return buf.String()
}
