package lang

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
)

// providerConfigTypes is a memoizing cache for results from ProviderConfigType,
// which we use primarily to ensure that our provider config types meet the
// expected cty contract that two capsule types are equal if they were created
// by the same call to cty.CapsuleWithOps.
//
// This design assumes that a typical Terraform run will only encounter a
// relatively small, finite number of distinct provider types, which will be
// constrained by whichever providers are listed in the given configuration's
// dependency lock file. Therefore this map should not grow uncontrolled and
// does not need to be explicitly cleaned up.
//
// ProviderConfigType must hold providerConfigTypesMu before accessing this.
var providerConfigTypes map[addrs.Provider]cty.Type
var providerConfigTypesMu sync.Mutex

// ProviderConfigType returns a cty capsule type which represents a reference
// to a configuration of the given provider.
func ProviderConfigType(addr addrs.Provider) cty.Type {
	providerConfigTypesMu.Lock()
	defer providerConfigTypesMu.Unlock()

	if existing, exists := providerConfigTypes[addr]; exists {
		return existing
	}

	// We start with a shallow copy of providerConfigTypeBaseCapsuleOps
	// and then customize it slightly, because many of our operations are
	// identical for all of our provider configuration types.
	ops := providerConfigTypeBaseCapsuleOps
	ops.TypeGoString = func(goTy reflect.Type) string {
		return fmt.Sprintf("lang.ProviderConfigType(addrs.MustParseProviderSourceString(%q))", addr)
	}

	ty := cty.CapsuleWithOps(
		fmt.Sprintf("configuration for %s", addr),
		reflect.TypeOf(addrs.AbsProviderConfig{}),
		&ops,
	)

	if providerConfigTypes == nil {
		providerConfigTypes = make(map[addrs.Provider]cty.Type)
	}
	providerConfigTypes[addr] = ty

	return ty
}

// ProviderConfigValue returns a value representing a particular absolute
// provider configuration.
//
// The type of the result is always equal to the type that would be returned
// by passing addr.Provider to [ProviderConfigType].
func ProviderConfigValue(addr addrs.AbsProviderConfig) cty.Value {
	ty := ProviderConfigType(addr.Provider)
	return cty.CapsuleVal(ty, &addr)
}

// ProviderConfigFromValue returns the provider configuration address
// encapsulated in the given value, which must be of a type previously
// returned by [ProviderConfigType].
//
// If the given value is not a known, non-known value of an appropriate
// type then this function will panic.
func ProviderConfigFromValue(v cty.Value) addrs.AbsProviderConfig {
	raw := v.EncapsulatedValue()
	return *raw.(*addrs.AbsProviderConfig)
}

// providerConfigTypeBaseCapsuleOps represents all of the common parts of
// the CapsuleOps objects produced by ProviderConfigType. That function must
// shallow-copy this object and then override the parts that need to vary
// depending on the actual provider type.
var providerConfigTypeBaseCapsuleOps = cty.CapsuleOps{
	GoString: func(val interface{}) string {
		return fmt.Sprintf("lang.ProviderConfigValue(%#v)", val)
	},
	// NOTE: ProviderConfigType must set TypeGoString, because it varies
	// for each provider type.

	// Two provider config values are equal if they refer to the same provider
	// configuration address.
	RawEquals: func(a, b interface{}) bool {
		addrA := *a.(*addrs.AbsProviderConfig)
		addrB := *b.(*addrs.AbsProviderConfig)
		return addrA.Equal(addrB)
	},
	HashKey: func(v interface{}) string {
		addr := *v.(*addrs.AbsProviderConfig)
		return addr.String()
	},
}
