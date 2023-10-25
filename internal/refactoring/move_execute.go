// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package refactoring

import (
	"fmt"
	"log"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
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
func ApplyMoves(moves *Moves, config *configs.Config, state *states.State, getProvider func(addrs.AbsProviderConfig) providers.Interface) {
	if len(moves.Statements) == 0 {
		return
	}

	// The methodology here is to construct a small graph of all of the move
	// statements where the edges represent where a particular statement
	// is either chained from or nested inside the effect of another statement.
	// That then means we can traverse the graph in topological sort order
	// to gradually move objects through potentially multiple moves each.

	g := buildMoveStatementGraph(moves.Statements)

	// If the graph is not valid the we will not take any action at all. The
	// separate validation step should detect this and return an error.
	if diags := validateMoveStatementGraph(g); diags.HasErrors() {
		log.Printf("[ERROR] ApplyMoves: %s", diags.ErrWithWarnings())
		return
	}

	// The graph must be reduced in order for ReverseDepthFirstWalk to work
	// correctly, since it is built from following edges and can skip over
	// dependencies if there is a direct edge to a transitive dependency.
	g.TransitiveReduction()

	// The starting nodes are the ones that don't depend on any other nodes.
	startNodes := make(dag.Set, len(moves.Statements))
	for _, v := range g.Vertices() {
		if len(g.DownEdges(v)) == 0 {
			startNodes.Add(v)
		}
	}

	if startNodes.Len() == 0 {
		log.Println("[TRACE] refactoring.ApplyMoves: No 'moved' statements to consider in this configuration")
		return
	}

	log.Printf("[TRACE] refactoring.ApplyMoves: Processing 'moved' statements in the configuration\n%s", logging.Indent(g.String()))
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
						moves.RecordBlockage(modAddr, newAddr)
						continue
					}

					// We need to visit all of the resource instances in the
					// module and record them individually as results.
					for _, rs := range ms.Resources {
						relAddr := rs.Addr.Resource
						for key := range rs.Instances {
							oldInst := relAddr.Instance(key).Absolute(modAddr)
							newInst := relAddr.Instance(key).Absolute(newAddr)
							moves.RecordMove(oldInst, newInst)
						}
					}

					state.MoveModuleInstance(modAddr, newAddr)
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
							moves.RecordBlockage(rAddr, newAddr)
							continue
						}

						crossTypeMove, crossTypeMovedRequired := initialiseCrossTypeMove(stmt, rAddr, newAddr, state, config, getProvider)
						if crossTypeMovedRequired {
							for key := range rs.Instances {
								oldInst := rAddr.Instance(key)
								newInst := newAddr.Instance(key)
								crossTypeMove.completeCrossTypeMove(stmt, oldInst, newInst, state)
							}
						}
						moves.Diags = moves.Diags.Append(crossTypeMove.diags)

						for key := range rs.Instances {
							oldInst := rAddr.Instance(key)
							newInst := newAddr.Instance(key)
							moves.RecordMove(oldInst, newInst)
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
								moves.RecordBlockage(iAddr, newAddr)
								continue
							}

							crossTypeMove, crossTypeMovedRequired := initialiseCrossTypeMove(stmt, rAddr, newAddr.ContainingResource(), state, config, getProvider)
							if crossTypeMovedRequired {
								crossTypeMove.completeCrossTypeMove(stmt, iAddr, newAddr, state)
							}
							moves.Diags = moves.Diags.Append(crossTypeMove.diags)

							moves.RecordMove(iAddr, newAddr)
							state.MoveAbsResourceInstance(iAddr, newAddr)
							continue
						}
					}
				}
			default:
				panic(fmt.Sprintf("unhandled move object kind %s", kind))
			}
		}
	}
}

type crossTypeMove struct {
	targetProvider              providers.Interface
	targetProviderAddr          addrs.AbsProviderConfig
	targetResourceSchema        *configschema.Block
	targetResourceSchemaVersion uint64

	sourceResourceSchema *configschema.Block
	sourceProviderAddr   addrs.AbsProviderConfig

	diags tfdiags.Diagnostics
}

