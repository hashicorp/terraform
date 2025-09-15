// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package jsonplan

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/msgpack"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/terraform"
)

func TestMarshalActionInvocations(t *testing.T) {

	action := addrs.AbsActionInstance{
		Module: addrs.RootModuleInstance,
		Action: addrs.ActionInstance{
			Action: addrs.Action{
				Type: "test_action",
				Name: "test",
			},
			Key: addrs.NoKey,
		},
	}

	provider := addrs.AbsProviderConfig{
		Module: addrs.RootModule,
		Provider: addrs.Provider{
			Type:      "test",
			Namespace: "hashicorp",
			Hostname:  addrs.DefaultProviderRegistryHost,
		},
	}

	schemas := &terraform.Schemas{
		Providers: map[addrs.Provider]providers.ProviderSchema{
			provider.Provider: {
				Actions: map[string]providers.ActionSchema{
					"test_action": {
						ConfigSchema: &configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"optional": {
									Type: cty.String,
								},
								"sensitive": {
									Type:      cty.String,
									Sensitive: true,
								},
							},
						},
					},
				},
			},
		},
	}

	tcs := map[string]struct {
		input  *plans.ActionInvocationInstanceSrc
		output ActionInvocation
	}{
		"no metadata": {
			input: &plans.ActionInvocationInstanceSrc{
				Addr:          action,
				ActionTrigger: new(plans.InvokeActionTrigger),
				ConfigValue: mustDynamicValue(t, cty.ObjectVal(map[string]cty.Value{
					"optional":  cty.StringVal("hello"),
					"sensitive": cty.StringVal("world"),
				})),
				SensitiveConfigPaths: nil,
				ProviderAddr:         provider,
			},
			output: ActionInvocation{
				Address: "action.test_action.test",
				Type:    "test_action",
				Name:    "test",
				ConfigValues: mustJson(t, map[string]interface{}{
					"optional":  "hello",
					"sensitive": "world",
				}),
				ConfigSensitive: mustJson(t, map[string]interface{}{
					"sensitive": true,
				}),
				ConfigUnknown:       mustJson(t, map[string]interface{}{}),
				ProviderName:        "registry.terraform.io/hashicorp/test",
				InvokeActionTrigger: new(InvokeActionTrigger),
			},
		},
		"unknown value": {
			input: &plans.ActionInvocationInstanceSrc{
				Addr:          action,
				ActionTrigger: new(plans.InvokeActionTrigger),
				ConfigValue: mustDynamicValue(t, cty.ObjectVal(map[string]cty.Value{
					"optional":  cty.UnknownVal(cty.String),
					"sensitive": cty.StringVal("world"),
				})),
				SensitiveConfigPaths: nil,
				ProviderAddr:         provider,
			},
			output: ActionInvocation{
				Address: "action.test_action.test",
				Type:    "test_action",
				Name:    "test",
				ConfigValues: mustJson(t, map[string]interface{}{
					"sensitive": "world",
				}),
				ConfigSensitive: mustJson(t, map[string]interface{}{
					"sensitive": true,
				}),
				ConfigUnknown: mustJson(t, map[string]interface{}{
					"optional": true,
				}),
				ProviderName:        "registry.terraform.io/hashicorp/test",
				InvokeActionTrigger: new(InvokeActionTrigger),
			},
		},
		"extra sensitive": {
			input: &plans.ActionInvocationInstanceSrc{
				Addr:          action,
				ActionTrigger: new(plans.InvokeActionTrigger),
				ConfigValue: mustDynamicValue(t, cty.ObjectVal(map[string]cty.Value{
					"optional":  cty.StringVal("hello"),
					"sensitive": cty.StringVal("world"),
				})),
				SensitiveConfigPaths: []cty.Path{cty.GetAttrPath("optional")},
				ProviderAddr:         provider,
			},
			output: ActionInvocation{
				Address: "action.test_action.test",
				Type:    "test_action",
				Name:    "test",
				ConfigValues: mustJson(t, map[string]interface{}{
					"optional":  "hello",
					"sensitive": "world",
				}),
				ConfigSensitive: mustJson(t, map[string]interface{}{
					"optional":  true,
					"sensitive": true,
				}),
				ConfigUnknown:       mustJson(t, map[string]interface{}{}),
				ProviderName:        "registry.terraform.io/hashicorp/test",
				InvokeActionTrigger: new(InvokeActionTrigger),
			},
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			out, err := MarshalActionInvocation(tc.input, schemas)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(tc.output, out); len(diff) > 0 {
				t.Fatal(diff)
			}
		})
	}

}

func mustDynamicValue(t *testing.T, value cty.Value) []byte {
	out, err := msgpack.Marshal(value, value.Type())
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func mustJson(t *testing.T, data interface{}) json.RawMessage {
	out, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}
	return out
}
