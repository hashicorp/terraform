// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/zclconf/go-cty-debug/ctydebug"
	"github.com/zclconf/go-cty/cty"

	"github.com/apparentlymart/go-versions/versions/constraints"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/depsfile"
	"github.com/hashicorp/terraform/internal/getproviders/providerreqs"
	"github.com/hashicorp/terraform/internal/providers"
	testing_provider "github.com/hashicorp/terraform/internal/providers/testing"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestProviderConfig_CheckProviderArgs_EmptyConfig(t *testing.T) {
	cfg := testStackConfig(t, "provider", "single_instance")
	providerTypeAddr := addrs.NewBuiltInProvider("foo")
	newMockProvider := func(t *testing.T) (*testing_provider.MockProvider, providers.Factory) {
		t.Helper()
		mockProvider := &testing_provider.MockProvider{
			GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
				Provider: providers.Schema{},
			},
			ValidateProviderConfigFn: func(vpcr providers.ValidateProviderConfigRequest) providers.ValidateProviderConfigResponse {
				if vpcr.Config.ContainsMarked() {
					panic("config has marks")
				}
				var diags tfdiags.Diagnostics
				if vpcr.Config.Type().HasAttribute("test") {
					if vpcr.Config.GetAttr("test").RawEquals(cty.StringVal("invalid")) {
						diags = diags.Append(fmt.Errorf("invalid value checked by provider itself"))
					}
				}
				return providers.ValidateProviderConfigResponse{
					PreparedConfig: vpcr.Config,
					Diagnostics:    diags,
				}
			},
		}
		providerFactory := providers.FactoryFixed(mockProvider)
		return mockProvider, providerFactory
	}
	getProviderConfig := func(ctx context.Context, t *testing.T, main *Main) *ProviderConfig {
		t.Helper()
		mainStack := main.MainStack()
		provider := mainStack.Provider(stackaddrs.ProviderConfig{
			Provider: providerTypeAddr,
			Name:     "bar",
		})
		if provider == nil {
			t.Fatal("no provider.foo.bar is available")
		}
		return provider.config
	}

	subtestInPromisingTask(t, "valid", func(ctx context.Context, t *testing.T) {
		mockProvider, providerFactory := newMockProvider(t)
		main := testEvaluator(t, testEvaluatorOpts{
			Config: cfg,
			ProviderFactories: ProviderFactories{
				providerTypeAddr: providerFactory,
			},
		})
		config := getProviderConfig(ctx, t, main)

		want := cty.EmptyObjectVal
		got, diags := config.CheckProviderArgs(ctx, InspectPhase)
		assertNoDiags(t, diags)

		if diff := cmp.Diff(want, got, ctydebug.CmpOptions); diff != "" {
			t.Errorf("wrong result\n%s", diff)
		}

		if !mockProvider.ValidateProviderConfigCalled {
			t.Error("ValidateProviderConfig was not called; should've been")
		} else {
			got := mockProvider.ValidateProviderConfigRequest
			want := providers.ValidateProviderConfigRequest{
				Config: cty.EmptyObjectVal,
			}
			if diff := cmp.Diff(want, got, ctydebug.CmpOptions); diff != "" {
				t.Errorf("wrong request\n%s", diff)
			}
		}
	})
}

