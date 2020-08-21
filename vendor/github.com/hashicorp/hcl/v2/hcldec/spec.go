package hcldec

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/ext/customdecode"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"
	"github.com/zclconf/go-cty/cty/function"
)

// A Spec is a description of how to decode a hcl.Body to a cty.Value.
//
// The various other types in this package whose names end in "Spec" are
// the spec implementations. The most common top-level spec is ObjectSpec,
// which decodes body content into a cty.Value of an object type.
type Spec interface {
	// Perform the decode operation on the given body, in the context of
	// the given block (which might be null), using the given eval context.
	//
	// "block" is provided only by the nested calls performed by the spec
	// types that work on block bodies.
	decode(content *hcl.BodyContent, blockLabels []blockLabel, ctx *hcl.EvalContext) (cty.Value, hcl.Diagnostics)

	// Return the cty.Type that should be returned when decoding a body with
	// this spec.
	impliedType() cty.Type

	// Call the given callback once for each of the nested specs that would
	// get decoded with the same body and block as the receiver. This should
	// not descend into the nested specs used when decoding blocks.
	visitSameBodyChildren(cb visitFunc)

	// Determine the source range of the value that would be returned for the
	// spec in the given content, in the context of the given block
	// (which might be null). If the corresponding item is missing, return
	// a place where it might be inserted.
	sourceRange(content *hcl.BodyContent, blockLabels []blockLabel) hcl.Range
}

type visitFunc func(spec Spec)

// An ObjectSpec is a Spec that produces a cty.Value of an object type whose
// attributes correspond to the keys of the spec map.
type ObjectSpec map[string]Spec

// attrSpec is implemented by specs that require attributes from the body.
type attrSpec interface {
	attrSchemata() []hcl.AttributeSchema
}

// blockSpec is implemented by specs that require blocks from the body.
type blockSpec interface {
	blockHeaderSchemata() []hcl.BlockHeaderSchema
	nestedSpec() Spec
}

// specNeedingVariables is implemented by specs that can use variables
// from the EvalContext, to declare which variables they need.
type specNeedingVariables interface {
	variablesNeeded(content *hcl.BodyContent) []hcl.Traversal
}

func (s ObjectSpec) visitSameBodyChildren(cb visitFunc) {
	for _, c := range s {
		cb(c)
	}
}

func (s ObjectSpec) decode(content *hcl.BodyContent, blockLabels []blockLabel, ctx *hcl.EvalContext) (cty.Value, hcl.Diagnostics) {
	vals := make(map[string]cty.Value, len(s))
	var diags hcl.Diagnostics

	for k, spec := range s {
		var kd hcl.Diagnostics
		vals[k], kd = spec.decode(content, blockLabels, ctx)
		diags = append(diags, kd...)
	}

	return cty.ObjectVal(vals), diags
}

func (s ObjectSpec) impliedType() cty.Type {
	if len(s) == 0 {
		return cty.EmptyObject
	}

	attrTypes := make(map[string]cty.Type)
	for k, childSpec := range s {
		attrTypes[k] = childSpec.impliedType()
	}
	return cty.Object(attrTypes)
}

func (s ObjectSpec) sourceRange(content *hcl.BodyContent, blockLabels []blockLabel) hcl.Range {
	// This is not great, but the best we can do. In practice, it's rather
	// strange to ask for the source range of an entire top-level body, since
	// that's already readily available to the caller.
	return content.MissingItemRange
}

// A TupleSpec is a Spec that produces a cty.Value of a tuple type whose
// elements correspond to the elements of the spec slice.
type TupleSpec []Spec

func (s TupleSpec) visitSameBodyChildren(cb visitFunc) {
	for _, c := range s {
		cb(c)
	}
}

func (s TupleSpec) decode(content *hcl.BodyContent, blockLabels []blockLabel, ctx *hcl.EvalContext) (cty.Value, hcl.Diagnostics) {
	vals := make([]cty.Value, len(s))
	var diags hcl.Diagnostics

	for i, spec := range s {
		var ed hcl.Diagnostics
		vals[i], ed = spec.decode(content, blockLabels, ctx)
		diags = append(diags, ed...)
	}

	return cty.TupleVal(vals), diags
}

func (s TupleSpec) impliedType() cty.Type {
	if len(s) == 0 {
		return cty.EmptyTuple
	}

	attrTypes := make([]cty.Type, len(s))
	for i, childSpec := range s {
		attrTypes[i] = childSpec.impliedType()
	}
	return cty.Tuple(attrTypes)
}

func (s TupleSpec) sourceRange(content *hcl.BodyContent, blockLabels []blockLabel) hcl.Range {
	// This is not great, but the best we can do. In practice, it's rather
	// strange to ask for the source range of an entire top-level body, since
	// that's already readily available to the caller.
	return content.MissingItemRange
}

// An AttrSpec is a Spec that evaluates a particular attribute expression in
// the body and returns its resulting value converted to the requested type,
// or produces a diagnostic if the type is incorrect.
type AttrSpec struct {
	Name     string
	Type     cty.Type
	Required bool
}

func (s *AttrSpec) visitSameBodyChildren(cb visitFunc) {
	// leaf node
}

// specNeedingVariables implementation
func (s *AttrSpec) variablesNeeded(content *hcl.BodyContent) []hcl.Traversal {
	attr, exists := content.Attributes[s.Name]
	if !exists {
		return nil
	}

	return attr.Expr.Variables()
}

// attrSpec implementation
func (s *AttrSpec) attrSchemata() []hcl.AttributeSchema {
	return []hcl.AttributeSchema{
		{
			Name:     s.Name,
			Required: s.Required,
		},
	}
}

func (s *AttrSpec) sourceRange(content *hcl.BodyContent, blockLabels []blockLabel) hcl.Range {
	attr, exists := content.Attributes[s.Name]
	if !exists {
		return content.MissingItemRange
	}

	return attr.Expr.Range()
}

