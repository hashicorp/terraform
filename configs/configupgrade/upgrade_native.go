package configupgrade

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"sort"
	"strings"

	version "github.com/hashicorp/go-version"

	hcl1ast "github.com/hashicorp/hcl/hcl/ast"
	hcl1parser "github.com/hashicorp/hcl/hcl/parser"
	hcl1printer "github.com/hashicorp/hcl/hcl/printer"
	hcl1token "github.com/hashicorp/hcl/hcl/token"

	hcl2 "github.com/hashicorp/hcl2/hcl"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/addrs"
	backendinit "github.com/hashicorp/terraform/backend/init"
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

	log.Printf("[TRACE] configupgrade: Working on %q", filename)

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

			log.Printf("[TRACE] configupgrade: Upgrading %s at %s", rAddr, declRange)
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
			log.Printf("[TRACE] configupgrade: Upgrading provider.%s at %s", pType, declRange)
			moreDiags := u.upgradeNativeSyntaxProvider(filename, &buf, pType, item, an, adhocComments)
			diags = diags.Append(moreDiags)

		case "terraform":
			if len(labels) != 0 {
				diags = diags.Append(&hcl2.Diagnostic{
					Severity: hcl2.DiagError,
					Summary:  fmt.Sprintf("Invalid %s block", blockType),
					Detail:   fmt.Sprintf("A %s block must not have any labels.", blockType),
					Subject:  &declRange,
				})
				continue
			}
			moreDiags := u.upgradeNativeSyntaxTerraformBlock(filename, &buf, item, an, adhocComments)
			diags = diags.Append(moreDiags)

		case "variable":
			if len(labels) != 1 {
				diags = diags.Append(&hcl2.Diagnostic{
					Severity: hcl2.DiagError,
					Summary:  fmt.Sprintf("Invalid %s block", blockType),
					Detail:   fmt.Sprintf("A %s block must have one label: the variable name.", blockType),
					Subject:  &declRange,
				})
				continue
			}

			printComments(&buf, item.LeadComment)
			printBlockOpen(&buf, blockType, labels, item.LineComment)
			rules := bodyContentRules{
				"description": noInterpAttributeRule(filename, cty.String, an),
				"default":     noInterpAttributeRule(filename, cty.DynamicPseudoType, an),
				"type": maybeBareKeywordAttributeRule(filename, an, map[string]string{
					// "list" and "map" in older versions were documented to
					// mean list and map of strings, so we'll migrate to that
					// and let the user adjust it to some other type if desired.
					"list": `list(string)`,
					"map":  `map(string)`,
				}),
			}
			log.Printf("[TRACE] configupgrade: Upgrading var.%s at %s", labels[0], declRange)
			bodyDiags := upgradeBlockBody(filename, fmt.Sprintf("var.%s", labels[0]), &buf, body.List.Items, body.Rbrace, rules, adhocComments)
			diags = diags.Append(bodyDiags)
			buf.WriteString("}\n\n")

		case "output":
			if len(labels) != 1 {
				diags = diags.Append(&hcl2.Diagnostic{
					Severity: hcl2.DiagError,
					Summary:  fmt.Sprintf("Invalid %s block", blockType),
					Detail:   fmt.Sprintf("A %s block must have one label: the output name.", blockType),
					Subject:  &declRange,
				})
				continue
			}

			printComments(&buf, item.LeadComment)
			printBlockOpen(&buf, blockType, labels, item.LineComment)

			rules := bodyContentRules{
				"description": noInterpAttributeRule(filename, cty.String, an),
				"value":       normalAttributeRule(filename, cty.DynamicPseudoType, an),
				"sensitive":   noInterpAttributeRule(filename, cty.Bool, an),
				"depends_on":  dependsOnAttributeRule(filename, an),
			}
			log.Printf("[TRACE] configupgrade: Upgrading output.%s at %s", labels[0], declRange)
			bodyDiags := upgradeBlockBody(filename, fmt.Sprintf("output.%s", labels[0]), &buf, body.List.Items, body.Rbrace, rules, adhocComments)
			diags = diags.Append(bodyDiags)
			buf.WriteString("}\n\n")

		case "module":
			if len(labels) != 1 {
				diags = diags.Append(&hcl2.Diagnostic{
					Severity: hcl2.DiagError,
					Summary:  fmt.Sprintf("Invalid %s block", blockType),
					Detail:   fmt.Sprintf("A %s block must have one label: the module call name.", blockType),
					Subject:  &declRange,
				})
				continue
			}

			// Since upgrading is a single-module endeavor, we don't have access
			// to the configuration of the child module here, but we know that
			// in practice all arguments that aren't reserved meta-arguments
			// in a module block are normal expression attributes so we'll
			// start with the straightforward mapping of those and override
			// the special lifecycle arguments below.
			rules := justAttributesBodyRules(filename, body, an)
			rules["source"] = noInterpAttributeRule(filename, cty.String, an)
			rules["version"] = noInterpAttributeRule(filename, cty.String, an)
			rules["providers"] = func(buf *bytes.Buffer, blockAddr string, item *hcl1ast.ObjectItem) tfdiags.Diagnostics {
				var diags tfdiags.Diagnostics
				subBody, ok := item.Val.(*hcl1ast.ObjectType)
				if !ok {
					diags = diags.Append(&hcl2.Diagnostic{
						Severity: hcl2.DiagError,
						Summary:  "Invalid providers argument",
						Detail:   `The "providers" argument must be a map from provider addresses in the child module to corresponding provider addresses in this module.`,
						Subject:  &declRange,
					})
					return diags
				}

				// We're gonna cheat here and use justAttributesBodyRules to
				// find all the attribute names but then just rewrite them all
				// to be our specialized traversal-style mapping instead.
				subRules := justAttributesBodyRules(filename, subBody, an)
				for k := range subRules {
					subRules[k] = maybeBareTraversalAttributeRule(filename, an)
				}
				buf.WriteString("providers = {\n")
				bodyDiags := upgradeBlockBody(filename, blockAddr, buf, subBody.List.Items, body.Rbrace, subRules, adhocComments)
				diags = diags.Append(bodyDiags)
				buf.WriteString("}\n")

				return diags
			}

			printComments(&buf, item.LeadComment)
			printBlockOpen(&buf, blockType, labels, item.LineComment)
			log.Printf("[TRACE] configupgrade: Upgrading module.%s at %s", labels[0], declRange)
			bodyDiags := upgradeBlockBody(filename, fmt.Sprintf("module.%s", labels[0]), &buf, body.List.Items, body.Rbrace, rules, adhocComments)
			diags = diags.Append(bodyDiags)
			buf.WriteString("}\n\n")

		case "locals":
			log.Printf("[TRACE] configupgrade: Upgrading locals block at %s", declRange)
			printComments(&buf, item.LeadComment)
			printBlockOpen(&buf, blockType, labels, item.LineComment)

			// The "locals" block contents are free-form declarations, so
			// we'll just use the default attribute mapping rule for everything
			// inside it.
			rules := justAttributesBodyRules(filename, body, an)
			log.Printf("[TRACE] configupgrade: Upgrading locals block at %s", declRange)
			bodyDiags := upgradeBlockBody(filename, "locals", &buf, body.List.Items, body.Rbrace, rules, adhocComments)
			diags = diags.Append(bodyDiags)
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
	if schema == nil {
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

	rules := schemaDefaultBodyRules(filename, schema, an, adhocComments)
	rules["count"] = normalAttributeRule(filename, cty.Number, an)
	rules["provider"] = maybeBareTraversalAttributeRule(filename, an)

	printComments(buf, item.LeadComment)
	printBlockOpen(buf, blockType, labels, item.LineComment)
	bodyDiags := upgradeBlockBody(filename, addr.String(), buf, body.List.Items, body.Rbrace, rules, adhocComments)
	diags = diags.Append(bodyDiags)
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
	rules := schemaDefaultBodyRules(filename, schema, an, adhocComments)
	rules["alias"] = noInterpAttributeRule(filename, cty.String, an)
	rules["version"] = noInterpAttributeRule(filename, cty.String, an)

	printComments(buf, item.LeadComment)
	printBlockOpen(buf, "provider", []string{typeName}, item.LineComment)
	bodyDiags := upgradeBlockBody(filename, fmt.Sprintf("provider.%s", typeName), buf, body.List.Items, body.Rbrace, rules, adhocComments)
	diags = diags.Append(bodyDiags)
	buf.WriteString("}\n\n")

	return diags
}

