package configupgrade

import (
	"bytes"
	"fmt"

	hcl1ast "github.com/hashicorp/hcl/hcl/ast"
	hcl1token "github.com/hashicorp/hcl/hcl/token"
	hcl2 "github.com/hashicorp/hcl2/hcl"
	hcl2syntax "github.com/hashicorp/hcl2/hcl/hclsyntax"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/tfdiags"
)

// bodyContentRules is a mapping from item names (argument names and block type
// names) to a "rule" function defining what to do with an item of that type.
type bodyContentRules map[string]bodyItemRule

// bodyItemRule is just a function to write an upgraded representation of a
// particular given item to the given buffer. This is generic to handle various
// different mapping rules, though most values will be those constructed by
// other helper functions below.
type bodyItemRule func(buf *bytes.Buffer, blockAddr string, item *hcl1ast.ObjectItem) tfdiags.Diagnostics

func normalAttributeRule(filename string, wantTy cty.Type, an *analysis) bodyItemRule {
	exprRule := func(val interface{}) ([]byte, tfdiags.Diagnostics) {
		return upgradeExpr(val, filename, true, an)
	}
	return attributeRule(filename, wantTy, an, exprRule)
}

func noInterpAttributeRule(filename string, wantTy cty.Type, an *analysis) bodyItemRule {
	exprRule := func(val interface{}) ([]byte, tfdiags.Diagnostics) {
		return upgradeExpr(val, filename, false, an)
	}
	return attributeRule(filename, wantTy, an, exprRule)
}

func maybeBareKeywordAttributeRule(filename string, an *analysis, specials map[string]string) bodyItemRule {
	exprRule := func(val interface{}) ([]byte, tfdiags.Diagnostics) {
		// If the expression is a literal that would be valid as a naked keyword
		// then we'll turn it into one.
		if lit, isLit := val.(*hcl1ast.LiteralType); isLit {
			if lit.Token.Type == hcl1token.STRING {
				kw := lit.Token.Value().(string)
				if hcl2syntax.ValidIdentifier(kw) {

					// If we have a special mapping rule for this keyword,
					// we'll let that override what the user gave.
					if override := specials[kw]; override != "" {
						kw = override
					}

					return []byte(kw), nil
				}
			}
		}

		return upgradeExpr(val, filename, false, an)
	}
	return attributeRule(filename, cty.String, an, exprRule)
}

func maybeBareTraversalAttributeRule(filename string, an *analysis) bodyItemRule {
	exprRule := func(val interface{}) ([]byte, tfdiags.Diagnostics) {
		return upgradeTraversalExpr(val, filename, an)
	}
	return attributeRule(filename, cty.String, an, exprRule)
}

func dependsOnAttributeRule(filename string, an *analysis) bodyItemRule {
	// FIXME: Should dig into the individual list items here and try to unwrap
	// them as naked references, as well as upgrading any legacy-style index
	// references like aws_instance.foo.0 to be aws_instance.foo[0] instead.
	exprRule := func(val interface{}) ([]byte, tfdiags.Diagnostics) {
		return upgradeExpr(val, filename, false, an)
	}
	return attributeRule(filename, cty.List(cty.String), an, exprRule)
}

func attributeRule(filename string, wantTy cty.Type, an *analysis, upgradeExpr func(val interface{}) ([]byte, tfdiags.Diagnostics)) bodyItemRule {
	return func(buf *bytes.Buffer, blockAddr string, item *hcl1ast.ObjectItem) tfdiags.Diagnostics {
		var diags tfdiags.Diagnostics

		name := item.Keys[0].Token.Value().(string)

		// We'll tolerate a block with no labels here as a degenerate
		// way to assign a map, but we can't migrate a block that has
		// labels. In practice this should never happen because
		// nested blocks in resource blocks did not accept labels
		// prior to v0.12.
		if len(item.Keys) != 1 {
			diags = diags.Append(&hcl2.Diagnostic{
				Severity: hcl2.DiagError,
				Summary:  "Block where attribute was expected",
				Detail:   fmt.Sprintf("Within %s the name %q is an attribute name, not a block type.", blockAddr, name),
				Subject:  hcl1PosRange(filename, item.Keys[0].Pos()).Ptr(),
			})
			return diags
		}

		valSrc, valDiags := upgradeExpr(item.Val)
		diags = diags.Append(valDiags)
		printAttribute(buf, item.Keys[0].Token.Value().(string), valSrc, item.LineComment)

		return diags
	}
}

