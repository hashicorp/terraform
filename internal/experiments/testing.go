package experiments

import (
	"testing"
)

// OverrideForTesting temporarily overrides the global tables
// of experiments in order to allow for a predictable set when unit testing
// the experiments infrastructure code.
//
// The correct way to use this function is to defer a call to its result so
// that the original tables can be restored at the conclusion of the calling
// test:
//
//     defer experiments.OverrideForTesting(t, current, concluded)()
//
// This function modifies global variables that are normally fixed throughout
// our execution, so this function must not be called from non-test code and
// any test using it cannot safely run concurrently with other tests.
func OverrideForTesting(t *testing.T, current Set, concluded map[Experiment]string) func() {
	// We're not currently using the given *testing.T in here, but we're
	// requiring it anyway in case we might need it in future, and because
	// it hopefully reinforces that only test code should be calling this.

	realCurrents := currentExperiments
	realConcludeds := concludedExperiments
	currentExperiments = current
	concludedExperiments = concluded
	return func() {
		currentExperiments = realCurrents
		concludedExperiments = realConcludeds
	}
}
