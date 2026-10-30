// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/terminal"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestStateIdentitiesJSON_Identities(t *testing.T) {
	testCases := map[string]struct {
		identities map[string]json.RawMessage
		wantStdout string
	}{
		"empty": {
			identities: map[string]json.RawMessage{},
			wantStdout: "{}\n",
		},
		"multiple identities": {
			identities: map[string]json.RawMessage{
				"test_instance.bar": json.RawMessage(`{"id":"my-bar-id"}`),
				"test_instance.foo": json.RawMessage(`{"id":"my-foo-id"}`),
			},
			wantStdout: `{
  "test_instance.bar": {
    "id": "my-bar-id"
  },
  "test_instance.foo": {
    "id": "my-foo-id"
  }
}
`,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			streams, done := terminal.StreamsForTesting(t)
			view := NewView(streams)
			view.Configure(&arguments.View{NoColor: true})
			v := NewStateIdentities(arguments.ViewJSON, view)

			v.Identities(tc.identities)

			output := done(t)
			if gotErr := output.Stderr(); gotErr != "" {
				t.Errorf("unexpected stderr output:\n%s", gotErr)
			}
			if diff := cmp.Diff(tc.wantStdout, output.Stdout()); diff != "" {
				t.Errorf("unexpected stdout diff:\n%s", diff)
			}
		})
	}
}

func TestStateIdentitiesJSON_Diagnostics(t *testing.T) {
	testCases := map[string]struct {
		diags      tfdiags.Diagnostics
		wantStdout string
		wantStderr string
	}{
		"no diags": {
			diags:      tfdiags.Diagnostics{},
			wantStdout: "",
			wantStderr: "",
		},
		"warning": {
			diags: tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Warning,
					"Example warning",
					"This is an example warning message.",
				),
			},
			wantStdout: "\nWarning: Example warning\n\nThis is an example warning message.\n",
			wantStderr: "",
		},
		"error": {
			diags: tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Example error",
					"This is an example error message.",
				),
			},
			wantStdout: "",
			wantStderr: "\nError: Example error\n\nThis is an example error message.\n",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			streams, done := terminal.StreamsForTesting(t)
			view := NewView(streams)
			view.Configure(&arguments.View{NoColor: true})
			v := NewStateIdentities(arguments.ViewJSON, view)

			v.Diagnostics(tc.diags)

			output := done(t)
			if diff := cmp.Diff(tc.wantStdout, output.Stdout()); diff != "" {
				t.Errorf("unexpected stdout diff:\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantStderr, output.Stderr()); diff != "" {
				t.Errorf("unexpected stderr diff:\n%s", diff)
			}
		})
	}
}
