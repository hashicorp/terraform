package syntax

import (
	"fmt"
	"reflect"

	hcl1 "github.com/hashicorp/hcl"
	hcl1ast "github.com/hashicorp/hcl/hcl/ast"
	hcl2 "github.com/hashicorp/hcl/v2"
)

// LegacyBody is an implementation of hcl.Body in terms of the HCL 1 AST,
// mimicking HCL 1's decoding rules while working with an HCL 2 body schema.
//
// Unlike a normal HCL body, it does not support arbitrary expressions in
// attribute values. Instead, it treats most attribute values as literal
// constants but selectively processes some specific arguments using
// os.Expand, assuming that the evaluation context contains environment
// variables for substitution.
type LegacyBody struct {
	node     *hcl1ast.ObjectType
	filename string

	// expandAttrs is a set of attribute names that are subject to environment
	// variable expansion using os.Expand.
	expandAttrs map[string]struct{}

	// hidden is a set of attribute names and block type names that have
	// already been processed by an earlier call to PartialContent and are thus
	// ineligible for further processing. This is set only in the body returned
	// in the "remain" return value from Body.PartialContent.
	hidden map[string]struct{}
}

var _ hcl2.Body = (*LegacyBody)(nil)

// JustAttributes implements hcl.Body.JustAttributes by interpreting all of
// the items in the enclosed HCL 1 object as attributes.
func (b *LegacyBody) JustAttributes() (hcl2.Attributes, hcl2.Diagnostics) {
	// The HCL 1 equivalent of JustAttributes was to decode the object into
	// a map[string]interface{}, so we'll do that here to ensure that we
	// get an equivalent result.
	var vals map[string]interface{}
	err := hcl1.DecodeObject(&vals, b.node)
	if err != nil {
		return nil, hcl1ErrorAsDiagnostics(err, hcl1PosDefaultFilename(b.node.Lbrace, b.filename))
	}

	// NOTE: decoding lost details, so this position is inaccurate: it's the
	// containing object rather than the individual item.
	// It's not possible in general to find a single source range for an
	// attribute in HCL 1, because it allows piecemeal definition of an
	// attribute across multiple items that might not even be consecutive in
	// the body.
	pos := hcl1PosDefaultFilename(b.node.Lbrace, b.filename)
	rng := hcl1PosAsHCL2Range(pos)

	ret := make(hcl2.Attributes, len(vals))
	var diags hcl2.Diagnostics
	for k, raw := range vals {
		ret[k] = &hcl2.Attribute{
			Name: k,

			Range:     rng,
			NameRange: rng,
		}
		if _, expand := b.expandAttrs[k]; expand {
			ret[k].Expr = &ExpandExpression{
				raw: raw,
				pos: hcl1PosDefaultFilename(b.node.Lbrace, b.filename),
			}
		} else {
			expr, moreDiags := literalExpr(raw, pos)
			diags = append(diags, moreDiags...)
			ret[k].Expr = expr
		}
	}
	return ret, diags
}

