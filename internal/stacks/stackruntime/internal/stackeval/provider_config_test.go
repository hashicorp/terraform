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
	"github.com/zclconf/go-cty-debug/ctydebug"
	"github.com/zclconf/go-cty/cty"
)

func TestProviderConfigCheckProviderArgs(t *testing.T) {
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
	getProviderConfig := func(ctx context.Context, t *testing.T, main *Main) *ProviderConfig {
		t.Helper()
		mainStack := main.MainStack(ctx)
		provider := mainStack.Provider(ctx, stackaddrs.ProviderConfig{
			Provider: providerTypeAddr,
			Name:     "bar",
		})
		if provider == nil {
			t.Fatal("no provider.foo.bar is available")
		}
		return provider.Config(ctx)
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
		got, diags := config.CheckProviderArgs(ctx)
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
		got, diags := config.CheckProviderArgs(ctx)
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