func (s *AttrSpec) decode(content *hcl.BodyContent, blockLabels []blockLabel, ctx *hcl.EvalContext) (cty.Value, hcl.Diagnostics) {
	attr, exists := content.Attributes[s.Name]
	if !exists {
		// We don't need to check required and emit a diagnostic here, because
		// that would already have happened when building "content".
		return cty.NullVal(s.Type), nil
	}

	if decodeFn := customdecode.CustomExpressionDecoderForType(s.Type); decodeFn != nil {
		v, diags := decodeFn(attr.Expr, ctx)
		if v == cty.NilVal {
			v = cty.UnknownVal(s.Type)
		}
		return v, diags
	}

	val, diags := attr.Expr.Value(ctx)

	convVal, err := convert.Convert(val, s.Type)
	if err != nil {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Incorrect attribute value type",
			Detail: fmt.Sprintf(
				"Inappropriate value for attribute %q: %s.",
				s.Name, err.Error(),
			),
			Subject:     attr.Expr.Range().Ptr(),
			Context:     hcl.RangeBetween(attr.NameRange, attr.Expr.Range()).Ptr(),
			Expression:  attr.Expr,
			EvalContext: ctx,
		})
		// We'll return an unknown value of the _correct_ type so that the
		// incomplete result can still be used for some analysis use-cases.
		val = cty.UnknownVal(s.Type)
	} else {
		val = convVal
	}

	return val, diags
}

func (s *AttrSpec) impliedType() cty.Type {
	return s.Type
}

// A LiteralSpec is a Spec that produces the given literal value, ignoring
// the given body.
type LiteralSpec struct {
	Value cty.Value
}

func (s *LiteralSpec) visitSameBodyChildren(cb visitFunc) {
	// leaf node
}

func (s *LiteralSpec) decode(content *hcl.BodyContent, blockLabels []blockLabel, ctx *hcl.EvalContext) (cty.Value, hcl.Diagnostics) {
	return s.Value, nil
}

func (s *LiteralSpec) impliedType() cty.Type {
	return s.Value.Type()
}

func (s *LiteralSpec) sourceRange(content *hcl.BodyContent, blockLabels []blockLabel) hcl.Range {
	// No sensible range to return for a literal, so the caller had better
	// ensure it doesn't cause any diagnostics.
	return hcl.Range{
		Filename: "<unknown>",
	}
}

// An ExprSpec is a Spec that evaluates the given expression, ignoring the
// given body.
type ExprSpec struct {
	Expr hcl.Expression
}

func (s *ExprSpec) visitSameBodyChildren(cb visitFunc) {
	// leaf node
}

// specNeedingVariables implementation
func (s *ExprSpec) variablesNeeded(content *hcl.BodyContent) []hcl.Traversal {
	return s.Expr.Variables()
}

func (s *ExprSpec) decode(content *hcl.BodyContent, blockLabels []blockLabel, ctx *hcl.EvalContext) (cty.Value, hcl.Diagnostics) {
	return s.Expr.Value(ctx)
}

func (s *ExprSpec) impliedType() cty.Type {
	// We can't know the type of our expression until we evaluate it
	return cty.DynamicPseudoType
}

func (s *ExprSpec) sourceRange(content *hcl.BodyContent, blockLabels []blockLabel) hcl.Range {
	return s.Expr.Range()
}

// A BlockSpec is a Spec that produces a cty.Value by decoding the contents
// of a single nested block of a given type, using a nested spec.
//
// If the Required flag is not set, the nested block may be omitted, in which
// case a null value is produced. If it _is_ set, an error diagnostic is
// produced if there are no nested blocks of the given type.
type BlockSpec struct {
	TypeName string
	Nested   Spec
	Required bool
}

func (s *BlockSpec) visitSameBodyChildren(cb visitFunc) {
	// leaf node ("Nested" does not use the same body)
}

// blockSpec implementation
func (s *BlockSpec) blockHeaderSchemata() []hcl.BlockHeaderSchema {
	return []hcl.BlockHeaderSchema{
		{
			Type:       s.TypeName,
			LabelNames: findLabelSpecs(s.Nested),
		},
	}
}

// blockSpec implementation
func (s *BlockSpec) nestedSpec() Spec {
	return s.Nested
}

// specNeedingVariables implementation
func (s *BlockSpec) variablesNeeded(content *hcl.BodyContent) []hcl.Traversal {
	var childBlock *hcl.Block
	for _, candidate := range content.Blocks {
		if candidate.Type != s.TypeName {
			continue
		}

		childBlock = candidate
		break
	}

	if childBlock == nil {
		return nil
	}

	return Variables(childBlock.Body, s.Nested)
}

func (s *BlockSpec) decode(content *hcl.BodyContent, blockLabels []blockLabel, ctx *hcl.EvalContext) (cty.Value, hcl.Diagnostics) {
	var diags hcl.Diagnostics

	var childBlock *hcl.Block
	for _, candidate := range content.Blocks {
		if candidate.Type != s.TypeName {
			continue
		}

		if childBlock != nil {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  fmt.Sprintf("Duplicate %s block", s.TypeName),
				Detail: fmt.Sprintf(
					"Only one block of type %q is allowed. Previous definition was at %s.",
					s.TypeName, childBlock.DefRange.String(),
				),
				Subject: &candidate.DefRange,
			})
			break
		}

		childBlock = candidate
	}

	if childBlock == nil {
		if s.Required {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  fmt.Sprintf("Missing %s block", s.TypeName),
				Detail: fmt.Sprintf(
					"A block of type %q is required here.", s.TypeName,
				),
				Subject: &content.MissingItemRange,
			})
		}
		return cty.NullVal(s.Nested.impliedType()), diags
	}

	if s.Nested == nil {
		panic("BlockSpec with no Nested Spec")
	}
	val, _, childDiags := decode(childBlock.Body, labelsForBlock(childBlock), ctx, s.Nested, false)
	diags = append(diags, childDiags...)
	return val, diags
}

func (s *BlockSpec) impliedType() cty.Type {
	return s.Nested.impliedType()
}

func (s *BlockSpec) sourceRange(content *hcl.BodyContent, blockLabels []blockLabel) hcl.Range {
	var childBlock *hcl.Block
	for _, candidate := range content.Blocks {
		if candidate.Type != s.TypeName {
			continue
		}

		childBlock = candidate
		break
	}

	if childBlock == nil {
		return content.MissingItemRange
	}

	return sourceRange(childBlock.Body, labelsForBlock(childBlock), s.Nested)
}

// A BlockListSpec is a Spec that produces a cty list of the results of
// decoding all of the nested blocks of a given type, using a nested spec.
type BlockListSpec struct {
	TypeName string
	Nested   Spec
	MinItems int
	MaxItems int
}

func (s *BlockListSpec) visitSameBodyChildren(cb visitFunc) {
	// leaf node ("Nested" does not use the same body)
}

// blockSpec implementation
func (s *BlockListSpec) blockHeaderSchemata() []hcl.BlockHeaderSchema {
	return []hcl.BlockHeaderSchema{
		{
			Type:       s.TypeName,
			LabelNames: findLabelSpecs(s.Nested),
		},
	}
}

