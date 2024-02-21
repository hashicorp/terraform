// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package renderers

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/command/jsonformat/collections"
	"github.com/hashicorp/terraform/internal/command/jsonformat/computed"
	"github.com/hashicorp/terraform/internal/command/jsonformat/structured"
	"github.com/hashicorp/terraform/internal/command/jsonformat/structured/attribute_path"
	"github.com/hashicorp/terraform/internal/plans"
)

var _ computed.DiffRenderer = (*primitiveRenderer)(nil)

func Primitive(before, after interface{}, ctype cty.Type) computed.DiffRenderer {
	return &primitiveRenderer{
		before: before,
		after:  after,
		ctype:  ctype,
	}
}

type primitiveRenderer struct {
	NoWarningsRenderer

	before interface{}
	after  interface{}
	ctype  cty.Type
}

func (renderer primitiveRenderer) RenderHuman(diff computed.Diff, indent int, opts computed.RenderHumanOpts) string {
	if renderer.ctype == cty.String {
		return renderer.renderStringDiff(diff, indent, opts)
	}

	beforeValue := renderPrimitiveValue(renderer.before, renderer.ctype, opts)
	afterValue := renderPrimitiveValue(renderer.after, renderer.ctype, opts)

	switch diff.Action {
	case plans.Create:
		return fmt.Sprintf("%s%s", afterValue, forcesReplacement(diff.Replace, opts))
	case plans.Delete:
		return fmt.Sprintf("%s%s%s", beforeValue, nullSuffix(diff.Action, opts), forcesReplacement(diff.Replace, opts))
	case plans.NoOp:
		return fmt.Sprintf("%s%s", beforeValue, forcesReplacement(diff.Replace, opts))
	default:
		return fmt.Sprintf("%s %s %s%s", beforeValue, opts.Colorize.Color("[yellow]->[reset]"), afterValue, forcesReplacement(diff.Replace, opts))
	}
}

func renderPrimitiveValue(value interface{}, t cty.Type, opts computed.RenderHumanOpts) string {
	if value == nil {
		return opts.Colorize.Color("[dark_gray]null[reset]")
	}

	switch {
	case t == cty.Bool:
		if value.(bool) {
			return "true"
		}
		return "false"
	case t == cty.Number:
		number := value.(json.Number)
		return number.String()
	default:
		panic("unrecognized primitive type: " + t.FriendlyName())
	}
}

