package configupgrade

import (
	"bytes"
	"fmt"
	"io"
	"sort"
	"strings"

	version "github.com/hashicorp/go-version"

	hcl1ast "github.com/hashicorp/hcl/hcl/ast"
	hcl1parser "github.com/hashicorp/hcl/hcl/parser"
	hcl1printer "github.com/hashicorp/hcl/hcl/printer"
	hcl1token "github.com/hashicorp/hcl/hcl/token"

	hcl2 "github.com/hashicorp/hcl2/hcl"
	hcl2syntax "github.com/hashicorp/hcl2/hcl/hclsyntax"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/tfdiags"
)

type upgradeFileResult struct {
	Content              []byte
	ProviderRequirements map[string]version.Constraints
}

func (u *Upgrader) upgradeNativeSyntaxFile(filename string, src []byte, an *analysis) (upgradeFileResult, tfdiags.Diagnostics) {
	var result upgradeFileResult
	var diags tfdiags.Diagnostics

	var buf bytes.Buffer

	f, err := hcl1parser.Parse(src)
	if err != nil {
		return result, diags.Append(&hcl2.Diagnostic{
			Severity: hcl2.DiagError,
			Summary:  "Syntax error in configuration file",
			Detail:   fmt.Sprintf("Error while parsing: %s", err),
			Subject:  hcl1ErrSubjectRange(filename, err),
		})
	}

	rootList := f.Node.(*hcl1ast.ObjectList)
	rootItems := rootList.Items
	adhocComments := collectAdhocComments(f)

	for _, item := range rootItems {
		comments := adhocComments.TakeBefore(item)
		for _, group := range comments {
			printComments(&buf, group)
			buf.WriteByte('\n') // Extra separator after each group
		}

		blockType := item.Keys[0].Token.Value().(string)
		labels := make([]string, len(item.Keys)-1)
		for i, key := range item.Keys[1:] {
			labels[i] = key.Token.Value().(string)
		}
		body, isObject := item.Val.(*hcl1ast.ObjectType)
		if !isObject {
			// Should never happen for valid input, since we don't expect
			// any non-block items at our top level.
			diags = diags.Append(&hcl2.Diagnostic{
				Severity: hcl2.DiagWarning,
				Summary:  "Unsupported top-level attribute",
				Detail:   fmt.Sprintf("Attribute %q is not expected here, so its expression was not upgraded.", blockType),
				Subject:  hcl1PosRange(filename, item.Keys[0].Pos()).Ptr(),
			})
			// Preserve the item as-is, using the hcl1printer package.
			buf.WriteString("# TF-UPGRADE-TODO: Top-level attributes are not valid, so this was not automatically upgraded.\n")
			hcl1printer.Fprint(&buf, item)
			buf.WriteString("\n\n")
			continue
		}
		declRange := hcl1PosRange(filename, item.Keys[0].Pos())

		switch blockType {

		case "resource", "data":
			if len(labels) != 2 {
				// Should never happen for valid input.
				diags = diags.Append(&hcl2.Diagnostic{
					Severity: hcl2.DiagError,
					Summary:  fmt.Sprintf("Invalid %s block", blockType),
					Detail:   fmt.Sprintf("A %s block must have two labels: the type and the name.", blockType),
					Subject:  &declRange,
				})
				continue
			}

			rAddr := addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: labels[0],
				Name: labels[1],
			}
			if blockType == "data" {
				rAddr.Mode = addrs.DataResourceMode
			}

			moreDiags := u.upgradeNativeSyntaxResource(filename, &buf, rAddr, item, an, adhocComments)
			diags = diags.Append(moreDiags)

		case "provider":
			if len(labels) != 1 {
				diags = diags.Append(&hcl2.Diagnostic{
					Severity: hcl2.DiagError,
					Summary:  fmt.Sprintf("Invalid %s block", blockType),
					Detail:   fmt.Sprintf("A %s block must have one label: the provider type.", blockType),
					Subject:  &declRange,
				})
				continue
			}

			pType := labels[0]
			moreDiags := u.upgradeNativeSyntaxProvider(filename, &buf, pType, item, an, adhocComments)
			diags = diags.Append(moreDiags)

		case "variable":
			printComments(&buf, item.LeadComment)
			printBlockOpen(&buf, blockType, labels, item.LineComment)
			args := body.List.Items
			for i, arg := range args {
				if len(arg.Keys) != 1 {
					// Should never happen for valid input, since there are no nested blocks expected here.
					diags = diags.Append(&hcl2.Diagnostic{
						Severity: hcl2.DiagWarning,
						Summary:  "Invalid nested block",
						Detail:   fmt.Sprintf("Blocks of type %q are not expected here, so this was not automatically upgraded.", arg.Keys[0].Token.Value().(string)),
						Subject:  hcl1PosRange(filename, arg.Keys[0].Pos()).Ptr(),
					})
					// Preserve the item as-is, using the hcl1printer package.
					buf.WriteString("\n# TF-UPGRADE-TODO: Blocks are not expected here, so this was not automatically upgraded.\n")
					hcl1printer.Fprint(&buf, arg)
					buf.WriteString("\n\n")
					continue
				}

				comments := adhocComments.TakeBefore(arg)
				for _, group := range comments {
					printComments(&buf, group)
					buf.WriteByte('\n') // Extra separator after each group
				}

				printComments(&buf, arg.LeadComment)

				switch arg.Keys[0].Token.Value() {
				case "type":
					// It is no longer idiomatic to place the type keyword in quotes,
					// so we'll unquote it here as long as it looks like the result
					// will be valid.
					if lit, isLit := arg.Val.(*hcl1ast.LiteralType); isLit {
						if lit.Token.Type == hcl1token.STRING {
							kw := lit.Token.Value().(string)
							if hcl2syntax.ValidIdentifier(kw) {

								// "list" and "map" in older versions really meant
								// list and map of strings, so we'll migrate to
								// that and let the user adjust to "any" as
								// the element type if desired.
								switch strings.TrimSpace(kw) {
								case "list":
									kw = "list(string)"
								case "map":
									kw = "map(string)"
								}

								printAttribute(&buf, "type", []byte(kw), arg.LineComment)
								break
							}
						}
					}
					// If we got something invalid there then we'll just fall through
					// into the default case and migrate it as a normal expression.
					fallthrough
				default:
					valSrc, valDiags := upgradeExpr(arg.Val, filename, false, an)
					diags = diags.Append(valDiags)
					printAttribute(&buf, arg.Keys[0].Token.Value().(string), valSrc, arg.LineComment)
				}

				// If we have another item and it's more than one line away
				// from the current one then we'll print an extra blank line
				// to retain that separation.
				if (i + 1) < len(args) {
					next := args[i+1]
					thisPos := arg.Pos()
					nextPos := next.Pos()
					if nextPos.Line-thisPos.Line > 1 {
						buf.WriteByte('\n')
					}
				}
			}
			buf.WriteString("}\n\n")

		case "output":
			printComments(&buf, item.LeadComment)
			printBlockOpen(&buf, blockType, labels, item.LineComment)
			args := body.List.Items
			for i, arg := range args {
				if len(arg.Keys) != 1 {
					// Should never happen for valid input, since there are no nested blocks expected here.
					diags = diags.Append(&hcl2.Diagnostic{
						Severity: hcl2.DiagWarning,
						Summary:  "Invalid nested block",
						Detail:   fmt.Sprintf("Blocks of type %q are not expected here, so this was not automatically upgraded.", arg.Keys[0].Token.Value().(string)),
						Subject:  hcl1PosRange(filename, arg.Keys[0].Pos()).Ptr(),
					})
					// Preserve the item as-is, using the hcl1printer package.
					buf.WriteString("\n# TF-UPGRADE-TODO: Blocks are not expected here, so this was not automatically upgraded.\n")
					hcl1printer.Fprint(&buf, arg)
					buf.WriteString("\n\n")
					continue
				}

				comments := adhocComments.TakeBefore(arg)
				for _, group := range comments {
					printComments(&buf, group)
					buf.WriteByte('\n') // Extra separator after each group
				}

				printComments(&buf, arg.LeadComment)

				interp := false
				switch arg.Keys[0].Token.Value() {
				case "value":
					interp = true
				}

				valSrc, valDiags := upgradeExpr(arg.Val, filename, interp, an)
				diags = diags.Append(valDiags)
				printAttribute(&buf, arg.Keys[0].Token.Value().(string), valSrc, arg.LineComment)

				// If we have another item and it's more than one line away
				// from the current one then we'll print an extra blank line
				// to retain that separation.
				if (i + 1) < len(args) {
					next := args[i+1]
					thisPos := arg.Pos()
					nextPos := next.Pos()
					if nextPos.Line-thisPos.Line > 1 {
						buf.WriteByte('\n')
					}
				}
			}
			buf.WriteString("}\n\n")

		case "locals":
			printComments(&buf, item.LeadComment)
			printBlockOpen(&buf, blockType, labels, item.LineComment)

			args := body.List.Items
			for i, arg := range args {
				if len(arg.Keys) != 1 {
					// Should never happen for valid input, since there are no nested blocks expected here.
					diags = diags.Append(&hcl2.Diagnostic{
						Severity: hcl2.DiagWarning,
						Summary:  "Invalid nested block",
						Detail:   fmt.Sprintf("Blocks of type %q are not expected here, so this was not automatically upgraded.", arg.Keys[0].Token.Value().(string)),
						Subject:  hcl1PosRange(filename, arg.Keys[0].Pos()).Ptr(),
					})
					// Preserve the item as-is, using the hcl1printer package.
					buf.WriteString("\n# TF-UPGRADE-TODO: Blocks are not expected here, so this was not automatically upgraded.\n")
					hcl1printer.Fprint(&buf, arg)
					buf.WriteString("\n\n")
					continue
				}

				comments := adhocComments.TakeBefore(arg)
				for _, group := range comments {
					printComments(&buf, group)
					buf.WriteByte('\n') // Extra separator after each group
				}

				printComments(&buf, arg.LeadComment)

				name := arg.Keys[0].Token.Value().(string)
				expr := arg.Val
				exprSrc, exprDiags := upgradeExpr(expr, filename, true, an)
				diags = diags.Append(exprDiags)
				printAttribute(&buf, name, exprSrc, arg.LineComment)

				// If we have another item and it's more than one line away
				// from the current one then we'll print an extra blank line
				// to retain that separation.
				if (i + 1) < len(args) {
					next := args[i+1]
					thisPos := arg.Pos()
					nextPos := next.Pos()
					if nextPos.Line-thisPos.Line > 1 {
						buf.WriteByte('\n')
					}
				}
			}
			buf.WriteString("}\n\n")

		default:
			// Should never happen for valid input, because the above cases
			// are exhaustive for valid blocks as of Terraform 0.11.
			diags = diags.Append(&hcl2.Diagnostic{
				Severity: hcl2.DiagWarning,
				Summary:  "Unsupported root block type",
				Detail:   fmt.Sprintf("The block type %q is not expected here, so its content was not upgraded.", blockType),
				Subject:  hcl1PosRange(filename, item.Keys[0].Pos()).Ptr(),
			})

			// Preserve the block as-is, using the hcl1printer package.
			buf.WriteString("# TF-UPGRADE-TODO: Block type was not recognized, so this block and its contents were not automatically upgraded.\n")
			hcl1printer.Fprint(&buf, item)
			buf.WriteString("\n\n")
			continue
		}
	}

	// Print out any leftover comments
	for _, group := range *adhocComments {
		printComments(&buf, group)
	}

	result.Content = buf.Bytes()

	return result, diags
}

