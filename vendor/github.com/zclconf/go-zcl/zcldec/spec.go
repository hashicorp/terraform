package zcldec

import (
	"fmt"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"
	"github.com/zclconf/go-zcl/zcl"
)

// A Spec is a description of how to decode a zcl.Body to a cty.Value.
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
	decode(content *zcl.BodyContent, block *zcl.Block, ctx *zcl.EvalContext) (cty.Value, zcl.Diagnostics)

	// Call the given callback once for each of the nested specs that would
	// get decoded with the same body and block as the receiver. This should
	// not descend into the nested specs used when decoding blocks.
	visitSameBodyChildren(cb visitFunc)

	// Determine the source range of the value that would be returned for the
	// spec in the given content, in the context of the given block
	// (which might be null). If the corresponding item is missing, return
	// a place where it might be inserted.
	sourceRange(content *zcl.BodyContent, block *zcl.Block) zcl.Range
}

type visitFunc func(spec Spec)

// An ObjectSpec is a Spec that produces a cty.Value of an object type whose
// attributes correspond to the keys of the spec map.
type ObjectSpec map[string]Spec

// attrSpec is implemented by specs that require attributes from the body.
type attrSpec interface {
	attrSchemata() []zcl.AttributeSchema
}

// blockSpec is implemented by specs that require blocks from the body.
type blockSpec interface {
	blockHeaderSchemata() []zcl.BlockHeaderSchema
}

// specNeedingVariables is implemented by specs that can use variables
// from the EvalContext, to declare which variables they need.
type specNeedingVariables interface {
	variablesNeeded(content *zcl.BodyContent) []zcl.Traversal
}

func (s ObjectSpec) visitSameBodyChildren(cb visitFunc) {
	for _, c := range s {
		cb(c)
	}
}

func (s ObjectSpec) decode(content *zcl.BodyContent, block *zcl.Block, ctx *zcl.EvalContext) (cty.Value, zcl.Diagnostics) {
	vals := make(map[string]cty.Value, len(s))
	var diags zcl.Diagnostics

	for k, spec := range s {
		var kd zcl.Diagnostics
		vals[k], kd = spec.decode(content, block, ctx)
		diags = append(diags, kd...)
	}

	return cty.ObjectVal(vals), diags
}

