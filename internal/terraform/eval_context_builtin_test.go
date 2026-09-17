// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"reflect"
	"sync"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/namedvals"
	"github.com/hashicorp/terraform/internal/providers"
	testing_provider "github.com/hashicorp/terraform/internal/providers/testing"
	"github.com/hashicorp/terraform/internal/resources/ephemeral"
	"github.com/hashicorp/terraform/internal/states"
)

func TestBuiltinEvalContextProviderInput(t *testing.T) {
	var lock sync.Mutex
	cache := make(map[string]map[string]cty.Value)

	ctx1 := defaultTestCtx(t)
	ctx1 = ctx1.withScope(evalContextModuleInstance{Addr: addrs.RootModuleInstance}).(*BuiltinEvalContext)
	ctx1.ProviderInputConfig = cache
	ctx1.ProviderLock = &lock

	ctx2 := defaultTestCtx(t)
	ctx2 = ctx2.withScope(evalContextModuleInstance{Addr: addrs.RootModuleInstance.Child("child", addrs.NoKey)}).(*BuiltinEvalContext)
	ctx2.ProviderInputConfig = cache
	ctx2.ProviderLock = &lock

	providerAddr1 := addrs.AbsProviderConfig{
		Module:   addrs.RootModule,
		Provider: addrs.NewDefaultProvider("foo"),
	}
	providerAddr2 := addrs.AbsProviderConfig{
		Module:   addrs.RootModule.Child("child"),
		Provider: addrs.NewDefaultProvider("foo"),
	}

	expected1 := map[string]cty.Value{"value": cty.StringVal("foo")}
	ctx1.SetProviderInput(providerAddr1, expected1)

	try2 := map[string]cty.Value{"value": cty.StringVal("bar")}
	ctx2.SetProviderInput(providerAddr2, try2) // ignored because not a root module

	actual1 := ctx1.ProviderInput(providerAddr1)
	actual2 := ctx2.ProviderInput(providerAddr2)

	if !reflect.DeepEqual(actual1, expected1) {
		t.Errorf("wrong result 1\ngot:  %#v\nwant: %#v", actual1, expected1)
	}
	if actual2 != nil {
		t.Errorf("wrong result 2\ngot:  %#v\nwant: %#v", actual2, nil)
	}
}

func TestBuildingEvalContextInitProvider(t *testing.T) {
	var lock sync.Mutex

	testP := &testing_provider.MockProvider{}

	ctx := defaultTestCtx(t)
	ctx = ctx.withScope(evalContextModuleInstance{Addr: addrs.RootModuleInstance}).(*BuiltinEvalContext)
	ctx.ProviderLock = &lock
	ctx.ProviderCache = make(map[string]providers.Interface)
	ctx.Plugins = newContextPlugins(map[addrs.Provider]providers.Factory{
		addrs.NewDefaultProvider("test"): providers.FactoryFixed(testP),
	}, nil, nil)

	providerAddrDefault := addrs.AbsProviderConfig{
		Module:   addrs.RootModule,
		Provider: addrs.NewDefaultProvider("test"),
	}
	providerAddrAlias := addrs.AbsProviderConfig{
		Module:   addrs.RootModule,
		Provider: addrs.NewDefaultProvider("test"),
		Alias:    "foo",
	}
	providerAddrMock := addrs.AbsProviderConfig{
		Module:   addrs.RootModule,
		Provider: addrs.NewDefaultProvider("test"),
		Alias:    "mock",
	}

	_, err := ctx.InitProvider(providerAddrDefault, nil)
	if err != nil {
		t.Fatalf("error initializing provider test: %s", err)
	}
	_, err = ctx.InitProvider(providerAddrAlias, nil)
	if err != nil {
		t.Fatalf("error initializing provider test.foo: %s", err)
	}

	_, err = ctx.InitProvider(providerAddrMock, &configs.Provider{
		Mock: true,
	})
	if err != nil {
		t.Fatalf("error initializing provider test.mock: %s", err)
	}
}

