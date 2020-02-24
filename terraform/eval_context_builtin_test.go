package terraform

import (
	"reflect"
	"sync"
	"testing"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/providers"
	"github.com/zclconf/go-cty/cty"
)

func TestBuiltinEvalContextProviderInput(t *testing.T) {
	var lock sync.Mutex
	cache := make(map[string]map[string]cty.Value)

	ctx1 := testBuiltinEvalContext(t)
	ctx1.PathValue = addrs.RootModuleInstance
	ctx1.ProviderInputConfig = cache
	ctx1.ProviderLock = &lock

	ctx2 := testBuiltinEvalContext(t)
	ctx2.PathValue = addrs.RootModuleInstance.Child("child", addrs.NoKey)
	ctx2.ProviderInputConfig = cache
	ctx2.ProviderLock = &lock

	providerAddr1 := addrs.AbsProviderConfig{
		Module:   addrs.RootModuleInstance,
		Provider: addrs.NewLegacyProvider("foo"),
	}
	providerAddr2 := addrs.AbsProviderConfig{
		Module:   addrs.RootModuleInstance.Child("child", addrs.NoKey),
		Provider: addrs.NewLegacyProvider("foo"),
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
	ctx.ProviderLock = &lock
	ctx.ProviderCache = make(map[string]providers.Interface)
	ctx.Components = &basicComponentFactory{
		providers: map[addrs.Provider]providers.Factory{
			addrs.NewLegacyProvider("test"): providers.FactoryFixed(testP),
		},
	}

	providerAddrDefault := addrs.AbsProviderConfig{
		Module:   addrs.RootModuleInstance,
		Provider: addrs.NewLegacyProvider("test"),
	}
	providerAddrAlias := addrs.AbsProviderConfig{
		Module:   addrs.RootModuleInstance,
		Provider: addrs.NewLegacyProvider("test"),
		Alias:    "foo",
	}

	_, err := ctx.InitProvider(providerAddrDefault)
	if err != nil {
		t.Fatalf("error initializing provider test: %s", err)
	}
	_, err = ctx.InitProvider(providerAddrAlias)
	if err != nil {
		t.Fatalf("error initializing provider test.foo: %s", err)
	}
}

func testBuiltinEvalContext(t *testing.T) *BuiltinEvalContext {
	return &BuiltinEvalContext{}
}
