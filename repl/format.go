package repl

import (
	"fmt"
	"strconv"
	"strings"

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
			// FIXME: If it's a multi-line string, better to render it using
			// HEREDOC-style syntax.
			return strconv.Quote(v.AsString())
		case cty.Number:
			bf := v.AsBigFloat()
			return bf.Text('g', -1)
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