func nestedBlockRule(filename string, nestedRules bodyContentRules, an *analysis, adhocComments *commentQueue) bodyItemRule {
	return func(buf *bytes.Buffer, blockAddr string, item *hcl1ast.ObjectItem) tfdiags.Diagnostics {
		// This simpler nestedBlockRule is for contexts where the special
		// "dynamic" block type is not accepted and so only HCL1 object
		// constructs can be accepted. Attempts to assign arbitrary HIL
		// expressions will be rejected as errors.

		var diags tfdiags.Diagnostics
		declRange := hcl1PosRange(filename, item.Keys[0].Pos())
		blockType := item.Keys[0].Token.Value().(string)
		labels := make([]string, len(item.Keys)-1)
		for i, key := range item.Keys[1:] {
			labels[i] = key.Token.Value().(string)
		}

		var blockItems []*hcl1ast.ObjectType

		switch val := item.Val.(type) {

		case *hcl1ast.ObjectType:
			blockItems = []*hcl1ast.ObjectType{val}

		case *hcl1ast.ListType:
			for _, node := range val.List {
				switch listItem := node.(type) {
				case *hcl1ast.ObjectType:
					blockItems = append(blockItems, listItem)
				default:
					diags = diags.Append(&hcl2.Diagnostic{
						Severity: hcl2.DiagError,
						Summary:  "Invalid value for nested block",
						Detail:   fmt.Sprintf("In %s the name %q is a nested block type, so any value assigned to it must be an object.", blockAddr, blockType),
						Subject:  hcl1PosRange(filename, node.Pos()).Ptr(),
					})
				}
			}

		default:
			diags = diags.Append(&hcl2.Diagnostic{
				Severity: hcl2.DiagError,
				Summary:  "Invalid value for nested block",
				Detail:   fmt.Sprintf("In %s the name %q is a nested block type, so any value assigned to it must be an object.", blockAddr, blockType),
				Subject:  &declRange,
			})
			return diags
		}

		for _, blockItem := range blockItems {
			printBlockOpen(buf, blockType, labels, item.LineComment)
			bodyDiags := upgradeBlockBody(
				filename, fmt.Sprintf("%s.%s", blockAddr, blockType), buf,
				blockItem.List.Items, blockItem.Rbrace, nestedRules, adhocComments,
			)
			diags = diags.Append(bodyDiags)
			buf.WriteString("}\n")
		}

		return diags
	}
}

