package refactoring

import (
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// ValidateMoves tests whether all of the given move statements comply with
// both the single-statement validation rules and the "big picture" rules
// that constrain statements in relation to one another.
//
// The validation rules are primarily in terms of the configuration, but
// ValidateMoves also takes the expander that resulted from creating a plan
// so that it can see which instances are defined for each module and resource,
// to precisely validate move statements involving specific-instance addresses.
//
// Because validation depends on the planning result but move execution must
// happen _before_ planning, we have the unusual situation where sibling
// function ApplyMoves must run before ValidateMoves and must therefore
// tolerate and ignore any invalid statements. The plan walk will then
// construct in incorrect plan (because it'll be starting from the wrong
// prior state) but ValidateMoves will block actually showing that invalid
// plan to the user.
func ValidateMoves(stmts []MoveStatement, rootCfg *configs.Config, declaredInsts instances.Set) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	g := buildMoveStatementGraph(stmts)

	// We need to track the absolute versions of our endpoint addresses in
	// order to detect when there are ambiguous moves.
	type AbsMoveEndpoint struct {
		Other     addrs.AbsMoveable
		StmtRange tfdiags.SourceRange
	}
	stmtFrom := map[addrs.UniqueKey]AbsMoveEndpoint{}
	stmtTo := map[addrs.UniqueKey]AbsMoveEndpoint{}

	for _, stmt := range stmts {
		// Earlier code that constructs MoveStatement values should ensure that
		// both stmt.From and stmt.To always belong to the same statement and
		// thus to the same module.
		stmtMod, fromCallSteps := stmt.From.ModuleCallTraversals()
		_, toCallSteps := stmt.To.ModuleCallTraversals()

		modCfg := rootCfg.Descendent(stmtMod)
		if pkgAddr := callsThroughModulePackage(modCfg, fromCallSteps); pkgAddr != nil {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Cross-package move statement",
				Detail: fmt.Sprintf(
					"This statement declares a move from an object declared in external module package %q. Move statements can be only within a single module package.",
					pkgAddr,
				),
				Subject: stmt.DeclRange.ToHCL().Ptr(),
			})
		}
		if pkgAddr := callsThroughModulePackage(modCfg, toCallSteps); pkgAddr != nil {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Cross-package move statement",
				Detail: fmt.Sprintf(
					"This statement declares a move to an object declared in external module package %q. Move statements can be only within a single module package.",
					pkgAddr,
				),
				Subject: stmt.DeclRange.ToHCL().Ptr(),
			})
		}

		for _, modInst := range declaredInsts.InstancesForModule(stmtMod) {

			absFrom := stmt.From.InModuleInstance(modInst)
			absTo := stmt.To.InModuleInstance(modInst)
			fromKey := absFrom.UniqueKey()
			toKey := absTo.UniqueKey()

			if fromKey == toKey {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Redundant move statement",
					Detail: fmt.Sprintf(
						"This statement declares a move from %s to the same address, which is the same as not declaring this move at all.",
						absFrom,
					),
					Subject: stmt.DeclRange.ToHCL().Ptr(),
				})
				continue
			}

			var noun string
			var shortNoun string
			switch absFrom.(type) {
			case addrs.ModuleInstance:
				noun = "module instance"
				shortNoun = "instance"
			case addrs.AbsModuleCall:
				noun = "module call"
				shortNoun = "call"
			case addrs.AbsResourceInstance:
				noun = "resource instance"
				shortNoun = "instance"
			case addrs.AbsResource:
				noun = "resource"
				shortNoun = "resource"
			default:
				// The above cases should cover all of the AbsMoveable types
				panic("unsupported AbsMoveable address type")
			}

			// It's invalid to have a move statement whose "from" address
			// refers to something that is still declared in the configuration.
			if moveableObjectExists(absFrom, declaredInsts) {
				conflictRange, hasRange := movableObjectDeclRange(absFrom, rootCfg)
				declaredAt := ""
				if hasRange {
					// NOTE: It'd be pretty weird to _not_ have a range, since
					// we're only in this codepath because the plan phase
					// thought this object existed in the configuration.
					declaredAt = fmt.Sprintf(" at %s", conflictRange.StartString())
				}

				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Moved object still exists",
					Detail: fmt.Sprintf(
						"This statement declares a move from %s, but that %s is still declared%s.\n\nChange your configuration so that this %s will be declared as %s instead.",
						absFrom, noun, declaredAt, shortNoun, absTo,
					),
					Subject: stmt.DeclRange.ToHCL().Ptr(),
				})
			}

			// There can only be one destination for each source address.
			if existing, exists := stmtFrom[fromKey]; exists {
				if existing.Other.UniqueKey() != toKey {
					diags = diags.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Ambiguous move statements",
						Detail: fmt.Sprintf(
							"A statement at %s declared that %s moved to %s, but this statement instead declares that it moved to %s.\n\nEach %s can move to only one destination %s.",
							existing.StmtRange.StartString(), absFrom, existing.Other, absTo,
							noun, shortNoun,
						),
						Subject: stmt.DeclRange.ToHCL().Ptr(),
					})
				}
			} else {
				stmtFrom[fromKey] = AbsMoveEndpoint{
					Other:     absTo,
					StmtRange: stmt.DeclRange,
				}
			}

			// There can only be one source for each destination address.
			if existing, exists := stmtTo[toKey]; exists {
				if existing.Other.UniqueKey() != fromKey {
					diags = diags.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Ambiguous move statements",
						Detail: fmt.Sprintf(
							"A statement at %s declared that %s moved to %s, but this statement instead declares that %s moved there.\n\nEach %s can have moved from only one source %s.",
							existing.StmtRange.StartString(), existing.Other, absTo, absFrom,
							noun, shortNoun,
						),
						Subject: stmt.DeclRange.ToHCL().Ptr(),
					})
				}
			} else {
				stmtTo[toKey] = AbsMoveEndpoint{
					Other:     absFrom,
					StmtRange: stmt.DeclRange,
				}
			}

			// Resource types must match.
			if resourceTypesDiffer(absFrom, absTo) {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Resource type mismatch",
					Detail: fmt.Sprintf(
						"This statement declares a move from %s to %s, which is a %s of a different type.", absFrom, absTo, noun,
					),
				})
			}
		}
	}

	// If we're not already returning other errors then we'll also check for
	// and report cycles.
	//
	// Cycles alone are difficult to report in a helpful way because we don't
	// have enough context to guess the user's intent. However, some particular
	// mistakes that might lead to a cycle can also be caught by other
	// validation rules above where we can make better suggestions, and so
	// we'll use a cycle report only as a last resort.
	if !diags.HasErrors() {
		for _, cycle := range g.Cycles() {
			// Reporting cycles is awkward because there isn't any definitive
			// way to decide which of the objects in the cycle is the cause of
			// the problem. Therefore we'll just list them all out and leave
			// the user to figure it out. :(
			stmtStrs := make([]string, 0, len(cycle))
			for _, stmtI := range cycle {
				// move statement graph nodes are pointers to move statements
				stmt := stmtI.(*MoveStatement)
				stmtStrs = append(stmtStrs, fmt.Sprintf(
					"\n  - %s: %s → %s",
					stmt.DeclRange.StartString(),
					stmt.From.String(),
					stmt.To.String(),
				))
			}
			sort.Strings(stmtStrs) // just to make the order deterministic

			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Cyclic dependency in move statements",
				fmt.Sprintf(
					"The following chained move statements form a cycle, and so there is no final location to move objects to:%s\n\nA chain of move statements must end with an address that doesn't appear in any other statements, and which typically also refers to an object still declared in the configuration.",
					strings.Join(stmtStrs, ""),
				),
			))
		}
	}

	return diags
}