func (u *Upgrader) upgradeNativeSyntaxResource(filename string, buf *bytes.Buffer, addr addrs.Resource, item *hcl1ast.ObjectItem, an *analysis, adhocComments *commentQueue) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	body := item.Val.(*hcl1ast.ObjectType)
	declRange := hcl1PosRange(filename, item.Keys[0].Pos())

	// We should always have a schema for each provider in our analysis
	// object. If not, it's a bug in the analyzer.
	providerType, ok := an.ResourceProviderType[addr]
	if !ok {
		panic(fmt.Sprintf("unknown provider type for %s", addr.String()))
	}
	providerSchema, ok := an.ProviderSchemas[providerType]
	if !ok {
		panic(fmt.Sprintf("missing schema for provider type %q", providerType))
	}
	schema, _ := providerSchema.SchemaForResourceAddr(addr)
	if !ok {
		diags = diags.Append(&hcl2.Diagnostic{
			Severity: hcl2.DiagError,
			Summary:  "Unknown resource type",
			Detail:   fmt.Sprintf("The resource type %q is not known to the currently-selected version of provider %q.", addr.Type, providerType),
			Subject:  &declRange,
		})
		return diags
	}

	var blockType string
	switch addr.Mode {
	case addrs.ManagedResourceMode:
		blockType = "resource"
	case addrs.DataResourceMode:
		blockType = "data"
	}
	labels := []string{addr.Type, addr.Name}

	printComments(buf, item.LeadComment)
	printBlockOpen(buf, blockType, labels, item.LineComment)
	u.upgradeBlockBody(filename, addr.String(), buf, body.List.Items, schema, an, adhocComments)
	buf.WriteString("}\n\n")

	return diags
}