func nestedBlockRuleWithDynamic(filename string, nestedRules bodyContentRules, nestedSchema *configschema.NestedBlock, an *analysis, adhocComments *commentQueue) bodyItemRule {
	return func(buf *bytes.Buffer, blockAddr string, item *hcl1ast.ObjectItem) tfdiags.Diagnostics {
		// In Terraform v0.11 it was possible in some cases to trick Terraform
		// and providers into accepting HCL's attribute syntax and some HIL
		// expressions in places where blocks or sequences of blocks were
		// expected, since the information about the heritage of the values
		// was lost during decoding and interpolation.
		//
		// In order to avoid all of the weird rough edges that resulted from
		// those misinterpretations, Terraform v0.12 is stricter and requires
		// the use of block syntax for blocks in all cases. However, because
		// various abuses of attribute syntax _did_ work (with some caveats)
		// in v0.11 we will upgrade them as best we can to use proper block
		// syntax.
		//
		// There are a few different permutations supported by this code:
		//
		// - Assigning a single HCL1 "object" using attribute syntax. This is
		//   straightforward to migrate just by dropping the equals sign.
		//
		// - Assigning a HCL1 list of objects using attribute syntax. Each
		//   object in that list can be translated to a separate block.
		//
		// - Assigning a HCL1 list containing HIL expressions that evaluate
		//   to maps. This is a hard case because we can't know the internal
		//   structure of those maps during static analysis, and so we must
		//   generate a worst-case dynamic block structure for it.
		//
		// - Assigning a single HIL expression that evaluates to a list of
		//   maps. This is just like the previous case except additionally
		//   we cannot even predict the number of generated blocks, so we must
		//   generate a single "dynamic" block to iterate over the list at
		//   runtime.

		var diags tfdiags.Diagnostics
		blockType := item.Keys[0].Token.Value().(string)
		labels := make([]string, len(item.Keys)-1)
		for i, key := range item.Keys[1:] {
			labels[i] = key.Token.Value().(string)
		}

		var blockItems []hcl1ast.Node

		switch val := item.Val.(type) {

		case *hcl1ast.ObjectType:
			blockItems = append(blockItems, val)

		case *hcl1ast.ListType:
			for _, node := range val.List {
				switch listItem := node.(type) {
				case *hcl1ast.ObjectType:
					blockItems = append(blockItems, listItem)
				default:
					// We're going to cheat a bit here and construct a synthetic
					// HCL1 list just because that makes our logic
					// simpler below where we can just treat all non-objects
					// in the same way when producing "dynamic" blocks.
					synthList := &hcl1ast.ListType{
						List:   []hcl1ast.Node{listItem},
						Lbrack: listItem.Pos(),
						Rbrack: hcl1NodeEndPos(listItem),
					}
					blockItems = append(blockItems, synthList)
				}
			}

		default:
			blockItems = append(blockItems, item.Val)
		}

		for _, blockItem := range blockItems {
			switch ti := blockItem.(type) {
			case *hcl1ast.ObjectType:
				// If we have an object then we'll pass through its content
				// as a block directly. This is the most straightforward mapping
				// from the source input, since we know exactly which keys
				// are present.
				printBlockOpen(buf, blockType, labels, item.LineComment)
				bodyDiags := upgradeBlockBody(
					filename, fmt.Sprintf("%s.%s", blockAddr, blockType), buf,
					ti.List.Items, ti.Rbrace, nestedRules, adhocComments,
				)
				diags = diags.Append(bodyDiags)
				buf.WriteString("}\n")
			default:
				// For any other sort of value we can't predict what shape it
				// will have at runtime, so we must generate a very conservative
				// "dynamic" block that tries to assign everything from the
				// schema. The result of this is likely to be pretty ugly.
				printBlockOpen(buf, "dynamic", []string{blockType}, item.LineComment)
				eachSrc, eachDiags := upgradeExpr(blockItem, filename, true, an)
				diags = diags.Append(eachDiags)
				printAttribute(buf, "for_each", eachSrc, nil)
				if nestedSchema.Nesting == configschema.NestingMap {
					// This is a pretty odd situation since map-based blocks
					// didn't exist prior to Terraform v0.12, but we'll support
					// this anyway in case we decide to add support in a later
					// SDK release that is still somehow compatible with
					// Terraform v0.11.
					printAttribute(buf, "labels", []byte(fmt.Sprintf(`[%s.key]`, blockType)), nil)
				}
				printBlockOpen(buf, "content", nil, nil)
				buf.WriteString("# TF-UPGRADE-TODO: The automatic upgrade tool can't predict\n")
				buf.WriteString("# which keys might be set in maps assigned here, so it has\n")
				buf.WriteString("# produced a comprehensive set here. Consider simplifying\n")
				buf.WriteString("# this after confirming which keys can be set in practice.\n\n")
				printDynamicBlockBody(buf, blockType, &nestedSchema.Block)
				buf.WriteString("}\n")
				buf.WriteString("}\n")
				diags = diags.Append(&hcl2.Diagnostic{
					Severity: hcl2.DiagWarning,
					Summary:  "Approximate migration of invalid block type assignment",
					Detail:   fmt.Sprintf("In %s the name %q is a nested block type, but this configuration is exploiting some missing validation rules from Terraform v0.11 and prior to trick Terraform into creating blocks dynamically.\n\nThis has been upgraded to use the new Terraform v0.12 dynamic blocks feature, but since the upgrade tool cannot predict which map keys will be present a fully-comprehensive set has been generated.", blockAddr, blockType),
					Subject:  hcl1PosRange(filename, blockItem.Pos()).Ptr(),
				})
			}
		}

		return diags
	}
}

