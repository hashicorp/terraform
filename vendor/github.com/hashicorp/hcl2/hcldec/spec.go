package hcldec

import (
	"bytes"
	"fmt"

	"github.com/hashicorp/hcl2/hcl"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"
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
			Subject: attr.Expr.StartRange().Ptr(),
			Context: hcl.RangeBetween(attr.NameRange, attr.Expr.StartRange()).Ptr(),
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
		panic("BlockSetSpec with no Nested Spec")
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

func (s *DefaultSpec) sourceRange(content *hcl.BodyContent, blockLabels []blockLabel) hcl.Range {
	// We can't tell from here which of the two specs will ultimately be used
	// in our result, so we'll just assume the first. This is usually the right
	// choice because the default is often a literal spec that doesn't have a
	// reasonable source range to return anyway.
	return s.Primary.sourceRange(content, blockLabels)
}