func (s ObjectSpec) sourceRange(content *zcl.BodyContent, block *zcl.Block) zcl.Range {
	if block != nil {
		return block.DefRange
	}

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

func (s TupleSpec) decode(content *zcl.BodyContent, block *zcl.Block, ctx *zcl.EvalContext) (cty.Value, zcl.Diagnostics) {
	vals := make([]cty.Value, len(s))
	var diags zcl.Diagnostics

	for i, spec := range s {
		var ed zcl.Diagnostics
		vals[i], ed = spec.decode(content, block, ctx)
		diags = append(diags, ed...)
	}

	return cty.TupleVal(vals), diags
}

func (s TupleSpec) sourceRange(content *zcl.BodyContent, block *zcl.Block) zcl.Range {
	if block != nil {
		return block.DefRange
	}

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
func (s *AttrSpec) variablesNeeded(content *zcl.BodyContent) []zcl.Traversal {
	attr, exists := content.Attributes[s.Name]
	if !exists {
		return nil
	}

	return attr.Expr.Variables()
}

// attrSpec implementation
func (s *AttrSpec) attrSchemata() []zcl.AttributeSchema {
	return []zcl.AttributeSchema{
		{
			Name:     s.Name,
			Required: s.Required,
		},
	}
}

func (s *AttrSpec) sourceRange(content *zcl.BodyContent, block *zcl.Block) zcl.Range {
	attr, exists := content.Attributes[s.Name]
	if !exists {
		return content.MissingItemRange
	}

	return attr.Expr.Range()
}

func (s *AttrSpec) decode(content *zcl.BodyContent, block *zcl.Block, ctx *zcl.EvalContext) (cty.Value, zcl.Diagnostics) {
	attr, exists := content.Attributes[s.Name]
	if !exists {
		// We don't need to check required and emit a diagnostic here, because
		// that would already have happened when building "content".
		return cty.NullVal(s.Type), nil
	}

	val, diags := attr.Expr.Value(ctx)

	convVal, err := convert.Convert(val, s.Type)
	if err != nil {
		diags = append(diags, &zcl.Diagnostic{
			Severity: zcl.DiagError,
			Summary:  "Incorrect attribute value type",
			Detail: fmt.Sprintf(
				"Inappropriate value for attribute %q: %s.",
				s.Name, err.Error(),
			),
			Subject: attr.Expr.StartRange().Ptr(),
			Context: zcl.RangeBetween(attr.NameRange, attr.Expr.StartRange()).Ptr(),
		})
		// We'll return an unknown value of the _correct_ type so that the
		// incomplete result can still be used for some analysis use-cases.
		val = cty.UnknownVal(s.Type)
	} else {
		val = convVal
	}

	return val, diags
}

// A LiteralSpec is a Spec that produces the given literal value, ignoring
// the given body.
type LiteralSpec struct {
	Value cty.Value
}

func (s *LiteralSpec) visitSameBodyChildren(cb visitFunc) {
	// leaf node
}

func (s *LiteralSpec) decode(content *zcl.BodyContent, block *zcl.Block, ctx *zcl.EvalContext) (cty.Value, zcl.Diagnostics) {
	return s.Value, nil
}

func (s *LiteralSpec) sourceRange(content *zcl.BodyContent, block *zcl.Block) zcl.Range {
	// No sensible range to return for a literal, so the caller had better
	// ensure it doesn't cause any diagnostics.
	return zcl.Range{
		Filename: "<unknown>",
	}
}

// An ExprSpec is a Spec that evaluates the given expression, ignoring the
// given body.
type ExprSpec struct {
	Expr zcl.Expression
}

func (s *ExprSpec) visitSameBodyChildren(cb visitFunc) {
	// leaf node
}

// specNeedingVariables implementation
func (s *ExprSpec) variablesNeeded(content *zcl.BodyContent) []zcl.Traversal {
	return s.Expr.Variables()
}

func (s *ExprSpec) decode(content *zcl.BodyContent, block *zcl.Block, ctx *zcl.EvalContext) (cty.Value, zcl.Diagnostics) {
	return s.Expr.Value(ctx)
}

func (s *ExprSpec) sourceRange(content *zcl.BodyContent, block *zcl.Block) zcl.Range {
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
func (s *BlockSpec) blockHeaderSchemata() []zcl.BlockHeaderSchema {
	return []zcl.BlockHeaderSchema{
		{
			Type: s.TypeName,
		},
	}
}

// specNeedingVariables implementation
func (s *BlockSpec) variablesNeeded(content *zcl.BodyContent) []zcl.Traversal {
	var childBlock *zcl.Block
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

func (s *BlockSpec) decode(content *zcl.BodyContent, block *zcl.Block, ctx *zcl.EvalContext) (cty.Value, zcl.Diagnostics) {
	var diags zcl.Diagnostics

	var childBlock *zcl.Block
	for _, candidate := range content.Blocks {
		if candidate.Type != s.TypeName {
			continue
		}

		if childBlock != nil {
			diags = append(diags, &zcl.Diagnostic{
				Severity: zcl.DiagError,
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
			diags = append(diags, &zcl.Diagnostic{
				Severity: zcl.DiagError,
				Summary:  fmt.Sprintf("Missing %s block", s.TypeName),
				Detail: fmt.Sprintf(
					"A block of type %q is required here.", s.TypeName,
				),
				Subject: &content.MissingItemRange,
			})
		}
		return cty.NullVal(cty.DynamicPseudoType), diags
	}

	if s.Nested == nil {
		panic("BlockSpec with no Nested Spec")
	}
	val, _, childDiags := decode(childBlock.Body, childBlock, ctx, s.Nested, false)
	diags = append(diags, childDiags...)
	return val, diags
}

func (s *BlockSpec) sourceRange(content *zcl.BodyContent, block *zcl.Block) zcl.Range {
	var childBlock *zcl.Block
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

	return sourceRange(childBlock.Body, childBlock, s.Nested)
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
func (s *BlockListSpec) blockHeaderSchemata() []zcl.BlockHeaderSchema {
	return []zcl.BlockHeaderSchema{
		{
			Type: s.TypeName,
			// FIXME: Need to peek into s.Nested to see if it has any
			// BlockLabelSpec instances, which will define then how many
			// labels we need.
		},
	}
}

// specNeedingVariables implementation
func (s *BlockListSpec) variablesNeeded(content *zcl.BodyContent) []zcl.Traversal {
	var ret []zcl.Traversal

	for _, childBlock := range content.Blocks {
		if childBlock.Type != s.TypeName {
			continue
		}

		ret = append(ret, Variables(childBlock.Body, s.Nested)...)
	}

	return ret
}

func (s *BlockListSpec) decode(content *zcl.BodyContent, block *zcl.Block, ctx *zcl.EvalContext) (cty.Value, zcl.Diagnostics) {
	var diags zcl.Diagnostics

	if s.Nested == nil {
		panic("BlockSpec with no Nested Spec")
	}

	var elems []cty.Value
	var sourceRanges []zcl.Range
	for _, childBlock := range content.Blocks {
		if childBlock.Type != s.TypeName {
			continue
		}

		val, _, childDiags := decode(childBlock.Body, childBlock, ctx, s.Nested, false)
		diags = append(diags, childDiags...)
		elems = append(elems, val)
		sourceRanges = append(sourceRanges, sourceRange(childBlock.Body, childBlock, s.Nested))
	}

	if len(elems) < s.MinItems {
		diags = append(diags, &zcl.Diagnostic{
			Severity: zcl.DiagError,
			Summary:  fmt.Sprintf("insufficient %s blocks", s.TypeName),
			Detail:   fmt.Sprintf("at least %d blocks are required", s.MinItems),
			Subject:  &content.MissingItemRange,
		})
	} else if s.MaxItems > 0 && len(elems) > s.MaxItems {
		diags = append(diags, &zcl.Diagnostic{
			Severity: zcl.DiagError,
			Summary:  fmt.Sprintf("too many %s blocks", s.TypeName),
			Detail:   fmt.Sprintf("no more than %d blocks are allowed", s.MaxItems),
			Subject:  &sourceRanges[s.MaxItems],
		})
	}

	var ret cty.Value

	if len(elems) == 0 {
		// FIXME: We don't currently have enough info to construct a type for
		// an empty list, so we'll just stub it out.
		ret = cty.ListValEmpty(cty.DynamicPseudoType)
	} else {
		ret = cty.ListVal(elems)
	}

	return ret, diags
}

func (s *BlockListSpec) sourceRange(content *zcl.BodyContent, block *zcl.Block) zcl.Range {
	// We return the source range of the _first_ block of the given type,
	// since they are not guaranteed to form a contiguous range.

	var childBlock *zcl.Block
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

	return sourceRange(childBlock.Body, childBlock, s.Nested)
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

func (s *BlockSetSpec) decode(content *zcl.BodyContent, block *zcl.Block, ctx *zcl.EvalContext) (cty.Value, zcl.Diagnostics) {
	panic("BlockSetSpec.decode not yet implemented")
}

func (s *BlockSetSpec) sourceRange(content *zcl.BodyContent, block *zcl.Block) zcl.Range {
	// We return the source range of the _first_ block of the given type,
	// since they are not guaranteed to form a contiguous range.

	var childBlock *zcl.Block
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

	return sourceRange(childBlock.Body, childBlock, s.Nested)
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

func (s *BlockMapSpec) decode(content *zcl.BodyContent, block *zcl.Block, ctx *zcl.EvalContext) (cty.Value, zcl.Diagnostics) {
	panic("BlockMapSpec.decode not yet implemented")
}

func (s *BlockMapSpec) sourceRange(content *zcl.BodyContent, block *zcl.Block) zcl.Range {
	// We return the source range of the _first_ block of the given type,
	// since they are not guaranteed to form a contiguous range.

	var childBlock *zcl.Block
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

	return sourceRange(childBlock.Body, childBlock, s.Nested)
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

func (s *BlockLabelSpec) decode(content *zcl.BodyContent, block *zcl.Block, ctx *zcl.EvalContext) (cty.Value, zcl.Diagnostics) {
	panic("BlockLabelSpec.decode not yet implemented")
}

func (s *BlockLabelSpec) sourceRange(content *zcl.BodyContent, block *zcl.Block) zcl.Range {
	if block == nil {
		panic("BlockListSpec used in non-block context")
	}

	return block.LabelRanges[s.Index]
}
