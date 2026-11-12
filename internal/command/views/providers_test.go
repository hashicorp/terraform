// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	tfaddr "github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/getproviders"
	"github.com/hashicorp/terraform/internal/getproviders/providerreqs"
	"github.com/hashicorp/terraform/internal/terminal"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestNewProviders_LogProviderRequirements(t *testing.T) {
	constraints, err := providerreqs.ParseVersionConstraints(">1.0.0")
	if err != nil {
		t.Fatalf("failed to parse version constraints: %s", err)
	}

	configReqs := &configs.ModuleRequirements{
		Requirements: providerreqs.Requirements{
			tfaddr.MustParseProviderSourceString("hashicorp/foobar1"): providerreqs.VersionConstraints{},
			tfaddr.MustParseProviderSourceString("hashicorp/foobar2"): constraints,
		},
	}

	// state cannot issue constraints, so they're ignored when rendering output even if present in the input data
	stateReqs := getproviders.Requirements{
		tfaddr.MustParseProviderSourceString("hashicorp/fizzbuzz1"): providerreqs.VersionConstraints{},
		tfaddr.MustParseProviderSourceString("hashicorp/fizzbuzz2"): constraints, // ignored
	}

	testCases := map[string]struct {
		diags      tfdiags.Diagnostics
		configReqs *configs.ModuleRequirements
		stateReqs  getproviders.Requirements
		wantStdout string
		wantStderr string
	}{
		"no requirements in config or state": {
			nil,
			nil,
			getproviders.Requirements{},
			emptyOutput,
			"",
		},
		"config reqs only": {
			nil,
			configReqs,
			getproviders.Requirements{},
			configOnlyOutput,
			"",
		},
		"state reqs only": {
			nil,
			nil,
			stateReqs,
			stateOnlyOutput,
			"",
		},
		"state and config reqs": {
			nil,
			configReqs,
			stateReqs,
			combinedOutput,
			"",
		},
		"reqs with warning": {
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Warning,
					"Example warning",
					"This is an example warning message.",
				),
			},
			configReqs,
			getproviders.Requirements{},
			fmt.Sprintf("Warning: Example warning\n\nThis is an example warning message.\n\n%s", configOnlyOutput),
			"",
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			streams, done := terminal.StreamsForTesting(t)
			view := NewView(streams)
			view.Configure(&arguments.View{NoColor: true})
			v := NewProviders(arguments.ViewHuman, view)

			v.LogProviderRequirements(tc.configReqs, tc.stateReqs, tc.diags)

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

func TestNewProviders_Diagnostics(t *testing.T) {
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
			v := NewProviders(arguments.ViewHuman, view)

			v.Diagnostics(tc.diags)

			output := done(t)

			// Assert contents
			// This must be done separately for stdout and stderr due to
			// the interleaving of output caused in tests by (TestOutput).All()
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

const (
	emptyOutput = `Providers required by configuration:
.
`

	configOnlyOutput = `Providers required by configuration:
.
├── provider[registry.terraform.io/hashicorp/foobar1]
└── provider[registry.terraform.io/hashicorp/foobar2] > 1.0.0`

	stateOnlyOutput = `Providers required by configuration:
.

Providers required by state:

    provider[registry.terraform.io/hashicorp/fizzbuzz1]

    provider[registry.terraform.io/hashicorp/fizzbuzz2]`

	combinedOutput = `Providers required by configuration:
.
├── provider[registry.terraform.io/hashicorp/foobar1]
└── provider[registry.terraform.io/hashicorp/foobar2] > 1.0.0

Providers required by state:

    provider[registry.terraform.io/hashicorp/fizzbuzz1]

    provider[registry.terraform.io/hashicorp/fizzbuzz2]`
)
