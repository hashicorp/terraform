package stressgen

import (
	"math/rand"
)

// GenerateConfigObject generates and returns a single configuration object,
// using the given random number generator to choose what kind of object
// to return and how to populate it.
func GenerateConfigObject(rnd *rand.Rand, ns *Namespace) ConfigObject {
	const (
		chooseVariable int = 0
		chooseOutput   int = 1
		chooseResource int = 2
		chooseModule   int = 3
	)

	// We adjust the chooseModule weight depending on how deep we are in the
	// module tree, because we want to prevent very deeply nested module
	// trees and also encourage examples with only shallow nesting.
	moduleAdj := len(ns.ModuleAddr)
	if len(ns.ModuleAddr) == 0 {
		moduleAdj = 1
	}

	which := decideIndex(rnd, []int{
		chooseVariable: 3,
		chooseOutput:   3,
		chooseResource: 8, // includes both managed and data resources, with an independent random choice
		chooseModule:   2 / moduleAdj,
	})

	switch which {
	case chooseVariable:
		return GenerateConfigVariable(rnd, ns)
	case chooseOutput:
		return GenerateConfigOutput(rnd, ns)
	case chooseResource:
		return GenerateConfigResource(rnd, ns)
	case chooseModule:
		return GenerateConfigModuleCall(rnd, ns)
	default:
		// This suggests either a bug in decideIndex or in our call
		// to decideIndex.
		panic("invalid decision")
	}
}
