package exprstress

import (
	"bytes"
	"io"
	"math/rand"
	"strings"

	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
)

// Expression is an interface implemented by various unexported types that
// represent different kinds of expression supported in the Terraform language.
//
// An Expression knows how to produce a Terraform language syntax
// representation of itself and how to describe its expected result value.
// Some expression types contain nested expressions, which will therefore
// have child expression syntax representations embedded in their own.
type Expression interface {
	// BuildSource writes to the given writer a series of bytes representing
	// a valid Terraform language expression.
	//
	// Each BuildSource implementation should behave as if it were generating
	// an entirely isolated expression, without any other expressions wrapping
	// it. For Expression implementations that wrap others, it's the parent's
	// responsibility to generate any necessary parentheses or punctuation
	// around a nested BuildSource call to ensure that the result is valid
	// and, where appropriate, gives operands the intended precedence.
	//
	// Package exprstress is primarily interested in the evaluation behavior
	// rather than the parsing behavior (parsing is already tested in HCL
	// itself) and so we don't typically try to generate multiple different
	// serializations that all produce the same evaluation effect, or worry
	// about making the generated string be formatted like a human might
	// write it.
	BuildSource(w SourceWriter)

	// ExpectedResults generates a description of the expected metadata for
	// the result of the expression.
	//
	// For Expression implementations that have other expressions embedded
	// in them, ExpectedResult should recursively call the downstream
	// ExpectedResult methods and then derive its own return value from those
	// using its knowledge about whichever expression operator or node it
	// is modeling.
	ExpectedResult() Expected
}

// ExpressionSourceString is a helper for capturing the result of an
// expression's BuildSource method into a string.
func ExpressionSourceString(expr Expression) string {
	var buf strings.Builder
	expr.BuildSource(&buf)
	return buf.String()
}

// ExpressionSourceBytes is a helper for capturing the result of an
// expression's BuildSource method into a byte slice.
func ExpressionSourceBytes(expr Expression) []byte {
	var buf bytes.Buffer
	expr.BuildSource(&buf)
	return buf.Bytes()
}

var topLevelGenerators = []expressionGenerator{
	// For the moment we only know how to build literals. This'll grow
	// later to include more interesting expression types too.
	generateExprLiteral(),
	generateArithmeticOperator(0),
}

// GenerateExpression uses the given random-number generator to generate an
// arbitrary Expression to potentially use as a test case.
func GenerateExpression(rand *rand.Rand) Expression {
	n := rand.Intn(len(topLevelGenerators))
	return topLevelGenerators[n](rand)
}

// SourceWriter is an interface used to append expression source code to a
// buffer. It's an aggregation of various interfaces and methods that
// standard library memory-backed writers typically offer, just to make it
// more convenient to write Expression.BuildSource implementations.
type SourceWriter interface {
	io.Writer
	io.StringWriter
	io.ByteWriter
	WriteRune(r rune) (n int, err error)
}

// exprLiteral is an Expression representing a constant value.
type exprLiteral struct {
	// Value is the value to return.
	//
	// Only the subset of values that are supported by hclwrite.TokensForValue
	// are allowed here; other values will cause BuildSource to panic.
	// Furthermore, TokensForValue sometimes represents a value using a
	// syntax that constructs a similar but non-identical type; don't use
	// such values in Value or else ExpectedResult will not declare the
	// correct result type.
	Value cty.Value
}

var exprLiteralNull = &exprLiteral{
	Value: cty.NullVal(cty.DynamicPseudoType),
}

func generateExprLiteral() expressionGenerator {
	return func(rand *rand.Rand) Expression {
		n := rand.Intn(10)
		if n < 2 {
			return exprLiteralNull
		}
		ty := randomLiteralType(rand)
		return generateExprLiteralOfType(ty)(rand)
	}
}

func generateExprLiteralOfType(ty cty.Type) expressionGenerator {
	return func(rand *rand.Rand) Expression {
		return &exprLiteral{
			Value: randomLiteralValueOfType(ty, rand),
		}
	}
}

func generateExprLiteralOfTypeOrNull(ty cty.Type) expressionGenerator {
	return func(rand *rand.Rand) Expression {
		n := rand.Intn(10)
		if n < 2 {
			return exprLiteralNull
		}
		return generateExprLiteralOfType(ty)(rand)
	}
}

func (e *exprLiteral) BuildSource(w SourceWriter) {
	toks := hclwrite.TokensForValue(e.Value)
	toks.WriteTo(w)
}

func (e *exprLiteral) ExpectedResult() Expected {
	// There's no literal syntax for writing an unknown value,
	// so an exprLiteral result must always be known, but can be null.
	// However, there's no literal syntax for a _typed_ null, so a null
	// value will always be of cty.DynamicPseudoType.
	// There's also no syntax for making a literal value be sensitive.

	var specialNum SpecialNumber

	v := e.Value
	switch {
	case v.IsNull():
		return Expected{
			Type: cty.DynamicPseudoType,
			Mode: NullValue,
		}
	default:
		switch {
		case v.RawEquals(cty.Zero) || v.RawEquals(cty.StringVal("0")):
			specialNum = NumberZero
		case v.RawEquals(cty.NumberIntVal(1)) || v.RawEquals(cty.StringVal("1")):
			specialNum = NumberOne
		case v.RawEquals(cty.NegativeInfinity) || v.RawEquals(cty.PositiveInfinity):
			specialNum = NumberInfinity
		}

		return Expected{
			Type:          v.Type(),
			Mode:          SpecifiedValue,
			SpecialNumber: specialNum,
		}
	}
}
