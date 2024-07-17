// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackruntime

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/depsfile"
	"github.com/hashicorp/terraform/internal/getproviders/providerreqs"
	"github.com/hashicorp/terraform/internal/providers"
	stacks_testing_provider "github.com/hashicorp/terraform/internal/stacks/stackruntime/testing"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type validateTestInput struct {
	// skip lets us write tests for behaviour we want to add in the future. Set
	// this to true for any tests that are not yet implemented.
	skip bool

	// diags is a function that returns the expected diagnostics for the
	// test.
	diags func() tfdiags.Diagnostics

	// planInputVars is used only in the plan tests to provide a set of input
	// variables to use for the plan request. Validate operates statically so
	// does not need any input variables.
	planInputVars map[string]cty.Value
}

var (
	// validConfigurations are shared between the validate and plan tests.
	validConfigurations = map[string]validateTestInput{
		"empty":                            {},
		"plan-variable-defaults":           {},
		"variable-output-roundtrip":        {},
		"variable-output-roundtrip-nested": {},
		"aliased-provider":                 {},
		filepath.Join("with-single-input", "input-from-component"): {},
		filepath.Join("with-single-input", "input-from-component-list"): {
			planInputVars: map[string]cty.Value{
				"components": cty.SetVal([]cty.Value{
					cty.StringVal("one"),
					cty.StringVal("two"),
					cty.StringVal("three"),
				}),
			},
		},
		filepath.Join("with-single-input", "provider-name-clash"): {
			planInputVars: map[string]cty.Value{
				"input": cty.StringVal("input"),
			},
		},
		filepath.Join("with-single-input", "valid"): {
			planInputVars: map[string]cty.Value{
				"input": cty.StringVal("input"),
			},
		},
		filepath.Join("with-single-input", "provider-for-each"): {
			planInputVars: map[string]cty.Value{
				"input": cty.StringVal("input"),
			},
		},
	}

	// invalidConfigurations are shared between the validate and plan tests.
	invalidConfigurations = map[string]validateTestInput{
		"validate-undeclared-variable": {
			diags: func() tfdiags.Diagnostics {
				var diags tfdiags.Diagnostics
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Reference to undeclared input variable",
					Detail:   `There is no variable "a" block declared in this stack.`,
					Subject: &hcl.Range{
						Filename: mainBundleSourceAddrStr("validate-undeclared-variable/validate-undeclared-variable.tfstack.hcl"),
						Start:    hcl.Pos{Line: 3, Column: 11, Byte: 40},
						End:      hcl.Pos{Line: 3, Column: 16, Byte: 45},
					},
				})
				return diags
			},
		},
		"invalid-configuration": {
			diags: func() tfdiags.Diagnostics {
				var diags tfdiags.Diagnostics
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Unsupported argument",
					Detail:   "An argument named \"invalid\" is not expected here.",
					Subject: &hcl.Range{
						Filename: mainBundleSourceAddrStr("invalid-configuration/invalid-configuration.tf"),
						Start:    hcl.Pos{Line: 11, Column: 3, Byte: 163},
						End:      hcl.Pos{Line: 11, Column: 10, Byte: 170},
					},
				})
				return diags
			},
		},
		filepath.Join("with-single-input", "undeclared-provider"): {
			diags: func() tfdiags.Diagnostics {
				var diags tfdiags.Diagnostics
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Reference to undeclared provider configuration",
					Detail:   "There is no provider \"testing\" \"default\" block declared in this stack.",
					Subject: &hcl.Range{
						Filename: mainBundleSourceAddrStr("with-single-input/undeclared-provider/undeclared-provider.tfstack.hcl"),
						Start:    hcl.Pos{Line: 10, Column: 15, Byte: 163},
						End:      hcl.Pos{Line: 10, Column: 39, Byte: 187},
					},
				})
				return diags
			},
		},
		filepath.Join("with-single-input", "missing-provider"): {
			diags: func() tfdiags.Diagnostics {
				var diags tfdiags.Diagnostics
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Missing required provider configuration",
					Detail:   "The root module for component.self requires a provider configuration named \"testing\" for provider \"hashicorp/testing\", which is not assigned in the component's \"providers\" argument.",
					Subject: &hcl.Range{
						Filename: mainBundleSourceAddrStr("with-single-input/missing-provider/missing-provider.tfstack.hcl"),
						Start:    hcl.Pos{Line: 14, Column: 1, Byte: 169},
						End:      hcl.Pos{Line: 14, Column: 17, Byte: 185},
					},
				})
				return diags
			},
		},
		filepath.Join("with-single-input", "invalid-provider-type"): {
			diags: func() tfdiags.Diagnostics {
				var diags tfdiags.Diagnostics
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid provider configuration",
					Detail:   "The provider configuration slot \"testing\" requires a configuration for provider \"registry.terraform.io/hashicorp/testing\", not for provider \"terraform.io/builtin/testing\".",
					Subject: &hcl.Range{
						Filename: mainBundleSourceAddrStr("with-single-input/invalid-provider-type/invalid-provider-type.tfstack.hcl"),
						Start:    hcl.Pos{Line: 22, Column: 15, Byte: 378},
						End:      hcl.Pos{Line: 22, Column: 39, Byte: 402},
					},
				})
				return diags
			},
		},
		filepath.Join("with-single-input", "invalid-provider-config"): {
			diags: func() tfdiags.Diagnostics {
				var diags tfdiags.Diagnostics
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Unsupported argument",
					Detail:   "An argument named \"imaginary\" is not expected here.",
					Subject: &hcl.Range{
						Filename: mainBundleSourceAddrStr("with-single-input/invalid-provider-config/invalid-provider-config.tfstack.hcl"),
						Start:    hcl.Pos{Line: 11, Column: 5, Byte: 218},
						End:      hcl.Pos{Line: 11, Column: 14, Byte: 227},
					},
				})
				return diags
			},
		},
		filepath.Join("with-single-input", "undeclared-variable"): {
			diags: func() tfdiags.Diagnostics {
				var diags tfdiags.Diagnostics
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Reference to undeclared input variable",
					Detail:   `There is no variable "input" block declared in this stack.`,
					Subject: &hcl.Range{
						Filename: mainBundleSourceAddrStr("with-single-input/undeclared-variable/undeclared-variable.tfstack.hcl"),
						Start:    hcl.Pos{Line: 19, Column: 13, Byte: 284},
						End:      hcl.Pos{Line: 19, Column: 22, Byte: 293},
					},
				})
				return diags
			},
		},
		filepath.Join("with-single-input", "missing-variable"): {
			diags: func() tfdiags.Diagnostics {
				var diags tfdiags.Diagnostics
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid inputs for component",
					Detail:   "Invalid input variable definition object: attribute \"input\" is required.",
					Subject: &hcl.Range{
						Filename: mainBundleSourceAddrStr("with-single-input/missing-variable/missing-variable.tfstack.hcl"),
						Start:    hcl.Pos{Line: 22, Column: 12, Byte: 338},
						End:      hcl.Pos{Line: 22, Column: 14, Byte: 340},
					},
				})
				return diags
			},
		},
		filepath.Join("with-single-input", "input-from-missing-component"): {
			diags: func() tfdiags.Diagnostics {
				var diags tfdiags.Diagnostics
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Reference to undeclared component",
					Detail:   "There is no component \"output\" block declared in this stack.",
					Subject: &hcl.Range{
						Filename: mainBundleSourceAddrStr("with-single-input/input-from-missing-component/input-from-missing-component.tfstack.hcl"),
						Start:    hcl.Pos{Line: 19, Column: 13, Byte: 314},
						End:      hcl.Pos{Line: 19, Column: 29, Byte: 330},
					},
				})
				return diags
			},
		},
		filepath.Join("with-single-input", "input-from-provider"): {
			diags: func() tfdiags.Diagnostics {
				var diags tfdiags.Diagnostics
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid inputs for component",
					Detail:   "Invalid input variable definition object: attribute \"input\": string required.",
					Subject: &hcl.Range{
						Filename: mainBundleSourceAddrStr("with-single-input/input-from-provider/input-from-provider.tfstack.hcl"),
						Start:    hcl.Pos{Line: 17, Column: 12, Byte: 239},
						End:      hcl.Pos{Line: 20, Column: 4, Byte: 339},
					},
				})
				return diags
			},
		},
	}
)

