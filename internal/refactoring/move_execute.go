// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package refactoring

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/logging"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

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
func ApplyMoves(stmts []MoveStatement, state *states.State, providerFactory map[addrs.Provider]providers.Factory) (MoveResults, tfdiags.Diagnostics) {
	ret := makeMoveResults()

	if len(stmts) == 0 {
		return ret, nil
	}

	// The methodology here is to construct a small graph of all of the move
	// statements where the edges represent where a particular statement
	// is either chained from or nested inside the effect of another statement.
	// That then means we can traverse the graph in topological sort order
	// to gradually move objects through potentially multiple moves each.

	g := buildMoveStatementGraph(stmts)

	// If the graph is not valid the we will not take any action at all. The
	// separate validation step should detect this and return an error.
	if diags := validateMoveStatementGraph(g); diags.HasErrors() {
		log.Printf("[ERROR] ApplyMoves: %s", diags.ErrWithWarnings())
		return ret, nil
	}

	// The graph must be reduced in order for ReverseDepthFirstWalk to work
	// correctly, since it is built from following edges and can skip over
	// dependencies if there is a direct edge to a transitive dependency.
	g.TransitiveReduction()

	// The starting nodes are the ones that don't depend on any other nodes.
	startNodes := make(dag.Set, len(stmts))
	for _, v := range g.Vertices() {
		if len(g.DownEdges(v)) == 0 {
			startNodes.Add(v)
		}
	}

	if startNodes.Len() == 0 {
		log.Println("[TRACE] refactoring.ApplyMoves: No 'moved' statements to consider in this configuration")
		return ret, nil
	}

	log.Printf("[TRACE] refactoring.ApplyMoves: Processing 'moved' statements in the configuration\n%s", logging.Indent(g.String()))

	recordOldAddr := func(oldAddr, newAddr addrs.AbsResourceInstance) {
		if prevMove, exists := ret.Changes.GetOk(oldAddr); exists {
			// If the old address was _already_ the result of a move then
			// we'll replace that entry so that our results summarize a chain
			// of moves into a single entry.
			ret.Changes.Remove(oldAddr)
			oldAddr = prevMove.From
		}
		ret.Changes.Put(newAddr, MoveSuccess{
			From: oldAddr,
			To:   newAddr,
		})
	}
	recordBlockage := func(newAddr, wantedAddr addrs.AbsMoveable) {
		ret.Blocked.Put(newAddr, MoveBlocked{
			Wanted: wantedAddr,
			Actual: newAddr,
		})
	}

	crossTypeMover := &crossTypeMover{
		State:             state,
		ProviderFactories: providerFactory,
		ProviderCache:     make(map[addrs.Provider]providers.Interface),
	}

	var diags tfdiags.Diagnostics

	syncState := state.SyncWrapper()
	for _, v := range g.ReverseTopologicalOrder() {
		stmt := v.(*MoveStatement)

		for _, ms := range state.Modules {
			modAddr := ms.Addr

			// We don't yet know that the current module is relevant, and
			// we determine that differently for each the object kind.
			switch kind := stmt.ObjectKind(); kind {
			case addrs.MoveEndpointModule:
				// For a module endpoint we just try the module address
				// directly, and execute the moves if it matches.
				if newAddr, matches := modAddr.MoveDestination(stmt.From, stmt.To); matches {
					log.Printf("[TRACE] refactoring.ApplyMoves: %s has moved to %s", modAddr, newAddr)

					// If we already have a module at the new address then
					// we'll skip this move and let the existing object take
					// priority.
					if ms := state.Module(newAddr); ms != nil {
						log.Printf("[WARN] Skipped moving %s to %s, because there's already another module instance at the destination", modAddr, newAddr)
						recordBlockage(modAddr, newAddr)
						continue
					}

					// We need to visit all of the resource instances in the
					// module and record them individually as results.
					for _, rs := range ms.Resources {
						relAddr := rs.Addr.Resource
						for key := range rs.Instances {
							oldInst := relAddr.Instance(key).Absolute(modAddr)
							newInst := relAddr.Instance(key).Absolute(newAddr)
							recordOldAddr(oldInst, newInst)
						}
					}

					syncState.MoveModuleInstance(modAddr, newAddr)
					continue
				}
			case addrs.MoveEndpointResource:
				// For a resource endpoint we require an exact containing
				// module match, because by definition a matching resource
				// cannot be nested any deeper than that.
				if !stmt.From.SelectsModule(modAddr) {
					continue
				}

				// We then need to search each of the resources and resource
				// instances in the module.
				for _, rs := range ms.Resources {
					rAddr := rs.Addr
					if newAddr, matches := rAddr.MoveDestination(stmt.From, stmt.To); matches {
						log.Printf("[TRACE] refactoring.ApplyMoves: resource %s has moved to %s", rAddr, newAddr)

						// If we already have a resource at the new address then
						// we'll skip this move and let the existing object take
						// priority.
						if rs := state.Resource(newAddr); rs != nil {
							log.Printf("[WARN] Skipped moving %s to %s, because there's already another resource at the destination", rAddr, newAddr)
							recordBlockage(rAddr, newAddr)
							continue
						}

						crossTypeMove, prepareDiags := crossTypeMover.prepareCrossTypeMove(stmt, rAddr, newAddr)
						diags = diags.Append(prepareDiags)
						for key := range rs.Instances {
							oldInst := rAddr.Instance(key)
							newInst := newAddr.Instance(key)
							diags = diags.Append(crossTypeMove.applyCrossTypeMove(stmt, oldInst, newInst, syncState))
							recordOldAddr(oldInst, newInst)
						}
						state.MoveAbsResource(rAddr, newAddr)
						continue
					}
					for key := range rs.Instances {
						iAddr := rAddr.Instance(key)
						if newAddr, matches := iAddr.MoveDestination(stmt.From, stmt.To); matches {
							log.Printf("[TRACE] refactoring.ApplyMoves: resource instance %s has moved to %s", iAddr, newAddr)

							// If we already have a resource instance at the new
							// address then we'll skip this move and let the existing
							// object take priority.
							if is := state.ResourceInstance(newAddr); is != nil {
								log.Printf("[WARN] Skipped moving %s to %s, because there's already another resource instance at the destination", iAddr, newAddr)
								recordBlockage(iAddr, newAddr)
								continue
							}

							crossTypeMove, crossTypeMoveDiags := crossTypeMover.prepareCrossTypeMove(stmt, iAddr.ContainingResource(), newAddr.ContainingResource())
							diags = diags.Append(crossTypeMoveDiags)
							diags = diags.Append(crossTypeMove.applyCrossTypeMove(stmt, iAddr, newAddr, syncState))
							recordOldAddr(iAddr, newAddr)
							syncState.MoveResourceInstance(iAddr, newAddr)
							continue
						}
					}
				}
			default:
				panic(fmt.Sprintf("unhandled move object kind %s", kind))
			}
		}
	}
	syncState.Close()

	diags = diags.Append(crossTypeMover.close())
	return ret, diags
}

