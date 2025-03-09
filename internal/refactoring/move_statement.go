// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package refactoring

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type MoveStatement struct {
	From, To  *addrs.MoveEndpointInModule
	DeclRange tfdiags.SourceRange

	// Provider is the provider configuration that applies to the "to" address
	// of this move. As in, the provider that will manage the resource after
	// it has been moved.
	//
	// This may be null if the "to" address points to a module instead of a
	// resource.
	Provider *addrs.AbsProviderConfig

	// Implied is true for statements produced by ImpliedMoveStatements, and
	// false for statements produced by FindMoveStatements.
	//
	// An "implied" statement is one that has no explicit "moved" block in
	// the configuration and was instead generated automatically based on a
	// comparison between current configuration and previous run state.
	// For implied statements, the DeclRange field contains the source location
	// of something in the source code that implied the statement, in which
	// case it would probably be confusing to show that source range to the
	// user, e.g. in an error message, without clearly mentioning that it's
	// related to an implied move statement.
	Implied bool
}

// FindMoveStatements recurses through the modules of the given configuration
// and returns a flat set of all "moved" blocks defined within, in a
// deterministic but undefined order.
func FindMoveStatements(rootCfg *configs.Config) []MoveStatement {
	return findMoveStatements(rootCfg, nil)
}

func findMoveStatements(cfg *configs.Config, into []MoveStatement) []MoveStatement {
	modAddr := cfg.Path
	for _, mc := range cfg.Module.Moved {
		fromAddr, toAddr := addrs.UnifyMoveEndpoints(modAddr, mc.From, mc.To)
		if fromAddr == nil || toAddr == nil {
			// Invalid combination should've been caught during original
			// configuration decoding, in the configs package.
			panic(fmt.Sprintf("incompatible move endpoints in %s", mc.DeclRange))
		}

		stmt := MoveStatement{
			From:      fromAddr,
			To:        toAddr,
			DeclRange: tfdiags.SourceRangeFromHCL(mc.DeclRange),
			Implied:   false,
		}

		// We have the statement, let's see if we should attach a provider to
		// it.
		if toResource, ok := mc.To.ConfigMoveable(addrs.RootModule).(addrs.ConfigResource); ok {
			// Only attach providers if we are moving resources, and we attach
			// the to resource provider from the config. We can retrieve the
			// from resource provider from the state later.
			modCfg := cfg.Descendant(toResource.Module)
			// It's possible that multiple refactorings have left a moved block
			// that points to a module which no longer exists. This may also be
			// a mistake, but the user will see the unexpected deletion in the
			// plan if it is.
			if modCfg != nil {
				resourceConfig := modCfg.Module.ResourceByAddr(toResource.Resource)
				if resourceConfig != nil {
					// Check the target resource config actually exists before we
					// try and extract the provider from them.

					stmt.Provider = &addrs.AbsProviderConfig{
						Module:   modAddr,
						Provider: resourceConfig.Provider,
					}

					if resourceConfig.ProviderConfigRef != nil {
						stmt.Provider.Alias = resourceConfig.ProviderConfigRef.Alias
					}
				}
			}
		}

		into = append(into, stmt)
	}

	for _, childCfg := range cfg.Children {
		into = findMoveStatements(childCfg, into)
	}

	return into
}

// ImpliedMoveStatements compares addresses in the given state with addresses
// in the given configuration and potentially returns additional MoveStatement
// objects representing moves we infer automatically, even though they aren't
// explicitly recorded in the configuration.
//
// We do this primarily for backward compatibility with behaviors of Terraform
// versions prior to introducing explicit "moved" blocks. Specifically, this
// function aims to achieve the same result as the "NodeCountBoundary"
// heuristic from Terraform v1.0 and earlier, where adding or removing the
// "count" meta-argument from an already-created resource can automatically
// preserve the zeroth or the NoKey instance, depending on the direction of
// the change. We do this only for resources that aren't mentioned already
// in at least one explicit move statement.
//
// As with the previous-version heuristics it replaces, this is a best effort
// and doesn't handle all situations. An explicit move statement is always
// preferred, but our goal here is to match exactly the same cases that the
// old heuristic would've matched, to retain compatibility for existing modules.
//
// We should think very hard before adding any _new_ implication rules for
// moved statements.
func ImpliedMoveStatements(rootCfg *configs.Config, prevRunState *states.State, explicitStmts []MoveStatement) []MoveStatement {
	return impliedMoveStatements(rootCfg, prevRunState, explicitStmts, nil)
}