// TestValidate_valid tests that a variety of configurations under the main
// test source bundle each generate no diagnostics at all, as a
// relatively-simple way to detect accidental regressions.
//
// Any stack configuration directory that we expect should be valid can
// potentially be included in here unless it depends on provider plugins
// to complete validation, since this test cannot supply provider plugins.
func TestValidate_valid(t *testing.T) {
	for name, tc := range validConfigurations {
		t.Run(name, func(t *testing.T) {
			if tc.skip {
				// We've added this test before the implementation was ready.
				t.SkipNow()
			}

			ctx := context.Background()
			cfg := loadMainBundleConfigForTest(t, name)
			lock := depsfile.NewLocks()
			lock.SetProvider(
				addrs.NewDefaultProvider("testing"),
				providerreqs.MustParseVersion("0.0.0"),
				providerreqs.MustParseVersionConstraints("=0.0.0"),
				providerreqs.PreferredHashes([]providerreqs.Hash{}),
			)
			lock.SetProvider(
				addrs.NewDefaultProvider("other"),
				providerreqs.MustParseVersion("0.0.0"),
				providerreqs.MustParseVersionConstraints("=0.0.0"),
				providerreqs.PreferredHashes([]providerreqs.Hash{}),
			)

			diags := Validate(ctx, &ValidateRequest{
				Config: cfg,
				ProviderFactories: map[addrs.Provider]providers.Factory{
					// We support both hashicorp/testing and
					// terraform.io/builtin/testing as providers. This lets us
					// test the provider aliasing feature. Both providers
					// support the same set of resources and data sources.
					addrs.NewDefaultProvider("testing"): func() (providers.Interface, error) {
						return stacks_testing_provider.NewProvider(), nil
					},
					addrs.NewBuiltInProvider("testing"): func() (providers.Interface, error) {
						return stacks_testing_provider.NewProvider(), nil
					},
					// We also support an "other" provider out of the box to
					// test the provider aliasing feature.
					addrs.NewDefaultProvider("other"): func() (providers.Interface, error) {
						return stacks_testing_provider.NewProvider(), nil
					},
				},
				DependencyLocks: *lock,
			})

			// The following will fail the test if there are any error
			// diagnostics.
			reportDiagnosticsForTest(t, diags)

			// We also want to fail if there are just warnings, since the
			// configurations here are supposed to be totally problem-free.
			if len(diags) != 0 {
				// reportDiagnosticsForTest already showed the diagnostics in
				// the log
				t.FailNow()
			}
		})
	}
}

