package configupgrade

import (
	"bytes"
	"fmt"
	"log"
	"strconv"
	"strings"

	hcl2 "github.com/hashicorp/hcl2/hcl"
	hcl2syntax "github.com/hashicorp/hcl2/hcl/hclsyntax"
	"github.com/zclconf/go-cty/cty"

	hcl1ast "github.com/hashicorp/hcl/hcl/ast"
	hcl1printer "github.com/hashicorp/hcl/hcl/printer"
	hcl1token "github.com/hashicorp/hcl/hcl/token"

	"github.com/hashicorp/hil"
	hilast "github.com/hashicorp/hil/ast"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/tfdiags"
)

func upgradeExpr(val interface{}, filename string, interp bool, an *analysis) ([]byte, tfdiags.Diagnostics) {
	var buf bytes.Buffer
	var diags tfdiags.Diagnostics

	// "val" here can be either a hcl1ast.Node or a hilast.Node, since both
	// of these correspond to expressions in HCL2. Therefore we need to
	// comprehensively handle every possible HCL1 *and* HIL AST node type
	// and, at minimum, print it out as-is in HCL2 syntax.
Value:
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
				return nil, diags
			}

			interpSrc, interpDiags := upgradeExpr(hilNode, filename, interp, an)
			buf.Write(interpSrc)
			diags = diags.Append(interpDiags)

		case hcl1token.HEREDOC:
			// HCL1's "Value" method for tokens pulls out the body and removes
			// any indents in the source for a flush heredoc, which throws away
			// information we need to upgrade. Therefore we're going to
			// re-implement a subset of that logic here where we want to retain
			// the whitespace verbatim even in flush mode.

			firstNewlineIdx := strings.IndexByte(tv.Text, '\n')
			if firstNewlineIdx < 0 {
				// Should never happen, because tv.Value would already have
				// panicked above in this case.
				panic("heredoc doesn't contain newline")
			}
			introducer := tv.Text[:firstNewlineIdx+1]
			marker := introducer[2:] // trim off << prefix
			if marker[0] == '-' {
				marker = marker[1:] // also trim of - prefix for flush heredoc
			}
			body := tv.Text[len(introducer) : len(tv.Text)-len(marker)]
			flush := introducer[2] == '-'
			if flush {
				// HCL1 treats flush heredocs differently, trimming off any
				// spare whitespace that might appear after the trailing
				// newline, and so we must replicate that here to avoid
				// introducing additional whitespace in the output.
				body = strings.TrimRight(body, " \t")
			}

			// Now we have:
			//  - introducer is the first line, like "<<-FOO\n"
			//  - marker is the end marker, like "FOO\n"
			//  - body is the raw data between the introducer and the marker,
			//    which we need to do recursive upgrading for.

			buf.WriteString(introducer)
			if !interp {
				// Easy case: escape all interpolation-looking sequences.
				printHeredocLiteralFromHILOutput(&buf, body)
			} else {
				hilNode, err := hil.Parse(body)
				if err != nil {
					diags = diags.Append(&hcl2.Diagnostic{
						Severity: hcl2.DiagError,
						Summary:  "Invalid interpolated string",
						Detail:   fmt.Sprintf("Interpolation parsing failed: %s", err),
						Subject:  hcl1PosRange(filename, tv.Pos).Ptr(),
					})
				}
				if _, ok := hilNode.(*hilast.Output); !ok {
					// hil.Parse usually produces an output, but it can sometimes
					// produce an isolated expression if the input is entirely
					// a single interpolation.
					hilNode = &hilast.Output{
						Exprs: []hilast.Node{hilNode},
						Posx:  hilNode.Pos(),
					}
				}
				interpDiags := upgradeHeredocBody(&buf, hilNode.(*hilast.Output), filename, an)
				diags = diags.Append(interpDiags)
			}
			if !strings.HasSuffix(body, "\n") {
				// The versions of HCL1 vendored into Terraform <=0.11
				// incorrectly allowed the end marker to appear at the end of
				// the final line of the body, rather than on a line of its own.
				// That is no longer valid in HCL2, so we need to fix it up.
				buf.WriteByte('\n')
			}
			// NOTE: Marker intentionally contains an extra newline here because
			// we need to ensure that any follow-on expression bits end up on
			// a separate line, or else the HCL2 parser won't be able to
			// recognize the heredoc marker. This causes an extra empty line
			// in some cases, which we accept for simplicity's sake.
			buf.WriteString(marker)

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
			buf.WriteString(strconv.FormatFloat(tl, 'f', -1, 64))
		case bool:
			if tl {
				buf.WriteString("true")
			} else {
				buf.WriteString("false")
			}
		}

	case *hilast.VariableAccess:
		// In HIL a variable access is just a single string which might contain
		// a mixture of identifiers, dots, integer indices, and splat expressions.
		// All of these concepts were formerly interpreted by Terraform itself,
		// rather than by HIL. We're going to process this one chunk at a time
		// here so we can normalize and introduce some newer syntax where it's
		// safe to do so.
		parts := strings.Split(tv.Name, ".")

		// First we need to deal with the .count pseudo-attributes that 0.11 and
		// prior allowed for resources. These no longer exist, because they
		// don't do anything we can't do with the length(...) function.
		if len(parts) > 0 {
			var rAddr addrs.Resource
			switch parts[0] {
			case "data":
				if len(parts) == 4 && parts[3] == "count" {
					rAddr.Mode = addrs.DataResourceMode
					rAddr.Type = parts[1]
					rAddr.Name = parts[2]
				}
			default:
				if len(parts) == 3 && parts[2] == "count" {
					rAddr.Mode = addrs.ManagedResourceMode
					rAddr.Type = parts[0]
					rAddr.Name = parts[1]
				}
			}

			// We need to check if the thing being referenced is actually an
			// existing resource, because other three-part traversals might
			// coincidentally end with "count".
			if hasCount, exists := an.ResourceHasCount[rAddr]; exists {
				if hasCount {
					buf.WriteString("length(")
					buf.WriteString(rAddr.String())
					buf.WriteString(")")
				} else {
					// If the resource does not have count, the .count
					// attr would've always returned 1 before.
					buf.WriteString("1")
				}
				break Value
			}
		}

		parts = upgradeTraversalParts(parts, an) // might add/remove/change parts
		first, remain := parts[0], parts[1:]
		buf.WriteString(first)
		seenSplat := false
		for _, part := range remain {
			if part == "*" {
				seenSplat = true
				buf.WriteString(".*")
				continue
			}

			// Other special cases apply only if we've not previously
			// seen a splat expression marker, since attribute vs. index
			// syntax have different interpretations after a simple splat.
			if !seenSplat {
				if v, err := strconv.Atoi(part); err == nil {
					// Looks like it's old-style index traversal syntax foo.0.bar
					// so we'll replace with canonical index syntax foo[0].bar.
					fmt.Fprintf(&buf, "[%d]", v)
					continue
				}
				if !hcl2syntax.ValidIdentifier(part) {
					// This should be rare since HIL's identifier syntax is _close_
					// to HCL2's, but we'll get here if one of the intervening
					// parts is not a valid identifier in isolation, since HIL
					// did not consider these to be separate identifiers.
					// e.g. foo.1bar would be invalid in HCL2; must instead be foo["1bar"].
					buf.WriteByte('[')
					printQuotedString(&buf, part)
					buf.WriteByte(']')
					continue
				}
			}

			buf.WriteByte('.')
			buf.WriteString(part)
		}

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

		// Some adaptations must happen prior to upgrading the arguments,
		// because they depend on the original argument AST nodes.
		switch name {
		case "base64sha256", "base64sha512", "md5", "sha1", "sha256", "sha512":
			// These functions were sometimes used in conjunction with the
			// file() function to take the hash of the contents of a file.
			// Prior to Terraform 0.11 there was a chance of silent corruption
			// of strings containing non-UTF8 byte sequences, and so we have
			// made it illegal to use file() with non-text files in 0.12 even
			// though in this _particular_ situation (passing the function
			// result directly to another function) there would not be any
			// corruption; the general rule keeps things consistent.
			// However, to still meet those use-cases we now have variants of
			// the hashing functions that have a "file" prefix on their names
			// and read the contents of a given file, rather than hashing
			// directly the given string.
			if len(args) > 0 {
				if subCall, ok := args[0].(*hilast.Call); ok && subCall.Func == "file" {
					// We're going to flatten this down into a single call, so
					// we actually want the arguments of the sub-call here.
					name = "file" + name
					args = subCall.Args

					// For this one, we'll fall through to the normal upgrade
					// handling now that we've fixed up the name and args...
				}
			}

		}

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
			break Value
		case "map":
			// Should now use object constructor syntax, but we can only
			// achieve that if the call is valid, which requires an even
			// number of arguments.
			if len(argExprs) == 0 {
				buf.WriteString("{}")
				break Value
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
				break Value
			}
		case "lookup":
			// A lookup call with only two arguments is equivalent to native
			// index syntax. (A third argument would specify a default value,
			// so calls like that must be left alone.)
			// (Note that we can't safely do this for element(...) because
			// the user may be relying on its wraparound behavior.)
			if len(argExprs) == 2 {
				buf.Write(argExprs[0])
				buf.WriteByte('[')
				buf.Write(argExprs[1])
				buf.WriteByte(']')
				break Value
			}
		case "element":
			// We cannot replace element with index syntax safely in general
			// because users may be relying on its special modulo wraparound
			// behavior that the index syntax doesn't do. However, if it seems
			// like the user is trying to use element with a set, we'll insert
			// an explicit conversion to list to mimic the implicit conversion
			// that we used to do as an unintended side-effect of how functions
			// work in HIL.
			if len(argExprs) > 0 {
				argTy := an.InferExpressionType(argExprs[0], nil)
				if argTy.IsSetType() {
					newExpr := []byte(`tolist(`)
					newExpr = append(newExpr, argExprs[0]...)
					newExpr = append(newExpr, ')')
					argExprs[0] = newExpr
				}
			}

			// HIL used some undocumented special functions to implement certain
			// operations, but since those were actually callable in real expressions
			// some users inevitably depended on them, so we'll fix them up here.
			// These each become two function calls to preserve the old behavior
			// of implicitly converting to the source type first. Usage of these
			// is relatively rare, so the result doesn't need to be too pretty.
		case "__builtin_BoolToString":
			buf.WriteString("tostring(tobool(")
			buf.Write(argExprs[0])
			buf.WriteString("))")
			break Value
		case "__builtin_FloatToString":
			buf.WriteString("tostring(tonumber(")
			buf.Write(argExprs[0])
			buf.WriteString("))")
			break Value
		case "__builtin_IntToString":
			buf.WriteString("tostring(floor(")
			buf.Write(argExprs[0])
			buf.WriteString("))")
			break Value
		case "__builtin_StringToInt":
			buf.WriteString("floor(tostring(")
			buf.Write(argExprs[0])
			buf.WriteString("))")
			break Value
		case "__builtin_StringToFloat":
			buf.WriteString("tonumber(tostring(")
			buf.Write(argExprs[0])
			buf.WriteString("))")
			break Value
		case "__builtin_StringToBool":
			buf.WriteString("tobool(tostring(")
			buf.Write(argExprs[0])
			buf.WriteString("))")
			break Value
		case "__builtin_FloatToInt", "__builtin_IntToFloat":
			// Since "floor" already has an implicit conversion of its argument
			// to number, and the result is a whole number in either case,
			// these ones are easier. (We no longer distinguish int and float
			// as types in HCL2, even though HIL did.)
			name = "floor"
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

