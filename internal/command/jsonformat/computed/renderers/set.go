package renderers

import (
	"bytes"
	"fmt"

	"github.com/hashicorp/terraform/internal/command/jsonformat/computed"

	"github.com/hashicorp/terraform/internal/command/format"
	"github.com/hashicorp/terraform/internal/plans"
)

var _ computed.DiffRenderer = (*setRenderer)(nil)

func Set(elements []computed.Diff) computed.DiffRenderer {
	return &setRenderer{
		elements: elements,
	}
}

type setRenderer struct {
	NoWarningsRenderer

	elements []computed.Diff
}

func (renderer setRenderer) RenderHuman(diff computed.Diff, indent int, opts computed.RenderHumanOpts) string {
	if len(renderer.elements) == 0 {
		return fmt.Sprintf("[]%s%s", nullSuffix(opts.OverrideNullSuffix, diff.Action), forcesReplacement(diff.Replace))
	}

	elementOpts := opts.Clone()
	elementOpts.OverrideNullSuffix = true

	unchangedElements := 0

	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("[%s\n", forcesReplacement(diff.Replace)))
	for _, element := range renderer.elements {
		if element.Action == plans.NoOp && !opts.ShowUnchangedChildren {
			unchangedElements++
			continue
		}

		for _, warning := range element.WarningsHuman(indent + 1) {
			buf.WriteString(fmt.Sprintf("%s%s\n", formatIndent(indent+1), warning))
		}
		buf.WriteString(fmt.Sprintf("%s%s %s,\n", formatIndent(indent+1), format.DiffActionSymbol(element.Action), element.RenderHuman(indent+1, elementOpts)))
	}

	if unchangedElements > 0 {
		buf.WriteString(fmt.Sprintf("%s%s %s\n", formatIndent(indent+1), format.DiffActionSymbol(plans.NoOp), unchanged("element", unchangedElements)))
	}

	buf.WriteString(fmt.Sprintf("%s%s ]%s", formatIndent(indent), format.DiffActionSymbol(plans.NoOp), nullSuffix(opts.OverrideNullSuffix, diff.Action)))
	return buf.String()
}
