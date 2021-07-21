package refactoring

import (
	"fmt"

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
//
// ApplyMoves expects exclusive access to the given state while it's running.
// Don't read or write any part of the state structure until ApplyMoves returns.
func ApplyMoves(stmts []MoveStatement, state *states.State) map[addrs.UniqueKey]MoveResult {
	// The methodology here is to construct a small graph of all of the move
	// statements where the edges represent where a particular statement
	// is either chained from or nested inside the effect of another statement.
	// That then means we can traverse the graph in topological sort order
	// to gradually move objects through potentially multiple moves each.

	g := buildMoveStatementGraph(stmts)

	// If there are any cycles in the graph then we'll not take any action
	// at all. The separate validation step should detect this and return
	// an error.
	if len(g.Cycles()) != 0 {
		return nil
	}

	// The starting nodes are the ones that don't depend on any other nodes.
	startNodes := make(dag.Set, len(stmts))
	for _, v := range g.Vertices() {
		if len(g.UpEdges(v)) == 0 {
			startNodes.Add(v)
		}
	}

	results := make(map[addrs.UniqueKey]MoveResult)
	g.DepthFirstWalk(startNodes, func(v dag.Vertex, depth int) error {
		stmt := v.(*MoveStatement)

		for _, ms := range state.Modules {
			modAddr := ms.Addr
			if !stmt.From.SelectsModule(modAddr) {
				continue
			}

			// We now know that the current module is relevant but what
			// we'll do with it depends on the object kind.
			switch kind := stmt.ObjectKind(); kind {
			case addrs.MoveEndpointModule:
				// For a module endpoint we just try the module address
				// directly.
				if newAddr, matches := modAddr.MoveDestination(stmt.From, stmt.To); matches {
					// We need to visit all of the resource instances in the
					// module and record them individually as results.
					for _, rs := range ms.Resources {
						relAddr := rs.Addr.Resource
						for key := range rs.Instances {
							oldInst := relAddr.Instance(key).Absolute(modAddr)
							newInst := relAddr.Instance(key).Absolute(newAddr)
							result := MoveResult{
								From: oldInst,
								To:   newInst,
							}
							results[oldInst.UniqueKey()] = result
							results[newInst.UniqueKey()] = result
						}
					}

					state.MoveModuleInstance(modAddr, newAddr)
					continue
				}
			case addrs.MoveEndpointResource:
				// For a resource endpoint we need to search each of the
				// resources and resource instances in the module.
				for _, rs := range ms.Resources {
					rAddr := rs.Addr
					if newAddr, matches := rAddr.MoveDestination(stmt.From, stmt.To); matches {
						for key := range rs.Instances {
							oldInst := rAddr.Instance(key)
							newInst := newAddr.Instance(key)
							result := MoveResult{
								From: oldInst,
								To:   newInst,
							}
							results[oldInst.UniqueKey()] = result
							results[newInst.UniqueKey()] = result
						}
						state.MoveAbsResource(rAddr, newAddr)
						continue
					}
					for key := range rs.Instances {
						iAddr := rAddr.Instance(key)
						if newAddr, matches := iAddr.MoveDestination(stmt.From, stmt.To); matches {
							result := MoveResult{From: iAddr, To: newAddr}
							results[iAddr.UniqueKey()] = result
							results[newAddr.UniqueKey()] = result

							state.MoveAbsResourceInstance(iAddr, newAddr)
							continue
						}
					}
				}
			default:
				panic(fmt.Sprintf("unhandled move object kind %s", kind))
			}
		}

		return nil
	})

	// FIXME: In the case of either chained or nested moves, "results" will
	// be left in a pretty interesting shape where the "old" address will
	// refer to a result that describes only the first step, while the "new"
	// address will refer to a result that describes only the last step.
	// To make that actually useful we'll need a different strategy where
	// the result describes the _effective_ source and destination, skipping
	// over any intermediate steps we took to get there, so that ultimately
	// we'll have enough information to annotate items in the plan with the
	// addresses the originally moved from.

	return results
}

// buildMoveStatementGraph constructs a dependency graph of the given move
// statements, where the nodes are all pointers to statements in the given
// slice and the edges represent either chaining or nesting relationships.
//
// buildMoveStatementGraph doesn't do any validation of the graph, so it
// may contain cycles and other sorts of invalidity.
func buildMoveStatementGraph(stmts []MoveStatement) *dag.AcyclicGraph {
	g := &dag.AcyclicGraph{}
	for _, stmt := range stmts {
		// The graph nodes are pointers to the actual statements directly.
		g.Add(&stmt)
	}

	// Now we'll add the edges representing chaining and nesting relationships.
	// We assume that a reasonable configuration will have at most tens of
	// move statements and thus this N*M algorithm is acceptable.
	for dependerI := range stmts {
		depender := &stmts[dependerI]
		for dependeeI := range stmts {
			dependee := &stmts[dependeeI]
			dependeeTo := dependee.To
			dependerFrom := depender.From
			if dependerFrom.CanChainFrom(dependeeTo) || dependerFrom.NestedWithin(dependeeTo) {
				g.Connect(dag.BasicEdge(depender, dependee))
			}
		}
	}

	return g
}
