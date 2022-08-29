package lang

import (
	"testing"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/zclconf/go-cty/cty"
)

func TestProviderConfigType(t *testing.T) {
	providerA := addrs.NewDefaultProvider("a")
	providerB := addrs.NewBuiltInProvider("b")

	tyA := ProviderConfigType(providerA)
	tyB := ProviderConfigType(providerB)

	valADefault := ProviderConfigValue(addrs.AbsProviderConfig{
		Module:   addrs.RootModule,
		Provider: providerA,
		Alias:    "",
	})
	valADefault2 := ProviderConfigValue(addrs.AbsProviderConfig{
		Module:   addrs.RootModule,
		Provider: providerA,
		Alias:    "",
	})
	valAAdditional := ProviderConfigValue(addrs.AbsProviderConfig{
		Module:   addrs.RootModule,
		Provider: providerA,
		Alias:    "foo",
	})
	valBDefault := ProviderConfigValue(addrs.AbsProviderConfig{
		Module:   addrs.RootModule,
		Provider: providerB,
		Alias:    "",
	})
	valBInModule := ProviderConfigValue(addrs.AbsProviderConfig{
		Module:   addrs.RootModule.Child("child"),
		Provider: providerB,
		Alias:    "",
	})

	if tyA.Equals(tyB) {
		t.Fatalf("type A and type B are equal; should not be")
	}
	if !tyA.Equals(ProviderConfigType(providerA)) {
		t.Fatalf("type A does not equal a newly-constructed instance of itself")
	}
	if !tyB.Equals(ProviderConfigType(providerB)) {
		t.Fatalf("type B does not equal a newly-constructed instance of itself")
	}

	if !valADefault.Type().Equals(tyA) {
		t.Fatalf("default provider value for provider A does not have provider A's type")
	}
	if !valAAdditional.Type().Equals(tyA) {
		t.Fatalf("additional provider value for provider A does not have provider A's type")
	}
	if !valBDefault.Type().Equals(tyB) {
		t.Fatalf("default provider value for provider B does not have provider B's type")
	}
	if !valBInModule.Type().Equals(tyB) {
		t.Fatalf("in-module provider value for provider B does not have provider B's type")
	}

	if valADefault.RawEquals(valAAdditional) {
		t.Errorf("%#v equals %#v; should be distinct", valADefault, valAAdditional)
	}
	if valADefault.RawEquals(valBDefault) {
		t.Errorf("%#v equals %#v; should be distinct", valADefault, valBDefault)
	}
	if valBDefault.RawEquals(valBInModule) {
		t.Errorf("%#v equals %#v; should be distinct", valBDefault, valBInModule)
	}
	if !valADefault.RawEquals(valADefault2) {
		t.Errorf("%#v does not equal %#v; should be equal", valADefault, valADefault2)
	}

	configSet := cty.SetVal([]cty.Value{
		valADefault,
		valADefault2, // (this one should coalesce with valADefault)
		valAAdditional,
	})
	if got, want := configSet.LengthInt(), 2; got != want {
		t.Errorf("set of configurations has %d elements; want %d", got, want)
	}
	vSet := configSet.AsValueSet()
	if want := valADefault; !vSet.Has(want) {
		t.Errorf("set of configurations missing expected element %#v", want)
	}
	if want := valADefault2; !vSet.Has(want) {
		t.Errorf("set of configurations missing expected element %#v", want)
	}
	if want := valAAdditional; !vSet.Has(want) {
		t.Errorf("set of configurations missing expected element %#v", want)
	}
	if doNotWant := valBDefault; vSet.Has(doNotWant) {
		t.Errorf("set of configurations has unexpected element %#v", doNotWant)
	}
}