func TestProviderConfig_CheckProviderArgs(t *testing.T) {
	cfg := testStackConfig(t, "provider", "single_instance_configured")
	providerTypeAddr := addrs.NewBuiltInProvider("foo")
	newMockProvider := func(t *testing.T) (*testing_provider.MockProvider, providers.Factory) {
		t.Helper()
		mockProvider := &testing_provider.MockProvider{
			GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
				Provider: providers.Schema{
					Body: &configschema.Block{
						Attributes: map[string]*configschema.Attribute{
							"test": {
								Type:     cty.String,
								Optional: true,
							},
						},
					},
				},
			},
			ValidateProviderConfigFn: func(vpcr providers.ValidateProviderConfigRequest) providers.ValidateProviderConfigResponse {
				if vpcr.Config.ContainsMarked() {
					panic("config has marks")
				}
				var diags tfdiags.Diagnostics
				if vpcr.Config.Type().HasAttribute("test") {
					if vpcr.Config.GetAttr("test").RawEquals(cty.StringVal("invalid")) {
						diags = diags.Append(fmt.Errorf("invalid value checked by provider itself"))
					}
				}
				return providers.ValidateProviderConfigResponse{
					PreparedConfig: vpcr.Config,
					Diagnostics:    diags,
				}
			},
		}
		providerFactory := providers.FactoryFixed(mockProvider)
		return mockProvider, providerFactory
	}
	getProviderConfig := func(ctx context.Context, t *testing.T, main *Main) *ProviderConfig {
		t.Helper()
		mainStack := main.MainStack()
		provider := mainStack.Provider(stackaddrs.ProviderConfig{
			Provider: providerTypeAddr,
			Name:     "bar",
		})
		if provider == nil {
			t.Fatal("no provider.foo.bar is available")
		}
		return provider.config
	}

	subtestInPromisingTask(t, "valid", func(ctx context.Context, t *testing.T) {
		mockProvider, providerFactory := newMockProvider(t)
		main := testEvaluator(t, testEvaluatorOpts{
			Config: cfg,
			TestOnlyGlobals: map[string]cty.Value{
				"provider_configuration": cty.StringVal("yep"),
			},
			ProviderFactories: ProviderFactories{
				providerTypeAddr: providerFactory,
			},
		})
		config := getProviderConfig(ctx, t, main)

		want := cty.ObjectVal(map[string]cty.Value{
			"test": cty.StringVal("yep"),
		})
		got, diags := config.CheckProviderArgs(ctx, InspectPhase)
		assertNoDiags(t, diags)

		if diff := cmp.Diff(want, got, ctydebug.CmpOptions); diff != "" {
			t.Errorf("wrong result\n%s", diff)
		}

		if !mockProvider.ValidateProviderConfigCalled {
			t.Error("ValidateProviderConfig was not called; should've been")
		} else {
			got := mockProvider.ValidateProviderConfigRequest
			want := providers.ValidateProviderConfigRequest{
				Config: cty.ObjectVal(map[string]cty.Value{
					"test": cty.StringVal("yep"),
				}),
			}
			if diff := cmp.Diff(want, got, ctydebug.CmpOptions); diff != "" {
				t.Errorf("wrong request\n%s", diff)
			}
		}
	})
	subtestInPromisingTask(t, "valid with marks", func(ctx context.Context, t *testing.T) {
		mockProvider, providerFactory := newMockProvider(t)
		main := testEvaluator(t, testEvaluatorOpts{
			Config: cfg,
			TestOnlyGlobals: map[string]cty.Value{
				"provider_configuration": cty.StringVal("yep").Mark("nope"),
			},
			ProviderFactories: ProviderFactories{
				providerTypeAddr: providerFactory,
			},
		})
		config := getProviderConfig(ctx, t, main)

		want := cty.ObjectVal(map[string]cty.Value{
			"test": cty.StringVal("yep").Mark("nope"),
		})
		got, diags := config.CheckProviderArgs(ctx, InspectPhase)
		assertNoDiags(t, diags)

		if diff := cmp.Diff(want, got, ctydebug.CmpOptions); diff != "" {
			t.Errorf("wrong result\n%s", diff)
		}

		if !mockProvider.ValidateProviderConfigCalled {
			t.Error("ValidateProviderConfig was not called; should've been")
		} else {
			got := mockProvider.ValidateProviderConfigRequest
			want := providers.ValidateProviderConfigRequest{
				Config: cty.ObjectVal(map[string]cty.Value{
					"test": cty.StringVal("yep"),
				}),
			}
			if diff := cmp.Diff(want, got, ctydebug.CmpOptions); diff != "" {
				t.Errorf("wrong request\n%s", diff)
			}
		}
	})
}

func TestCheckProviderInLockfile(t *testing.T) {
	providerAddr := addrs.NewDefaultProvider("testing")
	providerType := newProviderType(nil, providerAddr)

	mustConstraints := func(s string) constraints.IntersectionSpec {
		spec, err := constraints.ParseRubyStyleMulti(s)
		if err != nil {
			t.Fatalf("invalid constraint %q: %s", s, err)
		}
		return spec
	}

	// A lock file that records version 1.2.0 for the testing provider.
	lockedLocks := depsfile.NewLocks()
	lockedLocks.SetProvider(
		providerAddr,
		providerreqs.MustParseVersion("1.2.0"),
		providerreqs.MustParseVersionConstraints("~> 1.2.0"),
		providerreqs.PreferredHashes([]providerreqs.Hash{}),
	)

	tests := map[string]struct {
		locks       *depsfile.Locks
		constraints constraints.IntersectionSpec
		wantSummary string
	}{
		"present and satisfies constraints": {
			locks:       lockedLocks,
			constraints: mustConstraints("~> 1.2"),
			wantSummary: "",
		},
		"present with no constraints": {
			locks:       lockedLocks,
			constraints: nil,
			wantSummary: "",
		},
		"present but constraints no longer allow locked version": {
			locks:       lockedLocks,
			constraints: mustConstraints(">= 2.0.0"),
			wantSummary: "Provider version doesn't match the lockfile",
		},
		"missing from lockfile": {
			locks:       depsfile.NewLocks(),
			constraints: mustConstraints("~> 1.2"),
			wantSummary: "Provider missing from lockfile",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			diags := CheckProviderInLockfile(*tc.locks, providerType, tc.constraints, nil)

			if tc.wantSummary == "" {
				if diags.HasErrors() {
					t.Fatalf("unexpected diagnostics:\n%s", diags.Err())
				}
				return
			}

			if !diags.HasErrors() {
				t.Fatalf("expected error %q, got none", tc.wantSummary)
			}
			found := false
			for _, diag := range diags {
				if diag.Description().Summary == tc.wantSummary {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("expected diagnostic %q, got:\n%s", tc.wantSummary, diags.Err())
			}
		})
	}
}