// blockSpec implementation
func (s *BlockListSpec) nestedSpec() Spec {
	return s.Nested
}

// specNeedingVariables implementation
func (s *BlockListSpec) variablesNeeded(content *hcl.BodyContent) []hcl.Traversal {
	var ret []hcl.Traversal

	for _, childBlock := range content.Blocks {
		if childBlock.Type != s.TypeName {
			continue
		}

		ret = append(ret, Variables(childBlock.Body, s.Nested)...)
	}

	return ret
}

func (s *BlockListSpec) decode(content *hcl.BodyContent, blockLabels []blockLabel, ctx *hcl.EvalContext) (cty.Value, hcl.Diagnostics) {
	var diags hcl.Diagnostics

	if s.Nested == nil {
		panic("BlockListSpec with no Nested Spec")
	}

	var elems []cty.Value
	var sourceRanges []hcl.Range
	for _, childBlock := range content.Blocks {
		if childBlock.Type != s.TypeName {
			continue
		}

		val, _, childDiags := decode(childBlock.Body, labelsForBlock(childBlock), ctx, s.Nested, false)
		diags = append(diags, childDiags...)
		elems = append(elems, val)
		sourceRanges = append(sourceRanges, sourceRange(childBlock.Body, labelsForBlock(childBlock), s.Nested))
	}

	if len(elems) < s.MinItems {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  fmt.Sprintf("Insufficient %s blocks", s.TypeName),
			Detail:   fmt.Sprintf("At least %d %q blocks are required.", s.MinItems, s.TypeName),
			Subject:  &content.MissingItemRange,
		})
	} else if s.MaxItems > 0 && len(elems) > s.MaxItems {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  fmt.Sprintf("Too many %s blocks", s.TypeName),
			Detail:   fmt.Sprintf("No more than %d %q blocks are allowed", s.MaxItems, s.TypeName),
			Subject:  &sourceRanges[s.MaxItems],
		})
	}

	var ret cty.Value

	if len(elems) == 0 {
		ret = cty.ListValEmpty(s.Nested.impliedType())
	} else {
		// Since our target is a list, all of the decoded elements must have the
		// same type or cty.ListVal will panic below. Different types can arise
		// if there is an attribute spec of type cty.DynamicPseudoType in the
		// nested spec; all given values must be convertable to a single type
		// in order for the result to be considered valid.
		etys := make([]cty.Type, len(elems))
		for i, v := range elems {
			etys[i] = v.Type()
		}
		ety, convs := convert.UnifyUnsafe(etys)
		if ety == cty.NilType {
			// FIXME: This is a pretty terrible error message.
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  fmt.Sprintf("Unconsistent argument types in %s blocks", s.TypeName),
				Detail:   "Corresponding attributes in all blocks of this type must be the same.",
				Subject:  &sourceRanges[0],
			})
			return cty.DynamicVal, diags
		}
		for i, v := range elems {
			if convs[i] != nil {
				newV, err := convs[i](v)
				if err != nil {
					// FIXME: This is a pretty terrible error message.
					diags = append(diags, &hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  fmt.Sprintf("Unconsistent argument types in %s blocks", s.TypeName),
						Detail:   fmt.Sprintf("Block with index %d has inconsistent argument types: %s.", i, err),
						Subject:  &sourceRanges[i],
					})
					// Bail early here so we won't panic below in cty.ListVal
					return cty.DynamicVal, diags
				}
				elems[i] = newV
			}
		}

		ret = cty.ListVal(elems)
	}

	return ret, diags
}

func (s *BlockListSpec) impliedType() cty.Type {
	return cty.List(s.Nested.impliedType())
}

func (s *BlockListSpec) sourceRange(content *hcl.BodyContent, blockLabels []blockLabel) hcl.Range {
	// We return the source range of the _first_ block of the given type,
	// since they are not guaranteed to form a contiguous range.

	var childBlock *hcl.Block
	for _, candidate := range content.Blocks {
		if candidate.Type != s.TypeName {
			continue
		}

		childBlock = candidate
		break
	}

	if childBlock == nil {
		return content.MissingItemRange
	}

	return sourceRange(childBlock.Body, labelsForBlock(childBlock), s.Nested)
}

// A BlockTupleSpec is a Spec that produces a cty tuple of the results of
// decoding all of the nested blocks of a given type, using a nested spec.
//
// This is similar to BlockListSpec, but it permits the nested blocks to have
// different result types in situations where cty.DynamicPseudoType attributes
// are present.
type BlockTupleSpec struct {
	TypeName string
	Nested   Spec
	MinItems int
	MaxItems int
}

func (s *BlockTupleSpec) visitSameBodyChildren(cb visitFunc) {
	// leaf node ("Nested" does not use the same body)
}

// blockSpec implementation
func (s *BlockTupleSpec) blockHeaderSchemata() []hcl.BlockHeaderSchema {
	return []hcl.BlockHeaderSchema{
		{
			Type:       s.TypeName,
			LabelNames: findLabelSpecs(s.Nested),
		},
	}
}

// blockSpec implementation
func (s *BlockTupleSpec) nestedSpec() Spec {
	return s.Nested
}

// specNeedingVariables implementation
func (s *BlockTupleSpec) variablesNeeded(content *hcl.BodyContent) []hcl.Traversal {
	var ret []hcl.Traversal

	for _, childBlock := range content.Blocks {
		if childBlock.Type != s.TypeName {
			continue
		}

		ret = append(ret, Variables(childBlock.Body, s.Nested)...)
	}

	return ret
}

func (s *BlockTupleSpec) decode(content *hcl.BodyContent, blockLabels []blockLabel, ctx *hcl.EvalContext) (cty.Value, hcl.Diagnostics) {
	var diags hcl.Diagnostics

	if s.Nested == nil {
		panic("BlockListSpec with no Nested Spec")
	}

	var elems []cty.Value
	var sourceRanges []hcl.Range
	for _, childBlock := range content.Blocks {
		if childBlock.Type != s.TypeName {
			continue
		}

		val, _, childDiags := decode(childBlock.Body, labelsForBlock(childBlock), ctx, s.Nested, false)
		diags = append(diags, childDiags...)
		elems = append(elems, val)
		sourceRanges = append(sourceRanges, sourceRange(childBlock.Body, labelsForBlock(childBlock), s.Nested))
	}

	if len(elems) < s.MinItems {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  fmt.Sprintf("Insufficient %s blocks", s.TypeName),
			Detail:   fmt.Sprintf("At least %d %q blocks are required.", s.MinItems, s.TypeName),
			Subject:  &content.MissingItemRange,
		})
	} else if s.MaxItems > 0 && len(elems) > s.MaxItems {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  fmt.Sprintf("Too many %s blocks", s.TypeName),
			Detail:   fmt.Sprintf("No more than %d %q blocks are allowed", s.MaxItems, s.TypeName),
			Subject:  &sourceRanges[s.MaxItems],
		})
	}

	var ret cty.Value

	if len(elems) == 0 {
		ret = cty.EmptyTupleVal
	} else {
		ret = cty.TupleVal(elems)
	}

	return ret, diags
}

