package hcltest

import (
	"fmt"

	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
)

// MockBody returns a hcl.Body implementation that works in terms of a
// caller-constructed hcl.BodyContent, thus avoiding the need to parse
// a "real" HCL config file to use as input to a test.
func MockBody(content *hcl.BodyContent) hcl.Body {
	return mockBody{content}
}

type mockBody struct {
	C *hcl.BodyContent
}

func (b mockBody) Content(schema *hcl.BodySchema) (*hcl.BodyContent, hcl.Diagnostics) {
	content, remainI, diags := b.PartialContent(schema)
	remain := remainI.(mockBody)
	for _, attr := range remain.C.Attributes {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Extraneous argument in mock body",
			Detail:   fmt.Sprintf("Mock body has extraneous argument %q.", attr.Name),
			Subject:  &attr.NameRange,
		})
	}
	for _, block := range remain.C.Blocks {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Extraneous block in mock body",
			Detail:   fmt.Sprintf("Mock body has extraneous block of type %q.", block.Type),
			Subject:  &block.DefRange,
		})
	}
	return content, diags
}

func (b mockBody) PartialContent(schema *hcl.BodySchema) (*hcl.BodyContent, hcl.Body, hcl.Diagnostics) {
	ret := &hcl.BodyContent{
		Attributes:       map[string]*hcl.Attribute{},
		Blocks:           []*hcl.Block{},
		MissingItemRange: b.C.MissingItemRange,
	}
	remain := &hcl.BodyContent{
		Attributes:       map[string]*hcl.Attribute{},
		Blocks:           []*hcl.Block{},
		MissingItemRange: b.C.MissingItemRange,
	}
	var diags hcl.Diagnostics

	if len(schema.Attributes) != 0 {
		for _, attrS := range schema.Attributes {
			name := attrS.Name
			attr, ok := b.C.Attributes[name]
			if !ok {
				if attrS.Required {
					diags = append(diags, &hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Missing required argument",
						Detail:   fmt.Sprintf("Mock body doesn't have argument %q", name),
						Subject:  b.C.MissingItemRange.Ptr(),
					})
				}
				continue
			}
			ret.Attributes[name] = attr
		}
	}

	for attrN, attr := range b.C.Attributes {
		if _, ok := ret.Attributes[attrN]; !ok {
			remain.Attributes[attrN] = attr
		}
	}

	wantedBlocks := map[string]hcl.BlockHeaderSchema{}
	for _, blockS := range schema.Blocks {
		wantedBlocks[blockS.Type] = blockS
	}

	for _, block := range b.C.Blocks {
		if blockS, ok := wantedBlocks[block.Type]; ok {
			if len(block.Labels) != len(blockS.LabelNames) {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Wrong number of block labels",
					Detail:   fmt.Sprintf("Block of type %q requires %d labels, but got %d", blockS.Type, len(blockS.LabelNames), len(block.Labels)),
					Subject:  b.C.MissingItemRange.Ptr(),
				})
			}

			ret.Blocks = append(ret.Blocks, block)
		} else {
			remain.Blocks = append(remain.Blocks, block)
		}
	}

	return ret, mockBody{remain}, diags
}

func (b mockBody) JustAttributes() (hcl.Attributes, hcl.Diagnostics) {
	var diags hcl.Diagnostics
	if len(b.C.Blocks) != 0 {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Mock body has blocks",
			Detail:   "Can't use JustAttributes on a mock body with blocks.",
			Subject:  b.C.MissingItemRange.Ptr(),
		})
	}

	return b.C.Attributes, diags
}

func (b mockBody) MissingItemRange() hcl.Range {
	return b.C.MissingItemRange
}

// MockExprLiteral returns a hcl.Expression that evaluates to the given literal
// value.
func MockExprLiteral(val cty.Value) hcl.Expression {
	return mockExprLiteral{val}
}

type mockExprLiteral struct {
	V cty.Value
}

func (e mockExprLiteral) Value(ctx *hcl.EvalContext) (cty.Value, hcl.Diagnostics) {
	return e.V, nil
}

func (e mockExprLiteral) Variables() []hcl.Traversal {
	return nil
}

func (e mockExprLiteral) Range() hcl.Range {
	return hcl.Range{
		Filename: "MockExprLiteral",
	}
}

func (e mockExprLiteral) StartRange() hcl.Range {
	return e.Range()
}

// Implementation for hcl.ExprList
func (e mockExprLiteral) ExprList() []hcl.Expression {
	v := e.V
	ty := v.Type()
	if v.IsKnown() && !v.IsNull() && (ty.IsListType() || ty.IsTupleType()) {
		ret := make([]hcl.Expression, 0, v.LengthInt())
		for it := v.ElementIterator(); it.Next(); {
			_, v := it.Element()
			ret = append(ret, MockExprLiteral(v))
		}
		return ret
	}
	return nil
}