func initialiseCrossTypeMove(stmt *MoveStatement, source, target addrs.AbsResource, state *states.State, config *configs.Config, getProvider func(addrs.AbsProviderConfig) providers.Interface) (crossTypeMove, bool) {
	var crossTypeMove crossTypeMove

	targetModule := config.DescendentForInstance(target.Module).Module
	targetResourceConfig := targetModule.ResourceByAddr(target.Resource)
	targetProviderAddr := targetResourceConfig.Provider

	crossTypeMove.sourceProviderAddr = state.Resource(source).ProviderConfig

	if targetProviderAddr.Equals(crossTypeMove.sourceProviderAddr.Provider) {
		if source.Resource.Type == target.Resource.Type {
			return crossTypeMove, false
		}
	}

	crossTypeMove.targetProviderAddr = addrs.AbsProviderConfig{
		Module:   target.Module.Module(),
		Provider: targetProviderAddr,
	}

	if targetResourceConfig.ProviderConfigRef != nil {
		crossTypeMove.targetProviderAddr.Alias = targetResourceConfig.ProviderConfigRef.Alias
	}

	sourceProvider := getProvider(crossTypeMove.sourceProviderAddr)
	crossTypeMove.targetProvider = getProvider(crossTypeMove.targetProviderAddr)

	if sourceProvider == nil {
		panic(fmt.Errorf("move source provider %s is not available", crossTypeMove.sourceProviderAddr))
	}

	if crossTypeMove.targetProvider == nil {
		panic(fmt.Errorf("move target provider %s is not available", crossTypeMove.targetProviderAddr))
	}

	targetSchema := crossTypeMove.targetProvider.GetProviderSchema()
	crossTypeMove.diags = crossTypeMove.diags.Append(targetSchema.Diagnostics)
	if targetSchema.Diagnostics.HasErrors() {
		return crossTypeMove, false
	}

	sourceSchema := sourceProvider.GetProviderSchema()
	crossTypeMove.diags = crossTypeMove.diags.Append(sourceSchema.Diagnostics)
	if sourceSchema.Diagnostics.HasErrors() {
		return crossTypeMove, false
	}

	if !targetSchema.ServerCapabilities.MoveResourceState {
		crossTypeMove.diags = crossTypeMove.diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Unsupported `moved` across resource types",
			Detail:   fmt.Sprintf("The provider %q does not support moved operations across resource types and providers.", targetProviderAddr.Type),
			Subject:  stmt.DeclRange.ToHCL().Ptr(),
		})
		return crossTypeMove, false
	}

	crossTypeMove.sourceResourceSchema, _ = sourceSchema.SchemaForResourceAddr(source.Resource)
	crossTypeMove.targetResourceSchema, crossTypeMove.targetResourceSchemaVersion = targetSchema.SchemaForResourceAddr(target.Resource)
	return crossTypeMove, true
}

func (crossTypeMove crossTypeMove) completeCrossTypeMove(stmt *MoveStatement, source, target addrs.AbsResourceInstance, state *states.State) {

	sourceInstance := state.ResourceInstance(source)
	value, err := sourceInstance.Current.Decode(crossTypeMove.sourceResourceSchema.ImpliedType())
	if err != nil {
		crossTypeMove.diags = crossTypeMove.diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Failed to decode source value",
			Detail:   fmt.Sprintf("Terraform failed to decode the value in state for %s: %v. This is a bug in Terraform; Please report it.", source.String(), err),
			Subject:  stmt.DeclRange.ToHCL().Ptr(),
		})
		return
	}

	resp := crossTypeMove.targetProvider.MoveResourceState(providers.MoveResourceStateRequest{
		SourceProviderAddress: crossTypeMove.sourceProviderAddr.String(),
		SourceTypeName:        source.Resource.Resource.Type,
		SourceSchemaVersion:   int64(sourceInstance.Current.SchemaVersion),
		SourceState:           value.Value,
		TargetTypeName:        target.Resource.Resource.Type,
	})
	crossTypeMove.diags = crossTypeMove.diags.Append(resp.Diagnostics)
	if resp.Diagnostics.HasErrors() {
		return
	}

	if resp.TargetSource == cty.NilVal {
		crossTypeMove.diags = crossTypeMove.diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Provider returned invalid value",
			Detail:   fmt.Sprintf("The provider returned an invalid value during an across type move operation to %s. This is a bug in the relevant provider; Please report it.", target),
			Subject:  stmt.DeclRange.ToHCL().Ptr(),
		})
	}

	newValue := &states.ResourceInstanceObject{
		Value:               resp.TargetSource,
		Private:             value.Private,
		Status:              value.Status,
		Dependencies:        value.Dependencies,
		CreateBeforeDestroy: value.CreateBeforeDestroy,
	}

	data, err := newValue.Encode(crossTypeMove.targetResourceSchema.ImpliedType(), crossTypeMove.targetResourceSchemaVersion)
	if err != nil {
		crossTypeMove.diags = crossTypeMove.diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Failed to encode source value",
			Detail:   fmt.Sprintf("Terraform failed to encode the value in state for %s: %v. This is a bug in Terraform; Please report it.", source.String(), err),
			Subject:  stmt.DeclRange.ToHCL().Ptr(),
		})
		return
	}

	// Finally, overwrite the value in state for the source address so that when
	// it is moved it has the correct value in the new place.
	state.SyncWrapper().SetResourceInstanceCurrent(source, data, crossTypeMove.targetProviderAddr)
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
