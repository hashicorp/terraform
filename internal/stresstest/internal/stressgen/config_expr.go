package stressgen

import (
	"crypto/md5"

	hcl "github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/tfdiags"
)

// ConfigExpr is an interface implemented by types that represent various
// kinds of expression that are relevant to our testing.
//
// Since stresstest is focused mainly on testing graph building and graph
// traversal behaviors, and not on expression evaluation details, we don't
// aim to cover every possible kind of expression here but should aim to model
// all kinds of expression that can contribute in some way to the graph shape.
type ConfigExpr interface {
	// BuildExpr builds the hclwrite representation of the recieving expression,
	// for inclusion in the generated configuration files.
	BuildExpr() *hclwrite.Expression

	// ExpectedValue returns the value this expression ought to return if
	// Terraform behaves correctly. This must be the specific, fully-known
	// value we expect to find in the final state, not any placeholder value
	// that might show up during planning if we were faking a computed resource
	// argument.
	ExpectedValue(reg *Registry) cty.Value
}

// ConfigExprConst is an implementation of ConfigExpr representing static,
// constant values.
type ConfigExprConst struct {
	Value cty.Value
}

var _ ConfigExpr = (*ConfigExprConst)(nil)

// BuildExpr implements ConfigExpr.BuildExpr
func (e *ConfigExprConst) BuildExpr() *hclwrite.Expression {
	return hclwrite.NewExpressionLiteral(e.Value)
}

// ExpectedValue implements ConfigExpr.ExpectedValue
func (e *ConfigExprConst) ExpectedValue(reg *Registry) cty.Value {
	return e.Value
}

// ConfigExprRef is an implementation of ConfigExpr representing a reference
// to some referencable object elsewhere in the configuration.
type ConfigExprRef struct {
	// Target is the object being referenced.
	Target addrs.Referenceable

	// Path is an optional extra set of path traversal steps into the object,
	// allowing for e.g. referring to an attribute of an object.
	Path cty.Path
}

// NewConfigExprRef constructs a new ConfigExprRef with the given base address
// and path.
func NewConfigExprRef(objAddr addrs.Referenceable, path cty.Path) *ConfigExprRef {
	return &ConfigExprRef{
		Target: objAddr,
		Path:   path,
	}
}

var _ ConfigExpr = (*ConfigExprRef)(nil)

// BuildExpr implements ConfigExpr.BuildExpr.
func (e *ConfigExprRef) BuildExpr() *hclwrite.Expression {
	// Walking backwards from an already-parsed traversal to the traversal it
	// came from is not something we typically do in normal Terraform use,
	// and so this is a pretty hacky implementation of it just mushing
	// together some utilities we have elsewhere. Perhaps we can improve on
	// this in future if we find other use-cases for doing stuff like this.
	str := e.Target.String()
	if len(e.Path) > 0 {
		// CAUTION! tfdiags.FormatCtyPath is intended for display to users
		// and doesn't guarantee to produce exactly-valid traversal source
		// code. However, it's currently good enough for our purposes here
		// because we're only using a subset of valid paths:
		// - we're not generating attribute names that require special quoting
		// - we're not trying to traverse through sets
		// - we're not trying to use unknown values in these paths
		// If any of these assumptions change in future then we might need
		// to seek a different approach here.
		pathStr := tfdiags.FormatCtyPath(e.Path)
		str = str + pathStr
	}
	traversal, diags := hclsyntax.ParseTraversalAbs([]byte(str), "", hcl.InitialPos)
	if diags.HasErrors() {
		panic("we generated an invalid traversal and thus can't parse it")
	}
	return hclwrite.NewExpressionAbsTraversal(traversal)
}

// ExpectedValue implements ConfigExpr.ExpectedValue by wrapping
// Registry.RefValue.
func (e *ConfigExprRef) ExpectedValue(reg *Registry) cty.Value {
	return reg.RefValue(e.Target, e.Path)
}

// ConfigExprForEach is a specialized implementation of ConfigExpr focused on
// the problem of generating valid for_each arguments.
//
// It wraps zero or more other expressions, which it assumes will return
// either strings or string-convertable values. The ConfigExprForEach result
// is a mapping with constant keys but possibly-variable values.
type ConfigExprForEach struct {
	Exprs map[string]ConfigExpr
}

var _ ConfigExpr = (*ConfigExprForEach)(nil)