// buildMoveStatementGraph constructs a dependency graph of the given move
// statements, where the nodes are all pointers to statements in the given
// slice and the edges represent either chaining or nesting relationships.
//
// buildMoveStatementGraph doesn't do any validation of the graph, so it
// may contain cycles and other sorts of invalidity.
func buildMoveStatementGraph(stmts []MoveStatement) *dag.AcyclicGraph {
	g := &dag.AcyclicGraph{}
	for i := range stmts {
		// The graph nodes are pointers to the actual statements directly.
		g.Add(&stmts[i])
	}

	// Now we'll add the edges representing chaining and nesting relationships.
	// We assume that a reasonable configuration will have at most tens of
	// move statements and thus this N*M algorithm is acceptable.
	for dependerI := range stmts {
		depender := &stmts[dependerI]
		for dependeeI := range stmts {
			if dependerI == dependeeI {
				// skip comparing the statement to itself
				continue
			}
			dependee := &stmts[dependeeI]

			if statementDependsOn(depender, dependee) {
				g.Connect(dag.BasicEdge(depender, dependee))
			}
		}
	}

	return g
}

// statementDependsOn returns true if statement a depends on statement b;
// i.e. statement b must be executed before statement a.
func statementDependsOn(a, b *MoveStatement) bool {
	// chain-able moves are simple, as on the destination of one move could be
	// equal to the source of another.
	if a.From.CanChainFrom(b.To) {
		return true
	}

	// Statement nesting in more complex, as we have 8 possible combinations to
	// assess. Here we list all combinations, along with the statement which
	// must be executed first when one address is nested within another.
	// A.From  IsNestedWithin  B.From => A
	// A.From  IsNestedWithin  B.To   => B
	// A.To    IsNestedWithin  B.From => A
	// A.To    IsNestedWithin  B.To   => B
	// B.From  IsNestedWithin  A.From => B
	// B.From  IsNestedWithin  A.To   => A
	// B.To    IsNestedWithin  A.From => B
	// B.To    IsNestedWithin  A.To   => A
	//
	// Since we are only interested in checking if A depends on B, we only need
	// to check the 4 possibilities above which result in B being executed
	// first. If we're there's no dependency at all we can return immediately.
	if !(a.From.NestedWithin(b.To) || a.To.NestedWithin(b.To) ||
		b.From.NestedWithin(a.From) || b.To.NestedWithin(a.From)) {
		return false
	}

	// If a nested move has a dependency, we need to rule out the possibility
	// that this is a move inside a module only changing indexes. If an
	// ancestor module is only changing the index of a nested module, any
	// nested move statements are going to match both the From and To address
	// when the base name is not changing, causing a cycle in the order of
	// operations.

	// if A is not declared in an ancestor module, then we can't be nested
	// within a module index change.
	if len(a.To.Module()) >= len(b.To.Module()) {
		return true
	}
	// We only want the nested move statement to depend on the outer module
	// move, so we only test this in the reverse direction.
	if a.From.IsModuleReIndex(a.To) {
		return false
	}

	return true
}

