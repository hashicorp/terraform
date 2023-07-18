// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/provisioners"
)

// contextPlugins represents a library of available plugins (providers and
// provisioners) which we assume will all be used with the same
// terraform.Context, and thus it'll be safe to cache certain information
// about the providers for performance reasons.
type contextPlugins struct {
	providerFactories    map[addrs.Provider]providers.Factory
	provisionerFactories map[string]provisioners.Factory
}

func newContextPlugins(providerFactories map[addrs.Provider]providers.Factory, provisionerFactories map[string]provisioners.Factory) *contextPlugins {
	ret := &contextPlugins{
		providerFactories:    providerFactories,
		provisionerFactories: provisionerFactories,
	}
	return ret
}

func (cp *contextPlugins) HasProvider(addr addrs.Provider) bool {
	_, ok := cp.providerFactories[addr]
	return ok
}

func (cp *contextPlugins) NewProviderInstance(addr addrs.Provider) (providers.Interface, error) {
	f, ok := cp.providerFactories[addr]
	if !ok {
		return nil, fmt.Errorf("unavailable provider %q", addr.String())
	}

	return f()

}

func (cp *contextPlugins) HasProvisioner(typ string) bool {
	_, ok := cp.provisionerFactories[typ]
	return ok
}

func (cp *contextPlugins) NewProvisionerInstance(typ string) (provisioners.Interface, error) {
	f, ok := cp.provisionerFactories[typ]
	if !ok {
		return nil, fmt.Errorf("unavailable provisioner %q", typ)
	}

	return f()
}

// ProviderSchema uses a temporary instance of the provider with the given
// address to obtain the full schema for all aspects of that provider.
//
// ProviderSchema memoizes results by unique provider address, so it's fine
// to repeatedly call this method with the same address if various different
// parts of Terraform all need the same schema information.
func (cp *contextPlugins) ProviderSchema(addr addrs.Provider) (providers.ProviderSchema, error) {
	log.Printf("[TRACE] terraform.contextPlugins: Initializing provider %q to read its schema", addr)

	// Check the global schema cache first.
	// This cache is only written by the provider client, and transparently
	// used by GetProviderSchema, but we check it here because at this point we
	// may be able to avoid spinning up the provider instance at all.
	schemas, ok := providers.SchemaCache.Get(addr)
	if ok {
		return schemas, nil
	}

	provider, err := cp.NewProviderInstance(addr)
	if err != nil {
		return schemas, fmt.Errorf("failed to instantiate provider %q to obtain schema: %s", addr, err)
	}
	defer provider.Close()

	resp := provider.GetProviderSchema()
	if resp.Diagnostics.HasErrors() {
		return resp, fmt.Errorf("failed to retrieve schema from provider %q: %s", addr, resp.Diagnostics.Err())
	}

	if resp.Provider.Version < 0 {
		// We're not using the version numbers here yet, but we'll check
		// for validity anyway in case we start using them in future.
		return resp, fmt.Errorf("provider %s has invalid negative schema version for its configuration blocks,which is a bug in the provider ", addr)
	}

	for t, r := range resp.ResourceTypes {
		if err := r.Block.InternalValidate(); err != nil {
			return resp, fmt.Errorf("provider %s has invalid schema for managed resource type %q, which is a bug in the provider: %q", addr, t, err)
		}
		if r.Version < 0 {
			return resp, fmt.Errorf("provider %s has invalid negative schema version for managed resource type %q, which is a bug in the provider", addr, t)
		}
	}

	for t, d := range resp.DataSources {
		if err := d.Block.InternalValidate(); err != nil {
			return resp, fmt.Errorf("provider %s has invalid schema for data resource type %q, which is a bug in the provider: %q", addr, t, err)
		}
		if d.Version < 0 {
			// We're not using the version numbers here yet, but we'll check
			// for validity anyway in case we start using them in future.
			return resp, fmt.Errorf("provider %s has invalid negative schema version for data resource type %q, which is a bug in the provider", addr, t)
		}
	}

	return resp, nil
}

// ProviderConfigSchema is a helper wrapper around ProviderSchema which first
// reads the full schema of the given provider and then extracts just the
// provider's configuration schema, which defines what's expected in a
// "provider" block in the configuration when configuring this provider.
func (cp *contextPlugins) ProviderConfigSchema(providerAddr addrs.Provider) (*configschema.Block, error) {
	providerSchema, err := cp.ProviderSchema(providerAddr)
	if err != nil {
		return nil, err
	}

	return providerSchema.Provider.Block, nil
}

// ResourceTypeSchema is a helper wrapper around ProviderSchema which first
// reads the schema of the given provider and then tries to find the schema
// for the resource type of the given resource mode in that provider.
//
// ResourceTypeSchema will return an error if the provider schema lookup
// fails, but will return nil if the provider schema lookup succeeds but then
// the provider doesn't have a resource of the requested type.
//
// Managed resource types have versioned schemas, so the second return value
// is the current schema version number for the requested resource. The version
// is irrelevant for other resource modes.
func (cp *contextPlugins) ResourceTypeSchema(providerAddr addrs.Provider, resourceMode addrs.ResourceMode, resourceType string) (*configschema.Block, uint64, error) {
	providerSchema, err := cp.ProviderSchema(providerAddr)
	if err != nil {
		return nil, 0, err
	}

	schema, version := providerSchema.SchemaForResourceType(resourceMode, resourceType)
	return schema, version, nil
}

// ProvisionerSchema uses a temporary instance of the provisioner with the
// given type name to obtain the schema for that provisioner's configuration.
//
// ProvisionerSchema memoizes results by provisioner type name, so it's fine
// to repeatedly call this method with the same name if various different
// parts of Terraform all need the same schema information.
func (cp *contextPlugins) ProvisionerSchema(typ string) (*configschema.Block, error) {
	log.Printf("[TRACE] terraform.contextPlugins: Initializing provisioner %q to read its schema", typ)
	provisioner, err := cp.NewProvisionerInstance(typ)
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate provisioner %q to obtain schema: %s", typ, err)
	}
	defer provisioner.Close()

	resp := provisioner.GetSchema()
	if resp.Diagnostics.HasErrors() {
		return nil, fmt.Errorf("failed to retrieve schema from provisioner %q: %s", typ, resp.Diagnostics.Err())
	}

	return resp.Provisioner, nil
}
