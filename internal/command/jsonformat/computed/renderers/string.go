package renderers

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/command/format"
	"github.com/hashicorp/terraform/internal/command/jsonformat/collections"
	"github.com/hashicorp/terraform/internal/command/jsonformat/computed"
	"github.com/hashicorp/terraform/internal/plans"
)

type evaluatedString struct {
	String string
	Json   interface{}

	IsMultiline bool
}

func evaluatePrimitiveString(value interface{}) evaluatedString {
	if value == nil {
		return evaluatedString{String: "[dark_gray]null[reset]"}
	}

	str := value.(string)

	if strings.HasPrefix(str, "{") || strings.HasPrefix(str, "[") {
		var jv interface{}
		if err := json.Unmarshal([]byte(str), &jv); err == nil {
			return evaluatedString{
				String: str,
				Json:   jv,
			}
		}
	}

	if strings.Contains(str, "\n") {
		return evaluatedString{
			String:      strings.TrimSpace(str),
			IsMultiline: true,
		}
	}

	return evaluatedString{
		String: str,
	}
}

func (renderer primitiveRenderer) renderStringDiff(diff computed.Diff, indent int, opts computed.RenderHumanOpts) string {

	// We process multiline strings at the end of the switch statement.
	var lines []string

	switch diff.Action {
	case plans.Create, plans.NoOp:
		str := evaluatePrimitiveString(renderer.after)

		if str.Json != nil {
			if diff.Action == plans.NoOp {
				return renderer.renderStringDiffAsJson(diff, indent, opts, str, str)
			} else {
				return renderer.renderStringDiffAsJson(diff, indent, opts, evaluatedString{}, str)
			}
		}

		if !str.IsMultiline {
			return fmt.Sprintf("%q%s", str.String, forcesReplacement(diff.Replace))
		}

		// We are creating a single multiline string, so let's split by the new
		// line character. While we are doing this, we are going to insert our
		// indents and make sure each line is formatted correctly.
		lines = strings.Split(strings.ReplaceAll(str.String, "\n", fmt.Sprintf("\n%s%s ", formatIndent(indent), format.DiffActionSymbol(plans.NoOp))), "\n")

		// We now just need to do the same for the first entry in lines, because
		// we split on the new line characters which won't have been at the
		// beginning of the first line.
		lines[0] = fmt.Sprintf("%s%s %s", formatIndent(indent), format.DiffActionSymbol(plans.NoOp), lines[0])
	case plans.Delete:
		str := evaluatePrimitiveString(renderer.before)

		if str.Json != nil {
			return renderer.renderStringDiffAsJson(diff, indent, opts, str, evaluatedString{})
		}

		if !str.IsMultiline {
			return fmt.Sprintf("%q%s%s", str.String, nullSuffix(opts.OverrideNullSuffix, diff.Action), forcesReplacement(diff.Replace))
		}

		// We are creating a single multiline string, so let's split by the new
		// line character. While we are doing this, we are going to insert our
		// indents and make sure each line is formatted correctly.
		lines = strings.Split(strings.ReplaceAll(str.String, "\n", fmt.Sprintf("\n%s%s ", formatIndent(indent), format.DiffActionSymbol(plans.NoOp))), "\n")

		// We now just need to do the same for the first entry in lines, because
		// we split on the new line characters which won't have been at the
		// beginning of the first line.
		lines[0] = fmt.Sprintf("%s%s %s", formatIndent(indent), format.DiffActionSymbol(plans.NoOp), lines[0])
	default:
		beforeString := evaluatePrimitiveString(renderer.before)
		afterString := evaluatePrimitiveString(renderer.after)

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
			return fmt.Sprintf("%q [yellow]->[reset] %q%s", beforeString.String, afterString.String, forcesReplacement(diff.Replace))
		}

		beforeLines := strings.Split(beforeString.String, "\n")
		afterLines := strings.Split(afterString.String, "\n")

		processIndices := func(beforeIx, afterIx int) {
			if beforeIx < 0 || beforeIx >= len(beforeLines) {
				lines = append(lines, fmt.Sprintf("%s%s %s", formatIndent(indent), format.DiffActionSymbol(plans.Create), afterLines[afterIx]))
				return
			}

			if afterIx < 0 || afterIx >= len(afterLines) {
				lines = append(lines, fmt.Sprintf("%s%s %s", formatIndent(indent), format.DiffActionSymbol(plans.Delete), beforeLines[beforeIx]))
				return
			}

			lines = append(lines, fmt.Sprintf("%s%s %s", formatIndent(indent), format.DiffActionSymbol(plans.NoOp), beforeLines[beforeIx]))
		}
		isObjType := func(_ string) bool {
			return false
		}

		collections.ProcessSlice(beforeLines, afterLines, processIndices, isObjType)
	}

	// We return early if we find non-multiline strings or JSON strings, so we
	// know here that we just render the lines slice properly.
	return fmt.Sprintf("<<-EOT%s\n%s\n%sEOT%s",
		forcesReplacement(diff.Replace),
		strings.Join(lines, "\n"),
		formatIndent(indent),
		nullSuffix(opts.OverrideNullSuffix, diff.Action))
}
