// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/terminal"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestStateReplaceProviderHuman_DescribeChanges(t *testing.T) {
	from := addrs.NewDefaultProvider("beginning")
	to := addrs.NewDefaultProvider("end")

	cases := map[string]struct {
		resources      []*states.Resource
		from           addrs.Provider
		to             addrs.Provider
		expectedStdout string
	}{
		"basic": {
			resources: []*states.Resource{
				{
					Addr: addrs.AbsResource{
						Resource: addrs.Resource{
							Mode: addrs.ManagedResourceMode,
							Type: "test_resource",
							Name: "example",
						},
					},
				},
			},
			from: from,
			to:   to,
			expectedStdout: `Terraform will perform the following actions:

  ~ Updating provider:
    - registry.terraform.io/hashicorp/beginning
    + registry.terraform.io/hashicorp/end

Changing 1 resources:

  test_resource.example`,
		},
		"no resources changing": {
			resources:      []*states.Resource{},
			from:           from,
			to:             to,
			expectedStdout: `No matching resources found.`,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			streams, done := terminal.StreamsForTesting(t)
			view := NewView(streams)
			view.Configure(&arguments.View{NoColor: true})
			v := NewStateReplaceProvider(arguments.ViewHuman, view)

			v.DescribeChanges(tc.from, tc.to, tc.resources)

			output := done(t)
			if diff := cmp.Diff(tc.expectedStdout, strings.TrimSpace(output.Stdout())); diff != "" {
				t.Fatalf("unexpected stdout diff:\n%s", diff)
			}
			expectedStderr := ""
			if diff := cmp.Diff(expectedStderr, strings.TrimSpace(output.Stderr())); diff != "" {
				t.Fatalf("unexpected stderr diff:\n%s", diff)
			}
		})
	}
}

func TestStateReplaceProviderHuman_Success(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	view.Configure(&arguments.View{NoColor: true})
	v := NewStateReplaceProvider(arguments.ViewHuman, view)

	v.Success(2)

	output := done(t)
	expectedStdout := "Successfully replaced provider for 2 resources."
	if diff := cmp.Diff(expectedStdout, strings.TrimSpace(output.Stdout())); diff != "" {
		t.Fatalf("unexpected stdout diff:\n%s", diff)
	}
	expectedStderr := ""
	if diff := cmp.Diff(expectedStderr, strings.TrimSpace(output.Stderr())); diff != "" {
		t.Fatalf("unexpected stderr diff:\n%s", diff)
	}
}

func TestStateReplaceProviderHuman_Cancelled(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	view.Configure(&arguments.View{NoColor: true})
	v := NewStateReplaceProvider(arguments.ViewHuman, view)

	v.Cancelled()

	output := done(t)
	expectedStdout := "Cancelled replacing providers."
	if diff := cmp.Diff(expectedStdout, strings.TrimSpace(output.Stdout())); diff != "" {
		t.Fatalf("unexpected stdout diff:\n%s", diff)
	}
	expectedStderr := ""
	if diff := cmp.Diff(expectedStderr, strings.TrimSpace(output.Stderr())); diff != "" {
		t.Fatalf("unexpected stderr diff:\n%s", diff)
	}
}

func TestStateReplaceProviderHuman_Diagnostics(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	view.Configure(&arguments.View{NoColor: true})
	v := NewStateReplaceProvider(arguments.ViewHuman, view)

	diags := tfdiags.Diagnostics{
		tfdiags.Sourceless(
			tfdiags.Error,
			"Example error.",
			"This is an example error detail.",
		),
	}
	v.Diagnostics(diags)

	output := done(t)
	expectedStdout := ""
	if diff := cmp.Diff(expectedStdout, strings.TrimSpace(output.Stdout())); diff != "" {
		t.Fatalf("unexpected stdout diff:\n%s", diff)
	}
	expectedStderr := "Error: Example error.\n\nThis is an example error detail."
	if diff := cmp.Diff(expectedStderr, strings.TrimSpace(output.Stderr())); diff != "" {
		t.Fatalf("unexpected stderr diff:\n%s", diff)
	}
}