func (u *Upgrader) upgradeNativeSyntaxProvider(filename string, buf *bytes.Buffer, typeName string, item *hcl1ast.ObjectItem, an *analysis, adhocComments *commentQueue) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	body := item.Val.(*hcl1ast.ObjectType)

	// We should always have a schema for each provider in our analysis
	// object. If not, it's a bug in the analyzer.
	providerSchema, ok := an.ProviderSchemas[typeName]
	if !ok {
		panic(fmt.Sprintf("missing schema for provider type %q", typeName))
	}
	schema := providerSchema.Provider

	printComments(buf, item.LeadComment)
	printBlockOpen(buf, "provider", []string{typeName}, item.LineComment)
	u.upgradeBlockBody(filename, fmt.Sprintf("provider.%s", typeName), buf, body.List.Items, schema, an, adhocComments)
	buf.WriteString("}\n\n")

	return diags
}

func (u *Upgrader) upgradeBlockBody(filename string, blockAddr string, buf *bytes.Buffer, args []*hcl1ast.ObjectItem, schema *configschema.Block, an *analysis, adhocComments *commentQueue) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	for i, arg := range args {
		comments := adhocComments.TakeBefore(arg)
		for _, group := range comments {
			printComments(buf, group)
			buf.WriteByte('\n') // Extra separator after each group
		}

		printComments(buf, arg.LeadComment)

		name := arg.Keys[0].Token.Value().(string)
		//labelKeys := arg.Keys[1:]

		switch name {
		// TODO: Special case for all of the "meta-arguments" allowed
		// in a resource block, such as "count", "lifecycle",
		// "provisioner", etc.

		default:
			// We'll consult the schema to see how we ought to interpret
			// this item.

			if _, isAttr := schema.Attributes[name]; isAttr {
				// We'll tolerate a block with no labels here as a degenerate
				// way to assign a map, but we can't migrate a block that has
				// labels. In practice this should never happen because
				// nested blocks in resource blocks did not accept labels
				// prior to v0.12.
				if len(arg.Keys) != 1 {
					diags = diags.Append(&hcl2.Diagnostic{
						Severity: hcl2.DiagError,
						Summary:  "Block where attribute was expected",
						Detail:   fmt.Sprintf("Within %s the name %q is an attribute name, not a block type.", blockAddr, name),
						Subject:  hcl1PosRange(filename, arg.Keys[0].Pos()).Ptr(),
					})
					continue
				}

				valSrc, valDiags := upgradeExpr(arg.Val, filename, true, an)
				diags = diags.Append(valDiags)
				printAttribute(buf, arg.Keys[0].Token.Value().(string), valSrc, arg.LineComment)
			} else if _, isBlock := schema.BlockTypes[name]; isBlock {
				// TODO: Also upgrade blocks.
				// In particular we need to handle the tricky case where
				// a user attempts to treat a block type name like it's
				// an attribute, by producing a "dynamic" block.
				hcl1printer.Fprint(buf, arg)
				buf.WriteByte('\n')
			} else {
				if arg.Assign.IsValid() {
					diags = diags.Append(&hcl2.Diagnostic{
						Severity: hcl2.DiagError,
						Summary:  "Unrecognized attribute name",
						Detail:   fmt.Sprintf("No attribute named %q is expected in %s.", name, blockAddr),
						Subject:  hcl1PosRange(filename, arg.Keys[0].Pos()).Ptr(),
					})
				} else {
					diags = diags.Append(&hcl2.Diagnostic{
						Severity: hcl2.DiagError,
						Summary:  "Unrecognized block type",
						Detail:   fmt.Sprintf("Blocks of type %q are not expected in %s.", name, blockAddr),
						Subject:  hcl1PosRange(filename, arg.Keys[0].Pos()).Ptr(),
					})
				}
				continue
			}
		}

		// If we have another item and it's more than one line away
		// from the current one then we'll print an extra blank line
		// to retain that separation.
		if (i + 1) < len(args) {
			next := args[i+1]
			thisPos := arg.Pos()
			nextPos := next.Pos()
			if nextPos.Line-thisPos.Line > 1 {
				buf.WriteByte('\n')
			}
		}
	}

	return diags
}

