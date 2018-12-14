package json

import (
	"fmt"

	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hcl/hclsyntax"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"
)

// body is the implementation of "Body" used for files processed with the JSON
// parser.
type body struct {
	val node

	// If non-nil, the keys of this map cause the corresponding attributes to
	// be treated as non-existing. This is used when Body.PartialContent is
	// called, to produce the "remaining content" Body.
	hiddenAttrs map[string]struct{}
}

// expression is the implementation of "Expression" used for files processed
// with the JSON parser.
type expression struct {
	src node
}

func (b *body) Content(schema *hcl.BodySchema) (*hcl.BodyContent, hcl.Diagnostics) {
	content, newBody, diags := b.PartialContent(schema)

	hiddenAttrs := newBody.(*body).hiddenAttrs

	var nameSuggestions []string
	for _, attrS := range schema.Attributes {
		if _, ok := hiddenAttrs[attrS.Name]; !ok {
			// Only suggest an attribute name if we didn't use it already.
			nameSuggestions = append(nameSuggestions, attrS.Name)
		}
	}
	for _, blockS := range schema.Blocks {
		// Blocks can appear multiple times, so we'll suggest their type
		// names regardless of whether they've already been used.
		nameSuggestions = append(nameSuggestions, blockS.Type)
	}

	jsonAttrs, attrDiags := b.collectDeepAttrs(b.val, nil)
	diags = append(diags, attrDiags...)

	for _, attr := range jsonAttrs {
		k := attr.Name
		if k == "//" {
			// Ignore "//" keys in objects representing bodies, to allow
			// their use as comments.
			continue
		}

		if _, ok := hiddenAttrs[k]; !ok {
			suggestion := nameSuggestion(k, nameSuggestions)
			if suggestion != "" {
				suggestion = fmt.Sprintf(" Did you mean %q?", suggestion)
			}

			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Extraneous JSON object property",
				Detail:   fmt.Sprintf("No argument or block type is named %q.%s", k, suggestion),
				Subject:  &attr.NameRange,
				Context:  attr.Range().Ptr(),
			})
		}
	}

	return content, diags
}

func (b *body) PartialContent(schema *hcl.BodySchema) (*hcl.BodyContent, hcl.Body, hcl.Diagnostics) {
	var diags hcl.Diagnostics

	jsonAttrs, attrDiags := b.collectDeepAttrs(b.val, nil)
	diags = append(diags, attrDiags...)

	usedNames := map[string]struct{}{}
	if b.hiddenAttrs != nil {
		for k := range b.hiddenAttrs {
			usedNames[k] = struct{}{}
		}
	}

	content := &hcl.BodyContent{
		Attributes: map[string]*hcl.Attribute{},
		Blocks:     nil,

		MissingItemRange: b.MissingItemRange(),
	}

	// Create some more convenient data structures for our work below.
	attrSchemas := map[string]hcl.AttributeSchema{}
	blockSchemas := map[string]hcl.BlockHeaderSchema{}
	for _, attrS := range schema.Attributes {
		attrSchemas[attrS.Name] = attrS
	}
	for _, blockS := range schema.Blocks {
		blockSchemas[blockS.Type] = blockS
	}

	for _, jsonAttr := range jsonAttrs {
		attrName := jsonAttr.Name
		if _, used := b.hiddenAttrs[attrName]; used {
			continue
		}

		if attrS, defined := attrSchemas[attrName]; defined {
			if existing, exists := content.Attributes[attrName]; exists {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Duplicate argument",
					Detail:   fmt.Sprintf("The argument %q was already set at %s.", attrName, existing.Range),
					Subject:  &jsonAttr.NameRange,
					Context:  jsonAttr.Range().Ptr(),
				})
				continue
			}

			content.Attributes[attrS.Name] = &hcl.Attribute{
				Name:      attrS.Name,
				Expr:      &expression{src: jsonAttr.Value},
				Range:     hcl.RangeBetween(jsonAttr.NameRange, jsonAttr.Value.Range()),
				NameRange: jsonAttr.NameRange,
			}
			usedNames[attrName] = struct{}{}

		} else if blockS, defined := blockSchemas[attrName]; defined {
			bv := jsonAttr.Value
			blockDiags := b.unpackBlock(bv, blockS.Type, &jsonAttr.NameRange, blockS.LabelNames, nil, nil, &content.Blocks)
			diags = append(diags, blockDiags...)
			usedNames[attrName] = struct{}{}
		}

		// We ignore anything that isn't defined because that's the
		// PartialContent contract. The Content method will catch leftovers.
	}

	// Make sure we got all the required attributes.
	for _, attrS := range schema.Attributes {
		if !attrS.Required {
			continue
		}
		if _, defined := content.Attributes[attrS.Name]; !defined {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Missing required argument",
				Detail:   fmt.Sprintf("The argument %q is required, but no definition was found.", attrS.Name),
				Subject:  b.MissingItemRange().Ptr(),
			})
		}
	}

	unusedBody := &body{
		val:         b.val,
		hiddenAttrs: usedNames,
	}

	return content, unusedBody, diags
}

