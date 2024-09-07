// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"context"
	"fmt"
	"sync"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/promising"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type ProviderType struct {
	mu sync.Mutex

	addr addrs.Provider

	main *Main

	schema             promising.Once[providers.GetProviderSchemaResponse]
	unconfiguredClient providers.Interface
}

func newProviderType(main *Main, addr addrs.Provider) *ProviderType {
	return &ProviderType{
		addr: addr,
		main: main,
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
func (pt *ProviderType) UnconfiguredClient() (providers.Interface, error) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	if pt.unconfiguredClient == nil {
		client, err := pt.main.ProviderFactories().NewUnconfiguredClient(pt.Addr())
		if err != nil {
			return nil, err
		}
		pt.unconfiguredClient = client

		pt.main.RegisterCleanup(func(_ context.Context) tfdiags.Diagnostics {
			var diags tfdiags.Diagnostics
			if err := pt.unconfiguredClient.Close(); err != nil {
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Failed to terminate provider plugin",
					fmt.Sprintf(
						"Error closing the unconfigured instance of %s: %s.",
						pt.Addr(), err,
					),
				))
			}
			return diags
		})
	}

	return unconfigurableProvider{
		Interface: pt.unconfiguredClient,
	}, nil
}

func (pt *ProviderType) Schema(ctx context.Context) (providers.GetProviderSchemaResponse, error) {
	return pt.schema.Do(ctx, func(ctx context.Context) (providers.GetProviderSchemaResponse, error) {
		client, err := pt.UnconfiguredClient()
		if err != nil {
			return providers.GetProviderSchemaResponse{}, fmt.Errorf("provider startup failed: %w", err)
		}

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
