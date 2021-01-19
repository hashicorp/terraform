package stressgen

import (
	"math/rand"
)

// decideBool is a helper for making a weighted decision about whether or not
// to do something. "percent" is the percentage likelihood that the result
// will be true, so setting it to 100 will make this function always return
// true and to 0 will make this function always return false.
//
// (In practice you'll want to choose a number somewhere in between, of course.)
func decideBool(rnd *rand.Rand, percent int) bool {
	n := rnd.Intn(100)
	return n < percent
}

// decideIndex is a helper for making a weighted decision between a number of
// items.
//
// Each element of weights is a weight for its index, and the result is an
// index into the weights slice indicating which one was chosen. The result
// will therefore always be a valid index into weights, and consequently
// also a valid index into some other slice of the same size.
//
// At least one of weights must be greater than zero and all of the weights
// must be positive. Otherwise, the function will either panic or return a
// nonsense result.
func decideIndex(rnd *rand.Rand, weights []int) int {
	total := 0
	for _, weight := range weights {
		total += weight
	}
	n := rnd.Intn(total)
	total = 0
	for i, weight := range weights {
		if weight == 0 {
			continue
		}
		if total+weight >= n {
			return i
		}
		total += weight
	}
	panic("incorrect arguments") // shouldn't get here if the arguments are reasonable
}