func TestValidate_invalid(t *testing.T) {
	for name, tc := range invalidConfigurations {
		t.Run(name, func(t *testing.T) {
			if tc.skip {
				// We've added this test before the implementation was ready.
				t.SkipNow()
			}

			ctx := context.Background()
			cfg := loadMainBundleConfigForTest(t, name)

			lock := depsfile.NewLocks()
			lock.SetProvider(
				addrs.NewDefaultProvider("testing"),
				providerreqs.MustParseVersion("0.0.0"),
				providerreqs.MustParseVersionConstraints("=0.0.0"),
				providerreqs.PreferredHashes([]providerreqs.Hash{}),
			)

			gotDiags := Validate(ctx, &ValidateRequest{
				Config: cfg,
				ProviderFactories: map[addrs.Provider]providers.Factory{
					// We support both hashicorp/testing and
					// terraform.io/builtin/testing as providers. This lets us
					// test the provider aliasing feature. Both providers
					// support the same set of resources and data sources.
					addrs.NewDefaultProvider("testing"): func() (providers.Interface, error) {
						return stacks_testing_provider.NewProvider(), nil
					},
					addrs.NewBuiltInProvider("testing"): func() (providers.Interface, error) {
						return stacks_testing_provider.NewProvider(), nil
					},
				},
				DependencyLocks: *lock,
			}).ForRPC()
			wantDiags := tc.diags().ForRPC()

			if diff := cmp.Diff(wantDiags, gotDiags); diff != "" {
				t.Errorf("wrong diagnostics\n%s", diff)
			}
		})
	}
}

func TestValidate_embeddedStackSelfRef(t *testing.T) {
	ctx := context.Background()

	// One possible failure mode for this test is to deadlock itself if
	// our deadlock detection is incorrect, so we'll try to make it bail
	// if it runs too long.
	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	ctx, span := tracer.Start(ctx, "TestValidate_embeddedStackSelfRef")
	defer span.End()

	cfg := loadMainBundleConfigForTest(t, "validate-embedded-stack-selfref")

	gotDiags := Validate(ctx, &ValidateRequest{
		Config: cfg,
	})

	// We'll normalize the diagnostics to be of consistent underlying type
	// using ForRPC, so that we can easily diff them; we don't actually care
	// about which underlying implementation is in use.
	gotDiags = gotDiags.ForRPC()
	var wantDiags tfdiags.Diagnostics
	wantDiags = wantDiags.Append(tfdiags.Sourceless(
		tfdiags.Error,
		"Self-dependent items in configuration",
		`The following items in your configuration form a circular dependency chain through their references:
  - stack.a collected outputs
  - stack.a.output.a value
  - stack.a inputs

Terraform uses references to decide a suitable order for performing operations, so configuration items may not refer to their own results either directly or indirectly.`,
	))
	wantDiags = wantDiags.ForRPC()

	if diff := cmp.Diff(wantDiags, gotDiags); diff != "" {
		t.Errorf("wrong diagnostics\n%s", diff)
	}
}

