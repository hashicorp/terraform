package stressgen

import (
	"math/rand"
)

// newRand is a convenience wrapper around our common operation of constructing
// a random source with a particular seed and then wrapping it in a *rand.Rand
// object for more convenient use.
func newRand(seed int64) *rand.Rand {
	return rand.New(rand.NewSource(seed))
}

// generateRepetitionArgs uses the given random number generator to populate
// either a for_each expression, a count expression, or neither.
//
// At least one of the two return values will always be nil. If both are nil,
// the decision is to use neither of the repetition arguments.
func generateRepetitionArgs(rnd *rand.Rand, ns *Namespace) (*ConfigExprForEach, *ConfigExprCount) {
	// We support all three of the repetition modes for modules here: for_each
	// over a map, count with a number, and single-instance mode. However,
	// the rest of our generation strategy here works only with strings and
	// so we need to do some trickery here to produce suitable inputs for
	// the repetition arguments while still having them generate references
	// sometimes, because the repetition arguments play an important role in
	// constructing the dependency graph.
	// We achieve this as follows:
	// - for for_each, we generate a map with a random number of
	//   randomly-generated keys where each of the values is an expression
	//   randomly generated in our usual way.
	// - for count, we generate a random expression in the usual way, assume
	//   that the result will be convertable to a string (because that's our
	//   current standard) and apply some predictable string functions to it
	//   in order to deterministically derive a number.
	// Both cases therefore allow for the meta-argument to potentially depend
	// on other objects in the configuration, even though our current model
	// only allows for string dependencies directly.

	const (
		chooseSingleInstance int = 0
		chooseForEach        int = 1
		chooseCount          int = 2
	)
	which := decideIndex(rnd, []int{
		chooseSingleInstance: 4,
		chooseForEach:        2,
		chooseCount:          2,
	})
	switch which {
	case chooseSingleInstance:
		return nil, nil
	case chooseForEach:
		// We need to generate some randomly-selected instance keys, and then
		// associate each one with a randomly-selected expression.
		n := rnd.Intn(9)
		forEach := &ConfigExprForEach{
			Exprs: make(map[string]ConfigExpr, n),
		}
		for i := 0; i < n; i++ {
			k := ns.GenerateShortModifierName(rnd)
			expr := ns.GenerateExpression(rnd)
			forEach.Exprs[k] = expr
		}
		return forEach, nil
	case chooseCount:
		// We need to randomly select a source expression and then wrap it
		// in our special ConfigExprCount type to make it appear as a
		// randomly-chosen small integer instead of a string.
		expr := ns.GenerateExpression(rnd)
		return nil, &ConfigExprCount{Expr: expr}
	default:
		// This suggests either a bug in decideIndex or in our call
		// to decideIndex.
		panic("invalid decision")
	}

}