// schemaDefaultBodyRules constructs standard body content rules for the given
// schema. Each call is guaranteed to produce a distinct object so that
// callers can safely mutate the result in order to impose custom rules
// in addition to or instead of those created by default, for situations
// where schema-based and predefined items mix in a single body.
func schemaDefaultBodyRules(filename string, schema *configschema.Block, an *analysis, adhocComments *commentQueue) bodyContentRules {
	ret := make(bodyContentRules)
	if schema == nil {
		// Shouldn't happen in any real case, but often crops up in tests
		// where the mock schemas tend to be incomplete.
		return ret
	}

	for name, attrS := range schema.Attributes {
		ret[name] = normalAttributeRule(filename, attrS.Type, an)
	}
	for name, blockS := range schema.BlockTypes {
		nestedRules := schemaDefaultBodyRules(filename, &blockS.Block, an, adhocComments)
		ret[name] = nestedBlockRuleWithDynamic(filename, nestedRules, blockS, an, adhocComments)
	}

	return ret
}

// schemaNoInterpBodyRules constructs standard body content rules for the given
// schema. Each call is guaranteed to produce a distinct object so that
// callers can safely mutate the result in order to impose custom rules
// in addition to or instead of those created by default, for situations
// where schema-based and predefined items mix in a single body.
func schemaNoInterpBodyRules(filename string, schema *configschema.Block, an *analysis, adhocComments *commentQueue) bodyContentRules {
	ret := make(bodyContentRules)
	if schema == nil {
		// Shouldn't happen in any real case, but often crops up in tests
		// where the mock schemas tend to be incomplete.
		return ret
	}

	for name, attrS := range schema.Attributes {
		ret[name] = noInterpAttributeRule(filename, attrS.Type, an)
	}
	for name, blockS := range schema.BlockTypes {
		nestedRules := schemaDefaultBodyRules(filename, &blockS.Block, an, adhocComments)
		ret[name] = nestedBlockRule(filename, nestedRules, an, adhocComments)
	}

	return ret
}

// justAttributesBodyRules constructs body content rules that just use the
// standard interpolated attribute mapping for every name already present
// in the given body object.
//
// This is a little weird vs. just processing directly the attributes, but
// has the advantage that the caller can then apply overrides to the result
// as necessary to deal with any known names that need special handling.
//
// Any attribute rules created by this function do not have a specific wanted
// value type specified, instead setting it to just cty.DynamicPseudoType.
func justAttributesBodyRules(filename string, body *hcl1ast.ObjectType, an *analysis) bodyContentRules {
	rules := make(bodyContentRules, len(body.List.Items))
	args := body.List.Items
	for _, arg := range args {
		name := arg.Keys[0].Token.Value().(string)
		rules[name] = normalAttributeRule(filename, cty.DynamicPseudoType, an)
	}
	return rules
}

func lifecycleBlockBodyRules(filename string, an *analysis) bodyContentRules {
	return bodyContentRules{
		"create_before_destroy": noInterpAttributeRule(filename, cty.Bool, an),
		"prevent_destroy":       noInterpAttributeRule(filename, cty.Bool, an),
		"ignore_changes": func(buf *bytes.Buffer, blockAddr string, item *hcl1ast.ObjectItem) tfdiags.Diagnostics {
			var diags tfdiags.Diagnostics
			val, ok := item.Val.(*hcl1ast.ListType)
			if !ok {
				diags = diags.Append(&hcl2.Diagnostic{
					Severity: hcl2.DiagError,
					Summary:  "Invalid providers argument",
					Detail:   `The "providers" argument must be a map from provider addresses in the child module to corresponding provider addresses in this module.`,
					Subject:  hcl1PosRange(filename, item.Keys[0].Pos()).Ptr(),
				})
				return diags
			}

			// As a special case, we'll map the single-element list ["*"] to
			// the new keyword "all".
			if len(val.List) == 1 {
				if lit, ok := val.List[0].(*hcl1ast.LiteralType); ok {
					if lit.Token.Value() == "*" {
						printAttribute(buf, item.Keys[0].Token.Value().(string), []byte("all"), item.LineComment)
						return diags
					}
				}
			}

			var exprBuf bytes.Buffer
			multiline := len(val.List) > 1
			exprBuf.WriteByte('[')
			if multiline {
				exprBuf.WriteByte('\n')
			}
			for _, node := range val.List {
				itemSrc, moreDiags := upgradeTraversalExpr(node, filename, an)
				diags = diags.Append(moreDiags)
				exprBuf.Write(itemSrc)
				if multiline {
					exprBuf.WriteString(",\n")
				}
			}
			exprBuf.WriteByte(']')

			printAttribute(buf, item.Keys[0].Token.Value().(string), exprBuf.Bytes(), item.LineComment)

			return diags
		},
	}
}