func printComments(buf *bytes.Buffer, group *hcl1ast.CommentGroup) {
	if group == nil {
		return
	}
	for _, comment := range group.List {
		buf.WriteString(comment.Text)
		buf.WriteByte('\n')
	}
}

func printBlockOpen(buf *bytes.Buffer, blockType string, labels []string, commentGroup *hcl1ast.CommentGroup) {
	buf.WriteString(blockType)
	for _, label := range labels {
		buf.WriteByte(' ')
		printQuotedString(buf, label)
	}
	buf.WriteString(" {")
	if commentGroup != nil {
		for _, c := range commentGroup.List {
			buf.WriteByte(' ')
			buf.WriteString(c.Text)
		}
	}
	buf.WriteByte('\n')
}

func printAttribute(buf *bytes.Buffer, name string, valSrc []byte, commentGroup *hcl1ast.CommentGroup) {
	buf.WriteString(name)
	buf.WriteString(" = ")
	buf.Write(valSrc)
	if commentGroup != nil {
		for _, c := range commentGroup.List {
			buf.WriteByte(' ')
			buf.WriteString(c.Text)
		}
	}
	buf.WriteByte('\n')
}

func printQuotedString(buf *bytes.Buffer, val string) {
	buf.WriteByte('"')
	printStringLiteralFromHILOutput(buf, val)
	buf.WriteByte('"')
}