func (u *Upgrader) upgradeNativeSyntaxTerraformBlock(filename string, buf *bytes.Buffer, item *hcl1ast.ObjectItem, an *analysis, adhocComments *commentQueue) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	body := item.Val.(*hcl1ast.ObjectType)

	rules := bodyContentRules{
		"required_version": noInterpAttributeRule(filename, cty.String, an),
		"backend": func(buf *bytes.Buffer, blockAddr string, item *hcl1ast.ObjectItem) tfdiags.Diagnostics {
			var diags tfdiags.Diagnostics

			declRange := hcl1PosRange(filename, item.Keys[0].Pos())
			if len(item.Keys) != 2 {
				diags = diags.Append(&hcl2.Diagnostic{
					Severity: hcl2.DiagError,
					Summary:  `Invalid backend block`,
					Detail:   `A backend block must have one label: the backend type name.`,
					Subject:  &declRange,
				})
				return diags
			}

			typeName := item.Keys[1].Token.Value().(string)
			beFn := backendinit.Backend(typeName)
			if beFn == nil {
				diags = diags.Append(&hcl2.Diagnostic{
					Severity: hcl2.DiagError,
					Summary:  "Unsupported backend type",
					Detail:   fmt.Sprintf("Terraform does not support a backend type named %q.", typeName),
					Subject:  &declRange,
				})
				return diags
			}
			be := beFn()
			schema := be.ConfigSchema()
			rules := schemaNoInterpBodyRules(filename, schema, an, adhocComments)

			body := item.Val.(*hcl1ast.ObjectType)

			printComments(buf, item.LeadComment)
			printBlockOpen(buf, "backend", []string{typeName}, item.LineComment)
			bodyDiags := upgradeBlockBody(filename, fmt.Sprintf("terraform.backend.%s", typeName), buf, body.List.Items, body.Rbrace, rules, adhocComments)
			diags = diags.Append(bodyDiags)
			buf.WriteString("}\n")

			return diags
		},
	}

	printComments(buf, item.LeadComment)
	printBlockOpen(buf, "terraform", nil, item.LineComment)
	bodyDiags := upgradeBlockBody(filename, "terraform", buf, body.List.Items, body.Rbrace, rules, adhocComments)
	diags = diags.Append(bodyDiags)
	buf.WriteString("}\n\n")

	return diags
}

