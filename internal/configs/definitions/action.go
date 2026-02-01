// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package definitions

import (
	"github.com/hashicorp/hcl/v2"

	"github.com/hashicorp/terraform/internal/addrs"
)

// Action represents an "action" block inside a configuration
type Action struct {
	Name    string
	Type    string
	Config  hcl.Body
	Count   hcl.Expression
	ForEach hcl.Expression

	ProviderConfigRef *ProviderConfigRef
	Provider          addrs.Provider

	DeclRange hcl.Range
	TypeRange hcl.Range
}

// ModuleUniqueKey returns a unique key for this action within a module.
func (a *Action) ModuleUniqueKey() string {
	return a.Addr().String()
}

// Addr returns an action address for the receiver that is relative to the
// action's containing module.
func (a *Action) Addr() addrs.Action {
	return addrs.Action{
		Type: a.Type,
		Name: a.Name,
	}
}

// ProviderConfigAddr returns the address for the provider configuration that
// should be used for this action. This function returns a default provider
// config addr if an explicit "provider" argument was not provided.
func (a *Action) ProviderConfigAddr() addrs.LocalProviderConfig {
	if a.ProviderConfigRef == nil {
		// If no specific "provider" argument is given, we want to look up the
		// provider config where the local name matches the implied provider
		// from the resource type. This may be different from the resource's
		// provider type.
		return addrs.LocalProviderConfig{
			LocalName: a.Addr().ImpliedProvider(),
		}
	}

	return addrs.LocalProviderConfig{
		LocalName: a.ProviderConfigRef.Name,
		Alias:     a.ProviderConfigRef.Alias,
	}
}
