// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackruntime

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/providers"
	stacks_testing_provider "github.com/hashicorp/terraform/internal/stacks/stackruntime/testing"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// TestValidate_valid tests that a variety of configurations under the main
// test source bundle each generate no diagnostics at all, as a
// relatively-simple way to detect accidental regressions.
//
// Any stack configuration directory that we expect should be valid can
// potentially be included in here unless it depends on provider plugins
// to complete validation, since this test cannot supply provider plugins.
func TestValidate_valid(t *testing.T) {
	validConfigDirs := []string{
		"empty",
		"variable-output-roundtrip",
		"variable-output-roundtrip-nested",
	}

	for _, name := range validConfigDirs {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			cfg := loadMainBundleConfigForTest(t, name)

			diags := Validate(ctx, &ValidateRequest{
				Config: cfg,
			})

			// The following will fail the test if there are any error diagnostics.
			reportDiagnosticsForTest(t, diags)

			// We also want to fail if there are just warnings, since the
			// configurations here are supposed to be totally problem-free.
			if len(diags) != 0 {
				t.FailNow() // reportDiagnosticsForTest already showed the diagnostics in the log
			}
		})
	}
}

func TestValidate_invalid(t *testing.T) {
	tcs := map[string]struct {
		diags func() tfdiags.Diagnostics
	}{
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
						Filename: mainBundleLocalAddrStr("invalid-configuration/invalid-configuration.tf"),
						Start:    hcl.Pos{Line: 11, Column: 3, Byte: 163},
						End:      hcl.Pos{Line: 11, Column: 10, Byte: 170},
					},
				})
				return diags
			},
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			cfg := loadMainBundleConfigForTest(t, name)

			gotDiags := Validate(ctx, &ValidateRequest{
				Config: cfg,
				ProviderFactories: map[addrs.Provider]providers.Factory{
					addrs.NewDefaultProvider("testing"): func() (providers.Interface, error) {
						return stacks_testing_provider.NewProvider(), nil
					},
				},
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