// MoveResults describes the outcome of an ApplyMoves call.
type MoveResults struct {
	// Changes is a map from the unique keys of the final new resource
	// instance addresses to an object describing what changed.
	//
	// This includes one entry for each resource instance address that was
	// the destination of a move statement. It doesn't include resource
	// instances that were not affected by moves at all, but it does include
	// resource instance addresses that were "blocked" (also recorded in
	// BlockedAddrs) if and only if they were able to move at least
	// partially along a chain before being blocked.
	//
	// In the return value from ApplyMoves, all of the keys are guaranteed to
	// be unique keys derived from addrs.AbsResourceInstance values.
	Changes addrs.Map[addrs.AbsResourceInstance, MoveSuccess]

	// Blocked is a map from the unique keys of the final new
	// resource instances addresses to information about where they "wanted"
	// to move, but were blocked by a pre-existing object at the same address.
	//
	// "Blocking" can arise in unusual situations where multiple points along
	// a move chain were already bound to objects, and thus only one of them
	// can actually adopt the final position in the chain. It can also
	// occur in other similar situations, such as if a configuration contains
	// a move of an entire module and a move of an individual resource into
	// that module, such that the individual resource would collide with a
	// resource in the whole module that was moved.
	//
	// In the return value from ApplyMoves, all of the keys are guaranteed to
	// be unique keys derived from values of addrs.AbsMoveable types.
	Blocked addrs.Map[addrs.AbsMoveable, MoveBlocked]
}

func makeMoveResults() MoveResults {
	return MoveResults{
		Changes: addrs.MakeMap[addrs.AbsResourceInstance, MoveSuccess](),
		Blocked: addrs.MakeMap[addrs.AbsMoveable, MoveBlocked](),
	}
}

type MoveSuccess struct {
	From addrs.AbsResourceInstance
	To   addrs.AbsResourceInstance
}

type MoveBlocked struct {
	Wanted addrs.AbsMoveable
	Actual addrs.AbsMoveable
}

// AddrMoved returns true if and only if the given resource instance moved to
// a new address in the ApplyMoves call that the receiver is describing.
//
// If AddrMoved returns true, you can pass the same address to method OldAddr
// to find its original address prior to moving.
func (rs MoveResults) AddrMoved(newAddr addrs.AbsResourceInstance) bool {
	return rs.Changes.Has(newAddr)
}

// OldAddr returns the old address of the given resource instance address, or
// just returns back the same address if the given instance wasn't affected by
// any move statements.
func (rs MoveResults) OldAddr(newAddr addrs.AbsResourceInstance) addrs.AbsResourceInstance {
	change, ok := rs.Changes.GetOk(newAddr)
	if !ok {
		return newAddr
	}
	return change.From
}
