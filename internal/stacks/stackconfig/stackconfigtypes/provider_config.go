// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackconfigtypes

import (
	"fmt"
	"reflect"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
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
		reflect.TypeOf(stackaddrs.AbsProviderConfigInstance{}),
		&cty.CapsuleOps{
			TypeGoString: func(goTy reflect.Type) string {
				return fmt.Sprintf(
					"stackconfigtypes.ProviderConfigType(addrs.MustParseProviderSourceString(%q))",
					providerAddr.String(),
				)
			},
			RawEquals: func(a, b interface{}) bool {
				return a.(*stackaddrs.AbsProviderConfigInstance).UniqueKey() == b.(*stackaddrs.AbsProviderConfigInstance).UniqueKey()
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

// ProviderInstanceValue encapsulates a provider config instance address in
// a cty.Value of the given provider config type, or panics if the type and
// address are inconsistent with one another.
func ProviderInstanceValue(ty cty.Type, addr stackaddrs.AbsProviderConfigInstance) cty.Value {
	wantProvider := ProviderForProviderConfigType(ty)
	if addr.Item.ProviderConfig.Provider != wantProvider {
		panic(fmt.Sprintf("can't use %s instance for %s reference", addr.Item.ProviderConfig.Provider, wantProvider))
	}
	return cty.CapsuleVal(ty, &addr)
}

// ProviderInstanceForValue returns the provider configuration instance
// address encapsulated inside the given value, or panics if the value is
// not of a provider configuration reference type.
//
// Use [IsProviderConfigType] with the value's type to check first if a
// given value is suitable to pass to this function.
func ProviderInstanceForValue(v cty.Value) stackaddrs.AbsProviderConfigInstance {
	if !IsProviderConfigType(v.Type()) {
		panic("not a provider config value")
	}
	addrP := v.EncapsulatedValue().(*stackaddrs.AbsProviderConfigInstance)
	return *addrP
}

// ProviderInstancePathsInValue searches the leaves of the given value,
// which can be of any type, and returns all of the paths that lead to
// provider configuration references in no particular order.
//
// This is primarily intended for returning errors when values are traversing
// out of the stacks runtime into other subsystems, since provider configuration
// references are a stacks-language-specific concept.
func ProviderInstancePathsInValue(v cty.Value) []cty.Path {
	var ret []cty.Path
	cty.Transform(v, func(p cty.Path, v cty.Value) (cty.Value, error) {
		if IsProviderConfigType(v.Type()) {
			ret = append(ret, p)
		}
		return cty.NilVal, nil
	})
	return ret
}

// ProviderConfigPathsInType searches the leaves of the given type and returns
// all of the paths that lead to provider configuration references in no
// particular order.
//
// This is a type-oriented version of [ProviderInstancePathsInValue], for
// situations in the language where an author describes a specific type
// constraint that must not include provider configuration reference types
// regardless of final value.
//
// Because this function deals in types rather than values, the returned
// paths will include unknown value placeholders for any index operations
// traversing through collections.
func ProviderConfigPathsInType(ty cty.Type) []cty.Path {
	return providerConfigPathsInType(ty, make(cty.Path, 0, 2))
}

func providerConfigPathsInType(ty cty.Type, prefix cty.Path) []cty.Path {
	var ret []cty.Path
	switch {
	case IsProviderConfigType(ty):
		// The rest of our traversal is constantly modifying the
		// backing array of the prefix slice, so we must make
		// a snapshot copy of it here to return.
		result := make(cty.Path, len(prefix))
		copy(result, prefix)
		ret = append(ret, result)
	case ty.IsListType():
		ret = providerConfigPathsInType(ty.ElementType(), prefix.Index(cty.UnknownVal(cty.Number)))
	case ty.IsMapType():
		ret = providerConfigPathsInType(ty.ElementType(), prefix.Index(cty.UnknownVal(cty.String)))
	case ty.IsSetType():
		ret = providerConfigPathsInType(ty.ElementType(), prefix.Index(cty.DynamicVal))
	case ty.IsTupleType():
		etys := ty.TupleElementTypes()
		ret = make([]cty.Path, 0, len(etys))
		for i, ety := range etys {
			ret = append(ret, providerConfigPathsInType(ety, prefix.IndexInt(i))...)
		}
	case ty.IsObjectType():
		atys := ty.AttributeTypes()
		ret = make([]cty.Path, 0, len(atys))
		for n, aty := range atys {
			ret = append(ret, providerConfigPathsInType(aty, prefix.GetAttr(n))...)
		}
	default:
		// No other types can potentially have nested provider configurations.
	}
	return ret
}

type providerConfigExtDataKeyType int

const providerConfigExtDataKey = providerConfigExtDataKeyType(0)
