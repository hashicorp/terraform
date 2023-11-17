// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package rpcapi

import (
	"fmt"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/rpcapi/terraform1"
	"github.com/hashicorp/terraform/internal/tfdiags"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"
)

func TestDiagnosticsToProto(t *testing.T) {
	tests := map[string]struct {
		Input tfdiags.Diagnostics
		Want  []*terraform1.Diagnostic
	}{
		"nil": {
			Input: nil,
			Want:  nil,
		},
		"empty": {
			Input: make(tfdiags.Diagnostics, 0, 5),
			Want:  nil,
		},
		"sourceless": {
			Input: tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Something annoying",
					"But I'll get over it.",
				),
			},
			Want: []*terraform1.Diagnostic{
				{
					Severity: terraform1.Diagnostic_ERROR,
					Summary:  "Something annoying",
					Detail:   "But I'll get over it.",
				},
			},
		},
		"warning": {
			Input: tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Warning,
					"I have a very bad feeling about this",
					"That's no moon; it's a space station.",
				),
			},
			Want: []*terraform1.Diagnostic{
				{
					Severity: terraform1.Diagnostic_WARNING,
					Summary:  "I have a very bad feeling about this",
					Detail:   "That's no moon; it's a space station.",
				},
			},
		},
		"with subject": {
			Input: tfdiags.Diagnostics{}.Append(
				&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Something annoying",
					Detail:   "But I'll get over it.",
					Subject: &hcl.Range{
						Filename: "git::https://example.com/foo.git",
						Start:    hcl.InitialPos,
						End: hcl.Pos{
							Byte:   2,
							Line:   3,
							Column: 4,
						},
					},
				},
			),
			Want: []*terraform1.Diagnostic{
				{
					Severity: terraform1.Diagnostic_ERROR,
					Summary:  "Something annoying",
					Detail:   "But I'll get over it.",
					Subject: &terraform1.SourceRange{
						SourceAddr: "git::https://example.com/foo.git",
						Start: &terraform1.SourcePos{
							Byte:   0,
							Line:   1,
							Column: 1,
						},
						End: &terraform1.SourcePos{
							Byte:   2,
							Line:   3,
							Column: 4,
						},
					},
				},
			},
		},
		"with subject and context": {
			Input: tfdiags.Diagnostics{}.Append(
				&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Something annoying",
					Detail:   "But I'll get over it.",
					Subject: &hcl.Range{
						Filename: "git::https://example.com/foo.git",
						Start:    hcl.InitialPos,
						End: hcl.Pos{
							Byte:   2,
							Line:   3,
							Column: 4,
						},
					},
					Context: &hcl.Range{
						Filename: "git::https://example.com/foo.git",
						Start:    hcl.InitialPos,
						End: hcl.Pos{
							Byte:   5,
							Line:   6,
							Column: 7,
						},
					},
				},
			),
			Want: []*terraform1.Diagnostic{
				{
					Severity: terraform1.Diagnostic_ERROR,
					Summary:  "Something annoying",
					Detail:   "But I'll get over it.",
					Subject: &terraform1.SourceRange{
						SourceAddr: "git::https://example.com/foo.git",
						Start: &terraform1.SourcePos{
							Byte:   0,
							Line:   1,
							Column: 1,
						},
						End: &terraform1.SourcePos{
							Byte:   2,
							Line:   3,
							Column: 4,
						},
					},
					Context: &terraform1.SourceRange{
						SourceAddr: "git::https://example.com/foo.git",
						Start: &terraform1.SourcePos{
							Byte:   0,
							Line:   1,
							Column: 1,
						},
						End: &terraform1.SourcePos{
							Byte:   5,
							Line:   6,
							Column: 7,
						},
					},
				},
			},
		},
		"with only severity and summary": {
			// This is the kind of degenerate diagnostic we produce when
			// we're just naively wrapping a Go error, as tends to arise
			// in providers that are just passing through their SDK errors.
			Input: tfdiags.Diagnostics{}.Append(
				fmt.Errorf("oh no bad"),
			),
			Want: []*terraform1.Diagnostic{
				{
					Severity: terraform1.Diagnostic_ERROR,
					Summary:  "oh no bad",
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := diagnosticsToProto(test.Input)
			want := test.Want

			if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
				t.Errorf("wrong result\n%s", diff)
			}
		})
	}
}
