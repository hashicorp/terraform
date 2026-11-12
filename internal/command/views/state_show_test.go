// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/jsonformat"
	"github.com/hashicorp/terraform/internal/command/jsonprovider"
	"github.com/hashicorp/terraform/internal/command/jsonstate"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/terminal"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// The DisplayResourceInstanceState method in the human view is just a wrapper for logic defined in the jsonformat package.
// The tests here only test handling empty values and one example of handling non-empty data.
func TestStateShowHuman_DisplayResourceInstanceState(t *testing.T) {
	testCases := map[string]struct {
		jsonState  jsonformat.State
		diags      tfdiags.Diagnostics
		wantStdout string
		wantStderr string
	}{
		"empty state": {
			jsonformat.State{},
			nil,
			"The state file is empty. No resources are represented.",
			"",
		},
		"empty state 2": {
			jsonformat.State{
				StateFormatVersion:    jsonstate.FormatVersion,
				ProviderFormatVersion: jsonprovider.FormatVersion,
				RootModule:            jsonstate.Module{},
				RootModuleOutputs:     map[string]jsonstate.Output{},
				ProviderSchemas:       jsonprovider.MarshalForRenderer(&terraform.Schemas{}),
			},
			nil,
			"The state file is empty. No resources are represented.",
			"",
		},
		"individual resource in state": {
			jsonformat.State{
				StateFormatVersion:    jsonstate.FormatVersion,
				ProviderFormatVersion: jsonprovider.FormatVersion,
				RootModule: jsonstate.Module{
					Resources: []jsonstate.Resource{
						{
							Address:       "example_resource.foobar",
							Name:          "foobar",
							ProviderName:  addrs.NewDefaultProvider("example").String(),
							SchemaVersion: 0,
							Mode:          "managed",
							Type:          "example_resource",
						},
					},
				},
				RootModuleOutputs: map[string]jsonstate.Output{},
				ProviderSchemas: jsonprovider.MarshalForRenderer(&terraform.Schemas{
					Providers: map[addrs.Provider]providers.ProviderSchema{
						addrs.NewDefaultProvider("example"): {
							ResourceTypes: map[string]providers.Schema{
								"example_resource": {
									Body: &configschema.Block{
										Attributes: map[string]*configschema.Attribute{},
									},
								},
							},
						},
					},
				}),
			},
			nil,
			`# example_resource.foobar:
resource "example_resource" "foobar" {}
`,
			"",
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			streams, done := terminal.StreamsForTesting(t)
			view := NewView(streams)
			view.Configure(&arguments.View{NoColor: true})
			v := NewStateShow(arguments.ViewHuman, view)

			v.DisplayResourceInstanceState(tc.jsonState, tc.diags)

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

func TestStateShowHuman_Diagnostics(t *testing.T) {
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
			v := NewStateShow(arguments.ViewHuman, view)

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

func TestStateShowJSON_DisplayResourceInstanceState(t *testing.T) {
	testCases := map[string]struct {
		jsonState  jsonformat.State
		diags      tfdiags.Diagnostics
		wantStdout string
		wantStderr string
	}{
		"empty state": {
			jsonformat.State{},
			nil,
			`{
  "format_version": "1.0",
  "resource": {
    "provider_name": "",
    "schema_version": 0
  },
  "diagnostics": []
}`,
			"",
		},
		"empty state 2": {
			jsonformat.State{
				StateFormatVersion:    jsonstate.FormatVersion,
				ProviderFormatVersion: jsonprovider.FormatVersion,
				RootModule:            jsonstate.Module{},
				RootModuleOutputs:     map[string]jsonstate.Output{},
				ProviderSchemas:       jsonprovider.MarshalForRenderer(&terraform.Schemas{}),
			},
			nil,
			`{
  "format_version": "1.0",
  "resource": {
    "provider_name": "",
    "schema_version": 0
  },
  "diagnostics": []
}`,
			"",
		},
		"individual resource in state": {
			jsonformat.State{
				StateFormatVersion:    jsonstate.FormatVersion,
				ProviderFormatVersion: jsonprovider.FormatVersion,
				RootModule: jsonstate.Module{
					Resources: []jsonstate.Resource{
						{
							Address:       "example_resource.foobar",
							Name:          "foobar",
							ProviderName:  addrs.NewDefaultProvider("example").String(),
							SchemaVersion: 0,
							Mode:          "managed",
							Type:          "example_resource",
						},
					},
				},
				RootModuleOutputs: map[string]jsonstate.Output{},
				ProviderSchemas: jsonprovider.MarshalForRenderer(&terraform.Schemas{
					Providers: map[addrs.Provider]providers.ProviderSchema{
						addrs.NewDefaultProvider("example"): {
							ResourceTypes: map[string]providers.Schema{
								"example_resource": {
									Body: &configschema.Block{
										Attributes: map[string]*configschema.Attribute{},
									},
								},
							},
						},
					},
				}),
			},
			nil,
			`{
  "format_version": "1.0",
  "resource": {
    "address": "example_resource.foobar",
    "mode": "managed",
    "type": "example_resource",
    "name": "foobar",
    "provider_name": "registry.terraform.io/hashicorp/example",
    "schema_version": 0
  },
  "diagnostics": []
}`,
			"",
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			streams, done := terminal.StreamsForTesting(t)
			view := NewView(streams)
			view.Configure(&arguments.View{NoColor: true})
			v := NewStateShow(arguments.ViewJSON, view)

			v.DisplayResourceInstanceState(tc.jsonState, tc.diags)

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

func TestStateShowJSON_Diagnostics(t *testing.T) {
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
			`{
  "format_version": "1.0",
  "resource": {
    "provider_name": "",
    "schema_version": 0
  },
  "diagnostics": [
    {
      "severity": "error",
      "summary": "Example error",
      "detail": "This is an example error message."
    }
  ]
}`,
			"",
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			streams, done := terminal.StreamsForTesting(t)
			view := NewView(streams)
			view.Configure(&arguments.View{NoColor: true})
			v := NewStateShow(arguments.ViewJSON, view)

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
