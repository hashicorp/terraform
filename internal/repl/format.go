package repl

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/zclconf/go-cty/cty"
)

// FormatValue formats a value in a way that resembles Terraform language syntax
// and uses the type conversion functions where necessary to indicate exactly
// what type it is given, so that equality test failures can be quickly
// understood.
func FormatValue(v cty.Value, indent int) string {
	if !v.IsKnown() {
		return "(known after apply)"
	}
	if v.Type().Equals(cty.String) && v.HasMark(marks.Raw) {
		raw, _ := v.Unmark()
		return raw.AsString()
	}
	if v.HasMark(marks.Sensitive) {
		return "(sensitive)"
	}
	if v.IsNull() {
		ty := v.Type()
		switch {
		case ty == cty.DynamicPseudoType:
			return "null"
		case ty == cty.String:
			return "tostring(null)"
		case ty == cty.Number:
			return "tonumber(null)"
		case ty == cty.Bool:
			return "tobool(null)"
		case ty.IsListType():
			return fmt.Sprintf("tolist(null) /* of %s */", ty.ElementType().FriendlyName())
		case ty.IsSetType():
			return fmt.Sprintf("toset(null) /* of %s */", ty.ElementType().FriendlyName())
		case ty.IsMapType():
			return fmt.Sprintf("tomap(null) /* of %s */", ty.ElementType().FriendlyName())
		default:
			return fmt.Sprintf("null /* %s */", ty.FriendlyName())
		}
	}

	ty := v.Type()
	switch {
	case ty.IsPrimitiveType():
		switch ty {
		case cty.String:
			if formatted, isMultiline := formatMultilineString(v, indent); isMultiline {
				return formatted
			}
			return strconv.Quote(v.AsString())
		case cty.Number:
			bf := v.AsBigFloat()
			return bf.Text('f', -1)
		case cty.Bool:
			if v.True() {
				return "true"
			} else {
				return "false"
			}
		}
	case ty.IsObjectType():
		return formatMappingValue(v, indent)
	case ty.IsTupleType():
		return formatSequenceValue(v, indent)
	case ty.IsListType():
		return fmt.Sprintf("tolist(%s)", formatSequenceValue(v, indent))
	case ty.IsSetType():
		return fmt.Sprintf("toset(%s)", formatSequenceValue(v, indent))
	case ty.IsMapType():
		return fmt.Sprintf("tomap(%s)", formatMappingValue(v, indent))
	}

	// Should never get here because there are no other types
	return fmt.Sprintf("%#v", v)
}

func formatMultilineString(v cty.Value, indent int) (string, bool) {
	str := v.AsString()
	lines := strings.Split(str, "\n")
	if len(lines) < 2 {
		return "", false
	}

	// If the value is indented, we use the indented form of heredoc for readability.
	operator := "<<"
	if indent > 0 {
		operator = "<<-"
	}

	// Default delimiter is "End Of Text" by convention
	delimiter := "EOT"

OUTER:
	for {
		// Check if any of the lines are in conflict with the delimiter. The
		// parser allows leading and trailing whitespace, so we must remove it
		// before comparison.
		for _, line := range lines {
			// If the delimiter matches a line, extend it and start again
			if strings.TrimSpace(line) == delimiter {
				delimiter = delimiter + "_"
				continue OUTER
			}
		}

		// None of the lines match the delimiter, so we're ready
		break
	}

	// Write the heredoc, with indentation as appropriate.
	var buf strings.Builder

	buf.WriteString(operator)
	buf.WriteString(delimiter)
	for _, line := range lines {
		buf.WriteByte('\n')
		buf.WriteString(strings.Repeat(" ", indent))
		buf.WriteString(line)
	}
	buf.WriteByte('\n')
	buf.WriteString(strings.Repeat(" ", indent))
	buf.WriteString(delimiter)

	return buf.String(), true
}

func formatMappingValue(v cty.Value, indent int) string {
	var buf strings.Builder
	count := 0
	buf.WriteByte('{')
	indent += 2
	for it := v.ElementIterator(); it.Next(); {
		count++
		k, v := it.Element()
		buf.WriteByte('\n')
		buf.WriteString(strings.Repeat(" ", indent))
		buf.WriteString(FormatValue(k, indent))
		buf.WriteString(" = ")
		buf.WriteString(FormatValue(v, indent))
	}
	indent -= 2
	if count > 0 {
		buf.WriteByte('\n')
		buf.WriteString(strings.Repeat(" ", indent))
	}
	buf.WriteByte('}')
	return buf.String()
}

func formatSequenceValue(v cty.Value, indent int) string {
	var buf strings.Builder
	count := 0
	buf.WriteByte('[')
	indent += 2
	for it := v.ElementIterator(); it.Next(); {
		count++
		_, v := it.Element()
		buf.WriteByte('\n')
		buf.WriteString(strings.Repeat(" ", indent))
		buf.WriteString(FormatValue(v, indent))
		buf.WriteByte(',')
	}
	indent -= 2
	if count > 0 {
		buf.WriteByte('\n')
		buf.WriteString(strings.Repeat(" ", indent))
	}
	buf.WriteByte(']')
	return buf.String()
}