func (s *BlockTupleSpec) impliedType() cty.Type {
	// We can't predict our type, because we don't know how many blocks
	// there will be until we decode.
	return cty.DynamicPseudoType
}

func (s *BlockTupleSpec) sourceRange(content *hcl.BodyContent, blockLabels []blockLabel) hcl.Range {
	// We return the source range of the _first_ block of the given type,
	// since they are not guaranteed to form a contiguous range.

	var childBlock *hcl.Block
	for _, candidate := range content.Blocks {
		if candidate.Type != s.TypeName {
			continue
		}

		childBlock = candidate
		break
	}

	if childBlock == nil {
		return content.MissingItemRange
	}

	return sourceRange(childBlock.Body, labelsForBlock(childBlock), s.Nested)
}

// A BlockSetSpec is a Spec that produces a cty set of the results of
// decoding all of the nested blocks of a given type, using a nested spec.
type BlockSetSpec struct {
	TypeName string
	Nested   Spec
	MinItems int
	MaxItems int
}

func (s *BlockSetSpec) visitSameBodyChildren(cb visitFunc) {
	// leaf node ("Nested" does not use the same body)
}

// blockSpec implementation
func (s *BlockSetSpec) blockHeaderSchemata() []hcl.BlockHeaderSchema {
	return []hcl.BlockHeaderSchema{
		{
			Type:       s.TypeName,
			LabelNames: findLabelSpecs(s.Nested),
		},
	}
}

// blockSpec implementation
func (s *BlockSetSpec) nestedSpec() Spec {
	return s.Nested
}

// specNeedingVariables implementation
func (s *BlockSetSpec) variablesNeeded(content *hcl.BodyContent) []hcl.Traversal {
	var ret []hcl.Traversal

	for _, childBlock := range content.Blocks {
		if childBlock.Type != s.TypeName {
			continue
		}

		ret = append(ret, Variables(childBlock.Body, s.Nested)...)
	}

	return ret
}

func (s *BlockSetSpec) decode(content *hcl.BodyContent, blockLabels []blockLabel, ctx *hcl.EvalContext) (cty.Value, hcl.Diagnostics) {
	var diags hcl.Diagnostics

	if s.Nested == nil {
		panic("BlockSetSpec with no Nested Spec")
	}

	var elems []cty.Value
	var sourceRanges []hcl.Range
	for _, childBlock := range content.Blocks {
		if childBlock.Type != s.TypeName {
			continue
		}

		val, _, childDiags := decode(childBlock.Body, labelsForBlock(childBlock), ctx, s.Nested, false)
		diags = append(diags, childDiags...)
		elems = append(elems, val)
		sourceRanges = append(sourceRanges, sourceRange(childBlock.Body, labelsForBlock(childBlock), s.Nested))
	}

	if len(elems) < s.MinItems {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  fmt.Sprintf("Insufficient %s blocks", s.TypeName),
			Detail:   fmt.Sprintf("At least %d %q blocks are required.", s.MinItems, s.TypeName),
			Subject:  &content.MissingItemRange,
		})
	} else if s.MaxItems > 0 && len(elems) > s.MaxItems {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  fmt.Sprintf("Too many %s blocks", s.TypeName),
			Detail:   fmt.Sprintf("No more than %d %q blocks are allowed", s.MaxItems, s.TypeName),
			Subject:  &sourceRanges[s.MaxItems],
		})
	}

	var ret cty.Value

	if len(elems) == 0 {
		ret = cty.SetValEmpty(s.Nested.impliedType())
	} else {
		// Since our target is a set, all of the decoded elements must have the
		// same type or cty.SetVal will panic below. Different types can arise
		// if there is an attribute spec of type cty.DynamicPseudoType in the
		// nested spec; all given values must be convertable to a single type
		// in order for the result to be considered valid.
		etys := make([]cty.Type, len(elems))
		for i, v := range elems {
			etys[i] = v.Type()
		}
		ety, convs := convert.UnifyUnsafe(etys)
		if ety == cty.NilType {
			// FIXME: This is a pretty terrible error message.
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  fmt.Sprintf("Unconsistent argument types in %s blocks", s.TypeName),
				Detail:   "Corresponding attributes in all blocks of this type must be the same.",
				Subject:  &sourceRanges[0],
			})
			return cty.DynamicVal, diags
		}
		for i, v := range elems {
			if convs[i] != nil {
				newV, err := convs[i](v)
				if err != nil {
					// FIXME: This is a pretty terrible error message.
					diags = append(diags, &hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  fmt.Sprintf("Unconsistent argument types in %s blocks", s.TypeName),
						Detail:   fmt.Sprintf("Block with index %d has inconsistent argument types: %s.", i, err),
						Subject:  &sourceRanges[i],
					})
					// Bail early here so we won't panic below in cty.ListVal
					return cty.DynamicVal, diags
				}
				elems[i] = newV
			}
		}

		ret = cty.SetVal(elems)
	}

	return ret, diags
}

func (s *BlockSetSpec) impliedType() cty.Type {
	return cty.Set(s.Nested.impliedType())
}

func (s *BlockSetSpec) sourceRange(content *hcl.BodyContent, blockLabels []blockLabel) hcl.Range {
	// We return the source range of the _first_ block of the given type,
	// since they are not guaranteed to form a contiguous range.

	var childBlock *hcl.Block
	for _, candidate := range content.Blocks {
		if candidate.Type != s.TypeName {
			continue
		}

		childBlock = candidate
		break
	}

	if childBlock == nil {
		return content.MissingItemRange
	}

	return sourceRange(childBlock.Body, labelsForBlock(childBlock), s.Nested)
}

// A BlockMapSpec is a Spec that produces a cty map of the results of
// decoding all of the nested blocks of a given type, using a nested spec.
//
// One level of map structure is created for each of the given label names.
// There must be at least one given label name.
type BlockMapSpec struct {
	TypeName   string
	LabelNames []string
	Nested     Spec
}