// JustAttributes for JSON bodies interprets all properties of the wrapped
// JSON object as attributes and returns them.
func (b *body) JustAttributes() (hcl.Attributes, hcl.Diagnostics) {
	var diags hcl.Diagnostics
	attrs := make(map[string]*hcl.Attribute)

	obj, ok := b.val.(*objectVal)
	if !ok {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Incorrect JSON value type",
			Detail:   "A JSON object is required here, setting the arguments for this block.",
			Subject:  b.val.StartRange().Ptr(),
		})
		return attrs, diags
	}

	for _, jsonAttr := range obj.Attrs {
		name := jsonAttr.Name
		if name == "//" {
			// Ignore "//" keys in objects representing bodies, to allow
			// their use as comments.
			continue
		}

		if _, hidden := b.hiddenAttrs[name]; hidden {
			continue
		}

		if existing, exists := attrs[name]; exists {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Duplicate attribute definition",
				Detail:   fmt.Sprintf("The argument %q was already set at %s.", name, existing.Range),
				Subject:  &jsonAttr.NameRange,
			})
			continue
		}

		attrs[name] = &hcl.Attribute{
			Name:      name,
			Expr:      &expression{src: jsonAttr.Value},
			Range:     hcl.RangeBetween(jsonAttr.NameRange, jsonAttr.Value.Range()),
			NameRange: jsonAttr.NameRange,
		}
	}

	// No diagnostics possible here, since the parser already took care of
	// finding duplicates and every JSON value can be a valid attribute value.
	return attrs, diags
}

func (b *body) MissingItemRange() hcl.Range {
	switch tv := b.val.(type) {
	case *objectVal:
		return tv.CloseRange
	case *arrayVal:
		return tv.OpenRange
	default:
		// Should not happen in correct operation, but might show up if the
		// input is invalid and we are producing partial results.
		return tv.StartRange()
	}
}

