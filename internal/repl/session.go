package repl

import (
	"fmt"
	"sort"
	"strings"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform/internal/lang"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/lang/types"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Session represents the state for a single REPL session.
type Session struct {
	// Scope is the evaluation scope where expressions will be evaluated.
	Scope *lang.Scope
}

// Handle handles a single line of input from the REPL.
//
// This is a stateful operation if a command is given (such as setting
// a variable). This function should not be called in parallel.
//
// The return value is the output and the error to show.
func (s *Session) Handle(line string) (string, bool, tfdiags.Diagnostics) {
	switch {
	case strings.TrimSpace(line) == "":
		return "", false, nil
	case strings.TrimSpace(line) == "exit":
		return "", true, nil
	case strings.TrimSpace(line) == "help":
		ret, diags := s.handleHelp()
		return ret, false, diags
	default:
		ret, diags := s.handleEval(line)
		return ret, false, diags
	}
}

func (s *Session) handleEval(line string) (string, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	// Parse the given line as an expression
	expr, parseDiags := hclsyntax.ParseExpression([]byte(line), "<console-input>", hcl.Pos{Line: 1, Column: 1})
	diags = diags.Append(parseDiags)
	if parseDiags.HasErrors() {
		return "", diags
	}

	val, valDiags := s.Scope.EvalExpr(expr, cty.DynamicPseudoType)
	diags = diags.Append(valDiags)
	if valDiags.HasErrors() {
		return "", diags
	}

	// The TypeType mark is used only by the console-only `type` function, in
	// order to smuggle the type of a given value back here. We can then
	// display a representation of the type directly.
	if marks.Contains(val, marks.TypeType) {
		val, _ = val.UnmarkDeep()

		valType := val.Type()
		switch {
		case valType.Equals(types.TypeType):
			// An encapsulated type value, which should be displayed directly.
			valType := val.EncapsulatedValue().(*cty.Type)
			return typeString(*valType), diags
		default:
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Invalid use of type function",
				"The console-only \"type\" function cannot be used as part of an expression.",
			))
			return "", diags
		}
	}

	return FormatValue(val, 0), diags
}

func (s *Session) handleHelp() (string, tfdiags.Diagnostics) {
	text := `
The Terraform console allows you to experiment with Terraform interpolations.
You may access resources in the state (if you have one) just as you would
from a configuration. For example: "aws_instance.foo.id" would evaluate
to the ID of "aws_instance.foo" if it exists in your state.

Type in the interpolation to test and hit <enter> to see the result.

To exit the console, type "exit" and hit <enter>, or use Control-C or
Control-D.
`

	return strings.TrimSpace(text), nil
}

// Modified copy of TypeString from go-cty:
// https://github.com/zclconf/go-cty-debug/blob/master/ctydebug/type_string.go
//
// TypeString returns a string representation of a given type that is
// reminiscent of Go syntax calling into the cty package but is mainly
// intended for easy human inspection of values in tests, debug output, etc.
//
// The resulting string will include newlines and indentation in order to
// increase the readability of complex structures. It always ends with a
// newline, so you can print this result directly to your output.
func typeString(ty cty.Type) string {
	var b strings.Builder
	writeType(ty, &b, 0)
	return b.String()
}

func writeType(ty cty.Type, b *strings.Builder, indent int) {
	switch {
	case ty == cty.NilType:
		b.WriteString("nil")
		return
	case ty.IsObjectType():
		atys := ty.AttributeTypes()
		if len(atys) == 0 {
			b.WriteString("object({})")
			return
		}
		attrNames := make([]string, 0, len(atys))
		for name := range atys {
			attrNames = append(attrNames, name)
		}
		sort.Strings(attrNames)
		b.WriteString("object({\n")
		indent++
		for _, name := range attrNames {
			aty := atys[name]
			b.WriteString(indentSpaces(indent))
			fmt.Fprintf(b, "%s: ", name)
			writeType(aty, b, indent)
			b.WriteString(",\n")
		}
		indent--
		b.WriteString(indentSpaces(indent))
		b.WriteString("})")
	case ty.IsTupleType():
		etys := ty.TupleElementTypes()
		if len(etys) == 0 {
			b.WriteString("tuple([])")
			return
		}
		b.WriteString("tuple([\n")
		indent++
		for _, ety := range etys {
			b.WriteString(indentSpaces(indent))
			writeType(ety, b, indent)
			b.WriteString(",\n")
		}
		indent--
		b.WriteString(indentSpaces(indent))
		b.WriteString("])")
	case ty.IsCollectionType():
		ety := ty.ElementType()
		switch {
		case ty.IsListType():
			b.WriteString("list(")
		case ty.IsMapType():
			b.WriteString("map(")
		case ty.IsSetType():
			b.WriteString("set(")
		default:
			// At the time of writing there are no other collection types,
			// but we'll be robust here and just pass through the GoString
			// of anything we don't recognize.
			b.WriteString(ty.FriendlyName())
			return
		}
		// Because object and tuple types render split over multiple
		// lines, a collection type container around them can end up
		// being hard to see when scanning, so we'll generate some extra
		// indentation to make a collection of structural type more visually
		// distinct from the structural type alone.
		complexElem := ety.IsObjectType() || ety.IsTupleType()
		if complexElem {
			indent++
			b.WriteString("\n")
			b.WriteString(indentSpaces(indent))
		}
		writeType(ty.ElementType(), b, indent)
		if complexElem {
			indent--
			b.WriteString(",\n")
			b.WriteString(indentSpaces(indent))
		}
		b.WriteString(")")
	default:
		// For any other type we'll just use its GoString and assume it'll
		// follow the usual GoString conventions.
		b.WriteString(ty.FriendlyName())
	}
}

func indentSpaces(level int) string {
	return strings.Repeat("    ", level)
}