func upgradeHeredocBody(buf *bytes.Buffer, val *hilast.Output, filename string, an *analysis) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	for _, item := range val.Exprs {
		if lit, ok := item.(*hilast.LiteralNode); ok {
			if litStr, ok := lit.Value.(string); ok {
				printHeredocLiteralFromHILOutput(buf, litStr)
				continue
			}
		}
		interped, interpDiags := upgradeExpr(item, filename, true, an)
		diags = diags.Append(interpDiags)

		buf.WriteString("${")
		buf.Write(interped)
		buf.WriteString("}")
	}

	return diags
}

func upgradeTraversalExpr(val interface{}, filename string, an *analysis) ([]byte, tfdiags.Diagnostics) {
	if lit, ok := val.(*hcl1ast.LiteralType); ok && lit.Token.Type == hcl1token.STRING {
		trStr := lit.Token.Value().(string)
		if strings.HasSuffix(trStr, ".%") || strings.HasSuffix(trStr, ".#") {
			// Terraform 0.11 would often not validate traversals given in
			// strings and so users would get away with this sort of
			// flatmap-implementation-detail reference, particularly inside
			// ignore_changes. We'll just trim these off to tolerate it,
			// rather than failing below in ParseTraversalAbs.
			trStr = trStr[:len(trStr)-2]
		}
		trSrc := []byte(trStr)
		_, trDiags := hcl2syntax.ParseTraversalAbs(trSrc, "", hcl2.Pos{})
		if !trDiags.HasErrors() {
			return trSrc, nil
		}
	}
	return upgradeExpr(val, filename, false, an)
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

// upgradeTraversalParts might alter the given split parts from a HIL-style
// variable access to account for renamings made in Terraform v0.12.
func upgradeTraversalParts(parts []string, an *analysis) []string {
	parts = upgradeCountTraversalParts(parts, an)
	parts = upgradeTerraformRemoteStateTraversalParts(parts, an)
	return parts
}

func upgradeCountTraversalParts(parts []string, an *analysis) []string {
	// test_instance.foo.id needs to become test_instance.foo[0].id if
	// count is set for test_instance.foo. Likewise, if count _isn't_ set
	// then test_instance.foo.0.id must become test_instance.foo.id.
	if len(parts) < 3 {
		return parts
	}
	var addr addrs.Resource
	var idxIdx int
	switch parts[0] {
	case "data":
		addr.Mode = addrs.DataResourceMode
		addr.Type = parts[1]
		addr.Name = parts[2]
		idxIdx = 3
	default:
		addr.Mode = addrs.ManagedResourceMode
		addr.Type = parts[0]
		addr.Name = parts[1]
		idxIdx = 2
	}

	hasCount, exists := an.ResourceHasCount[addr]
	if !exists {
		// Probably not actually a resource instance at all, then.
		return parts
	}

	// Since at least one attribute is required after a resource reference
	// prior to Terraform v0.12, we can assume there will be at least enough
	// parts to contain the index even if no index is actually present.
	if idxIdx >= len(parts) {
		return parts
	}

	maybeIdx := parts[idxIdx]
	switch {
	case hasCount:
		if _, err := strconv.Atoi(maybeIdx); err == nil || maybeIdx == "*" {
			// Has an index already, so no changes required.
			return parts
		}
		// Need to insert index zero at idxIdx.
		log.Printf("[TRACE] configupgrade: %s has count but reference does not have index, so adding one", addr)
		newParts := make([]string, len(parts)+1)
		copy(newParts, parts[:idxIdx])
		newParts[idxIdx] = "0"
		copy(newParts[idxIdx+1:], parts[idxIdx:])
		return newParts
	default:
		// For removing indexes we'll be more conservative and only remove
		// exactly index "0", because other indexes on a resource without
		// count are invalid anyway and we're better off letting the normal
		// configuration parser deal with that.
		if maybeIdx != "0" {
			return parts
		}

		// Need to remove the index zero.
		log.Printf("[TRACE] configupgrade: %s does not have count but reference has index, so removing it", addr)
		newParts := make([]string, len(parts)-1)
		copy(newParts, parts[:idxIdx])
		copy(newParts[idxIdx:], parts[idxIdx+1:])
		return newParts
	}
}

func upgradeTerraformRemoteStateTraversalParts(parts []string, an *analysis) []string {
	// data.terraform_remote_state.x.foo needs to become
	// data.terraform_remote_state.x.outputs.foo unless "foo" is a real
	// attribute in the object type implied by the remote state schema.
	if len(parts) < 4 {
		return parts
	}
	if parts[0] != "data" || parts[1] != "terraform_remote_state" {
		return parts
	}

	attrIdx := 3
	if parts[attrIdx] == "*" {
		attrIdx = 4 // data.terraform_remote_state.x.*.foo
	} else if _, err := strconv.Atoi(parts[attrIdx]); err == nil {
		attrIdx = 4 // data.terraform_remote_state.x.1.foo
	}
	if attrIdx >= len(parts) {
		return parts
	}

	attrName := parts[attrIdx]

	// Now we'll use the schema of data.terraform_remote_state to decide if
	// the user intended this to be an output, or whether it's one of the real
	// attributes of this data source.
	var schema *configschema.Block
	if providerSchema := an.ProviderSchemas["terraform"]; providerSchema != nil {
		schema, _ = providerSchema.SchemaForResourceType(addrs.DataResourceMode, "terraform_remote_state")
	}
	// Schema should be available in all reasonable cases, but might be nil
	// if input configuration contains a reference to a remote state data resource
	// without actually defining that data resource. In that weird edge case,
	// we'll just assume all attributes are outputs.
	if schema != nil && schema.ImpliedType().HasAttribute(attrName) {
		// User is accessing one of the real attributes, then, and we have
		// no need to rewrite it.
		return parts
	}

	// If we get down here then our task is to produce a new parts slice
	// that has the fixed additional attribute name "outputs" inserted at
	// attrIdx, retaining all other parts.
	newParts := make([]string, len(parts)+1)
	copy(newParts, parts[:attrIdx])
	newParts[attrIdx] = "outputs"
	copy(newParts[attrIdx+1:], parts[attrIdx:])
	return newParts
}

func typeIsSettableFromTupleCons(ty cty.Type) bool {
	return ty.IsListType() || ty.IsTupleType() || ty.IsSetType()
}