func (b *body) unpackBlock(v node, typeName string, typeRange *hcl.Range, labelsLeft []string, labelsUsed []string, labelRanges []hcl.Range, blocks *hcl.Blocks) (diags hcl.Diagnostics) {
	if len(labelsLeft) > 0 {
		labelName := labelsLeft[0]
		jsonAttrs, attrDiags := b.collectDeepAttrs(v, &labelName)
		diags = append(diags, attrDiags...)

		if len(jsonAttrs) == 0 {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Missing block label",
				Detail:   fmt.Sprintf("At least one object property is required, whose name represents the %s block's %s.", typeName, labelName),
				Subject:  v.StartRange().Ptr(),
			})
			return
		}
		labelsUsed := append(labelsUsed, "")
		labelRanges := append(labelRanges, hcl.Range{})
		for _, p := range jsonAttrs {
			pk := p.Name
			labelsUsed[len(labelsUsed)-1] = pk
			labelRanges[len(labelRanges)-1] = p.NameRange
			diags = append(diags, b.unpackBlock(p.Value, typeName, typeRange, labelsLeft[1:], labelsUsed, labelRanges, blocks)...)
		}
		return
	}

	// By the time we get here, we've peeled off all the labels and we're ready
	// to deal with the block's actual content.

	// need to copy the label slices because their underlying arrays will
	// continue to be mutated after we return.
	labels := make([]string, len(labelsUsed))
	copy(labels, labelsUsed)
	labelR := make([]hcl.Range, len(labelRanges))
	copy(labelR, labelRanges)

	switch tv := v.(type) {
	case *objectVal:
		// Single instance of the block
		*blocks = append(*blocks, &hcl.Block{
			Type:   typeName,
			Labels: labels,
			Body: &body{
				val: tv,
			},

			DefRange:    tv.OpenRange,
			TypeRange:   *typeRange,
			LabelRanges: labelR,
		})
	case *arrayVal:
		// Multiple instances of the block
		for _, av := range tv.Values {
			*blocks = append(*blocks, &hcl.Block{
				Type:   typeName,
				Labels: labels,
				Body: &body{
					val: av, // might be mistyped; we'll find out when content is requested for this body
				},

				DefRange:    tv.OpenRange,
				TypeRange:   *typeRange,
				LabelRanges: labelR,
			})
		}
	default:
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Incorrect JSON value type",
			Detail:   fmt.Sprintf("Either a JSON object or a JSON array is required, representing the contents of one or more %q blocks.", typeName),
			Subject:  v.StartRange().Ptr(),
		})
	}
	return
}

// collectDeepAttrs takes either a single object or an array of objects and
// flattens it into a list of object attributes, collecting attributes from
// all of the objects in a given array.
//
// Ordering is preserved, so a list of objects that each have one property
// will result in those properties being returned in the same order as the
// objects appeared in the array.
//
// This is appropriate for use only for objects representing bodies or labels
// within a block.
//
// The labelName argument, if non-null, is used to tailor returned error
// messages to refer to block labels rather than attributes and child blocks.
// It has no other effect.
func (b *body) collectDeepAttrs(v node, labelName *string) ([]*objectAttr, hcl.Diagnostics) {
	var diags hcl.Diagnostics
	var attrs []*objectAttr

	switch tv := v.(type) {

	case *objectVal:
		attrs = append(attrs, tv.Attrs...)

	case *arrayVal:
		for _, ev := range tv.Values {
			switch tev := ev.(type) {
			case *objectVal:
				attrs = append(attrs, tev.Attrs...)
			default:
				if labelName != nil {
					diags = append(diags, &hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Incorrect JSON value type",
						Detail:   fmt.Sprintf("A JSON object is required here, to specify %s labels for this block.", *labelName),
						Subject:  ev.StartRange().Ptr(),
					})
				} else {
					diags = append(diags, &hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Incorrect JSON value type",
						Detail:   "A JSON object is required here, to define arguments and child blocks.",
						Subject:  ev.StartRange().Ptr(),
					})
				}
			}
		}

	default:
		if labelName != nil {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Incorrect JSON value type",
				Detail:   fmt.Sprintf("Either a JSON object or JSON array of objects is required here, to specify %s labels for this block.", *labelName),
				Subject:  v.StartRange().Ptr(),
			})
		} else {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Incorrect JSON value type",
				Detail:   "Either a JSON object or JSON array of objects is required here, to define arguments and child blocks.",
				Subject:  v.StartRange().Ptr(),
			})
		}
	}

	return attrs, diags
}

