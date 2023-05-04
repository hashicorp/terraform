// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package renderers

import (
	"bytes"
	"fmt"

	"github.com/hashicorp/terraform/internal/command/jsonformat/computed"
	"github.com/hashicorp/terraform/internal/plans"
)

var _ computed.DiffRenderer = (*listRenderer)(nil)

func List(elements []computed.Diff) computed.DiffRenderer {
	return &listRenderer{
		displayContext: true,
		elements:       elements,
	}
}

func NestedList(elements []computed.Diff) computed.DiffRenderer {
	return &listRenderer{
		elements: elements,
	}
}

type listRenderer struct {
	NoWarningsRenderer

	displayContext bool
	elements       []computed.Diff
}

func (renderer listRenderer) RenderHuman(diff computed.Diff, indent int, opts computed.RenderHumanOpts) string {
	if len(renderer.elements) == 0 {
		return fmt.Sprintf("[]%s%s", nullSuffix(diff.Action, opts), forcesReplacement(diff.Replace, opts))
	}

	elementOpts := opts.Clone()
	elementOpts.OverrideNullSuffix = true

	unchangedElementOpts := opts.Clone()
	unchangedElementOpts.ShowUnchangedChildren = true

	var unchangedElements []computed.Diff

	// renderNext tells the renderer to print out the next element in the list
	// whatever state it is in. So, even if a change is a NoOp we will still
	// print it out if the last change we processed wants us to.
	renderNext := false

	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("[%s\n", forcesReplacement(diff.Replace, opts)))
	for _, element := range renderer.elements {
		if element.Action == plans.NoOp && !renderNext && !opts.ShowUnchangedChildren {
			unchangedElements = append(unchangedElements, element)
			continue
		}
		renderNext = false

		opts := elementOpts

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
				buf.WriteString(fmt.Sprintf("%s%s%s\n", formatIndent(indent+1), writeDiffActionSymbol(plans.NoOp, opts), unchanged("element", len(unchangedElements)-1, opts)))
			}
			// If our list of unchanged elements contains at least one entry,
			// we're going to print out the most recent change in full. That's
			// what happens here.
			if len(unchangedElements) > 0 {
				lastElement := unchangedElements[len(unchangedElements)-1]
				buf.WriteString(fmt.Sprintf("%s%s%s,\n", formatIndent(indent+1), writeDiffActionSymbol(lastElement.Action, unchangedElementOpts), lastElement.RenderHuman(indent+1, unchangedElementOpts)))
			}
			// We now reset the unchanged elements list, we've printed out a
			// count of all the elements we skipped so we start counting from
			// scratch again. This means that if we process a run of changed
			// elements, they won't all start printing out summaries of every
			// change that happened previously.
			unchangedElements = nil

			if element.Action == plans.NoOp {
				// If this is a NoOp action then we're going to render it below
				// so we need to just override the opts we're going to use to
				// make sure we use the unchanged opts.
				opts = unchangedElementOpts
			} else {
				// As we also want to render the element immediately after any
				// changes, we make a note here to say we should render the next
				// change whatever it is. But, we only want to render the next
				// change if the current change isn't a NoOp. If the current change
				// is a NoOp then it was told to print by the last change and we
				// don't want to cascade and print all changes from now on.
				renderNext = true
			}
		}

		for _, warning := range element.WarningsHuman(indent+1, opts) {
			buf.WriteString(fmt.Sprintf("%s%s\n", formatIndent(indent+1), warning))
		}
		buf.WriteString(fmt.Sprintf("%s%s%s,\n", formatIndent(indent+1), writeDiffActionSymbol(element.Action, opts), element.RenderHuman(indent+1, opts)))
	}

	// If we were not displaying any context alongside our changes then the
	// unchangedElements list will contain every unchanged element, and we'll
	// print that out as we do with every other collection.
	//
	// If we were displaying context, then this will contain any unchanged
	// elements since our last change, so we should also print it out.
	if len(unchangedElements) > 0 {
		buf.WriteString(fmt.Sprintf("%s%s%s\n", formatIndent(indent+1), writeDiffActionSymbol(plans.NoOp, opts), unchanged("element", len(unchangedElements), opts)))
	}

	buf.WriteString(fmt.Sprintf("%s%s]%s", formatIndent(indent), writeDiffActionSymbol(plans.NoOp, opts), nullSuffix(diff.Action, opts)))
	return buf.String()
}
