// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/promising"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/zclconf/go-cty/cty"
)

type ProviderType struct {
	addr addrs.Provider

	main *Main

	schema             promising.Once[providers.GetProviderSchemaResponse]
	unconfiguredClient rcProviderClient
}

func newProviderType(main *Main, addr addrs.Provider) *ProviderType {
	return &ProviderType{
		addr: addr,
		main: main,
		unconfiguredClient: rcProviderClient{
			Factory: func() (providers.Interface, error) {
				return main.ProviderFactories().NewUnconfiguredClient(addr)
			},
		},
	}
}

func (pt *ProviderType) Addr() addrs.Provider {
	return pt.addr
}

// ProviderRefType returns the cty capsule type that represents references to
// providers of this type when passed through expressions.
func (pt *ProviderType) ProviderRefType() cty.Type {
	allTypes := pt.main.ProviderRefTypes()
	return allTypes[pt.Addr()]
}

// UnconfiguredClient returns the client for the singleton unconfigured
// provider of this type, initializing the provider first if necessary.
//
// Callers must call Close on the returned client once they are finished
// with it, which will internally decrement a reference count so that
// the shared provider can be eventually closed once no longer needed.
func (pt *ProviderType) UnconfiguredClient(ctx context.Context) (providers.Interface, error) {
	return pt.unconfiguredClient.Borrow(ctx)
}

func (pt *ProviderType) Schema(ctx context.Context) (providers.GetProviderSchemaResponse, error) {
	return pt.schema.Do(ctx, func(ctx context.Context) (providers.GetProviderSchemaResponse, error) {
		client, err := pt.UnconfiguredClient(ctx)
		if err != nil {
			return providers.GetProviderSchemaResponse{}, fmt.Errorf("provider startup failed: %w", err)
		}
		defer client.Close()

		ret := client.GetProviderSchema()
		if ret.Diagnostics.HasErrors() {
			return providers.GetProviderSchemaResponse{}, fmt.Errorf("provider failed to return its schema")
		}
		return ret, nil
	})
}

// reportNamedPromises implements namedPromiseReporter.
func (pt *ProviderType) reportNamedPromises(cb func(id promising.PromiseID, name string)) {
	cb(pt.schema.PromiseID(), pt.Addr().String()+" schema")
}