func (s *BlockMapSpec) visitSameBodyChildren(cb visitFunc) {
	// leaf node ("Nested" does not use the same body)
}

// blockSpec implementation
func (s *BlockMapSpec) blockHeaderSchemata() []hcl.BlockHeaderSchema {
	return []hcl.BlockHeaderSchema{
		{
			Type:       s.TypeName,
			LabelNames: append(s.LabelNames, findLabelSpecs(s.Nested)...),
		},
	}
}

// blockSpec implementation
func (s *BlockMapSpec) nestedSpec() Spec {
	return s.Nested
}

// specNeedingVariables implementation
func (s *BlockMapSpec) variablesNeeded(content *hcl.BodyContent) []hcl.Traversal {
	var ret []hcl.Traversal

	for _, childBlock := range content.Blocks {
		if childBlock.Type != s.TypeName {
			continue
		}

		ret = append(ret, Variables(childBlock.Body, s.Nested)...)
	}

	return ret
}

func (s *BlockMapSpec) decode(content *hcl.BodyContent, blockLabels []blockLabel, ctx *hcl.EvalContext) (cty.Value, hcl.Diagnostics) {
	var diags hcl.Diagnostics

	if s.Nested == nil {
		panic("BlockMapSpec with no Nested Spec")
	}
	if ImpliedType(s).HasDynamicTypes() {
		panic("cty.DynamicPseudoType attributes may not be used inside a BlockMapSpec")
	}

	elems := map[string]interface{}{}
	for _, childBlock := range content.Blocks {
		if childBlock.Type != s.TypeName {
			continue
		}

		childLabels := labelsForBlock(childBlock)
		val, _, childDiags := decode(childBlock.Body, childLabels[len(s.LabelNames):], ctx, s.Nested, false)
		targetMap := elems
		for _, key := range childBlock.Labels[:len(s.LabelNames)-1] {
			if _, exists := targetMap[key]; !exists {
				targetMap[key] = make(map[string]interface{})
			}
			targetMap = targetMap[key].(map[string]interface{})
		}

		diags = append(diags, childDiags...)

		key := childBlock.Labels[len(s.LabelNames)-1]
		if _, exists := targetMap[key]; exists {
			labelsBuf := bytes.Buffer{}
			for _, label := range childBlock.Labels {
				fmt.Fprintf(&labelsBuf, " %q", label)
			}
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  fmt.Sprintf("Duplicate %s block", s.TypeName),
				Detail: fmt.Sprintf(
					"A block for %s%s was already defined. The %s labels must be unique.",
					s.TypeName, labelsBuf.String(), s.TypeName,
				),
				Subject: &childBlock.DefRange,
			})
			continue
		}

		targetMap[key] = val
	}

	if len(elems) == 0 {
		return cty.MapValEmpty(s.Nested.impliedType()), diags
	}

	var ctyMap func(map[string]interface{}, int) cty.Value
	ctyMap = func(raw map[string]interface{}, depth int) cty.Value {
		vals := make(map[string]cty.Value, len(raw))
		if depth == 1 {
			for k, v := range raw {
				vals[k] = v.(cty.Value)
			}
		} else {
			for k, v := range raw {
				vals[k] = ctyMap(v.(map[string]interface{}), depth-1)
			}
		}
		return cty.MapVal(vals)
	}

	return ctyMap(elems, len(s.LabelNames)), diags
}

func (s *BlockMapSpec) impliedType() cty.Type {
	ret := s.Nested.impliedType()
	for _ = range s.LabelNames {
		ret = cty.Map(ret)
	}
	return ret
}

func (s *BlockMapSpec) sourceRange(content *hcl.BodyContent, blockLabels []blockLabel) hcl.Range {
	// We return the source range of the _first_ block of the given type,
	// since they are not guaranteed to form a contiguous range.

	var childBlock *hcl.Block
	for _, candidate := range content.Blocks {
		if candidate.Type != s.TypeName {
			continue
		}

		childBlock = candidate
		break
	}

	if childBlock == nil {
		return content.MissingItemRange
	}

	return sourceRange(childBlock.Body, labelsForBlock(childBlock), s.Nested)
}

// A BlockObjectSpec is a Spec that produces a cty object of the results of
// decoding all of the nested blocks of a given type, using a nested spec.
//
// One level of object structure is created for each of the given label names.
// There must be at least one given label name.
//
// This is similar to BlockMapSpec, but it permits the nested blocks to have
// different result types in situations where cty.DynamicPseudoType attributes
// are present.
type BlockObjectSpec struct {
	TypeName   string
	LabelNames []string
	Nested     Spec
}

func (s *BlockObjectSpec) visitSameBodyChildren(cb visitFunc) {
	// leaf node ("Nested" does not use the same body)
}

// blockSpec implementation
func (s *BlockObjectSpec) blockHeaderSchemata() []hcl.BlockHeaderSchema {
	return []hcl.BlockHeaderSchema{
		{
			Type:       s.TypeName,
			LabelNames: append(s.LabelNames, findLabelSpecs(s.Nested)...),
		},
	}
}

// blockSpec implementation
func (s *BlockObjectSpec) nestedSpec() Spec {
	return s.Nested
}

// specNeedingVariables implementation
func (s *BlockObjectSpec) variablesNeeded(content *hcl.BodyContent) []hcl.Traversal {
	var ret []hcl.Traversal

	for _, childBlock := range content.Blocks {
		if childBlock.Type != s.TypeName {
			continue
		}

		ret = append(ret, Variables(childBlock.Body, s.Nested)...)
	}

	return ret
}

