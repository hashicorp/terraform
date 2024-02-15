// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/hashicorp/terraform/version"
	"github.com/zclconf/go-cty-debug/ctydebug"
	"github.com/zclconf/go-cty/cty"
)

func TestProviderInstanceCheckProviderArgs(t *testing.T) {
	cfg := testStackConfig(t, "provider", "single_instance_configured")
	providerTypeAddr := addrs.NewBuiltInProvider("foo")
	newMockProvider := func(t *testing.T) (*terraform.MockProvider, providers.Factory) {
		t.Helper()
		mockProvider := &terraform.MockProvider{
			GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
				Provider: providers.Schema{
					Block: &configschema.Block{
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
	getProviderInstance := func(ctx context.Context, t *testing.T, main *Main) *ProviderInstance {
		t.Helper()
		mainStack := main.MainStack(ctx)
		provider := mainStack.Provider(ctx, stackaddrs.ProviderConfig{
			Provider: providerTypeAddr,
			Name:     "bar",
		})
		if provider == nil {
			t.Fatal("no provider.foo.bar is available")
		}
		insts := provider.Instances(ctx, InspectPhase)
		inst, ok := insts[addrs.NoKey]
		if !ok {
			t.Fatal("missing NoKey instance of provider.foo.bar")
		}
		return inst
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
		inst := getProviderInstance(ctx, t, main)

		want := cty.ObjectVal(map[string]cty.Value{
			"test": cty.StringVal("yep"),
		})
		got, diags := inst.CheckProviderArgs(ctx, InspectPhase)
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
		inst := getProviderInstance(ctx, t, main)

		want := cty.ObjectVal(map[string]cty.Value{
			"test": cty.StringVal("yep").Mark("nope"),
		})
		got, diags := inst.CheckProviderArgs(ctx, InspectPhase)
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
	subtestInPromisingTask(t, "valid with no config block at all", func(ctx context.Context, t *testing.T) {
		// For this one we'll use a different configuration fixture that
		// doesn't include a "config" block at all.
		cfg := testStackConfig(t, "provider", "single_instance")

		mockProvider, providerFactory := newMockProvider(t)
		main := testEvaluator(t, testEvaluatorOpts{
			Config: cfg,
			ProviderFactories: ProviderFactories{
				providerTypeAddr: providerFactory,
			},
		})
		inst := getProviderInstance(ctx, t, main)

		// We'll make sure the configuration really does omit the config
		// block, in case someone modifies the fixture in future without
		// realizing we're relying on that invariant here.
		decl := inst.provider.Declaration(ctx)
		if decl.Config != nil {
			t.Fatal("test fixture has a config block for the provider; should omit it")
		}

		want := cty.ObjectVal(map[string]cty.Value{
			"test": cty.NullVal(cty.String),
		})
		got, diags := inst.CheckProviderArgs(ctx, InspectPhase)
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
					"test": cty.NullVal(cty.String),
				}),
			}
			if diff := cmp.Diff(want, got, ctydebug.CmpOptions); diff != "" {
				t.Errorf("wrong request\n%s", diff)
			}
		}
	})
	subtestInPromisingTask(t, "invalid per schema", func(ctx context.Context, t *testing.T) {
		mockProvider, providerFactory := newMockProvider(t)
		main := testEvaluator(t, testEvaluatorOpts{
			Config: cfg,
			TestOnlyGlobals: map[string]cty.Value{
				"provider_configuration": cty.EmptyObjectVal,
			},
			ProviderFactories: ProviderFactories{
				providerTypeAddr: providerFactory,
			},
		})
		inst := getProviderInstance(ctx, t, main)

		_, diags := inst.CheckProviderArgs(ctx, InspectPhase)
		assertMatchingDiag(t, diags, func(diag tfdiags.Diagnostic) bool {
			// The "test" argument expects a string, but we assigned an object
			return diag.Severity() == tfdiags.Error && diag.Description().Summary == `Incorrect attribute value type`
		})
		if mockProvider.ValidateProviderConfigCalled {
			t.Error("ValidateProviderConfig was called, but should not have been because the config didn't conform to the schema")
		}
	})
	subtestInPromisingTask(t, "invalid per provider logic", func(ctx context.Context, t *testing.T) {
		mockProvider, providerFactory := newMockProvider(t)
		main := testEvaluator(t, testEvaluatorOpts{
			Config: cfg,
			TestOnlyGlobals: map[string]cty.Value{
				"provider_configuration": cty.StringVal("invalid"),
			},
			ProviderFactories: ProviderFactories{
				providerTypeAddr: providerFactory,
			},
		})
		inst := getProviderInstance(ctx, t, main)

		_, diags := inst.CheckProviderArgs(ctx, InspectPhase)
		assertMatchingDiag(t, diags, func(diag tfdiags.Diagnostic) bool {
			return diag.Severity() == tfdiags.Error && diag.Description().Summary == `invalid value checked by provider itself`
		})
		if !mockProvider.ValidateProviderConfigCalled {
			// It would be strange to get here because that would suggest
			// that we got the diagnostic from the provider without asking
			// the provider for it. Is terraform.MockProvider broken?
			t.Error("ValidateProviderConfig was not called, but should have been")
		} else {
			got := mockProvider.ValidateProviderConfigRequest
			want := providers.ValidateProviderConfigRequest{
				Config: cty.ObjectVal(map[string]cty.Value{
					"test": cty.StringVal("invalid"),
				}),
			}
			if diff := cmp.Diff(want, got, ctydebug.CmpOptions); diff != "" {
				t.Errorf("wrong request\n%s", diff)
			}
		}
	})
	subtestInPromisingTask(t, "can't fetch schema at all", func(ctx context.Context, t *testing.T) {
		mockProvider, providerFactory := newMockProvider(t)
		mockProvider.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
			Diagnostics: tfdiags.Diagnostics(nil).Append(fmt.Errorf("nope")),
		}
		main := testEvaluator(t, testEvaluatorOpts{
			Config: cfg,
			TestOnlyGlobals: map[string]cty.Value{
				"provider_configuration": cty.EmptyObjectVal,
			},
			ProviderFactories: ProviderFactories{
				providerTypeAddr: providerFactory,
			},
		})
		inst := getProviderInstance(ctx, t, main)

		_, diags := inst.CheckProviderArgs(ctx, InspectPhase)
		assertMatchingDiag(t, diags, func(diag tfdiags.Diagnostic) bool {
			return diag.Severity() == tfdiags.Error && diag.Description().Summary == `Failed to read provider schema`
		})
	})
	subtestInPromisingTask(t, "provider doesn't even start up", func(ctx context.Context, t *testing.T) {
		providerFactory := providers.Factory(func() (providers.Interface, error) {
			return nil, fmt.Errorf("uh-oh")
		})
		main := testEvaluator(t, testEvaluatorOpts{
			Config: cfg,
			TestOnlyGlobals: map[string]cty.Value{
				"provider_configuration": cty.EmptyObjectVal,
			},
			ProviderFactories: ProviderFactories{
				providerTypeAddr: providerFactory,
			},
		})
		inst := getProviderInstance(ctx, t, main)

		_, diags := inst.CheckProviderArgs(ctx, InspectPhase)
		assertMatchingDiag(t, diags, func(diag tfdiags.Diagnostic) bool {
			return diag.Severity() == tfdiags.Error && diag.Description().Summary == `Failed to read provider schema`
		})
	})
}

