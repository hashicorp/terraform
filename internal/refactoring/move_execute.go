package refactoring

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/states"
)

type MoveResult struct {
	From, To addrs.AbsResourceInstance
}

// ApplyMoves modifies in-place the given state object so that any existing
// objects that are matched by a "from" argument of one of the move statements
// will be moved to instead appear at the "to" argument of that statement.
//
// The result is a map from the unique key of each absolute address that was
// either the source or destination of a move to a MoveResult describing
// what happened at that address.
//
// ApplyMoves does not have any error situations itself, and will instead just
// ignore any unresolvable move statements. Validation of a set of moves is
// a separate concern applied to the configuration, because validity of
// moves is always dependent only on the configuration, not on the state.
func ApplyMoves(stmts []MoveStatement, state *states.State) map[addrs.UniqueKey]MoveResult {
	// The methodology here is to construct a small graph of all of the move
	// statements where the edges represent where a particular statement
	// is either chained from or nested inside the effect of another statement.
	// That then means we can traverse the graph in topological sort order
	// to gradually move objects through potentially multiple moves each.

	g := &dag.AcyclicGraph{}
	for _, stmt := range stmts {
		// The graph nodes are pointers to the actual statements directly.
		g.Add(&stmt)
	}

	// Now we'll add the edges representing chaining and nesting relationships.
	// We assume that a reasonable configuration will have at most tens of
	// move statements and thus this N*M algorithm is acceptable.
	for _, depender := range stmts {
		for _, dependee := range stmts {
			dependeeTo := dependee.To
			dependerFrom := depender.From
			if dependerFrom.CanChainFrom(dependeeTo) || dependerFrom.NestedWithin(dependeeTo) {
				g.Connect(dag.BasicEdge(depender, dependee))
			}
		}
	}

	// If there are any cycles in the graph then we'll not take any action
	// at all. The separate validation step should detect this and return
	// an error.
	if len(g.Cycles()) != 0 {
		return nil
	}

	// The starting nodes are the ones that don't depend on any other nodes.
	//startNodes := make(dag.Set, len(stmts))
	//g.DepthFirstWalk()

	return nil
}
