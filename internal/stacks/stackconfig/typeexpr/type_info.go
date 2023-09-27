// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package typeexpr

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/zclconf/go-cty/cty"
)

// TypeInformation is an interface used to give [TypeConstraint] information
// about its surrounding environment to use during type expression decoding.
//
// [TypeInformation]'s API is not concurrency-safe, so the same object should
// not be passed to multiple concurrent calls of [TypeConstraint].
type TypeInformation interface {
	// SetProviderConfigType stores a capsule type allocated to represent
	// provider configurations for the given provider. The same type
	// will then be returned from subsequent calls to ProviderConfigType
	// using the same provider address.
	SetProviderConfigType(providerAddr addrs.Provider, ty cty.Type)

	// ProviderConfigType retrieves a provider configurationc capsule type
	// previously stored by SetProviderConfigType, or [cty.NilType] if
	// there was no previous call for the given provider address.
	ProviderConfigType(providerAddr addrs.Provider) cty.Type

	// ProviderForLocalName translates a provider local name into its
	// corresponding fully-qualified provider address, or sets its second
	// return value to false if there is no such local name defined.
	ProviderForLocalName(localName string) (addrs.Provider, bool)
}