func (e *expression) Value(ctx *hcl.EvalContext) (cty.Value, hcl.Diagnostics) {
	switch v := e.src.(type) {
	case *stringVal:
		if ctx != nil {
			// Parse string contents as a HCL native language expression.
			// We only do this if we have a context, so passing a nil context
			// is how the caller specifies that interpolations are not allowed
			// and that the string should just be returned verbatim.
			templateSrc := v.Value
			expr, diags := hclsyntax.ParseTemplate(
				[]byte(templateSrc),
				v.SrcRange.Filename,

				// This won't produce _exactly_ the right result, since
				// the hclsyntax parser can't "see" any escapes we removed
				// while parsing JSON, but it's better than nothing.
				hcl.Pos{
					Line: v.SrcRange.Start.Line,

					// skip over the opening quote mark
					Byte:   v.SrcRange.Start.Byte + 1,
					Column: v.SrcRange.Start.Column + 1,
				},
			)
			if diags.HasErrors() {
				return cty.DynamicVal, diags
			}
			val, evalDiags := expr.Value(ctx)
			diags = append(diags, evalDiags...)
			return val, diags
		}

		return cty.StringVal(v.Value), nil
	case *numberVal:
		return cty.NumberVal(v.Value), nil
	case *booleanVal:
		return cty.BoolVal(v.Value), nil
	case *arrayVal:
		vals := []cty.Value{}
		for _, jsonVal := range v.Values {
			val, _ := (&expression{src: jsonVal}).Value(ctx)
			vals = append(vals, val)
		}
		return cty.TupleVal(vals), nil
	case *objectVal:
		var diags hcl.Diagnostics
		attrs := map[string]cty.Value{}
		attrRanges := map[string]hcl.Range{}
		known := true
		for _, jsonAttr := range v.Attrs {
			// In this one context we allow keys to contain interpolation
			// expressions too, assuming we're evaluating in interpolation
			// mode. This achieves parity with the native syntax where
			// object expressions can have dynamic keys, while block contents
			// may not.
			name, nameDiags := (&expression{src: &stringVal{
				Value:    jsonAttr.Name,
				SrcRange: jsonAttr.NameRange,
			}}).Value(ctx)
			valExpr := &expression{src: jsonAttr.Value}
			val, valDiags := valExpr.Value(ctx)
			diags = append(diags, nameDiags...)
			diags = append(diags, valDiags...)

			var err error
			name, err = convert.Convert(name, cty.String)
			if err != nil {
				diags = append(diags, &hcl.Diagnostic{
					Severity:    hcl.DiagError,
					Summary:     "Invalid object key expression",
					Detail:      fmt.Sprintf("Cannot use this expression as an object key: %s.", err),
					Subject:     &jsonAttr.NameRange,
					Expression:  valExpr,
					EvalContext: ctx,
				})
				continue
			}
			if name.IsNull() {
				diags = append(diags, &hcl.Diagnostic{
					Severity:    hcl.DiagError,
					Summary:     "Invalid object key expression",
					Detail:      "Cannot use null value as an object key.",
					Subject:     &jsonAttr.NameRange,
					Expression:  valExpr,
					EvalContext: ctx,
				})
				continue
			}
			if !name.IsKnown() {
				// This is a bit of a weird case, since our usual rules require
				// us to tolerate unknowns and just represent the result as
				// best we can but if we don't know the key then we can't
				// know the type of our object at all, and thus we must turn
				// the whole thing into cty.DynamicVal. This is consistent with
				// how this situation is handled in the native syntax.
				// We'll keep iterating so we can collect other errors in
				// subsequent attributes.
				known = false
				continue
			}
			nameStr := name.AsString()
			if _, defined := attrs[nameStr]; defined {
				diags = append(diags, &hcl.Diagnostic{
					Severity:    hcl.DiagError,
					Summary:     "Duplicate object attribute",
					Detail:      fmt.Sprintf("An attribute named %q was already defined at %s.", nameStr, attrRanges[nameStr]),
					Subject:     &jsonAttr.NameRange,
					Expression:  e,
					EvalContext: ctx,
				})
				continue
			}
			attrs[nameStr] = val
			attrRanges[nameStr] = jsonAttr.NameRange
		}
		if !known {
			// We encountered an unknown key somewhere along the way, so
			// we can't know what our type will eventually be.
			return cty.DynamicVal, diags
		}
		return cty.ObjectVal(attrs), diags
	default:
		// Default to DynamicVal so that ASTs containing invalid nodes can
		// still be partially-evaluated.
		return cty.DynamicVal, nil
	}
}

