// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package configload

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
)

func assertNoDiagnostics(t *testing.T, diags hcl.Diagnostics) bool {
	t.Helper()
	return assertDiagnosticCount(t, diags, 0)
}

func assertDiagnosticCount(t *testing.T, diags hcl.Diagnostics, want int) bool {
	t.Helper()
	if len(diags) != want {
		t.Errorf("wrong number of diagnostics %d; want %d", len(diags), want)
		for _, diag := range diags {
			t.Logf("- %s", diag)
		}
		return true
	}
	return false
}
func assertResultCtyEqual(t *testing.T, got, want cty.Value) bool {
	t.Helper()
	if !got.RawEquals(want) {
		t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, want)
		return true
	}
	return false
}
