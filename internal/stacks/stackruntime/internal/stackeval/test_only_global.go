// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"context"
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

func (m *Main) resolveTestOnlyGlobalReference(ctx context.Context, addr stackaddrs.TestOnlyGlobal, rng tfdiags.SourceRange) (Referenceable, tfdiags.Diagnostics) {
	if m.testOnlyGlobals == nil {
		var diags tfdiags.Diagnostics
		// We don't seem to be running in a testing context, so we'll pretend
		// that test-only globals don't exist at all.
		//
		// This diagnostic is designed to resemble the one that
		// stackaddrs.ParseReference would return if given a traversal
		// that has no recognizable prefix, since this reference type should
		// behave as if it doesn't exist at all when we're not doing internal
		// testing.
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Reference to unknown symbol",
			Detail:   "There is no symbol _test_only_global defined in the current scope.",
			Subject:  rng.ToHCL().Ptr(),
		})
		return nil, diags
	}
	if _, exists := m.testOnlyGlobals[addr.Name]; !exists {
		var diags tfdiags.Diagnostics
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Reference to undefined test-only global",
			Detail:   fmt.Sprintf("Test-only globals are available here, but there's no definition for one named %q.", addr.Name),
			Subject:  rng.ToHCL().Ptr(),
		})
		return nil, diags
	}
	return &testOnlyGlobal{name: addr.Name, main: m}, nil
}

type testOnlyGlobal struct {
	name string
	main *Main
}

// ExprReferenceValue implements Referenceable.
func (g *testOnlyGlobal) ExprReferenceValue(ctx context.Context, phase EvalPhase) cty.Value {
	// By the time we get here we can assume that we represent an
	// actually-defined test-only global, because
	// Main.resolveTestOnlyGlobalReference checks that.
	return g.main.testOnlyGlobals[g.name]
}
