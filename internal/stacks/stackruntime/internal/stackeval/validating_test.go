// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/providers"
	testing_provider "github.com/hashicorp/terraform/internal/providers/testing"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestValidate_modulesWithProviderConfigs(t *testing.T) {
	// This test checks that we're correctly prohibiting inline provider
	// configurations in Terraform modules used as stack components, which
	// is forbidden because the stacks language is responsible for provider
	// configurations.
	//
	// The underlying modules runtime isn't configured with any ability to
	// instantiate provider plugins itself, so failing to prohibit this
	// at the stacks language layer would just cause a lower-quality and
	// more confusing error message to be emited by the modules runtime.

	cfg := testStackConfig(t, "validating", "modules_with_provider_configs")
	main := NewForValidating(cfg, ValidateOpts{
		ProviderFactories: map[addrs.Provider]providers.Factory{
			addrs.NewBuiltInProvider("test"): func() (providers.Interface, error) {
				// The test fails before it has to do any schema validation so
				// we can safely return an empty mock provider here.
				return &testing_provider.MockProvider{}, nil
			},
		},
	})

	inPromisingTask(t, func(ctx context.Context, t *testing.T) {
		diags := main.ValidateAll(ctx)
		if !diags.HasErrors() {
			t.Fatalf("succeeded; want errors")
		}
		diags.Sort()

		// We'll use the ForRPC method just as a convenient way to discard
		// the specific diagnostic object types, so that we can compare
		// the objects without worrying about exactly which diagnostic
		// implementation each is using.
		gotDiags := diags.ForRPC()

		var wantDiags tfdiags.Diagnostics
		// Configurations in the root module get a different detail message
		// than those in descendent modules, because for descendents we don't
		// assume that the author is empowered to make the module
		// stacks-compatible, while for the root it's more likely to be
		// directly intended for stacks use, at least for now while things are
		// relatively early. (We could revisit this tradeoff later.)
		wantDiags = wantDiags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Inline provider configuration not allowed",
			Detail:   `A module used as a stack component must have all of its provider configurations passed from the stack configuration, using the "providers" argument within the component configuration block.`,
			Subject: &hcl.Range{
				Filename: "https://testing.invalid/validating.tar.gz//modules_with_provider_configs/module-a/modules-with-provider-configs-a.tf",
				Start:    hcl.Pos{Line: 9, Column: 1, Byte: 104},
				End:      hcl.Pos{Line: 9, Column: 16, Byte: 119},
			},
		})
		wantDiags = wantDiags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Inline provider configuration not allowed",
			Detail:   "This module is not compatible with Terraform Stacks, because it declares an inline provider configuration.\n\nTo be used with stacks, this module must instead accept provider configurations from its caller.",
			Subject: &hcl.Range{
				Filename: "https://testing.invalid/validating.tar.gz//modules_with_provider_configs/module-b/modules-with-provider-configs-b.tf",
				Start:    hcl.Pos{Line: 9, Column: 1, Byte: 104},
				End:      hcl.Pos{Line: 9, Column: 16, Byte: 119},
			},
		})
		wantDiags = wantDiags.ForRPC()

		if diff := cmp.Diff(wantDiags, gotDiags); diff != "" {
			t.Errorf("wrong diagnostics\n%s", diff)
		}
	})
}

func TestValidate_nestedModuleDiagnostics(t *testing.T) {
	// This test verifies that our source bundle aware module loader correctly
	// builds diagnostic source addresses for various kinds of nested modules.
	// It covers both in-repo components and remote components, both having
	// top-level and nested diagnostic errors.

	cfg := testStackConfig(t, "validating", "nested_module_diagnostics")
	main := NewForValidating(cfg, ValidateOpts{})

	inPromisingTask(t, func(ctx context.Context, t *testing.T) {
		diags := main.ValidateAll(ctx)
		if !diags.HasErrors() {
			t.Fatalf("succeeded; want errors")
		}
		diags.Sort()

		// We'll use the ForRPC method just as a convenient way to discard
		// the specific diagnostic object types, so that we can compare
		// the objects without worrying about exactly which diagnostic
		// implementation each is using.
		gotDiags := diags.ForRPC()

		var wantDiags tfdiags.Diagnostics
		// This configuration has the same errors repeated multiple times,
		// varying only on filename (source address).
		filenames := []string{
			"https://testing.invalid/invalid.tar.gz//invalid.tf",
			"https://testing.invalid/invalid_child.tar.gz//child/invalid_child.tf",
			"https://testing.invalid/invalid_child.tar.gz//child/invalid_child.tf",
			"https://testing.invalid/invalid_grandchildren.tar.gz//first/child/invalid_child.tf",
			"https://testing.invalid/invalid_grandchildren.tar.gz//second/child/invalid_child.tf",
			"https://testing.invalid/validating.tar.gz//nested_module_diagnostics/invalid/invalid.tf",
			"https://testing.invalid/validating.tar.gz//nested_module_diagnostics/invalid_child/child/invalid_child.tf",
		}
		for _, filename := range filenames {
			wantDiags = wantDiags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Unsupported block type",
				Detail:   `Blocks of type "invalid" are not expected here.`,
				Subject: &hcl.Range{
					Filename: filename,
					Start:    hcl.Pos{Line: 1, Column: 1, Byte: 0},
					End:      hcl.Pos{Line: 1, Column: 8, Byte: 7},
				},
			})
		}
		wantDiags = wantDiags.ForRPC()

		if diff := cmp.Diff(wantDiags, gotDiags); diff != "" {
			for i, diag := range gotDiags {
				t.Logf("diagnostic %d: %s", i, diag)
			}
			t.Errorf("wrong diagnostics\n%s", diff)
		}
	})
}
