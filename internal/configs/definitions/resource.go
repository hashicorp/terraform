// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package definitions

import (
	"github.com/hashicorp/hcl/v2"

	"github.com/hashicorp/terraform/internal/addrs"
)

// Resource represents a "resource" or "data" block in a module or file.
type Resource struct {
	Mode    addrs.ResourceMode
	Name    string
	Type    string
	Config  hcl.Body
	Count   hcl.Expression
	ForEach hcl.Expression

	ProviderConfigRef *ProviderConfigRef
	Provider          addrs.Provider

	Preconditions  []*CheckRule
	Postconditions []*CheckRule

	DependsOn []hcl.Traversal

	TriggersReplacement []hcl.Expression

	// Managed is populated only for Mode = addrs.ManagedResourceMode,
	// containing the additional fields that apply to managed resources.
	// For all other resource modes, this field is nil.
	Managed *ManagedResource

	// List is populated only for Mode = addrs.ListResourceMode,
	// containing the additional fields that apply to list resources.
	List *ListResource

	// Container links a scoped resource back up to the resources that contains
	// it. This field is referenced during static analysis to check whether any
	// references are also made from within the same container.
	//
	// If this is nil, then this resource is essentially public.
	Container Container

	DeclRange hcl.Range
	TypeRange hcl.Range
}

// ModuleUniqueKey returns a unique key for this resource within a module.
func (r *Resource) ModuleUniqueKey() string {
	return r.Addr().String()
}

// Addr returns a resource address for the receiver that is relative to the
// resource's containing module.
func (r *Resource) Addr() addrs.Resource {
	return addrs.Resource{
		Mode: r.Mode,
		Type: r.Type,
		Name: r.Name,
	}
}

// ProviderConfigAddr returns the address for the provider configuration that
// should be used for this resource. This function returns a default provider
// config addr if an explicit "provider" argument was not provided.
func (r *Resource) ProviderConfigAddr() addrs.LocalProviderConfig {
	if r.ProviderConfigRef == nil {
		// If no specific "provider" argument is given, we want to look up the
		// provider config where the local name matches the implied provider
		// from the resource type. This may be different from the resource's
		// provider type.
		return addrs.LocalProviderConfig{
			LocalName: r.Addr().ImpliedProvider(),
		}
	}

	return addrs.LocalProviderConfig{
		LocalName: r.ProviderConfigRef.Name,
		Alias:     r.ProviderConfigRef.Alias,
	}
}

// HasCustomConditions returns true if and only if the resource has at least
// one author-specified custom condition.
func (r *Resource) HasCustomConditions() bool {
	return len(r.Postconditions) != 0 || len(r.Preconditions) != 0
}