func TestValidate_missing_provider_from_lockfile(t *testing.T) {
	ctx := context.Background()
	cfg := loadMainBundleConfigForTest(t, filepath.Join("with-single-input", "input-from-component"))
	lock := depsfile.NewLocks()

	diags := Validate(ctx, &ValidateRequest{
		Config: cfg,
		ProviderFactories: map[addrs.Provider]providers.Factory{
			// We support both hashicorp/testing and
			// terraform.io/builtin/testing as providers. This lets us
			// test the provider aliasing feature. Both providers
			// support the same set of resources and data sources.
			addrs.NewDefaultProvider("testing"): func() (providers.Interface, error) {
				return stacks_testing_provider.NewProvider(), nil
			},
			addrs.NewBuiltInProvider("testing"): func() (providers.Interface, error) {
				return stacks_testing_provider.NewProvider(), nil
			},
		},
		DependencyLocks: *lock,
	})

	if len(diags) != 1 {
		t.Fatalf("expected exactly one diagnostic, got %d", len(diags))
	}

	diag := diags[0]
	if diag.Severity() != tfdiags.Error {
		t.Fatalf("expected error diagnostic, got %s", diag.Severity())
	}

	if diag.Description().Summary != "Provider missing from lockfile" {
		t.Fatalf("expected diagnostic summary 'Provider missing from lockfile', got %q", diag.Description().Summary)
	}

	if diag.Description().Detail != "Provider \"registry.terraform.io/hashicorp/testing\" is not in the lockfile. This provider must be in the lockfile to be used in the configuration. Please run `tfstacks provider lock` to update the lockfile and run this operation again with an updated configuration." {
		t.Fatalf("expected diagnostic detail to be a specific message, got %q", diag.Description().Detail)
	}
}

func TestValidate_impliedProviderTypes(t *testing.T) {

	tcs := []struct {
		directory string
		providers map[addrs.Provider]providers.Factory
		wantDiags func() tfdiags.Diagnostics
	}{
		{
			directory: "with-hashicorp-provider",
			providers: map[addrs.Provider]providers.Factory{
				addrs.NewDefaultProvider("testing"): func() (providers.Interface, error) {
					return stacks_testing_provider.NewProvider(), nil
				},
			},
		},
		{
			directory: "with-non-hashicorp-provider",
			providers: map[addrs.Provider]providers.Factory{
				addrs.NewProvider(addrs.DefaultProviderRegistryHost, "other", "testing"): func() (providers.Interface, error) {
					return stacks_testing_provider.NewProvider(), nil
				},
			},
			wantDiags: func() tfdiags.Diagnostics {
				var diags tfdiags.Diagnostics
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid provider configuration",
					Detail: "The provider configuration slot \"testing\" requires a configuration for provider \"registry.terraform.io/hashicorp/testing\", not for provider \"registry.terraform.io/other/testing\"." +
						"\n\nThe module does not declare a source address for \"testing\" in its required_providers block, so Terraform assumed \"hashicorp/testing\" for backward-compatibility with older versions of Terraform",
					Subject: &hcl.Range{
						Filename: mainBundleSourceAddrStr("legacy-module/with-non-hashicorp-provider/with-non-hashicorp-provider.tfstack.hcl"),
						Start:    hcl.Pos{Line: 21, Column: 15, Byte: 447},
						End:      hcl.Pos{Line: 21, Column: 39, Byte: 471},
					},
				})
				return diags
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.directory, func(t *testing.T) {

			ctx := context.Background()
			lock := depsfile.NewLocks()
			for addr := range tc.providers {
				lock.SetProvider(
					addr,
					providerreqs.MustParseVersion("0.0.0"),
					providerreqs.MustParseVersionConstraints("=0.0.0"),
					providerreqs.PreferredHashes([]providerreqs.Hash{}),
				)
			}

			cfg := loadMainBundleConfigForTest(t, filepath.Join("legacy-module", tc.directory))
			gotDiags := Validate(ctx, &ValidateRequest{
				Config:            cfg,
				ProviderFactories: tc.providers,
				DependencyLocks:   *lock,
			}).ForRPC()

			wantDiags := tfdiags.Diagnostics{}.ForRPC()
			if tc.wantDiags != nil {
				wantDiags = tc.wantDiags().ForRPC()
			}

			if diff := cmp.Diff(wantDiags, gotDiags); diff != "" {
				t.Errorf("wrong diagnostics\n%s", diff)
			}
		})
	}
}
