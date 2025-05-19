// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package renderers

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/hashicorp/terraform/internal/command/jsonformat/computed"

	"github.com/hashicorp/terraform/internal/plans"
)

var _ computed.DiffRenderer = (*mapRenderer)(nil)

func Map(elements map[string]computed.Diff) computed.DiffRenderer {
	return &mapRenderer{
		elements:  elements,
		alignKeys: true,
	}
}

func NestedMap(elements map[string]computed.Diff) computed.DiffRenderer {
	return &mapRenderer{
		elements:                  elements,
		overrideNullSuffix:        true,
		overrideForcesReplacement: true,
	}
}

type mapRenderer struct {
	NoWarningsRenderer

	elements map[string]computed.Diff

	overrideNullSuffix        bool
	overrideForcesReplacement bool
	alignKeys                 bool
}

func (renderer mapRenderer) RenderHuman(diff computed.Diff, indent int, opts computed.RenderHumanOpts) string {
	forcesReplacementSelf := diff.Replace && !renderer.overrideForcesReplacement
	forcesReplacementChildren := diff.Replace && renderer.overrideForcesReplacement

	if len(renderer.elements) == 0 {
		return fmt.Sprintf("{}%s%s", nullSuffix(diff.Action, opts), forcesReplacement(forcesReplacementSelf, opts))
	}

	// Sort the map elements by key, so we have a deterministic ordering in
	// the output.
	var keys []string

	// We need to make sure the keys are capable of rendering properly.
	escapedKeys := make(map[string]string)

	maximumKeyLen := 0
	for key := range renderer.elements {
		keys = append(keys, key)

		escapedKey := hclEscapeString(key)
		escapedKeys[key] = escapedKey
		if maximumKeyLen < len(escapedKey) {
			maximumKeyLen = len(escapedKey)
		}
	}
	sort.Strings(keys)

	unchangedElements := 0

	elementOpts := opts.Clone()
	elementOpts.OverrideNullSuffix = diff.Action == plans.Delete || renderer.overrideNullSuffix
	elementOpts.ForceForcesReplacement = forcesReplacementChildren

	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("{%s\n", forcesReplacement(forcesReplacementSelf, opts)))
	for _, key := range keys {
		element := renderer.elements[key]

		if element.Action == plans.NoOp && !opts.ShowUnchangedChildren {
			// Don't render NoOp operations when we are compact display.
			unchangedElements++
			continue
		}

		for _, warning := range element.WarningsHuman(indent+1, opts) {
			buf.WriteString(fmt.Sprintf("%s%s\n", formatIndent(indent+1), warning))
		}
		// Only show commas between elements for objects.
		comma := ""
		if _, ok := element.Renderer.(*objectRenderer); ok {
			comma = ","
		}

		if renderer.alignKeys {
			buf.WriteString(fmt.Sprintf("%s%s%-*s = %s%s\n", formatIndent(indent+1), writeDiffActionSymbol(element.Action, elementOpts), maximumKeyLen, escapedKeys[key], element.RenderHuman(indent+1, elementOpts), comma))
		} else {
			buf.WriteString(fmt.Sprintf("%s%s%s = %s%s\n", formatIndent(indent+1), writeDiffActionSymbol(element.Action, elementOpts), escapedKeys[key], element.RenderHuman(indent+1, elementOpts), comma))
		}

	}

	if unchangedElements > 0 {
		buf.WriteString(fmt.Sprintf("%s%s%s\n", formatIndent(indent+1), writeDiffActionSymbol(plans.NoOp, opts), unchanged("element", unchangedElements, opts)))
	}

	buf.WriteString(fmt.Sprintf("%s%s}%s", formatIndent(indent), writeDiffActionSymbol(plans.NoOp, opts), nullSuffix(diff.Action, opts)))
	return buf.String()
}