func TestProviderInstanceCheckClient(t *testing.T) {
	cfg := testStackConfig(t, "provider", "single_instance_configured")
	providerTypeAddr := addrs.NewBuiltInProvider("foo")
	newMockProvider := func(t *testing.T) (*terraform.MockProvider, providers.Factory) {
		t.Helper()
		mockProvider := &terraform.MockProvider{
			GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
				Provider: providers.Schema{
					Block: &configschema.Block{
						Attributes: map[string]*configschema.Attribute{
							"test": {
								Type:     cty.String,
								Optional: true,
							},
						},
					},
				},
			},
			ConfigureProviderFn: func(vpcr providers.ConfigureProviderRequest) providers.ConfigureProviderResponse {
				if vpcr.Config.ContainsMarked() {
					panic("config has marks")
				}
				var diags tfdiags.Diagnostics
				if vpcr.Config.Type().HasAttribute("test") {
					if vpcr.Config.GetAttr("test").RawEquals(cty.StringVal("invalid")) {
						diags = diags.Append(fmt.Errorf("invalid value checked by provider itself"))
					}
				}
				return providers.ConfigureProviderResponse{
					Diagnostics: diags,
				}
			},
		}
		providerFactory := providers.FactoryFixed(mockProvider)
		return mockProvider, providerFactory
	}
	getProviderInstance := func(ctx context.Context, t *testing.T, main *Main) *ProviderInstance {
		t.Helper()
		mainStack := main.MainStack(ctx)
		provider := mainStack.Provider(ctx, stackaddrs.ProviderConfig{
			Provider: providerTypeAddr,
			Name:     "bar",
		})
		if provider == nil {
			t.Fatal("no provider.foo.bar is available")
		}
		insts := provider.Instances(ctx, InspectPhase)
		inst, ok := insts[addrs.NoKey]
		if !ok {
			t.Fatal("missing NoKey instance of provider.foo.bar")
		}
		return inst
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
		inst := getProviderInstance(ctx, t, main)

		client, diags := inst.CheckClient(ctx, InspectPhase)
		assertNoDiags(t, diags)

		switch c := client.(type) {
		case providerClose:
			break
		default:
			t.Errorf("unexpected client type %#T", c)
		}

		if !mockProvider.ConfigureProviderCalled {
			t.Error("ConfigureProvider was not called; should've been")
		} else {
			got := mockProvider.ConfigureProviderRequest
			want := providers.ConfigureProviderRequest{
				TerraformVersion: version.SemVer.String(),
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
		inst := getProviderInstance(ctx, t, main)

		client, diags := inst.CheckClient(ctx, InspectPhase)
		assertNoDiags(t, diags)

		switch c := client.(type) {
		case providerClose:
			break
		default:
			t.Errorf("unexpected client type %#T", c)
		}

		if !mockProvider.ConfigureProviderCalled {
			t.Error("ConfigureProvider was not called; should've been")
		} else {
			got := mockProvider.ConfigureProviderRequest
			want := providers.ConfigureProviderRequest{
				TerraformVersion: version.SemVer.String(),
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