func (s *BlockObjectSpec) decode(content *hcl.BodyContent, blockLabels []blockLabel, ctx *hcl.EvalContext) (cty.Value, hcl.Diagnostics) {
	var diags hcl.Diagnostics

	if s.Nested == nil {
		panic("BlockObjectSpec with no Nested Spec")
	}

	elems := map[string]interface{}{}
	for _, childBlock := range content.Blocks {
		if childBlock.Type != s.TypeName {
			continue
		}

		childLabels := labelsForBlock(childBlock)
		val, _, childDiags := decode(childBlock.Body, childLabels[len(s.LabelNames):], ctx, s.Nested, false)
		targetMap := elems
		for _, key := range childBlock.Labels[:len(s.LabelNames)-1] {
			if _, exists := targetMap[key]; !exists {
				targetMap[key] = make(map[string]interface{})
			}
			targetMap = targetMap[key].(map[string]interface{})
		}

		diags = append(diags, childDiags...)

		key := childBlock.Labels[len(s.LabelNames)-1]
		if _, exists := targetMap[key]; exists {
			labelsBuf := bytes.Buffer{}
			for _, label := range childBlock.Labels {
				fmt.Fprintf(&labelsBuf, " %q", label)
			}
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  fmt.Sprintf("Duplicate %s block", s.TypeName),
				Detail: fmt.Sprintf(
					"A block for %s%s was already defined. The %s labels must be unique.",
					s.TypeName, labelsBuf.String(), s.TypeName,
				),
				Subject: &childBlock.DefRange,
			})
			continue
		}

		targetMap[key] = val
	}

	if len(elems) == 0 {
		return cty.EmptyObjectVal, diags
	}

	var ctyObj func(map[string]interface{}, int) cty.Value
	ctyObj = func(raw map[string]interface{}, depth int) cty.Value {
		vals := make(map[string]cty.Value, len(raw))
		if depth == 1 {
			for k, v := range raw {
				vals[k] = v.(cty.Value)
			}
		} else {
			for k, v := range raw {
				vals[k] = ctyObj(v.(map[string]interface{}), depth-1)
			}
		}
		return cty.ObjectVal(vals)
	}

	return ctyObj(elems, len(s.LabelNames)), diags
}

func (s *BlockObjectSpec) impliedType() cty.Type {
	// We can't predict our type, since we don't know how many blocks are
	// present and what labels they have until we decode.
	return cty.DynamicPseudoType
}

func (s *BlockObjectSpec) sourceRange(content *hcl.BodyContent, blockLabels []blockLabel) hcl.Range {
	// We return the source range of the _first_ block of the given type,
	// since they are not guaranteed to form a contiguous range.

	var childBlock *hcl.Block
	for _, candidate := range content.Blocks {
		if candidate.Type != s.TypeName {
			continue
		}

		childBlock = candidate
		break
	}

	if childBlock == nil {
		return content.MissingItemRange
	}

	return sourceRange(childBlock.Body, labelsForBlock(childBlock), s.Nested)
}

// A BlockAttrsSpec is a Spec that interprets a single block as if it were
// a map of some element type. That is, each attribute within the block
// becomes a key in the resulting map and the attribute's value becomes the
// element value, after conversion to the given element type. The resulting
// value is a cty.Map of the given element type.
//
// This spec imposes a validation constraint that there be exactly one block
// of the given type name and that this block may contain only attributes. The
// block does not accept any labels.
//
// This is an alternative to an AttrSpec of a map type for situations where
// block syntax is desired. Note that block syntax does not permit dynamic
// keys, construction of the result via a "for" expression, etc. In most cases
// an AttrSpec is preferred if the desired result is a map whose keys are
// chosen by the user rather than by schema.
type BlockAttrsSpec struct {
	TypeName    string
	ElementType cty.Type
	Required    bool
}

func (s *BlockAttrsSpec) visitSameBodyChildren(cb visitFunc) {
	// leaf node
}

// blockSpec implementation
func (s *BlockAttrsSpec) blockHeaderSchemata() []hcl.BlockHeaderSchema {
	return []hcl.BlockHeaderSchema{
		{
			Type:       s.TypeName,
			LabelNames: nil,
		},
	}
}

// blockSpec implementation
func (s *BlockAttrsSpec) nestedSpec() Spec {
	// This is an odd case: we aren't actually going to apply a nested spec
	// in this case, since we're going to interpret the body directly as
	// attributes, but we need to return something non-nil so that the
	// decoder will recognize this as a block spec. We won't actually be
	// using this for anything at decode time.
	return noopSpec{}
}

// specNeedingVariables implementation
func (s *BlockAttrsSpec) variablesNeeded(content *hcl.BodyContent) []hcl.Traversal {

	block, _ := s.findBlock(content)
	if block == nil {
		return nil
	}

	var vars []hcl.Traversal

	attrs, diags := block.Body.JustAttributes()
	if diags.HasErrors() {
		return nil
	}

	for _, attr := range attrs {
		vars = append(vars, attr.Expr.Variables()...)
	}

	// We'll return the variables references in source order so that any
	// error messages that result are also in source order.
	sort.Slice(vars, func(i, j int) bool {
		return vars[i].SourceRange().Start.Byte < vars[j].SourceRange().Start.Byte
	})

	return vars
}

func (s *BlockAttrsSpec) decode(content *hcl.BodyContent, blockLabels []blockLabel, ctx *hcl.EvalContext) (cty.Value, hcl.Diagnostics) {
	var diags hcl.Diagnostics

	block, other := s.findBlock(content)
	if block == nil {
		if s.Required {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  fmt.Sprintf("Missing %s block", s.TypeName),
				Detail: fmt.Sprintf(
					"A block of type %q is required here.", s.TypeName,
				),
				Subject: &content.MissingItemRange,
			})
		}
		return cty.NullVal(cty.Map(s.ElementType)), diags
	}
	if other != nil {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  fmt.Sprintf("Duplicate %s block", s.TypeName),
			Detail: fmt.Sprintf(
				"Only one block of type %q is allowed. Previous definition was at %s.",
				s.TypeName, block.DefRange.String(),
			),
			Subject: &other.DefRange,
		})
	}

	attrs, attrDiags := block.Body.JustAttributes()
	diags = append(diags, attrDiags...)

	if len(attrs) == 0 {
		return cty.MapValEmpty(s.ElementType), diags
	}

	vals := make(map[string]cty.Value, len(attrs))
	for name, attr := range attrs {
		if decodeFn := customdecode.CustomExpressionDecoderForType(s.ElementType); decodeFn != nil {
			attrVal, attrDiags := decodeFn(attr.Expr, ctx)
			diags = append(diags, attrDiags...)
			if attrVal == cty.NilVal {
				attrVal = cty.UnknownVal(s.ElementType)
			}
			vals[name] = attrVal
			continue
		}

		attrVal, attrDiags := attr.Expr.Value(ctx)
		diags = append(diags, attrDiags...)

		attrVal, err := convert.Convert(attrVal, s.ElementType)
		if err != nil {
			diags = append(diags, &hcl.Diagnostic{
				Severity:    hcl.DiagError,
				Summary:     "Invalid attribute value",
				Detail:      fmt.Sprintf("Invalid value for attribute of %q block: %s.", s.TypeName, err),
				Subject:     attr.Expr.Range().Ptr(),
				Context:     hcl.RangeBetween(attr.NameRange, attr.Expr.Range()).Ptr(),
				Expression:  attr.Expr,
				EvalContext: ctx,
			})
			attrVal = cty.UnknownVal(s.ElementType)
		}

		vals[name] = attrVal
	}

	return cty.MapVal(vals), diags
}