func upgradeBlockBody(filename string, blockAddr string, buf *bytes.Buffer, args []*hcl1ast.ObjectItem, end hcl1token.Pos, rules bodyContentRules, adhocComments *commentQueue) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	for i, arg := range args {
		comments := adhocComments.TakeBefore(arg)
		for _, group := range comments {
			printComments(buf, group)
			buf.WriteByte('\n') // Extra separator after each group
		}

		printComments(buf, arg.LeadComment)

		name := arg.Keys[0].Token.Value().(string)

		rule, expected := rules[name]
		if !expected {
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

		itemDiags := rule(buf, blockAddr, arg)
		diags = diags.Append(itemDiags)

		// If we have another item and it's more than one line away
		// from the current one then we'll print an extra blank line
		// to retain that separation.
		if (i + 1) < len(args) {
			next := args[i+1]
			thisPos := hcl1NodeEndPos(arg)
			nextPos := next.Pos()
			if nextPos.Line-thisPos.Line > 1 {
				buf.WriteByte('\n')
			}
		}
	}

	// Before we return, we must also print any remaining adhocComments that
	// appear between our last item and the closing brace.
	comments := adhocComments.TakeBeforePos(end)
	for i, group := range comments {
		printComments(buf, group)
		if i < len(comments)-1 {
			buf.WriteByte('\n') // Extra separator after each group
		}
	}

	return diags
}

