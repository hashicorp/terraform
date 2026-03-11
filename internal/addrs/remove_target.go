// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package addrs

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Like MoveEndpoint, RemoveTarget is a wrapping struct that captures the result
// of decoding an HCL traversal representing a relative path from the current
// module to a removeable object.
//
// Remove targets are somewhat simpler than move endpoints, in that they deal
// only with resources and modules defined in configuration, not instances of
// those objects as recorded in state. We are therefore able to determine the
// ConfigMoveable up front, since specifying any resource or module instance key
// in a removed block is invalid.
//
// An interesting quirk of RemoveTarget is that RelSubject denotes a
// configuration object that, if the removed block is valid, should no longer
// exist in configuration. This "last known address" is used to locate and delete
// the appropriate state objects, or, in the case in which the user has forgotten
// to remove the object from configuration, to report the address of that block
// in an error diagnostic.
type RemoveTarget struct {
	// SourceRange is the location of the target address in configuration.
	SourceRange tfdiags.SourceRange

	// RelSubject, like MoveEndpoint's relSubject, abuses an absolute address
	// type to represent a relative address.
	RelSubject ConfigMoveable
}

func (t *RemoveTarget) ObjectKind() RemoveTargetKind {
	return removeTargetKind(t.RelSubject)
}

func (t *RemoveTarget) String() string {
	if t.ObjectKind() == RemoveTargetModule {
		return t.RelSubject.(Module).String()
	} else if t.ObjectKind() == RemoveTargetResource {
		return t.RelSubject.(ConfigResource).String()
	}
	// No other valid address types
	panic("Usupported remove target kind")
}

func (t *RemoveTarget) Equal(other *RemoveTarget) bool {
	switch {
	case (t == nil) != (other == nil):
		return false
	case t == nil:
		return true
	default:
		// We can safely compare string representations, since the Subject is a
		// simple module or resource address.
		return t.String() == other.String() && t.SourceRange == other.SourceRange
	}
}

func ParseRemoveTarget(traversal hcl.Traversal) (*RemoveTarget, tfdiags.Diagnostics) {
	path, remain, diags := parseModulePrefix(traversal)
	if diags.HasErrors() {
		return nil, diags
	}

	rng := tfdiags.SourceRangeFromHCL(traversal.SourceRange())

	if len(remain) == 0 {
		return &RemoveTarget{
			RelSubject:  path,
			SourceRange: rng,
		}, diags
	}

	rAddr, moreDiags := parseConfigResourceUnderModule(path, remain)
	diags = diags.Append(moreDiags)
	if diags.HasErrors() {
		return nil, diags
	}

	if rAddr.Resource.Mode == DataResourceMode {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Data source address not allowed",
			Detail:   "Data sources are never destroyed, so they are not valid targets of removed blocks. To remove the data source from state, remove the data source block from configuration.",
			Subject:  rng.ToHCL().Ptr(),
		})
	}

	return &RemoveTarget{
		RelSubject:  rAddr,
		SourceRange: rng,
	}, diags
}
