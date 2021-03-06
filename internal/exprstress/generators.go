package exprstress

import (
	"math/rand"

	"github.com/zclconf/go-cty/cty"
)

type exprFixed struct {
	Source   string
	Expected Expected
}

func (e exprFixed) BuildSource(w SourceWriter) {
	w.WriteString(e.Source)
}

func (e exprFixed) ExpectedResult() Expected {
	return e.Expected
}

func (e exprFixed) generator(rand *rand.Rand) Expression {
	return e
}

// fixedExpressions is a set of expressions that generate fixed outcomes.
//
// We create various subsets of this in other collections below, because many
// of these generators can be useful for a number of different purposes.
var fixedExpressions = []exprFixed{
	{
		`local.string`,
		Expected{
			Type: cty.String,
			Mode: SpecifiedValue,
		},
	},
	{
		`local.unknown.any`,
		Expected{
			Type: cty.DynamicPseudoType,
			Mode: UnknownValue,
		},
	},
	{
		`local.unknown.string`,
		Expected{
			Type: cty.String,
			Mode: UnknownValue,
		},
	},
	{
		`local.unknown.number`,
		Expected{
			Type: cty.Number,
			Mode: UnknownValue,
		},
	},
	{
		`local.unknown.bool`,
		Expected{
			Type: cty.Bool,
			Mode: UnknownValue,
		},
	},
	{
		`local.unknown.empty_object`,
		Expected{
			Type: cty.EmptyObject,
			Mode: UnknownValue,
		},
	},
	{
		`local.unknown.object`,
		Expected{
			Type: cty.Object(map[string]cty.Type{"a": cty.String}),
			Mode: UnknownValue,
		},
	},
	{
		`local.unknown.object.a`,
		Expected{
			Type: cty.String,
			Mode: UnknownValue,
		},
	},
}

func generateArithmeticOperand(depth int) expressionGenerator {
	generators := []expressionGenerator{
		generateExprLiteralOfType(cty.Number),
		generateAny(
			filterFixedExpressions(func(outcome Expected) bool {
				return outcome.CouldConvertTo(cty.Number) && (outcome.Mode == SpecifiedValue || outcome.Mode == UnknownValue)
			})...,
		),
	}
	if depth <= 8 {
		// We only generate recursive arithmetic operators up to a certain
		// depth, to avoid potential infinite recursion during generation.
		generators = append(generators, generateArithmeticOperator(depth))
	}
	return generateAny(generators...)
}

func filterFixedExpressions(cond func(outcome Expected) bool) []expressionGenerator {
	var ret []expressionGenerator
	for _, candidate := range fixedExpressions {
		if !cond(candidate.Expected) {
			continue
		}
		ret = append(ret, candidate.generator)
	}
	return ret
}