func (s *BlockAttrsSpec) impliedType() cty.Type {
	return cty.Map(s.ElementType)
}

func (s *BlockAttrsSpec) sourceRange(content *hcl.BodyContent, blockLabels []blockLabel) hcl.Range {
	block, _ := s.findBlock(content)
	if block == nil {
		return content.MissingItemRange
	}
	return block.DefRange
}

func (s *BlockAttrsSpec) findBlock(content *hcl.BodyContent) (block *hcl.Block, other *hcl.Block) {
	for _, candidate := range content.Blocks {
		if candidate.Type != s.TypeName {
			continue
		}
		if block != nil {
			return block, candidate
		}
		block = candidate
	}

	return block, nil
}

// A BlockLabelSpec is a Spec that returns a cty.String representing the
// label of the block its given body belongs to, if indeed its given body
// belongs to a block. It is a programming error to use this in a non-block
// context, so this spec will panic in that case.
//
// This spec only works in the nested spec within a BlockSpec, BlockListSpec,
// BlockSetSpec or BlockMapSpec.
//
// The full set of label specs used against a particular block must have a
// consecutive set of indices starting at zero. The maximum index found
// defines how many labels the corresponding blocks must have in cty source.
type BlockLabelSpec struct {
	Index int
	Name  string
}

func (s *BlockLabelSpec) visitSameBodyChildren(cb visitFunc) {
	// leaf node
}

func (s *BlockLabelSpec) decode(content *hcl.BodyContent, blockLabels []blockLabel, ctx *hcl.EvalContext) (cty.Value, hcl.Diagnostics) {
	if s.Index >= len(blockLabels) {
		panic("BlockListSpec used in non-block context")
	}

	return cty.StringVal(blockLabels[s.Index].Value), nil
}

func (s *BlockLabelSpec) impliedType() cty.Type {
	return cty.String // labels are always strings
}

func (s *BlockLabelSpec) sourceRange(content *hcl.BodyContent, blockLabels []blockLabel) hcl.Range {
	if s.Index >= len(blockLabels) {
		panic("BlockListSpec used in non-block context")
	}

	return blockLabels[s.Index].Range
}

func findLabelSpecs(spec Spec) []string {
	maxIdx := -1
	var names map[int]string

	var visit visitFunc
	visit = func(s Spec) {
		if ls, ok := s.(*BlockLabelSpec); ok {
			if maxIdx < ls.Index {
				maxIdx = ls.Index
			}
			if names == nil {
				names = make(map[int]string)
			}
			names[ls.Index] = ls.Name
		}
		s.visitSameBodyChildren(visit)
	}

	visit(spec)

	if maxIdx < 0 {
		return nil // no labels at all
	}

	ret := make([]string, maxIdx+1)
	for i := range ret {
		name := names[i]
		if name == "" {
			// Should never happen if the spec is conformant, since we require
			// consecutive indices starting at zero.
			name = fmt.Sprintf("missing%02d", i)
		}
		ret[i] = name
	}

	return ret
}

// DefaultSpec is a spec that wraps two specs, evaluating the primary first
// and then evaluating the default if the primary returns a null value.
//
// The two specifications must have the same implied result type for correct
// operation. If not, the result is undefined.
//
// Any requirements imposed by the "Default" spec apply even if "Primary" does
// not return null. For example, if the "Default" spec is for a required
// attribute then that attribute is always required, regardless of the result
// of the "Primary" spec.
//
// The "Default" spec must not describe a nested block, since otherwise the
// result of ChildBlockTypes would not be decidable without evaluation. If
// the default spec _does_ describe a nested block then the result is
// undefined.
type DefaultSpec struct {
	Primary Spec
	Default Spec
}

func (s *DefaultSpec) visitSameBodyChildren(cb visitFunc) {
	cb(s.Primary)
	cb(s.Default)
}

func (s *DefaultSpec) decode(content *hcl.BodyContent, blockLabels []blockLabel, ctx *hcl.EvalContext) (cty.Value, hcl.Diagnostics) {
	val, diags := s.Primary.decode(content, blockLabels, ctx)
	if val.IsNull() {
		var moreDiags hcl.Diagnostics
		val, moreDiags = s.Default.decode(content, blockLabels, ctx)
		diags = append(diags, moreDiags...)
	}
	return val, diags
}

func (s *DefaultSpec) impliedType() cty.Type {
	return s.Primary.impliedType()
}

// attrSpec implementation
func (s *DefaultSpec) attrSchemata() []hcl.AttributeSchema {
	// We must pass through the union of both of our nested specs so that
	// we'll have both values available in the result.
	var ret []hcl.AttributeSchema
	if as, ok := s.Primary.(attrSpec); ok {
		ret = append(ret, as.attrSchemata()...)
	}
	if as, ok := s.Default.(attrSpec); ok {
		ret = append(ret, as.attrSchemata()...)
	}
	return ret
}

// blockSpec implementation
func (s *DefaultSpec) blockHeaderSchemata() []hcl.BlockHeaderSchema {
	// Only the primary spec may describe a block, since otherwise
	// our nestedSpec method below can't know which to return.
	if bs, ok := s.Primary.(blockSpec); ok {
		return bs.blockHeaderSchemata()
	}
	return nil
}

// blockSpec implementation
func (s *DefaultSpec) nestedSpec() Spec {
	if bs, ok := s.Primary.(blockSpec); ok {
		return bs.nestedSpec()
	}
	return nil
}

func (s *DefaultSpec) sourceRange(content *hcl.BodyContent, blockLabels []blockLabel) hcl.Range {
	// We can't tell from here which of the two specs will ultimately be used
	// in our result, so we'll just assume the first. This is usually the right
	// choice because the default is often a literal spec that doesn't have a
	// reasonable source range to return anyway.
	return s.Primary.sourceRange(content, blockLabels)
}

// TransformExprSpec is a spec that wraps another and then evaluates a given
// hcl.Expression on the result.
//
// The implied type of this spec is determined by evaluating the expression
// with an unknown value of the nested spec's implied type, which may cause
// the result to be imprecise. This spec should not be used in situations where
// precise result type information is needed.
type TransformExprSpec struct {
	Wrapped      Spec
	Expr         hcl.Expression
	TransformCtx *hcl.EvalContext
	VarName      string
}

