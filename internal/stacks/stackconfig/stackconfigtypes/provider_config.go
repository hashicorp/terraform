package stackconfigtypes

import (
	"fmt"
	"reflect"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/zclconf/go-cty/cty"
)

// ProviderConfigType constructs a new cty capsule type for representing an
// active provider configuration.
//
// Each call to this function will produce a distinct type, even if the
// given provider address matches a previous call. Callers should retain their
// own data structure of previously-constructed to ensure that they will use
// only a single type per distinct provider address.
func ProviderConfigType(providerAddr addrs.Provider) cty.Type {
	return cty.CapsuleWithOps(
		fmt.Sprintf("configuration for %s provider", providerAddr.ForDisplay()),
		reflect.TypeOf(providers.Interface(nil)),
		&cty.CapsuleOps{
			TypeGoString: func(goTy reflect.Type) string {
				return fmt.Sprintf(
					"stackconfigtypes.ProviderConfigType(addrs.MustParseProviderSourceString(%q))",
					providerAddr.String(),
				)
			},
			RawEquals: func(a, b interface{}) bool {
				// NOTE: This assumes that providers.Interface implementations
				// are always comparable. That's true for the real ones we
				// use to represent external plugins, since they are pointers,
				// but this will fail for e.g. a mock implementation used in
				// tests if it isn't a pointer and contains something
				// non-comparable.
				return a == b
			},
			ExtensionData: func(key interface{}) interface{} {
				switch key {
				case providerConfigExtDataKey:
					return providerAddr
				default:
					return nil
				}
			},
		},
	)
}

// IsProviderConfigType returns true if the given type is one that was
// previously constructed with [ProviderConfigType], or false otherwise.
//
// If and only if this function returns true, callers can use
// [ProviderForProviderConfigType] to learn which specific provider the
// type is representing configurations for.
func IsProviderConfigType(ty cty.Type) bool {
	if !ty.IsCapsuleType() {
		return false
	}
	ops := ty.CapsuleOps()
	if ops == nil {
		return false
	}
	providerAddrI := ops.ExtensionData(providerConfigExtDataKey)
	return providerAddrI != nil
}

// ProviderForProviderConfigType returns the address of the provider that
// the given type can store instances of, or panics if the given type is
// not one produced by an earlier call to [ProviderConfigType].
//
// Use [IsProviderConfigType] before calling to confirm whether an unknown
// type is safe to pass to this function.
func ProviderForProviderConfigType(ty cty.Type) addrs.Provider {
	if !ty.IsCapsuleType() {
		panic("not a provider config type")
	}
	ops := ty.CapsuleOps()
	if ops == nil {
		panic("not a provider config type")
	}
	providerAddrI := ops.ExtensionData(providerConfigExtDataKey)
	return providerAddrI.(addrs.Provider)
}

type providerConfigExtDataKeyType int

const providerConfigExtDataKey = providerConfigExtDataKeyType(0)