// printDynamicBody prints out a conservative, exhaustive dynamic block body
// for every attribute and nested block in the given schema, for situations
// when a dynamic expression was being assigned to a block type name in input
// configuration and so we can assume it's a list of maps but can't make
// any assumptions about what subset of the schema-specified keys might be
// present in the map values.
func printDynamicBlockBody(buf *bytes.Buffer, iterName string, schema *configschema.Block) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	attrNames := make([]string, 0, len(schema.Attributes))
	for name := range schema.Attributes {
		attrNames = append(attrNames, name)
	}
	sort.Strings(attrNames)
	for _, name := range attrNames {
		attrS := schema.Attributes[name]
		if !(attrS.Required || attrS.Optional) { // no Computed-only attributes
			continue
		}
		if attrS.Required {
			// For required attributes we can generate a simpler expression
			// that just assumes the presence of the key representing the
			// attribute value.
			printAttribute(buf, name, []byte(fmt.Sprintf(`%s.value.%s`, iterName, name)), nil)
		} else {
			// Otherwise we must be conservative and generate a conditional
			// lookup that will just populate nothing at all if the expected
			// key is not present.
			printAttribute(buf, name, []byte(fmt.Sprintf(`lookup(%s.value, %q, null)`, iterName, name)), nil)
		}
	}

	blockTypeNames := make([]string, 0, len(schema.BlockTypes))
	for name := range schema.BlockTypes {
		blockTypeNames = append(blockTypeNames, name)
	}
	sort.Strings(blockTypeNames)
	for i, name := range blockTypeNames {
		blockS := schema.BlockTypes[name]

		// We'll disregard any block type that consists only of computed
		// attributes, since otherwise we'll just create weird empty blocks
		// that do nothing except create confusion.
		if !schemaHasSettableArguments(&blockS.Block) {
			continue
		}

		if i > 0 || len(attrNames) > 0 {
			buf.WriteByte('\n')
		}
		printBlockOpen(buf, "dynamic", []string{name}, nil)
		switch blockS.Nesting {
		case configschema.NestingMap:
			printAttribute(buf, "for_each", []byte(fmt.Sprintf(`lookup(%s.value, %q, {})`, iterName, name)), nil)
			printAttribute(buf, "labels", []byte(fmt.Sprintf(`[%s.key]`, name)), nil)
		case configschema.NestingSingle:
			printAttribute(buf, "for_each", []byte(fmt.Sprintf(`lookup(%s.value, %q, null) != null ? [%s.value.%s] : []`, iterName, name, iterName, name)), nil)
		default:
			printAttribute(buf, "for_each", []byte(fmt.Sprintf(`lookup(%s.value, %q, [])`, iterName, name)), nil)
		}
		printBlockOpen(buf, "content", nil, nil)
		moreDiags := printDynamicBlockBody(buf, name, &blockS.Block)
		diags = diags.Append(moreDiags)
		buf.WriteString("}\n")
		buf.WriteString("}\n")
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

func (q *commentQueue) TakeBeforeToken(token hcl1token.Token) []*hcl1ast.CommentGroup {
	return q.TakeBeforePos(token.Pos)
}

func (q *commentQueue) TakeBefore(node hcl1ast.Node) []*hcl1ast.CommentGroup {
	return q.TakeBeforePos(node.Pos())
}

func (q *commentQueue) TakeBeforePos(pos hcl1token.Pos) []*hcl1ast.CommentGroup {
	toPos := pos
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

// hcl1NodeEndPos tries to find the latest possible position in the given
// node. This is primarily to try to find the last line number of a multi-line
// construct and is a best-effort sort of thing because HCL1 only tracks
// start positions for tokens and has no generalized way to find the full
// range for a single node.
func hcl1NodeEndPos(node hcl1ast.Node) hcl1token.Pos {
	switch tn := node.(type) {
	case *hcl1ast.ObjectItem:
		if tn.LineComment != nil && len(tn.LineComment.List) > 0 {
			return tn.LineComment.List[len(tn.LineComment.List)-1].Start
		}
		return hcl1NodeEndPos(tn.Val)
	case *hcl1ast.ListType:
		return tn.Rbrack
	case *hcl1ast.ObjectType:
		return tn.Rbrace
	default:
		// If all else fails, we'll just return the position of what we were given.
		return tn.Pos()
	}
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

func schemaHasSettableArguments(schema *configschema.Block) bool {
	for _, attrS := range schema.Attributes {
		if attrS.Optional || attrS.Required {
			return true
		}
	}
	for _, blockS := range schema.BlockTypes {
		if schemaHasSettableArguments(&blockS.Block) {
			return true
		}
	}
	return false
}