// PartialContent implements hcl.Body.PartialContent against the enclosed
// HCL 1 object.
//
// This implementation preserves the HCL 1 decoding behaviors as closely as
// possible within the HCL 2 API, including ignoring attribute and block type
// names that don't appear in the schema at all.
func (b *LegacyBody) PartialContent(schema *hcl2.BodySchema) (*hcl2.BodyContent, hcl2.Body, hcl2.Diagnostics) {
	newHidden := map[string]struct{}{}
	for k, v := range b.hidden { // propagate anything already hidden
		newHidden[k] = v
	}
	remain := *b // shallow copy
	remain.hidden = newHidden

	var diags hcl2.Diagnostics
	content := &hcl2.BodyContent{
		Attributes: hcl2.Attributes{},
		Blocks:     hcl2.Blocks{},
	}

	pos := hcl1PosDefaultFilename(b.node.Lbrace, b.filename)
	rng := hcl1PosAsHCL2Range(pos)

	for _, attrS := range schema.Attributes {
		newHidden[attrS.Name] = struct{}{}

		// HCL 1 "Filter" looks for all of the nested items whose key sequences
		// start with the given name.
		filter := b.node.List.Filter(attrS.Name)
		var raw interface{} // where we'll put our result; like decoding into a struct field of type interface{}

		// Here we're mimicking what HCL 1's decoder would do when decoding
		// into a struct field: it goes hunting both for direct assignments
		// of the given name and nested blocks whose first key is the given
		// name and then just processes both, letting the cards fall where
		// they may if both are set. The sequence of operations below is the
		// same as in HCL 1 so we can achieve as close a result as possible.
		// See: https://github.com/hashicorp/hcl/blob/914dc3f8dd7c463188c73fc47e9ced82a6e421ca/decoder.go#L700-L726
		prefixMatches := filter.Children()
		matches := filter.Elem()
		if len(matches.Items) == 0 && len(prefixMatches.Items) == 0 {
			continue
		}

		if len(prefixMatches.Items) > 0 {
			if err := hcl1.DecodeObject(raw, prefixMatches); err != nil {
				return content, &remain, hcl1ErrorAsDiagnostics(err, pos)
			}
		}
		for _, match := range matches.Items {
			var decodeNode hcl1ast.Node = match.Val
			if ot, ok := decodeNode.(*hcl1ast.ObjectType); ok {
				decodeNode = &hcl1ast.ObjectList{Items: ot.List.Items}
			}

			if err := hcl1.DecodeObject(raw, decodeNode); err != nil {
				return content, &remain, hcl1ErrorAsDiagnostics(err, pos)
			}
		}

		expr, moreDiags := literalExpr(raw, pos)
		diags = append(diags, moreDiags...)

		attr := &hcl2.Attribute{
			Name:      attrS.Name,
			Expr:      expr,
			Range:     rng,
			NameRange: rng,
		}
		content.Attributes[attrS.Name] = attr
	}

	blockSByType := make(map[string]hcl2.BlockHeaderSchema)
	for _, blockS := range schema.Blocks {
		if len(blockS.LabelNames) > 1 {
			// This is currently not supported because assuming only one
			// level makes this decidedly simpler and the CLI config doesn't
			// currently have any multi-label block types.
			diags = diags.Append(&hcl2.Diagnostic{
				Severity: hcl2.DiagError,
				Summary:  "Unsupported block type in CLI config schema",
				Detail:   fmt.Sprintf("The block type %q is defined with more than one label. The CLI config decoder doesn't currently support that. This is a bug in Terraform.", blockS.Type),
				Subject:  rng.Ptr(),
			})
			continue
		}
		blockSByType[blockS.Type] = blockS
	}

	// We must iterate all the items here, rather than iterating the schema,
	// because we want to preserve the lexical ordering of the nested blocks.
	for _, item := range b.node.List.Items {
		blockS, ok := blockSByType[item.Keys[0].Token.Value().(string)]
		if !ok {
			continue // not a block type
		}
		newHidden[blockS.Type] = struct{}{}

		if len(blockS.LabelNames) > 0 {
			// Deal with JSON key ambiguity: the JSON parser tries to guess
			// what the user might mean but it doesn't have any schema to
			// base that on, so we now need to give it a hint that we're
			// trying to expand nested objects.
			item = expandObject(item)
		}

		labelKeys := item.Keys[1:] // the first key signifies the block type

		if len(labelKeys) != len(blockS.LabelNames) {
			diags = diags.Append(&hcl2.Diagnostic{
				Severity: hcl2.DiagError,
				Summary:  "Wrong number of block labels",
				Detail:   fmt.Sprintf("The block type %q expects %d label(s), but we found %d here.", blockS.Type, len(blockS.LabelNames), len(labelKeys)),
				Subject:  rng.Ptr(),
			})
			continue
		}

		nested, ok := item.Val.(*hcl1ast.ObjectType)
		if !ok {
			diags = diags.Append(&hcl2.Diagnostic{
				Severity: hcl2.DiagError,
				Summary:  "Attribute where block was expected",
				Detail:   fmt.Sprintf("The name %q must be a nested block, not an attribute.", blockS.Type),
				Subject:  rng.Ptr(),
			})
			continue
		}

		var labels []string
		if len(labelKeys) > 0 {
			labels = make([]string, len(labelKeys))
		}
		for i, k := range labelKeys {
			labels[i] = k.Token.Value().(string)
		}

		block := &hcl2.Block{
			Type:   blockS.Type,
			Labels: labels,
			Body: &LegacyBody{
				node:     nested,
				filename: b.filename,
			},

			DefRange:  rng,
			TypeRange: rng,
		}
		content.Blocks = append(content.Blocks, block)
	}

	// By the time we get here we should've assigned values to all of our
	// required attributes.
	for _, attrS := range schema.Attributes {
		if !attrS.Required {
			continue
		}
		if _, exists := content.Attributes[attrS.Name]; !exists {
			diags = diags.Append(&hcl2.Diagnostic{
				Severity: hcl2.DiagError,
				Summary:  "Missing required argument",
				Detail:   fmt.Sprintf("The argument %q is required.", attrS.Name),
				Subject:  rng.Ptr(),
			})
		}
	}

	return content, &remain, diags
}

// MissingItemRange returns the location of the opening brace of the body,
// if any. Otherwise, it returns an invalid range.
func (b *LegacyBody) MissingItemRange() hcl2.Range {
	return hcl1PosAsHCL2Range(hcl1PosDefaultFilename(b.node.Lbrace, b.filename))
}

// Content implements hcl.Body.Content against the enclosed HCL 1 object.
//
// Because this body implementation ignores names that are not specified in
// the schema, Content is really just PartialContent but discarding any
// remaining items in the body.
func (b *LegacyBody) Content(schema *hcl2.BodySchema) (*hcl2.BodyContent, hcl2.Diagnostics) {
	ret, _, diags := b.PartialContent(schema)
	return ret, diags
}

// expandObject detects if an ambiguous JSON object was flattened to a List which
// should be decoded into a struct, and expands the ast to properly decode.
// This is based on the function of the same name in HCL 1.
func expandObject(item *hcl1ast.ObjectItem) *hcl1ast.ObjectItem {
	// A list value will have a key and field name. If it had more fields,
	// it wouldn't have been flattened.
	if len(item.Keys) != 2 {
		return item
	}

	keyToken := item.Keys[0].Token
	item.Keys = item.Keys[1:]

	// we need to un-flatten the ast enough to decode
	newNode := &hcl1ast.ObjectItem{
		Keys: []*hcl1ast.ObjectKey{
			{
				Token: keyToken,
			},
		},
		Val: &hcl1ast.ObjectType{
			List: &hcl1ast.ObjectList{
				Items: []*hcl1ast.ObjectItem{item},
			},
		},
	}

	return newNode
}

// Getting hold of the empty interface type _itself_ requires some indirection,
// because otherwise reflect.TypeOf will try to take the dynamic type of this
// nil value and will panic.
var emptyInterfaceType = reflect.TypeOf((*interface{})(nil)).Elem()
