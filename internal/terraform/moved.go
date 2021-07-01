package terraform

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// movedStatement represents a single validated move statement discovered
// from the configuration.
type movedStatement struct {
	// BaseModule is the module where the move was declared, which
	// the From and To addresses are therefore relative to.
	BaseModule addrs.Module

	// From and To are the source and destination addresses for the move,
	// respectively. Both addresses are relative to the module given
	// in BaseModule.
	//
	// In objects returned from decodeModes, From and To are guaranteed to
	// be of the same address type. If they are addrs.ModuleInstance then
	// one or both might refer to a whole module rather than an instance
	// if the last step's instance key is addrs.NoKey.
	From, To addrs.Targetable

	// DeclRange is the source location of the configuration block
	// that this statement was derived from.
	DeclRange hcl.Range
}

// decodeMoves searches the given configuration for "moved" blocks,
// checks that each one matches our static validation rules, and then
// produces a list of moves to be applied.
//
// The rules for what is considered a valid move are defined only in
// terms of configuration, and then the separate logic for _applying_
// the moves must be able to handle anything returned from here
// without returning any additional errors (though possibly ignoring
// moves that have already been applied).
func decodeMoves(config *configs.Config, schemas *Schemas) ([]movedStatement, tfdiags.Diagnostics) {
	var stmts []movedStatement
	var diags tfdiags.Diagnostics

	moves := config.Module.Moved

	for _, mc := range moves {
		stmt, moreDiags := decodeSingleMove(config, mc, schemas)
		diags = diags.Append(moreDiags)
		if !diags.HasErrors() {
			// We only append valid moves to the result
			stmts = append(stmts, stmt)
		}
	}

	// Also collect from child configurations, so we'll
	// recursively visit the entire tree.
	for _, childCfg := range config.Children {
		moreStmts, moreDiags := decodeMoves(childCfg, schemas)
		stmts = append(stmts, moreStmts...)
		diags = diags.Append(moreDiags)
	}

	return stmts, diags
}

func decodeSingleMove(config *configs.Config, mc *configs.Moved, schemas *Schemas) (movedStatement, tfdiags.Diagnostics) {
	ret := movedStatement{
		BaseModule: config.Path,
		DeclRange:  mc.DeclRange,
	}
	var diags tfdiags.Diagnostics

	ret.From, ret.To, diags = unifyMoveAddrs(mc.From, mc.To)
	if diags.HasErrors() {
		return ret, diags
	}

	// Now we'll make sure that the given addresses actually make sense.
	// By the time we get here ret.From and ret.To are guaranteed by
	// unifyMoveAddrs to be of the same type, which makes things a little
	// easier here.
	fromRange := mc.From.SourceRange
	toRange := mc.To.SourceRange
	const invalidMove = "Invalid move"

	switch ret.From.AddrType() {
	case addrs.AbsResourceInstanceAddrType:
		from := ret.From.(addrs.AbsResourceInstance)
		to := ret.To.(addrs.AbsResourceInstance)

		// Can only move between resources of the same type
		if fromType, toType := from.Resource.Resource.Type, to.Resource.Resource.Type; fromType != toType {
			// TODO: Once we have a way for providers to declare
			// deprecation-related renames, check in "schemas"
			// to see if this change is permitted by one and
			// allow it if so.
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  invalidMove,
				Detail:   fmt.Sprintf("Can't move a resource of type %q to a resource of type %q.", fromType, toType),
				Subject:  toRange.ToHCL().Ptr(),
			})
		}

		fromModCfg, moreDiags := findMoveAddrModule(config, from.Module, fromRange)
		diags = diags.Append(moreDiags)
		if fromModCfg != nil {
			// If the module is still present in configuration then it
			// mustn't declare the from address as currently existing,
			// because that would be ambiguous.
			if rc := fromModCfg.Module.ResourceByAddr(from.Resource.Resource); rc != nil {
				if from.Resource.Key == addrs.NoKey {
					// A no-key instance is declared in the configuration
					// if the resource block has neither count nor for_each
					// set.
					if rc.Count == nil && rc.ForEach == nil {

					}
				}
			}
		}

	case addrs.AbsResourceAddrType:
		from := ret.From.(addrs.AbsResource)
		to := ret.To.(addrs.AbsResource)

		// Can only move between resources of the same type
		if fromType, toType := from.Resource.Type, to.Resource.Type; fromType != toType {
			// TODO: Once we have a way for providers to declare
			// deprecation-related renames, check in "schemas"
			// to see if this change is permitted by one and
			// allow it if so.
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  invalidMove,
				Detail:   fmt.Sprintf("Can't move a resource of type %q to a resource of type %q.", fromType, toType),
				Subject:  toRange.ToHCL().Ptr(),
			})
		}

	default:
		// Shouldn't get here, because the cases above are the
		// subset used by unifyMoveAddrs.
		panic(fmt.Sprintf("unexpected move address type %#v", ret.From.AddrType()))
	}

	return ret, diags
}

