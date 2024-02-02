// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"reflect"
	"sync"
	"testing"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/providers"
)

func TestBuiltinEvalContextProviderInput(t *testing.T) {
	var lock sync.Mutex
	cache := make(map[string]map[string]cty.Value)

	ctx1 := testBuiltinEvalContext(t)
	ctx1 = ctx1.withScope(evalContextModuleInstance{Addr: addrs.RootModuleInstance}).(*BuiltinEvalContext)
	ctx1.ProviderInputConfig = cache
	ctx1.ProviderLock = &lock

	ctx2 := testBuiltinEvalContext(t)
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

	testP := &MockProvider{}

	ctx := testBuiltinEvalContext(t)
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

func testBuiltinEvalContext(t *testing.T) *BuiltinEvalContext {
	return &BuiltinEvalContext{}
}
