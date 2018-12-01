package configupgrade

import (
	"bytes"
	"fmt"

	hcl1ast "github.com/hashicorp/hcl/hcl/ast"
	hcl1printer "github.com/hashicorp/hcl/hcl/printer"
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
		// If the expression is a literal that would be valid as a naked
		// absolute traversal then we'll turn it into one.
		if lit, isLit := val.(*hcl1ast.LiteralType); isLit {
			if lit.Token.Type == hcl1token.STRING {
				trStr := lit.Token.Value().(string)
				trSrc := []byte(trStr)
				_, trDiags := hcl2syntax.ParseTraversalAbs(trSrc, "", hcl2.Pos{})
				if !trDiags.HasErrors() {
					return trSrc, nil
				}
			}
		}

		return upgradeExpr(val, filename, false, an)
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
				blockItem.List.Items, nestedRules, adhocComments,
			)
			diags = diags.Append(bodyDiags)
			buf.WriteString("}\n")
		}

		return diags
	}
}

func nestedBlockRuleWithDynamic(filename string, nestedRules bodyContentRules, an *analysis, adhocComments *commentQueue) bodyItemRule {
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

		// TODO: Implement this
		hcl1printer.Fprint(buf, item)
		buf.WriteByte('\n')
		return nil
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
		ret[name] = nestedBlockRule(filename, nestedRules, an, adhocComments)
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