func printStringLiteralFromHILOutput(buf *bytes.Buffer, val string) {
	val = strings.Replace(val, `\`, `\\`, -1)
	val = strings.Replace(val, `"`, `\"`, -1)
	val = strings.Replace(val, "\n", `\n`, -1)
	val = strings.Replace(val, "\r", `\r`, -1)
	val = strings.Replace(val, `${`, `$${`, -1)
	val = strings.Replace(val, `%{`, `%%{`, -1)
	buf.WriteString(val)
}

func collectAdhocComments(f *hcl1ast.File) *commentQueue {
	comments := make(map[hcl1token.Pos]*hcl1ast.CommentGroup)
	for _, c := range f.Comments {
		comments[c.Pos()] = c
	}

	// We'll remove from our map any comments that are attached to specific
	// nodes as lead or line comments, since we'll find those during our
	// walk anyway.
	hcl1ast.Walk(f, func(nn hcl1ast.Node) (hcl1ast.Node, bool) {
		switch t := nn.(type) {
		case *hcl1ast.LiteralType:
			if t.LeadComment != nil {
				for _, comment := range t.LeadComment.List {
					delete(comments, comment.Pos())
				}
			}

			if t.LineComment != nil {
				for _, comment := range t.LineComment.List {
					delete(comments, comment.Pos())
				}
			}
		case *hcl1ast.ObjectItem:
			if t.LeadComment != nil {
				for _, comment := range t.LeadComment.List {
					delete(comments, comment.Pos())
				}
			}

			if t.LineComment != nil {
				for _, comment := range t.LineComment.List {
					delete(comments, comment.Pos())
				}
			}
		}

		return nn, true
	})

	if len(comments) == 0 {
		var ret commentQueue
		return &ret
	}

	ret := make([]*hcl1ast.CommentGroup, 0, len(comments))
	for _, c := range comments {
		ret = append(ret, c)
	}
	sort.Slice(ret, func(i, j int) bool {
		return ret[i].Pos().Before(ret[j].Pos())
	})
	queue := commentQueue(ret)
	return &queue
}

type commentQueue []*hcl1ast.CommentGroup

func (q *commentQueue) TakeBefore(node hcl1ast.Node) []*hcl1ast.CommentGroup {
	toPos := node.Pos()
	var i int
	for i = 0; i < len(*q); i++ {
		if (*q)[i].Pos().After(toPos) {
			break
		}
	}
	if i == 0 {
		return nil
	}

	ret := (*q)[:i]
	*q = (*q)[i:]

	return ret
}

func hcl1ErrSubjectRange(filename string, err error) *hcl2.Range {
	if pe, isPos := err.(*hcl1parser.PosError); isPos {
		return hcl1PosRange(filename, pe.Pos).Ptr()
	}
	return nil
}

func hcl1PosRange(filename string, pos hcl1token.Pos) hcl2.Range {
	return hcl2.Range{
		Filename: filename,
		Start: hcl2.Pos{
			Line:   pos.Line,
			Column: pos.Column,
			Byte:   pos.Offset,
		},
		End: hcl2.Pos{
			Line:   pos.Line,
			Column: pos.Column,
			Byte:   pos.Offset,
		},
	}
}

func passthruBlockTodo(w io.Writer, node hcl1ast.Node, msg string) {
	fmt.Fprintf(w, "\n# TF-UPGRADE-TODO: %s\n", msg)
	hcl1printer.Fprint(w, node)
	w.Write([]byte{'\n', '\n'})
}