func moveableObjectExists(addr addrs.AbsMoveable, in instances.Set) bool {
	switch addr := addr.(type) {
	case addrs.ModuleInstance:
		return in.HasModuleInstance(addr)
	case addrs.AbsModuleCall:
		return in.HasModuleCall(addr)
	case addrs.AbsResourceInstance:
		return in.HasResourceInstance(addr)
	case addrs.AbsResource:
		return in.HasResource(addr)
	default:
		// The above cases should cover all of the AbsMoveable types
		panic("unsupported AbsMoveable address type")
	}
}

func resourceTypesDiffer(absFrom, absTo addrs.AbsMoveable) bool {
	switch absFrom := absFrom.(type) {
	case addrs.AbsMoveableResource:
		// addrs.UnifyMoveEndpoints guarantees that both addresses are of the
		// same kind, so at this point we can assume that absTo is also an
		// addrs.AbsResourceInstance or addrs.AbsResource.
		absTo := absTo.(addrs.AbsMoveableResource)
		return absFrom.AffectedAbsResource().Resource.Type != absTo.AffectedAbsResource().Resource.Type
	default:
		return false
	}
}

func movableObjectDeclRange(addr addrs.AbsMoveable, cfg *configs.Config) (tfdiags.SourceRange, bool) {
	switch addr := addr.(type) {
	case addrs.ModuleInstance:
		// For a module instance we're actually looking for the call that
		// declared it, which belongs to the parent module.
		// (NOTE: This assumes "addr" can never be the root module instance,
		// because the root module is never moveable.)
		parentAddr, callAddr := addr.Call()
		modCfg := cfg.DescendentForInstance(parentAddr)
		if modCfg == nil {
			return tfdiags.SourceRange{}, false
		}
		call := modCfg.Module.ModuleCalls[callAddr.Name]
		if call == nil {
			return tfdiags.SourceRange{}, false
		}

		// If the call has either count or for_each set then we'll "blame"
		// that expression, rather than the block as a whole, because it's
		// the expression that decides which instances are available.
		switch {
		case call.ForEach != nil:
			return tfdiags.SourceRangeFromHCL(call.ForEach.Range()), true
		case call.Count != nil:
			return tfdiags.SourceRangeFromHCL(call.Count.Range()), true
		default:
			return tfdiags.SourceRangeFromHCL(call.DeclRange), true
		}
	case addrs.AbsModuleCall:
		modCfg := cfg.DescendentForInstance(addr.Module)
		if modCfg == nil {
			return tfdiags.SourceRange{}, false
		}
		call := modCfg.Module.ModuleCalls[addr.Call.Name]
		if call == nil {
			return tfdiags.SourceRange{}, false
		}
		return tfdiags.SourceRangeFromHCL(call.DeclRange), true
	case addrs.AbsResourceInstance:
		modCfg := cfg.DescendentForInstance(addr.Module)
		if modCfg == nil {
			return tfdiags.SourceRange{}, false
		}
		rc := modCfg.Module.ResourceByAddr(addr.Resource.Resource)
		if rc == nil {
			return tfdiags.SourceRange{}, false
		}

		// If the resource has either count or for_each set then we'll "blame"
		// that expression, rather than the block as a whole, because it's
		// the expression that decides which instances are available.
		switch {
		case rc.ForEach != nil:
			return tfdiags.SourceRangeFromHCL(rc.ForEach.Range()), true
		case rc.Count != nil:
			return tfdiags.SourceRangeFromHCL(rc.Count.Range()), true
		default:
			return tfdiags.SourceRangeFromHCL(rc.DeclRange), true
		}
	case addrs.AbsResource:
		modCfg := cfg.DescendentForInstance(addr.Module)
		if modCfg == nil {
			return tfdiags.SourceRange{}, false
		}
		rc := modCfg.Module.ResourceByAddr(addr.Resource)
		if rc == nil {
			return tfdiags.SourceRange{}, false
		}
		return tfdiags.SourceRangeFromHCL(rc.DeclRange), true
	default:
		// The above cases should cover all of the AbsMoveable types
		panic("unsupported AbsMoveable address type")
	}
}

func callsThroughModulePackage(modCfg *configs.Config, callSteps []addrs.ModuleCall) addrs.ModuleSource {
	var sourceAddr addrs.ModuleSource
	current := modCfg
	for _, step := range callSteps {
		call := current.Module.ModuleCalls[step.Name]
		if call == nil {
			break
		}
		if call.EntersNewPackage() {
			sourceAddr = call.SourceAddr
		}
		current = modCfg.Children[step.Name]
		if current == nil {
			// Weird to have a call but not a config, but we'll tolerate
			// it to avoid crashing here.
			break
		}
	}
	return sourceAddr
}