// unifyMoveAddrs is responsible for either making the two given addresses
// have the same address type or returning errors explaining why it cannot.
func unifyMoveAddrs(from, to *addrs.Target) (fromAddr, toAddr addrs.Targetable, diags tfdiags.Diagnostics) {
	fromType := from.Subject.AddrType()
	toType := to.Subject.AddrType()

	if fromType == toType {
		// Easy case: they already match!
		return from.Subject, to.Subject, nil
	}

	var retType addrs.TargetableAddrType
	switch {
	case fromType == addrs.AbsResourceInstanceAddrType || toType == addrs.AbsResourceInstanceAddrType:
		retType = addrs.AbsResourceAddrType
	case fromType == addrs.AbsResourceAddrType || toType == addrs.AbsResourceAddrType:
		retType = addrs.AbsResourceAddrType
	case fromType == addrs.ModuleInstanceAddrType || toType == addrs.ModuleInstanceAddrType:
		retType = addrs.ModuleInstanceAddrType
	case fromType == addrs.ModuleAddrType || toType == addrs.ModuleAddrType:
		// We only really want module instance addresses,
		// so we'll just force static module addresses
		// to be no-key module instances.
		retType = addrs.ModuleInstanceAddrType
	default:
		// Shouldn't get here because the above should be exhaustive.
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Unsupported move address type",
			Detail:   "Can move only from resource or module addresses.",
			Subject:  from.SourceRange.ToHCL().Ptr(),
		})
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Unsupported move address type",
			Detail:   "Can move only to resource or module addresses.",
			Subject:  to.SourceRange.ToHCL().Ptr(),
		})
		return from.Subject, to.Subject, diags
	}

	var moreDiags tfdiags.Diagnostics
	fromAddr, moreDiags = convertMoveAddr(from, "source", retType)
	diags = append(diags, moreDiags...)
	toAddr, moreDiags = convertMoveAddr(to, "destination", retType)
	diags = append(diags, moreDiags...)

	if diags.HasErrors() {
		return from.Subject, to.Subject, diags
	}
	return fromAddr, toAddr, diags
}

func convertMoveAddr(given *addrs.Target, which string, wantType addrs.TargetableAddrType) (addrs.Targetable, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	gotType := given.Subject.AddrType()
	if gotType == wantType {
		return given.Subject, diags
	}

	switch wantType {
	case addrs.AbsResourceInstanceAddrType:
		switch gotAddr := given.Subject.(type) {
		case addrs.AbsResource:
			// We can reinterpret a whole-resource address as a no-key instance
			return gotAddr.Instance(addrs.NoKey), diags
		default:
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  fmt.Sprintf("Invalid move %s", which),
				Detail:   fmt.Sprintf("The %s address must refer to a resource instance, to match the other given address.", which),
				Subject:  given.SourceRange.ToHCL().Ptr(),
			})
			return given.Subject, diags
		}
	case addrs.AbsResourceAddrType:
		// A whole resource is compatible only with another whole resource,
		// so if we get here then it's always wrong.
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  fmt.Sprintf("Invalid move %s", which),
			Detail:   fmt.Sprintf("The %s address must refer to a resource, to match the other given address.", which),
			Subject:  given.SourceRange.ToHCL().Ptr(),
		})
		return given.Subject, diags
	case addrs.ModuleInstanceAddrType:
		switch gotAddr := given.Subject.(type) {
		case addrs.Module:
			// We just represent all module addresses as
			// addrs.ModuleInstance, for simplicity's sake.
			return gotAddr.UnkeyedInstanceShim(), diags
		default:
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  fmt.Sprintf("Invalid move %s", which),
				Detail:   fmt.Sprintf("The %s address must refer to a module instance, to match the other given address.", which),
				Subject:  given.SourceRange.ToHCL().Ptr(),
			})
			return given.Subject, diags
		}
	default:
		// unifyMoveAddrs shouldn't pass any other types
		panic(fmt.Sprintf("unexpected address type %#v", wantType))
	}
}

// findMoveAddrModule walks through the module tree starting at "start" to
// find the descendent module at the given relative address.
//
// The result can be nil without any diagnostics if the path walks into
// a module call that isn't present in the configuration. That case is
// valid because the module path might still have resources tracked
// in the state if only recently removed from configuration.
//
// While walking it verifies that none of the steps traverse into a
// different module package, returning error diagnostics if so because
// such a traversal isn't allowed.
func findMoveAddrModule(start *configs.Config, addr addrs.ModuleInstance, addrRange tfdiags.SourceRange) (*configs.Config, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	current := start

	for _, step := range addr {
		next := current.Children[step.Name]
		if next == nil {
			// No longer configured, but that's okay because it's
			// allowed to move from or two an unconfigured
			// address: it might've existed previously and thus
			// still have resources tracked in the state.
			return nil, diags
		}

		if next.EntersNewPackage() {
			if !diags.HasErrors() {
				// Only return one error per address
				pkgAddr := next.SourceAddr
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Move crosses module package boundary",
					Detail:   fmt.Sprintf("Can't move objects to or from modules in the separate module package %q.", pkgAddr),
					Subject:  addrRange.ToHCL().Ptr(),
				})
			}
		}
		current = next
	}

	return current, diags
}