// BuildExpr implements ConfigExpr.BuildExpr.
func (e *ConfigExprForEach) BuildExpr() *hclwrite.Expression {
	// hclwrite doesn't currently have any built-in support for generating an
	// object constructor expression whose attribute values are themselves
	// arbitrary expressions, so we'll construct it manually as a raw token
	// sequence instead.
	var tokens hclwrite.Tokens
	tokens = append(tokens, &hclwrite.Token{
		Type:  hclsyntax.TokenOBrace,
		Bytes: []byte{'{'},
	})
	tokens = append(tokens, &hclwrite.Token{
		Type:  hclsyntax.TokenNewline,
		Bytes: []byte{'\n'},
	})
	for k, expr := range e.Exprs {
		tokens = append(tokens, hclwrite.TokensForValue(cty.StringVal(k))...)
		tokens = append(tokens, &hclwrite.Token{
			Type:  hclsyntax.TokenEqual,
			Bytes: []byte{'='},
		})
		exprTokens := expr.BuildExpr().BuildTokens(nil)
		tokens = append(tokens, exprTokens...)
		tokens = append(tokens, &hclwrite.Token{
			Type:  hclsyntax.TokenComma,
			Bytes: []byte{','},
		})
		tokens = append(tokens, &hclwrite.Token{
			Type:  hclsyntax.TokenNewline,
			Bytes: []byte{'\n'},
		})
	}
	tokens = append(tokens, &hclwrite.Token{
		Type:  hclsyntax.TokenCBrace,
		Bytes: []byte{'}'},
	})
	return hclwrite.NewExpressionRaw(tokens)
}

// ExpectedValue implements ConfigExpr.ExpectedValue.
func (e *ConfigExprForEach) ExpectedValue(reg *Registry) cty.Value {
	attrs := make(map[string]cty.Value, len(e.Exprs))
	for k, expr := range e.Exprs {
		attrs[k] = expr.ExpectedValue(reg)
	}
	return cty.ObjectVal(attrs)
}

// ConfigExprCount is a specialized implementation of ConfigExpr focus on
// generating small integers value for "count" arguments.
//
// Because we're typically using randomly-generated strings as our placeholder
// values in generated configuration, this expression does something rather
// bizarre to derive a hopefully-unbiased random number from a random string:
// it takes an MD5 hash of the string, takes the first nybble of that hash,
// and uses it as a number between 0 and 15. That's not a realistic thing that
// someone would typically do in a real module, but our goal here is to test
// various different graph shapes rather than to test expression evaluation.
//
// The result of ConfigExprCount is always a whole number between zero and
// fifteen, as long as the expression it is wrapping is a string.
type ConfigExprCount struct {
	Expr ConfigExpr
}

var _ ConfigExpr = (*ConfigExprCount)(nil)

// BuildExpr implements ConfigExpr.BuildExpr.
func (e *ConfigExprCount) BuildExpr() *hclwrite.Expression {
	// hclwrite doesn't currently have any built-in support for generating a
	// complex bunch of nested function calls, so we'll construct it manually
	// as a raw token sequence instead.
	// The result we're trying to build here is:
	// parseint(substr(md5(input), 0, 1), 16)
	// ...where 'input' is whatever tokens e.Expr represents.

	var tokens hclwrite.Tokens
	tokens = append(tokens, &hclwrite.Token{
		Type:  hclsyntax.TokenIdent,
		Bytes: []byte("parseint"),
	})
	tokens = append(tokens, &hclwrite.Token{
		Type:  hclsyntax.TokenOParen,
		Bytes: []byte{'('},
	})
	tokens = append(tokens, &hclwrite.Token{
		Type:  hclsyntax.TokenIdent,
		Bytes: []byte("substr"),
	})
	tokens = append(tokens, &hclwrite.Token{
		Type:  hclsyntax.TokenOParen,
		Bytes: []byte{'('},
	})
	tokens = append(tokens, &hclwrite.Token{
		Type:  hclsyntax.TokenIdent,
		Bytes: []byte("md5"),
	})
	tokens = append(tokens, &hclwrite.Token{
		Type:  hclsyntax.TokenOParen,
		Bytes: []byte{'('},
	})
	exprTokens := e.Expr.BuildExpr().BuildTokens(nil)
	tokens = append(tokens, exprTokens...)
	tokens = append(tokens, &hclwrite.Token{
		Type:  hclsyntax.TokenCParen,
		Bytes: []byte{')'},
	})
	tokens = append(tokens, &hclwrite.Token{
		Type:  hclsyntax.TokenComma,
		Bytes: []byte{','},
	})
	tokens = append(tokens, &hclwrite.Token{
		Type:  hclsyntax.TokenNumberLit,
		Bytes: []byte{'0'},
	})
	tokens = append(tokens, &hclwrite.Token{
		Type:  hclsyntax.TokenComma,
		Bytes: []byte{','},
	})
	tokens = append(tokens, &hclwrite.Token{
		Type:  hclsyntax.TokenNumberLit,
		Bytes: []byte{'1'},
	})
	tokens = append(tokens, &hclwrite.Token{
		Type:  hclsyntax.TokenCParen,
		Bytes: []byte{')'},
	})
	tokens = append(tokens, &hclwrite.Token{
		Type:  hclsyntax.TokenComma,
		Bytes: []byte{','},
	})
	tokens = append(tokens, &hclwrite.Token{
		Type:  hclsyntax.TokenNumberLit,
		Bytes: []byte{'1', '6'},
	})
	tokens = append(tokens, &hclwrite.Token{
		Type:  hclsyntax.TokenCParen,
		Bytes: []byte{')'},
	})
	return hclwrite.NewExpressionRaw(tokens)
}