func (e *expression) Variables() []hcl.Traversal {
	var vars []hcl.Traversal

	switch v := e.src.(type) {
	case *stringVal:
		templateSrc := v.Value
		expr, diags := hclsyntax.ParseTemplate(
			[]byte(templateSrc),
			v.SrcRange.Filename,

			// This won't produce _exactly_ the right result, since
			// the hclsyntax parser can't "see" any escapes we removed
			// while parsing JSON, but it's better than nothing.
			hcl.Pos{
				Line: v.SrcRange.Start.Line,

				// skip over the opening quote mark
				Byte:   v.SrcRange.Start.Byte + 1,
				Column: v.SrcRange.Start.Column + 1,
			},
		)
		if diags.HasErrors() {
			return vars
		}
		return expr.Variables()

	case *arrayVal:
		for _, jsonVal := range v.Values {
			vars = append(vars, (&expression{src: jsonVal}).Variables()...)
		}
	case *objectVal:
		for _, jsonAttr := range v.Attrs {
			keyExpr := &stringVal{ // we're going to treat key as an expression in this context
				Value:    jsonAttr.Name,
				SrcRange: jsonAttr.NameRange,
			}
			vars = append(vars, (&expression{src: keyExpr}).Variables()...)
			vars = append(vars, (&expression{src: jsonAttr.Value}).Variables()...)
		}
	}

	return vars
}

func (e *expression) Range() hcl.Range {
	return e.src.Range()
}

func (e *expression) StartRange() hcl.Range {
	return e.src.StartRange()
}

// Implementation for hcl.AbsTraversalForExpr.
func (e *expression) AsTraversal() hcl.Traversal {
	// In JSON-based syntax a traversal is given as a string containing
	// traversal syntax as defined by hclsyntax.ParseTraversalAbs.

	switch v := e.src.(type) {
	case *stringVal:
		traversal, diags := hclsyntax.ParseTraversalAbs([]byte(v.Value), v.SrcRange.Filename, v.SrcRange.Start)
		if diags.HasErrors() {
			return nil
		}
		return traversal
	default:
		return nil
	}
}

// Implementation for hcl.ExprCall.
func (e *expression) ExprCall() *hcl.StaticCall {
	// In JSON-based syntax a static call is given as a string containing
	// an expression in the native syntax that also supports ExprCall.

	switch v := e.src.(type) {
	case *stringVal:
		expr, diags := hclsyntax.ParseExpression([]byte(v.Value), v.SrcRange.Filename, v.SrcRange.Start)
		if diags.HasErrors() {
			return nil
		}

		call, diags := hcl.ExprCall(expr)
		if diags.HasErrors() {
			return nil
		}

		return call
	default:
		return nil
	}
}

// Implementation for hcl.ExprList.
func (e *expression) ExprList() []hcl.Expression {
	switch v := e.src.(type) {
	case *arrayVal:
		ret := make([]hcl.Expression, len(v.Values))
		for i, node := range v.Values {
			ret[i] = &expression{src: node}
		}
		return ret
	default:
		return nil
	}
}

// Implementation for hcl.ExprMap.
func (e *expression) ExprMap() []hcl.KeyValuePair {
	switch v := e.src.(type) {
	case *objectVal:
		ret := make([]hcl.KeyValuePair, len(v.Attrs))
		for i, jsonAttr := range v.Attrs {
			ret[i] = hcl.KeyValuePair{
				Key: &expression{src: &stringVal{
					Value:    jsonAttr.Name,
					SrcRange: jsonAttr.NameRange,
				}},
				Value: &expression{src: jsonAttr.Value},
			}
		}
		return ret
	default:
		return nil
	}
}
