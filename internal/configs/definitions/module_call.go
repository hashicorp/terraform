// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package definitions

import (
	"github.com/hashicorp/hcl/v2"

	"github.com/hashicorp/terraform/internal/addrs"
)

// ModuleCall represents a "module" block in a module or file.
type ModuleCall struct {
	Name string

	SourceAddr      addrs.ModuleSource
	SourceAddrRaw   string
	SourceAddrRange hcl.Range
	SourceSet       bool

	Config hcl.Body

	Version VersionConstraint

	Count   hcl.Expression
	ForEach hcl.Expression

	Providers []PassedProviderConfig

	DependsOn []hcl.Traversal

	DeclRange hcl.Range
}

// EntersNewPackage returns true if this call is to an external module, either
// directly via a remote source address or indirectly via a registry source
// address.
//
// Other behaviors in Terraform may treat package crossings as a special
// situation, because that indicates that the caller and callee can change
// independently of one another and thus we should disallow using any features
// where the caller assumes anything about the callee other than its input
// variables, required provider configurations, and output values.
func (mc *ModuleCall) EntersNewPackage() bool {
	return ModuleSourceAddrEntersNewPackage(mc.SourceAddr)
}

// ModuleSourceAddrEntersNewPackage returns true if the given source address
// represents a call to an external module package.
func ModuleSourceAddrEntersNewPackage(addr addrs.ModuleSource) bool {
	switch addr.(type) {
	case nil:
		// There are only two situations where we should get here:
		// - We've been asked about the source address of the root module,
		//   which is always nil.
		// - We've been asked about a ModuleCall that is part of the partial
		//   result of a failed decode.
		// The root module exists outside of all module packages, so we'll
		// just return false for that case. For the error case it doesn't
		// really matter what we return as long as we don't panic, because
		// we only make a best-effort to allow careful inspection of objects
		// representing invalid configuration.
		return false
	case addrs.ModuleSourceLocal:
		// Local source addresses are the only address type that remains within
		// the same package.
		return false
	default:
		// All other address types enter a new package.
		return true
	}
}
