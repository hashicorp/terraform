package configupgrade

import (
	"bytes"
	"fmt"
	"log"
	"strconv"

	hcl2 "github.com/hashicorp/hcl2/hcl"

	hcl1ast "github.com/hashicorp/hcl/hcl/ast"
	hcl1printer "github.com/hashicorp/hcl/hcl/printer"
	hcl1token "github.com/hashicorp/hcl/hcl/token"

	"github.com/hashicorp/hil"
	hilast "github.com/hashicorp/hil/ast"

	"github.com/hashicorp/terraform/tfdiags"
)

func upgradeExpr(val interface{}, filename string, interp bool, an *analysis) ([]byte, tfdiags.Diagnostics) {
	var buf bytes.Buffer
	var diags tfdiags.Diagnostics

	// "val" here can be either a hcl1ast.Node or a hilast.Node, since both
	// of these correspond to expressions in HCL2. Therefore we need to
	// comprehensively handle every possible HCL1 *and* HIL AST node type
	// and, at minimum, print it out as-is in HCL2 syntax.
	switch tv := val.(type) {

	case *hcl1ast.LiteralType:
		return upgradeExpr(tv.Token, filename, interp, an)

	case hcl1token.Token:
		litVal := tv.Value()
		switch tv.Type {
		case hcl1token.STRING:
			if !interp {
				// Easy case, then.
				printQuotedString(&buf, litVal.(string))
				break
			}

			hilNode, err := hil.Parse(litVal.(string))
			if err != nil {
				diags = diags.Append(&hcl2.Diagnostic{
					Severity: hcl2.DiagError,
					Summary:  "Invalid interpolated string",
					Detail:   fmt.Sprintf("Interpolation parsing failed: %s", err),
					Subject:  hcl1PosRange(filename, tv.Pos).Ptr(),
				})
			}

			interpSrc, interpDiags := upgradeExpr(hilNode, filename, interp, an)
			buf.Write(interpSrc)
			diags = diags.Append(interpDiags)

		case hcl1token.HEREDOC:
			// TODO: Implement
			panic("HEREDOC not supported yet")

		case hcl1token.BOOL:
			if litVal.(bool) {
				buf.WriteString("true")
			} else {
				buf.WriteString("false")
			}

		default:
			// For everything else (NUMBER, FLOAT) we'll just pass through the given bytes verbatim.
			buf.WriteString(tv.Text)

		}

	case *hcl1ast.ListType:
		multiline := tv.Lbrack.Line != tv.Rbrack.Line
		buf.WriteString("[")
		if multiline {
			buf.WriteString("\n")
		}
		for i, node := range tv.List {
			src, moreDiags := upgradeExpr(node, filename, interp, an)
			diags = diags.Append(moreDiags)
			buf.Write(src)
			if multiline {
				buf.WriteString(",\n")
			} else if i < len(tv.List)-1 {
				buf.WriteString(", ")
			}
		}
		buf.WriteString("]")

	case *hcl1ast.ObjectType:
		buf.WriteString("{\n")
		for _, item := range tv.List.Items {
			if len(item.Keys) != 1 {
				diags = diags.Append(&hcl2.Diagnostic{
					Severity: hcl2.DiagError,
					Summary:  "Invalid map element",
					Detail:   "A map element may not have any block-style labels.",
					Subject:  hcl1PosRange(filename, item.Pos()).Ptr(),
				})
				continue
			}
			keySrc, moreDiags := upgradeExpr(item.Keys[0].Token, filename, interp, an)
			diags = diags.Append(moreDiags)
			valueSrc, moreDiags := upgradeExpr(item.Val, filename, interp, an)
			diags = diags.Append(moreDiags)
			buf.Write(keySrc)
			buf.WriteString(" = ")
			buf.Write(valueSrc)
			buf.WriteString("\n")
		}
		buf.WriteString("}")

	case hcl1ast.Node:
		// If our more-specific cases above didn't match this then we'll
		// ask the hcl1printer package to print the expression out
		// itself, and assume it'll still be valid in HCL2.
		// (We should rarely end up here, since our cases above should
		// be comprehensive.)
		log.Printf("[TRACE] configupgrade: Don't know how to upgrade %T as expression, so just passing it through as-is", tv)
		hcl1printer.Fprint(&buf, tv)

	case *hilast.LiteralNode:
		switch tl := tv.Value.(type) {
		case string:
			// This shouldn't generally happen because literal strings are
			// always wrapped in hilast.Output in HIL, but we'll allow it anyway.
			printQuotedString(&buf, tl)
		case int:
			buf.WriteString(strconv.Itoa(tl))
		case float64:
			buf.WriteString(strconv.FormatFloat(tl, 'f', 64, 64))
		case bool:
			if tl {
				buf.WriteString("true")
			} else {
				buf.WriteString("false")
			}
		}

	case *hilast.VariableAccess:
		buf.WriteString(tv.Name)

	case *hilast.Arithmetic:
		op, exists := hilArithmeticOpSyms[tv.Op]
		if !exists {
			panic(fmt.Errorf("arithmetic node with unsupported operator %#v", tv.Op))
		}

		lhsExpr := tv.Exprs[0]
		rhsExpr := tv.Exprs[1]
		lhsSrc, exprDiags := upgradeExpr(lhsExpr, filename, true, an)
		diags = diags.Append(exprDiags)
		rhsSrc, exprDiags := upgradeExpr(rhsExpr, filename, true, an)
		diags = diags.Append(exprDiags)

		// HIL's AST represents -foo as (0 - foo), so we'll recognize
		// that here and normalize it back.
		if tv.Op == hilast.ArithmeticOpSub && len(lhsSrc) == 1 && lhsSrc[0] == '0' {
			buf.WriteString("-")
			buf.Write(rhsSrc)
			break
		}

		buf.Write(lhsSrc)
		buf.WriteString(op)
		buf.Write(rhsSrc)

	case *hilast.Call:
		name := tv.Func
		args := tv.Args

		argExprs := make([][]byte, len(args))
		multiline := false
		totalLen := 0
		for i, arg := range args {
			if i > 0 {
				totalLen += 2
			}
			exprSrc, exprDiags := upgradeExpr(arg, filename, true, an)
			diags = diags.Append(exprDiags)
			argExprs[i] = exprSrc
			if bytes.Contains(exprSrc, []byte{'\n'}) {
				// If any of our arguments are multi-line then we'll also be multiline
				multiline = true
			}
			totalLen += len(exprSrc)
		}

		if totalLen > 60 { // heuristic, since we don't know here how indented we are already
			multiline = true
		}

		// Some functions are now better expressed as native language constructs.
		// These cases will return early if they emit anything, or otherwise
		// fall through to the default emitter.
		switch name {
		case "list":
			// Should now use tuple constructor syntax
			buf.WriteByte('[')
			if multiline {
				buf.WriteByte('\n')
			}
			for i, exprSrc := range argExprs {
				buf.Write(exprSrc)
				if multiline {
					buf.WriteString(",\n")
				} else {
					if i < len(args)-1 {
						buf.WriteString(", ")
					}
				}
			}
			buf.WriteByte(']')
			return buf.Bytes(), diags
		case "map":
			// Should now use object constructor syntax, but we can only
			// achieve that if the call is valid, which requires an even
			// number of arguments.
			if len(argExprs) == 0 {
				buf.WriteString("{}")
				return buf.Bytes(), diags
			} else if len(argExprs)%2 == 0 {
				buf.WriteString("{\n")
				for i := 0; i < len(argExprs); i += 2 {
					k := argExprs[i]
					v := argExprs[i+1]

					buf.Write(k)
					buf.WriteString(" = ")
					buf.Write(v)
					buf.WriteByte('\n')
				}
				buf.WriteByte('}')
				return buf.Bytes(), diags
			}
		}

		buf.WriteString(name)
		buf.WriteByte('(')
		if multiline {
			buf.WriteByte('\n')
		}
		for i, exprSrc := range argExprs {
			buf.Write(exprSrc)
			if multiline {
				buf.WriteString(",\n")
			} else {
				if i < len(args)-1 {
					buf.WriteString(", ")
				}
			}
		}
		buf.WriteByte(')')

	case *hilast.Conditional:
		condSrc, exprDiags := upgradeExpr(tv.CondExpr, filename, true, an)
		diags = diags.Append(exprDiags)
		trueSrc, exprDiags := upgradeExpr(tv.TrueExpr, filename, true, an)
		diags = diags.Append(exprDiags)
		falseSrc, exprDiags := upgradeExpr(tv.FalseExpr, filename, true, an)
		diags = diags.Append(exprDiags)

		buf.Write(condSrc)
		buf.WriteString(" ? ")
		buf.Write(trueSrc)
		buf.WriteString(" : ")
		buf.Write(falseSrc)

	case *hilast.Index:
		targetSrc, exprDiags := upgradeExpr(tv.Target, filename, true, an)
		diags = diags.Append(exprDiags)
		keySrc, exprDiags := upgradeExpr(tv.Key, filename, true, an)
		diags = diags.Append(exprDiags)
		buf.Write(targetSrc)
		buf.WriteString("[")
		buf.Write(keySrc)
		buf.WriteString("]")

	case *hilast.Output:
		if len(tv.Exprs) == 1 {
			item := tv.Exprs[0]
			naked := true
			if lit, ok := item.(*hilast.LiteralNode); ok {
				if _, ok := lit.Value.(string); ok {
					naked = false
				}
			}
			if naked {
				// If there's only one expression and it isn't a literal string
				// then we'll just output it naked, since wrapping a single
				// expression in interpolation is no longer idiomatic.
				interped, interpDiags := upgradeExpr(item, filename, true, an)
				diags = diags.Append(interpDiags)
				buf.Write(interped)
				break
			}
		}

		buf.WriteString(`"`)
		for _, item := range tv.Exprs {
			if lit, ok := item.(*hilast.LiteralNode); ok {
				if litStr, ok := lit.Value.(string); ok {
					printStringLiteralFromHILOutput(&buf, litStr)
					continue
				}
			}

			interped, interpDiags := upgradeExpr(item, filename, true, an)
			diags = diags.Append(interpDiags)

			buf.WriteString("${")
			buf.Write(interped)
			buf.WriteString("}")
		}
		buf.WriteString(`"`)

	case hilast.Node:
		// Nothing reasonable we can do here, so we should've handled all of
		// the possibilities above.
		panic(fmt.Errorf("upgradeExpr doesn't handle HIL node type %T", tv))

	default:
		// If we end up in here then the caller gave us something completely invalid.
		panic(fmt.Errorf("upgradeExpr on unsupported type %T", val))

	}

	return buf.Bytes(), diags
}

var hilArithmeticOpSyms = map[hilast.ArithmeticOp]string{
	hilast.ArithmeticOpAdd: " + ",
	hilast.ArithmeticOpSub: " - ",
	hilast.ArithmeticOpMul: " * ",
	hilast.ArithmeticOpDiv: " / ",
	hilast.ArithmeticOpMod: " % ",

	hilast.ArithmeticOpLogicalAnd: " && ",
	hilast.ArithmeticOpLogicalOr:  " || ",

	hilast.ArithmeticOpEqual:              " == ",
	hilast.ArithmeticOpNotEqual:           " != ",
	hilast.ArithmeticOpLessThan:           " < ",
	hilast.ArithmeticOpLessThanOrEqual:    " <= ",
	hilast.ArithmeticOpGreaterThan:        " > ",
	hilast.ArithmeticOpGreaterThanOrEqual: " >= ",
}