// MockExprVariable returns a hcl.Expression that evaluates to the value of
// the variable with the given name.
func MockExprVariable(name string) hcl.Expression {
	return mockExprVariable(name)
}

type mockExprVariable string

func (e mockExprVariable) Value(ctx *hcl.EvalContext) (cty.Value, hcl.Diagnostics) {
	name := string(e)
	for ctx != nil {
		if val, ok := ctx.Variables[name]; ok {
			return val, nil
		}
		ctx = ctx.Parent()
	}

	// If we fall out here then there is no variable with the given name
	return cty.DynamicVal, hcl.Diagnostics{
		{
			Severity: hcl.DiagError,
			Summary:  "Reference to undefined variable",
			Detail:   fmt.Sprintf("Variable %q is not defined.", name),
		},
	}
}

func (e mockExprVariable) Variables() []hcl.Traversal {
	return []hcl.Traversal{
		{
			hcl.TraverseRoot{
				Name:     string(e),
				SrcRange: e.Range(),
			},
		},
	}
}

func (e mockExprVariable) Range() hcl.Range {
	return hcl.Range{
		Filename: "MockExprVariable",
	}
}

func (e mockExprVariable) StartRange() hcl.Range {
	return e.Range()
}

// Implementation for hcl.AbsTraversalForExpr and hcl.RelTraversalForExpr.
func (e mockExprVariable) AsTraversal() hcl.Traversal {
	return hcl.Traversal{
		hcl.TraverseRoot{
			Name:     string(e),
			SrcRange: e.Range(),
		},
	}
}

// MockExprTraversal returns a hcl.Expression that evaluates the given
// absolute traversal.
func MockExprTraversal(traversal hcl.Traversal) hcl.Expression {
	return mockExprTraversal{
		Traversal: traversal,
	}
}

// MockExprTraversalSrc is like MockExprTraversal except it takes a
// traversal string as defined by the native syntax and parses it first.
//
// This method is primarily for testing with hard-coded traversal strings, so
// it will panic if the given string is not syntactically correct.
func MockExprTraversalSrc(src string) hcl.Expression {
	traversal, diags := hclsyntax.ParseTraversalAbs([]byte(src), "MockExprTraversal", hcl.Pos{})
	if diags.HasErrors() {
		panic("invalid traversal string")
	}
	return MockExprTraversal(traversal)
}

type mockExprTraversal struct {
	Traversal hcl.Traversal
}

func (e mockExprTraversal) Value(ctx *hcl.EvalContext) (cty.Value, hcl.Diagnostics) {
	return e.Traversal.TraverseAbs(ctx)
}

func (e mockExprTraversal) Variables() []hcl.Traversal {
	return []hcl.Traversal{e.Traversal}
}

func (e mockExprTraversal) Range() hcl.Range {
	return e.Traversal.SourceRange()
}

func (e mockExprTraversal) StartRange() hcl.Range {
	return e.Range()
}

// Implementation for hcl.AbsTraversalForExpr and hcl.RelTraversalForExpr.
func (e mockExprTraversal) AsTraversal() hcl.Traversal {
	return e.Traversal
}

func MockExprList(exprs []hcl.Expression) hcl.Expression {
	return mockExprList{
		Exprs: exprs,
	}
}

type mockExprList struct {
	Exprs []hcl.Expression
}

func (e mockExprList) Value(ctx *hcl.EvalContext) (cty.Value, hcl.Diagnostics) {
	if len(e.Exprs) == 0 {
		return cty.ListValEmpty(cty.DynamicPseudoType), nil
	}
	vals := make([]cty.Value, 0, len(e.Exprs))
	var diags hcl.Diagnostics

	for _, expr := range e.Exprs {
		val, valDiags := expr.Value(ctx)
		diags = append(diags, valDiags...)
		vals = append(vals, val)
	}

	return cty.ListVal(vals), diags
}

func (e mockExprList) Variables() []hcl.Traversal {
	var traversals []hcl.Traversal
	for _, expr := range e.Exprs {
		traversals = append(traversals, expr.Variables()...)
	}
	return traversals
}

func (e mockExprList) Range() hcl.Range {
	return hcl.Range{
		Filename: "MockExprList",
	}
}

func (e mockExprList) StartRange() hcl.Range {
	return e.Range()
}

// Implementation for hcl.ExprList
func (e mockExprList) ExprList() []hcl.Expression {
	return e.Exprs
}

// MockAttrs constructs and returns a hcl.Attributes map with attributes
// derived from the given expression map.
//
// Each entry in the map becomes an attribute whose name is the key and
// whose expression is the value.
func MockAttrs(exprs map[string]hcl.Expression) hcl.Attributes {
	ret := make(hcl.Attributes)
	for name, expr := range exprs {
		ret[name] = &hcl.Attribute{
			Name: name,
			Expr: expr,
			Range: hcl.Range{
				Filename: "MockAttrs",
			},
			NameRange: hcl.Range{
				Filename: "MockAttrs",
			},
		}
	}
	return ret
}