func TestBuiltinEvalContextEvaluationScopeProviderFunctions(t *testing.T) {
	providerAddr := addrs.NewDefaultProvider("scope-functions")
	for name, tc := range map[string]struct {
		operation     walkOperation
		provider      addrs.Provider
		wantFunctions bool
	}{
		"init with unresolved provider": {
			operation: walkInit,
		},
		"init with resolved provider": {
			operation: walkInit,
			provider:  providerAddr,
		},
		"plan with resolved provider": {
			operation:     walkPlan,
			provider:      providerAddr,
			wantFunctions: true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			cfg := configs.NewEmptyConfig()
			cfg.Module.ProviderRequirements = &configs.RequiredProviders{
				RequiredProviders: map[string]*configs.RequiredProvider{
					"custom": {Name: "custom", Type: tc.provider},
				},
			}
			p := &testing_provider.MockProvider{
				GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
					Functions: map[string]providers.FunctionDecl{
						"example": {ReturnType: cty.String},
					},
				},
				CallFunctionFn: func(req providers.CallFunctionRequest) providers.CallFunctionResponse {
					return providers.CallFunctionResponse{Result: cty.StringVal("result")}
				},
			}
			ctx := testBuiltinEvalContext(t, tc.operation, cfg, nil, nil)
			ctx = ctx.withScope(evalContextModuleInstance{Addr: addrs.RootModuleInstance}).(*BuiltinEvalContext)
			ctx.Plugins = newContextPlugins(map[addrs.Provider]providers.Factory{
				providerAddr: providers.FactoryFixed(p),
			}, nil, nil)

			scope := ctx.EvaluationScope(nil, nil, EvalDataForNoInstanceKey)
			if _, exists := scope.ExternalFuncs.Provider["custom"]["example"]; exists != tc.wantFunctions {
				t.Errorf("provider function available = %t, want %t", exists, tc.wantFunctions)
			}

			expr, hclDiags := hclsyntax.ParseExpression([]byte(`provider::custom::example()`), "test.tf", hcl.InitialPos)
			if hclDiags.HasErrors() {
				t.Fatal(hclDiags.Error())
			}
			got, diags := scope.EvalExpr(expr, cty.String)
			if diags.HasErrors() {
				t.Fatal(diags.Err())
			}
			want := cty.UnknownVal(cty.String)
			if tc.wantFunctions {
				want = cty.StringVal("result")
			}
			if !got.RawEquals(want) {
				t.Errorf("wrong function result: got %#v, want %#v", got, want)
			}
			if p.GetProviderSchemaCalled != tc.wantFunctions {
				t.Errorf("provider schema requested = %t, want %t", p.GetProviderSchemaCalled, tc.wantFunctions)
			}
			if p.CallFunctionCalled != tc.wantFunctions {
				t.Errorf("provider function called = %t, want %t", p.CallFunctionCalled, tc.wantFunctions)
			}
		})
	}
}

var defaultTestCtx = func(t *testing.T) *BuiltinEvalContext {
	return testBuiltinEvalContext(t, walkPlan, nil, nil, nil)
}

func testBuiltinEvalContext(t *testing.T, op walkOperation, cfg *configs.Config, state *states.State, valState *namedvals.State) *BuiltinEvalContext {
	t.Helper()
	if state == nil {
		state = states.NewState()
	}
	if cfg == nil {
		cfg = configs.NewEmptyConfig()
	}
	if valState == nil {
		valState = namedvals.NewState()
	}
	ex := instances.NewExpander(nil)
	eph := ephemeral.NewResources()
	ev := &Evaluator{
		Config:             cfg,
		State:              state.SyncWrapper(),
		Operation:          op,
		NamedValues:        valState,
		Instances:          ex,
		EphemeralResources: eph,
	}
	return &BuiltinEvalContext{
		Evaluator:               ev,
		StateValue:              state.SyncWrapper(),
		PrevRunStateValue:       state.DeepCopy().SyncWrapper(),
		RefreshStateValue:       state.DeepCopy().SyncWrapper(),
		PolicyGraphValue:        newPolicySubgraph(),
		NamedValuesValue:        valState,
		ProviderLock:            &sync.Mutex{},
		ProviderCache:           make(map[string]providers.Interface),
		ProviderFuncCache:       make(map[string]providers.Interface),
		InstanceExpanderValue:   ex,
		EphemeralResourcesValue: eph,
	}
}
