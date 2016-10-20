package terraform

import (
	"log"
)

// EvalCountFixZeroOneBoundaryGlobal is an EvalNode that fixes up the state
// when there is a resource count with zero/one boundary, i.e. fixing
// a resource named "aws_instance.foo" to "aws_instance.foo.0" and vice-versa.
//
// This works on the global state.
type EvalCountFixZeroOneBoundaryGlobal struct{}

// TODO: test
func (n *EvalCountFixZeroOneBoundaryGlobal) Eval(ctx EvalContext) (interface{}, error) {
	// Get the state and lock it since we'll potentially modify it
	state, lock := ctx.State()
	lock.Lock()
	defer lock.Unlock()

	// Prune the state since we require a clean state to work
	state.prune()

	// Go through each modules since the boundaries are restricted to a
	// module scope.
	for _, m := range state.Modules {
		if err := n.fixModule(m); err != nil {
			return nil, err
		}
	}

	return nil, nil
}

func (n *EvalCountFixZeroOneBoundaryGlobal) fixModule(m *ModuleState) error {
	// Counts keeps track of keys and their counts
	counts := make(map[string]int)
	for k, _ := range m.Resources {
		// Parse the key
		key, err := ParseResourceStateKey(k)
		if err != nil {
			return err
		}

		// Set the index to -1 so that we can keep count
		key.Index = -1

		// Increment
		counts[key.String()]++
	}

	// Go through the counts and do the fixup for each resource
	for raw, count := range counts {
		// Search and replace this resource
		search := raw
		replace := raw + ".0"
		if count < 2 {
			search, replace = replace, search
		}
		log.Printf("[TRACE] EvalCountFixZeroOneBoundaryGlobal: count %d, search %q, replace %q", count, search, replace)

		// Look for the resource state. If we don't have one, then it is okay.
		rs, ok := m.Resources[search]
		if !ok {
			continue
		}

		// If the replacement key exists, we just keep both
		if _, ok := m.Resources[replace]; ok {
			continue
		}

		m.Resources[replace] = rs
		delete(m.Resources, search)
	}

	return nil
}