func (s *TransformExprSpec) visitSameBodyChildren(cb visitFunc) {
	cb(s.Wrapped)
}

func (s *TransformExprSpec) decode(content *hcl.BodyContent, blockLabels []blockLabel, ctx *hcl.EvalContext) (cty.Value, hcl.Diagnostics) {
	wrappedVal, diags := s.Wrapped.decode(content, blockLabels, ctx)
	if diags.HasErrors() {
		// We won't try to run our function in this case, because it'll probably
		// generate confusing additional errors that will distract from the
		// root cause.
		return cty.UnknownVal(s.impliedType()), diags
	}

	chiCtx := s.TransformCtx.NewChild()
	chiCtx.Variables = map[string]cty.Value{
		s.VarName: wrappedVal,
	}
	resultVal, resultDiags := s.Expr.Value(chiCtx)
	diags = append(diags, resultDiags...)
	return resultVal, diags
}

func (s *TransformExprSpec) impliedType() cty.Type {
	wrappedTy := s.Wrapped.impliedType()
	chiCtx := s.TransformCtx.NewChild()
	chiCtx.Variables = map[string]cty.Value{
		s.VarName: cty.UnknownVal(wrappedTy),
	}
	resultVal, _ := s.Expr.Value(chiCtx)
	return resultVal.Type()
}

func (s *TransformExprSpec) sourceRange(content *hcl.BodyContent, blockLabels []blockLabel) hcl.Range {
	// We'll just pass through our wrapped range here, even though that's
	// not super-accurate, because there's nothing better to return.
	return s.Wrapped.sourceRange(content, blockLabels)
}

// TransformFuncSpec is a spec that wraps another and then evaluates a given
// cty function with the result. The given function must expect exactly one
// argument, where the result of the wrapped spec will be passed.
//
// The implied type of this spec is determined by type-checking the function
// with an unknown value of the nested spec's implied type, which may cause
// the result to be imprecise. This spec should not be used in situations where
// precise result type information is needed.
//
// If the given function produces an error when run, this spec will produce
// a non-user-actionable diagnostic message. It's the caller's responsibility
// to ensure that the given function cannot fail for any non-error result
// of the wrapped spec.
type TransformFuncSpec struct {
	Wrapped Spec
	Func    function.Function
}

func (s *TransformFuncSpec) visitSameBodyChildren(cb visitFunc) {
	cb(s.Wrapped)
}

func (s *TransformFuncSpec) decode(content *hcl.BodyContent, blockLabels []blockLabel, ctx *hcl.EvalContext) (cty.Value, hcl.Diagnostics) {
	wrappedVal, diags := s.Wrapped.decode(content, blockLabels, ctx)
	if diags.HasErrors() {
		// We won't try to run our function in this case, because it'll probably
		// generate confusing additional errors that will distract from the
		// root cause.
		return cty.UnknownVal(s.impliedType()), diags
	}

	resultVal, err := s.Func.Call([]cty.Value{wrappedVal})
	if err != nil {
		// This is not a good example of a diagnostic because it is reporting
		// a programming error in the calling application, rather than something
		// an end-user could act on.
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Transform function failed",
			Detail:   fmt.Sprintf("Decoder transform returned an error: %s", err),
			Subject:  s.sourceRange(content, blockLabels).Ptr(),
		})
		return cty.UnknownVal(s.impliedType()), diags
	}

	return resultVal, diags
}

func (s *TransformFuncSpec) impliedType() cty.Type {
	wrappedTy := s.Wrapped.impliedType()
	resultTy, err := s.Func.ReturnType([]cty.Type{wrappedTy})
	if err != nil {
		// Should never happen with a correctly-configured spec
		return cty.DynamicPseudoType
	}

	return resultTy
}

func (s *TransformFuncSpec) sourceRange(content *hcl.BodyContent, blockLabels []blockLabel) hcl.Range {
	// We'll just pass through our wrapped range here, even though that's
	// not super-accurate, because there's nothing better to return.
	return s.Wrapped.sourceRange(content, blockLabels)
}

// ValidateFuncSpec is a spec that allows for extended
// developer-defined validation. The validation function receives the
// result of the wrapped spec.
//
// The Subject field of the returned Diagnostic is optional. If not
// specified, it is automatically populated with the range covered by
// the wrapped spec.
//
type ValidateSpec struct {
	Wrapped Spec
	Func    func(value cty.Value) hcl.Diagnostics
}

func (s *ValidateSpec) visitSameBodyChildren(cb visitFunc) {
	cb(s.Wrapped)
}

func (s *ValidateSpec) decode(content *hcl.BodyContent, blockLabels []blockLabel, ctx *hcl.EvalContext) (cty.Value, hcl.Diagnostics) {
	wrappedVal, diags := s.Wrapped.decode(content, blockLabels, ctx)
	if diags.HasErrors() {
		// We won't try to run our function in this case, because it'll probably
		// generate confusing additional errors that will distract from the
		// root cause.
		return cty.UnknownVal(s.impliedType()), diags
	}

	validateDiags := s.Func(wrappedVal)
	// Auto-populate the Subject fields if they weren't set.
	for i := range validateDiags {
		if validateDiags[i].Subject == nil {
			validateDiags[i].Subject = s.sourceRange(content, blockLabels).Ptr()
		}
	}

	diags = append(diags, validateDiags...)
	return wrappedVal, diags
}

func (s *ValidateSpec) impliedType() cty.Type {
	return s.Wrapped.impliedType()
}

func (s *ValidateSpec) sourceRange(content *hcl.BodyContent, blockLabels []blockLabel) hcl.Range {
	return s.Wrapped.sourceRange(content, blockLabels)
}

// noopSpec is a placeholder spec that does nothing, used in situations where
// a non-nil placeholder spec is required. It is not exported because there is
// no reason to use it directly; it is always an implementation detail only.
type noopSpec struct {
}

func (s noopSpec) decode(content *hcl.BodyContent, blockLabels []blockLabel, ctx *hcl.EvalContext) (cty.Value, hcl.Diagnostics) {
	return cty.NullVal(cty.DynamicPseudoType), nil
}

func (s noopSpec) impliedType() cty.Type {
	return cty.DynamicPseudoType
}

func (s noopSpec) visitSameBodyChildren(cb visitFunc) {
	// nothing to do
}

func (s noopSpec) sourceRange(content *hcl.BodyContent, blockLabels []blockLabel) hcl.Range {
	// No useful range for a noopSpec, and nobody should be calling this anyway.
	return hcl.Range{
		Filename: "noopSpec",
	}
}