func impliedMoveStatements(cfg *configs.Config, prevRunState *states.State, explicitStmts []MoveStatement, into []MoveStatement) []MoveStatement {
	modAddr := cfg.Path

	// There can be potentially many instances of the module, so we need
	// to consider each of them separately.
	for _, modState := range prevRunState.ModuleInstances(modAddr) {
		// What we're looking for here is either a no-key resource instance
		// where the configuration has count set or a zero-key resource
		// instance where the configuration _doesn't_ have count set.
		// If so, we'll generate a statement replacing no-key with zero-key or
		// vice-versa.
		for _, rState := range modState.Resources {
			rAddr := rState.Addr
			rCfg := cfg.Module.ResourceByAddr(rAddr.Resource)
			if rCfg == nil {
				// If there's no configuration at all then there can't be any
				// automatic move fixup to do.
				continue
			}
			approxSrcRange := tfdiags.SourceRangeFromHCL(rCfg.DeclRange)

			// NOTE: We're intentionally not checking to see whether the
			// "to" addresses in our implied statements already have
			// instances recorded in state, because ApplyMoves should
			// deal with such conflicts in a deterministic way for both
			// explicit and implicit moves, and we'd rather have that
			// handled all in one place.

			var fromKey, toKey addrs.InstanceKey

			switch {
			case rCfg.Count != nil:
				// If we have a count expression then we'll use _that_ as
				// a slightly-more-precise approximate source range.
				approxSrcRange = tfdiags.SourceRangeFromHCL(rCfg.Count.Range())

				if riState := rState.Instances[addrs.NoKey]; riState != nil {
					fromKey = addrs.NoKey
					toKey = addrs.IntKey(0)
				}
			case rCfg.Count == nil && rCfg.ForEach == nil: // no repetition at all
				if riState := rState.Instances[addrs.IntKey(0)]; riState != nil {
					fromKey = addrs.IntKey(0)
					toKey = addrs.NoKey
				}
			}

			if fromKey != toKey {
				// We mustn't generate an implied statement if the user already
				// wrote an explicit statement referring to this resource,
				// because they may wish to select an instance key other than
				// zero as the one to retain.
				if !haveMoveStatementForResource(rAddr, explicitStmts) {

					resource := cfg.Descendant(addrs.RootModule).Module.ResourceByAddr(rAddr.Resource)
					provider := &addrs.AbsProviderConfig{
						Module:   rAddr.Module.Module(),
						Provider: resource.Provider,
					}
					if resource.ProviderConfigRef != nil {
						provider.Alias = resource.ProviderConfigRef.Alias
					}

					into = append(into, MoveStatement{
						From:      addrs.ImpliedMoveStatementEndpoint(rAddr.Instance(fromKey), approxSrcRange),
						To:        addrs.ImpliedMoveStatementEndpoint(rAddr.Instance(toKey), approxSrcRange),
						Provider:  provider,
						DeclRange: approxSrcRange,
						Implied:   true,
					})
				}
			}
		}
	}

	for _, childCfg := range cfg.Children {
		into = impliedMoveStatements(childCfg, prevRunState, explicitStmts, into)
	}

	return into
}

func (s *MoveStatement) ObjectKind() addrs.MoveEndpointKind {
	// addrs.UnifyMoveEndpoints guarantees that both of our addresses have
	// the same kind, so we can just arbitrary use From and assume To will
	// match it.
	return s.From.ObjectKind()
}

// Name is used internally for displaying the statement graph
func (s *MoveStatement) Name() string {
	return fmt.Sprintf("%s->%s", s.From, s.To)
}

func haveMoveStatementForResource(addr addrs.AbsResource, stmts []MoveStatement) bool {
	// This is not a particularly optimal way to answer this question,
	// particularly since our caller calls this function in a loop already,
	// but we expect the total number of explicit statements to be small
	// in any reasonable Terraform configuration and so a more complicated
	// approach wouldn't be justified here.

	for _, stmt := range stmts {
		if stmt.From.SelectsResource(addr) {
			return true
		}
		if stmt.To.SelectsResource(addr) {
			return true
		}
	}
	return false
}