// ExpectedValue implements ConfigExpr.ExpectedValue.
func (e *ConfigExprCount) ExpectedValue(reg *Registry) cty.Value {
	// Here we mimic in Go the same operation that BuildExpr implemented in
	// the Terraform language. As with the BuildExpr-generated expression,
	// we assume that the input is a known, non-null string.

	strVal := e.Expr.ExpectedValue(reg)
	str := strVal.AsString()
	hash := md5.Sum([]byte(str))

	// We take the high-order nybble of the first octet, to mimic what the
	// Terraform language implementation does with its string manipulation.
	result := int64(hash[0] >> 4)
	return cty.NumberIntVal(result)
}

// ConfigExprCountIndexString is a specialized implementation of ConfigExpr
// which wraps the expression "count.index" in a string interpolation expression
// so that the result will be a string, so we can preserve our assumption that
// we primarily pass strings around in our random configurations.
//
// We use this only in one special situation: when Namespace.GenerateExpression
// would've otherwise returned a direct reference to count.index, we substitute
// this expression type instead so that the result will be a string.
type ConfigExprCountIndexString struct {
}

var _ ConfigExpr = (*ConfigExprCountIndexString)(nil)

// BuildExpr implements ConfigExpr.BuildExpr.
func (e *ConfigExprCountIndexString) BuildExpr() *hclwrite.Expression {
	var tokens hclwrite.Tokens
	tokens = append(tokens, &hclwrite.Token{
		Type:  hclsyntax.TokenOQuote,
		Bytes: []byte{'"'},
	})
	tokens = append(tokens, &hclwrite.Token{
		Type:  hclsyntax.TokenStringLit,
		Bytes: []byte(`index `),
	})
	tokens = append(tokens, &hclwrite.Token{
		Type:  hclsyntax.TokenTemplateInterp,
		Bytes: []byte{'$', '{'},
	})
	tokens = append(tokens, &hclwrite.Token{
		Type:  hclsyntax.TokenIdent,
		Bytes: []byte("count"),
	})
	tokens = append(tokens, &hclwrite.Token{
		Type:  hclsyntax.TokenDot,
		Bytes: []byte{'.'},
	})
	tokens = append(tokens, &hclwrite.Token{
		Type:  hclsyntax.TokenIdent,
		Bytes: []byte("index"),
	})
	tokens = append(tokens, &hclwrite.Token{
		Type:  hclsyntax.TokenTemplateSeqEnd,
		Bytes: []byte{'}'},
	})
	tokens = append(tokens, &hclwrite.Token{
		Type:  hclsyntax.TokenCQuote,
		Bytes: []byte{'"'},
	})
	return hclwrite.NewExpressionRaw(tokens)
}

// ExpectedValue implements ConfigExpr.ExpectedValue.
func (e *ConfigExprCountIndexString) ExpectedValue(reg *Registry) cty.Value {
	numVal := reg.RefValue(addrs.CountAttr{Name: "index"}, nil)
	strVal, err := convert.Convert(numVal, cty.String)
	if err != nil {
		// This should never happen if the registry is populated with sensible
		// values for count.index.
		panic("count.index predicted value cannot convert to string")
	}
	return cty.StringVal("index " + strVal.AsString())
}