func (renderer primitiveRenderer) renderStringDiff(diff computed.Diff, indent int, opts computed.RenderHumanOpts) string {

	// We process multiline strings at the end of the switch statement.
	var lines []string

	switch diff.Action {
	case plans.Create, plans.NoOp:
		str := evaluatePrimitiveString(renderer.after, opts)

		if str.Json != nil {
			if diff.Action == plans.NoOp {
				return renderer.renderStringDiffAsJson(diff, indent, opts, str, str)
			} else {
				return renderer.renderStringDiffAsJson(diff, indent, opts, evaluatedString{}, str)
			}
		}

		if !str.IsMultiline {
			return fmt.Sprintf("%s%s", str.RenderSimple(), forcesReplacement(diff.Replace, opts))
		}

		// We are creating a single multiline string, so let's split by the new
		// line character. While we are doing this, we are going to insert our
		// indents and make sure each line is formatted correctly.
		lines = strings.Split(strings.ReplaceAll(str.String, "\n", fmt.Sprintf("\n%s%s", formatIndent(indent+1), writeDiffActionSymbol(plans.NoOp, opts))), "\n")

		// We now just need to do the same for the first entry in lines, because
		// we split on the new line characters which won't have been at the
		// beginning of the first line.
		lines[0] = fmt.Sprintf("%s%s%s", formatIndent(indent+1), writeDiffActionSymbol(plans.NoOp, opts), lines[0])
	case plans.Delete:
		str := evaluatePrimitiveString(renderer.before, opts)
		if str.IsNull {
			// We don't put the null suffix (-> null) here because the final
			// render or null -> null would look silly.
			return fmt.Sprintf("%s%s", str.RenderSimple(), forcesReplacement(diff.Replace, opts))
		}

		if str.Json != nil {
			return renderer.renderStringDiffAsJson(diff, indent, opts, str, evaluatedString{})
		}

		if !str.IsMultiline {
			return fmt.Sprintf("%s%s%s", str.RenderSimple(), nullSuffix(diff.Action, opts), forcesReplacement(diff.Replace, opts))
		}

		// We are creating a single multiline string, so let's split by the new
		// line character. While we are doing this, we are going to insert our
		// indents and make sure each line is formatted correctly.
		lines = strings.Split(strings.ReplaceAll(str.String, "\n", fmt.Sprintf("\n%s%s", formatIndent(indent+1), writeDiffActionSymbol(plans.NoOp, opts))), "\n")

		// We now just need to do the same for the first entry in lines, because
		// we split on the new line characters which won't have been at the
		// beginning of the first line.
		lines[0] = fmt.Sprintf("%s%s%s", formatIndent(indent+1), writeDiffActionSymbol(plans.NoOp, opts), lines[0])
	default:
		beforeString := evaluatePrimitiveString(renderer.before, opts)
		afterString := evaluatePrimitiveString(renderer.after, opts)

		if beforeString.Json != nil && afterString.Json != nil {
			return renderer.renderStringDiffAsJson(diff, indent, opts, beforeString, afterString)
		}

		if beforeString.Json != nil || afterString.Json != nil {
			// This means one of the strings is JSON and one isn't. We're going
			// to be a little inefficient here, but we can just reuse another
			// renderer for this so let's keep it simple.
			return computed.NewDiff(
				TypeChange(
					computed.NewDiff(Primitive(renderer.before, nil, cty.String), plans.Delete, false),
					computed.NewDiff(Primitive(nil, renderer.after, cty.String), plans.Create, false)),
				diff.Action,
				diff.Replace).RenderHuman(indent, opts)
		}

		if !beforeString.IsMultiline && !afterString.IsMultiline {
			return fmt.Sprintf("%s %s %s%s", beforeString.RenderSimple(), opts.Colorize.Color("[yellow]->[reset]"), afterString.RenderSimple(), forcesReplacement(diff.Replace, opts))
		}

		beforeLines := strings.Split(beforeString.String, "\n")
		afterLines := strings.Split(afterString.String, "\n")

		processIndices := func(beforeIx, afterIx int) {
			if beforeIx < 0 || beforeIx >= len(beforeLines) {
				lines = append(lines, fmt.Sprintf("%s%s%s", formatIndent(indent+1), writeDiffActionSymbol(plans.Create, opts), afterLines[afterIx]))
				return
			}

			if afterIx < 0 || afterIx >= len(afterLines) {
				lines = append(lines, fmt.Sprintf("%s%s%s", formatIndent(indent+1), writeDiffActionSymbol(plans.Delete, opts), beforeLines[beforeIx]))
				return
			}

			if beforeLines[beforeIx] != afterLines[afterIx] {
				lines = append(lines, fmt.Sprintf("%s%s%s", formatIndent(indent+1), writeDiffActionSymbol(plans.Delete, opts), beforeLines[beforeIx]))
				lines = append(lines, fmt.Sprintf("%s%s%s", formatIndent(indent+1), writeDiffActionSymbol(plans.Create, opts), afterLines[afterIx]))
				return
			}

			lines = append(lines, fmt.Sprintf("%s%s%s", formatIndent(indent+1), writeDiffActionSymbol(plans.NoOp, opts), beforeLines[beforeIx]))
		}
		isObjType := func(_ string) bool {
			return false
		}

		collections.ProcessSlice(beforeLines, afterLines, processIndices, isObjType)
	}

	// We return early if we find non-multiline strings or JSON strings, so we
	// know here that we just render the lines slice properly.
	return fmt.Sprintf("<<-EOT%s\n%s\n%s%sEOT%s",
		forcesReplacement(diff.Replace, opts),
		strings.Join(lines, "\n"),
		formatIndent(indent),
		writeDiffActionSymbol(plans.NoOp, opts),
		nullSuffix(diff.Action, opts))
}

func (renderer primitiveRenderer) renderStringDiffAsJson(diff computed.Diff, indent int, opts computed.RenderHumanOpts, before evaluatedString, after evaluatedString) string {
	jsonDiff := RendererJsonOpts().Transform(structured.Change{
		BeforeExplicit:     diff.Action != plans.Create,
		AfterExplicit:      diff.Action != plans.Delete,
		Before:             before.Json,
		After:              after.Json,
		Unknown:            false,
		BeforeSensitive:    false,
		AfterSensitive:     false,
		ReplacePaths:       attribute_path.Empty(false),
		RelevantAttributes: attribute_path.AlwaysMatcher(),
	})

	action := diff.Action

	jsonOpts := opts.Clone()
	jsonOpts.OverrideNullSuffix = true

	var whitespace, replace string
	if jsonDiff.Action == plans.NoOp && diff.Action == plans.Update {
		// Then this means we are rendering a whitespace only change. The JSON
		// differ will have ignored the whitespace changes so that makes the
		// diff we are about to print out very confusing without extra
		// explanation.
		if diff.Replace {
			whitespace = " # whitespace changes force replacement"
		} else {
			whitespace = " # whitespace changes"
		}

		// Because we'd be showing no changes otherwise:
		jsonOpts.ShowUnchangedChildren = true

		// Whitespace changes should not appear as if edited.
		action = plans.NoOp
	} else {
		// We only show the replace suffix if we didn't print something out
		// about whitespace changes.
		replace = forcesReplacement(diff.Replace, opts)
	}

	renderedJsonDiff := jsonDiff.RenderHuman(indent+1, jsonOpts)

	if diff.Action == plans.Create || diff.Action == plans.Delete {
		// We don't display the '+' or '-' symbols on the JSON diffs, we should
		// still display the '~' for an update action though.
		action = plans.NoOp
	}

	if strings.Contains(renderedJsonDiff, "\n") {
		return fmt.Sprintf("jsonencode(%s\n%s%s%s%s\n%s%s)%s", whitespace, formatIndent(indent+1), writeDiffActionSymbol(action, opts), renderedJsonDiff, replace, formatIndent(indent), writeDiffActionSymbol(plans.NoOp, opts), nullSuffix(diff.Action, opts))
	}
	return fmt.Sprintf("jsonencode(%s)%s%s", renderedJsonDiff, whitespace, replace)
}
