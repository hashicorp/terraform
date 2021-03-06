package exprstress

import (
	"fmt"
	"math/rand"

	"github.com/zclconf/go-cty/cty"
)

// This file contains the definition of a generator and some general
// generator combinators that we use elsewhere. There are also generators
// in other files in this package; in particular, each Expression
// implementation will typically have at least one associated generator
// or generator constructor, with function names that start with "generate".

// expressionGenerator defines the signature of an expression generator
// function. We have this as a separate type only to help us define combinator
// functions to help with operations that support complex combinations of
// different values.
type expressionGenerator func(rand *rand.Rand) Expression

func generateAny(gens ...expressionGenerator) expressionGenerator {
	return func(rand *rand.Rand) Expression {
		n := rand.Intn(len(gens))
		return gens[n](rand)
	}
}

var testStrings = []string{
	"",
	"bleep",
	"bloop",
	"foo",
	"bar",
	"baz",
	"beep",
	"boop",
	"doo doo doo lah lah lah lah",
}

func randomString(rand *rand.Rand) string {
	// Since our exprstress checks are focused on metadata about the results
	// rather than the results themselves, there's no need to be _super_
	// random in our string values, but we will choose randomly from a
	// small set just because it's harder for humans doing debugging to read
	// and think about an expression where all of the values are the same.
	n := rand.Intn(len(testStrings))
	return testStrings[n]
}

func randomBool(rand *rand.Rand) bool {
	n := rand.Intn(2)
	return n == 1
}

// randomPrimitiveType returns one of cty's three primitive types, chosen at
// random.
func randomPrimitiveType(rand *rand.Rand) cty.Type {
	n := rand.Intn(3)
	switch n {
	case 0:
		return cty.String
	case 1:
		return cty.Number
	default:
		return cty.Bool
	}
}

// randomLiteralType returns a randomly-chosen type which belongs to the
// subset of types that randomLiteralValueOfType can accept without
// panicking.
func randomLiteralType(rand *rand.Rand) cty.Type {
	// We give a one in 20 chance of returning each of the two
	// structural types, and return a primitive type otherwise.
	// We want to give preference to generating relatively-shallow data
	// structures because literal deeply nested structures are not very
	// common in real-world configurations and relatively expensive for us
	// to try to evaluate against them.
	n := rand.Intn(20)
	switch {
	case n < 1: // tuple type
		n := rand.Intn(15)
		if n == 0 {
			return cty.EmptyTuple
		}
		etys := make([]cty.Type, n)
		for i := range etys {
			etys[i] = randomLiteralType(rand)
		}
		return cty.Tuple(etys)
	case n < 2: // object type
		n := rand.Intn(len(testStrings) - 1)
		if n == 0 {
			return cty.EmptyObject
		}
		atys := make(map[string]cty.Type, n)
		for i := 0; i < n; i++ {
			k := testStrings[i+1]
			atys[k] = randomLiteralType(rand)
		}
		return cty.Object(atys)
	default:
		return randomPrimitiveType(rand)
	}
}

// randomLiteralValueOfType returns a randomly-chosen non-null literal value of
// the given type which belongs to the subset of values that
// hclwrite.TokensForValue can faithfully serialize, or panics if the
// given type isn't one that has a literal syntax available.
func randomLiteralValueOfType(ty cty.Type, rand *rand.Rand) cty.Value {
	switch {
	case ty == cty.String:
		return cty.StringVal(randomString(rand))
	case ty == cty.Number:
		n := rand.Intn(105) - 5 // some chance of being a negative number
		return cty.NumberIntVal(int64(n))
	case ty == cty.Bool:
		return cty.BoolVal(randomBool(rand))
	case ty.IsObjectType():
		atys := ty.AttributeTypes()
		if len(atys) == 0 {
			return cty.EmptyObjectVal
		}
		attrs := make(map[string]cty.Value)
		for n, aty := range atys {
			attrs[n] = randomLiteralValueOfType(aty, rand)
		}
		return cty.ObjectVal(attrs)
	case ty.IsTupleType():
		etys := ty.TupleElementTypes()
		if len(etys) == 0 {
			return cty.EmptyTupleVal
		}
		elems := make([]cty.Value, len(etys))
		for i, ety := range etys {
			elems[i] = randomLiteralValueOfType(ety, rand)
		}
		return cty.TupleVal(elems)
	default:
		panic(fmt.Sprintf("there is no literal syntax to produce a value of type %#v", ty))
	}
}
