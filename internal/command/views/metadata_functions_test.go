// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/terminal"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

func TestMetadataFunctionsJSON_Functions(t *testing.T) {
	testCases := map[string]struct {
		functions map[string]function.Function
		want      string
	}{
		"minimal function": {
			functions: map[string]function.Function{
				"fun": function.New(&function.Spec{
					Type: function.StaticReturnType(cty.Bool),
				}),
			},
			want: `{"format_version":"1.0","function_signatures":{"fun":{"return_type":"bool"}}}`,
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			streams, done := terminal.StreamsForTesting(t)
			view := NewView(streams)
			view.Configure(&arguments.View{NoColor: true})
			v := NewMetadataFunctions(arguments.ViewJSON, view)

			v.Functions(tc.functions)

			output := done(t)

			// Assert contents
			gotStdout := strings.TrimSpace(output.Stdout())
			wantStdout := strings.TrimSpace(tc.want)
			if diff := cmp.Diff(wantStdout, gotStdout); diff != "" {
				t.Fatalf("unexpected diff in human output:\n%s", diff)
			}
			if output.Stderr() != "" {
				t.Fatalf("unexpected output in stderr:\n%s", output.Stderr())
			}
		})
	}
}

func TestMetadataFunctionsHuman_Diagnostics(t *testing.T) {
	testCases := map[string]struct {
		diags      tfdiags.Diagnostics
		wantStdout string
		wantStderr string
	}{
		"error": {
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Example error",
					"This is an example error message.",
				),
			},
			"",
			"Error: Example error\n\nThis is an example error message.\n\n",
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			streams, done := terminal.StreamsForTesting(t)
			view := NewView(streams)
			view.Configure(&arguments.View{NoColor: true})
			v := NewMetadataFunctions(arguments.ViewHuman, view)

			v.Diagnostics(tc.diags)

			output := done(t)

			// Assert contents
			gotStdout := strings.TrimSpace(output.Stdout())
			wantStdout := strings.TrimSpace(tc.wantStdout)
			if diff := cmp.Diff(wantStdout, gotStdout); diff != "" {
				t.Fatalf("unexpected diff in human output:\n%s", diff)
			}
			gotStderr := strings.TrimSpace(output.Stderr())
			wantStderr := strings.TrimSpace(tc.wantStderr)
			if diff := cmp.Diff(wantStderr, gotStderr); diff != "" {
				t.Fatalf("unexpected diff in human output:\n%s", diff)
			}
		})
	}
}

func TestMetadataFunctionsJSON_Diagnostics(t *testing.T) {
	testCases := map[string]struct {
		diags      tfdiags.Diagnostics
		wantStdout string
		wantStderr string
	}{
		"error": {
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Example error",
					"This is an example error message.",
				),
			},
			"",
			// Human output produced despite JSON view being used.
			"Error: Example error\n\nThis is an example error message.\n\n",
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			streams, done := terminal.StreamsForTesting(t)
			view := NewView(streams)
			view.Configure(&arguments.View{NoColor: true})
			v := NewMetadataFunctions(arguments.ViewJSON, view)

			v.Diagnostics(tc.diags)

			output := done(t)

			// Assert contents
			gotStdout := strings.TrimSpace(output.Stdout())
			wantStdout := strings.TrimSpace(tc.wantStdout)
			if diff := cmp.Diff(wantStdout, gotStdout); diff != "" {
				t.Fatalf("unexpected diff in JSON output:\n%s", diff)
			}
			gotStderr := strings.TrimSpace(output.Stderr())
			wantStderr := strings.TrimSpace(tc.wantStderr)
			if diff := cmp.Diff(wantStderr, gotStderr); diff != "" {
				t.Fatalf("unexpected diff in JSON output:\n%s", diff)
			}
		})
	}
}
