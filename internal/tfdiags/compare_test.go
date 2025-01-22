// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1
package tfdiags

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/hcl/v2"
)

func TestDiagnosticComparer(t *testing.T) {

	baseError := hcl.Diagnostic{
		Severity: hcl.DiagError,
		Summary:  "error",
		Detail:   "this is an error",
	}

	cases := map[string]struct {
		diag1      Diagnostic
		diag2      Diagnostic
		expectDiff bool
	}{
		"reports that identical diagnostics match": {
			diag1:      hclDiagnostic{&baseError},
			diag2:      hclDiagnostic{&baseError},
			expectDiff: false,
		},
		"reports that diagnostics don't match if severity differs": {
			diag1: hclDiagnostic{&baseError},
			diag2: func() Diagnostic {
				d := baseError
				d.Severity = hcl.DiagWarning
				return hclDiagnostic{&d}
			}(),
			expectDiff: true,
		},
		"reports that diagnostics don't match if summary differs": {
			diag1: hclDiagnostic{&baseError},
			diag2: func() Diagnostic {
				d := baseError
				d.Summary = "altered summary"
				return hclDiagnostic{&d}
			}(),
			expectDiff: true,
		},
		"reports that diagnostics don't match if detail differs": {
			diag1: hclDiagnostic{&baseError},
			diag2: func() Diagnostic {
				d := baseError
				d.Detail = "altered detail"
				return hclDiagnostic{&d}
			}(),
			expectDiff: true,
		},
		"reports that diagnostics don't match if attribute path differs": {
			diag1: func() Diagnostic {
				return AttributeValue(Error, "summary here", "detail here", cty.Path{cty.GetAttrStep{Name: "foobar1"}})
			}(),
			diag2: func() Diagnostic {
				return AttributeValue(Error, "summary here", "detail here", cty.Path{cty.GetAttrStep{Name: "foobar2"}})
			}(),
			expectDiff: true,
		},
		"reports that diagnostics don't match if attribute path missing from one differs": {
			diag1: func() Diagnostic {
				return AttributeValue(Error, "summary here", "detail here", cty.Path{cty.GetAttrStep{Name: "foobar1"}})
			}(),
			diag2: func() Diagnostic {
				d := hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "summary here",
					Detail:   "detail here",
				}
				return hclDiagnostic{&d}
			}(),
			expectDiff: true,
		},
	}

	for tn, tc := range cases {
		t.Run(tn, func(t *testing.T) {
			output := cmp.Diff(tc.diag1, tc.diag2, DiagnosticComparer)

			diffFound := output != ""
			if diffFound && !tc.expectDiff {
				t.Fatalf("unexpected diff detected:\n%s", output)
			}
			if !diffFound && tc.expectDiff {
				t.Fatal("expected a diff but none was detected")
			}
		})
	}

}
