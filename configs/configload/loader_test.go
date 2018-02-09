package configload

import (
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/hcl2/hcl"
	"github.com/zclconf/go-cty/cty"
)

func assertNoDiagnostics(t *testing.T, diags hcl.Diagnostics) bool {
	t.Helper()
	return assertDiagnosticCount(t, diags, 0)
}

func assertDiagnosticCount(t *testing.T, diags hcl.Diagnostics, want int) bool {
	t.Helper()
	if len(diags) != 0 {
		t.Errorf("wrong number of diagnostics %d; want %d", len(diags), want)
		for _, diag := range diags {
			t.Logf("- %s", diag)
		}
		return true
	}
	return false
}

func assertDiagnosticSummary(t *testing.T, diags hcl.Diagnostics, want string) bool {
	t.Helper()

	for _, diag := range diags {
		if diag.Summary == want {
			return false
		}
	}

	t.Errorf("missing diagnostic summary %q", want)
	for _, diag := range diags {
		t.Logf("- %s", diag)
	}
	return true
}

func assertResultDeepEqual(t *testing.T, got, want interface{}) bool {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("wrong result\ngot: %swant: %s", spew.Sdump(got), spew.Sdump(want))
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
