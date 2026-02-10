// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/getproviders"
	"github.com/hashicorp/terraform/internal/terminal"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestProvidersHuman_Output(t *testing.T) {
	testCases := map[string]struct {
		reqs         *configs.ModuleRequirements
		stateReqs    getproviders.Requirements
		wantContains []string
	}{
		"basic provider": {
			reqs: &configs.ModuleRequirements{
				Requirements: getproviders.Requirements{
					addrs.NewDefaultProvider("foo"): nil,
				},
			},
			stateReqs: nil,
			wantContains: []string{
				"Providers required by configuration:",
				"provider[registry.terraform.io/hashicorp/foo]",
			},
		},
		"provider with version constraint": {
			reqs: &configs.ModuleRequirements{
				Requirements: getproviders.Requirements{
					addrs.NewDefaultProvider("foo"): getproviders.MustParseVersionConstraints(">= 1.0.0"),
				},
			},
			stateReqs: nil,
			wantContains: []string{
				"Providers required by configuration:",
				"provider[registry.terraform.io/hashicorp/foo] >= 1.0.0",
			},
		},
		"with child module": {
			reqs: &configs.ModuleRequirements{
				Requirements: getproviders.Requirements{
					addrs.NewDefaultProvider("foo"): nil,
				},
				Children: map[string]*configs.ModuleRequirements{
					"child": {
						Name: "child",
						Requirements: getproviders.Requirements{
							addrs.NewDefaultProvider("bar"): nil,
						},
					},
				},
			},
			stateReqs: nil,
			wantContains: []string{
				"Providers required by configuration:",
				"provider[registry.terraform.io/hashicorp/foo]",
				"module.child",
				"provider[registry.terraform.io/hashicorp/bar]",
			},
		},
		"with state providers": {
			reqs: &configs.ModuleRequirements{
				Requirements: getproviders.Requirements{
					addrs.NewDefaultProvider("foo"): nil,
				},
			},
			stateReqs: getproviders.Requirements{
				addrs.NewDefaultProvider("baz"): nil,
			},
			wantContains: []string{
				"Providers required by configuration:",
				"provider[registry.terraform.io/hashicorp/foo]",
				"Providers required by state:",
				"provider[registry.terraform.io/hashicorp/baz]",
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			streams, done := terminal.StreamsForTesting(t)
			view := NewView(streams)
			view.Configure(&arguments.View{NoColor: true})
			v := NewProviders(view)

			v.Output(tc.reqs, tc.stateReqs)

			got := done(t).All()
			for _, want := range tc.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("output missing %q:\n%s", want, got)
				}
			}
		})
	}
}

func TestProvidersHuman_Diagnostics(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	view.Configure(&arguments.View{NoColor: true})
	v := NewProviders(view)

	diags := tfdiags.Diagnostics{
		tfdiags.Sourceless(
			tfdiags.Error,
			"Test error",
			"This is a test error message.",
		),
	}

	v.Diagnostics(diags)

	got := done(t).All()
	if !strings.Contains(got, "Error: Test error") {
		t.Errorf("expected error message in output:\n%s", got)
	}
	if !strings.Contains(got, "This is a test error message.") {
		t.Errorf("expected error detail in output:\n%s", got)
	}
}

func TestNewProviders(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	defer done(t)
	view := NewView(streams)

	got := NewProviders(view)

	if _, ok := got.(*ProvidersHuman); !ok {
		t.Errorf("expected *ProvidersHuman, got %T", got)
	}
}
