// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// ProviderFactories is a collection of factory functions for starting new
// instances of various providers.
type ProviderFactories map[addrs.Provider]providers.Factory

func (pf ProviderFactories) ProviderAvailable(providerAddr addrs.Provider) bool {
	_, available := pf[providerAddr]
	return available
}

// NewUnconfiguredClient launches a new instance of the requested provider,
// if available, and returns it in an unconfigured state.
//
// Callers that need a _configured_ provider can then call
// [providers.Interface.Configure] on the result to configure it, making it
// ready for the majority of operations that require a configured provider.
func (pf ProviderFactories) NewUnconfiguredClient(providerAddr addrs.Provider) (providers.Interface, error) {
	f, ok := pf[providerAddr]
	if !ok {
		return nil, fmt.Errorf("provider is not available in this execution context")
	}
	return f()
}

// unconfigurableProvider is a wrapper around a provider.Interface that
// prevents it from being configured. This is because the underlying interface
// should already have been configured by the time we get here, or should never
// be configured.
//
// In addition, unconfigurableProviders are not closeable, because they should
// be closed by the external thing that configured them when they are done
// with them.
type unconfigurableProvider struct {
	providers.Interface
}

var _ providers.Interface = unconfigurableProvider{}

func (p unconfigurableProvider) Close() error {
	// whatever created the underlying provider should be responsible for
	// closing it, so we'll do nothing here.
	return nil
}

func (p unconfigurableProvider) ConfigureProvider(request providers.ConfigureProviderRequest) providers.ConfigureProviderResponse {
	// the real provider should either already have been configured by the time
	// we get here or should never get configured, so we should never see this
	// method called.
	return providers.ConfigureProviderResponse{
		Diagnostics: tfdiags.Diagnostics{
			tfdiags.AttributeValue(
				tfdiags.Error,
				"Called ConfigureProvider on an unconfigurable provider",
				"This provider should have already been configured, or should never be configured. This is a bug in Terraform - please report it.",
				nil, // nil attribute path means the overall configuration block
			),
		},
	}
}
