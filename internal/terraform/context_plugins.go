package terraform

import (
	"fmt"
	"log"
	"sync"

	"github.com/hashicorp/hcl/v2/hclsyntax"

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

	// We memoize the schemas we've previously loaded in here, to avoid
	// repeatedly paying the cost of activating the same plugins to access
	// their schemas in various different spots. We use schemas for many
	// purposes in Terraform, so there isn't a single choke point where
	// it makes sense to preload all of them.
	providerSchemas    map[addrs.Provider]*ProviderSchema
	provisionerSchemas map[string]*configschema.Block
	schemasLock        sync.Mutex
}

func newContextPlugins(providerFactories map[addrs.Provider]providers.Factory, provisionerFactories map[string]provisioners.Factory) *contextPlugins {
	ret := &contextPlugins{
		providerFactories:    providerFactories,
		provisionerFactories: provisionerFactories,
	}
	ret.init()
	return ret
}

func (cp *contextPlugins) init() {
	cp.providerSchemas = make(map[addrs.Provider]*ProviderSchema, len(cp.providerFactories))
	cp.provisionerSchemas = make(map[string]*configschema.Block, len(cp.provisionerFactories))
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
func (cp *contextPlugins) ProviderSchema(addr addrs.Provider) (*ProviderSchema, error) {
	cp.schemasLock.Lock()
	defer cp.schemasLock.Unlock()

	if schema, ok := cp.providerSchemas[addr]; ok {
		return schema, nil
	}

	log.Printf("[TRACE] terraform.contextPlugins: Initializing provider %q to read its schema", addr)

	provider, err := cp.NewProviderInstance(addr)
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate provider %q to obtain schema: %s", addr, err)
	}
	defer provider.Close()

	resp := provider.GetProviderSchema()
	if resp.Diagnostics.HasErrors() {
		return nil, fmt.Errorf("failed to retrieve schema from provider %q: %s", addr, resp.Diagnostics.Err())
	}

	s := &ProviderSchema{
		Provider:      resp.Provider.Block,
		ResourceTypes: make(map[string]*configschema.Block),
		DataSources:   make(map[string]*configschema.Block),
		Functions:     make(map[string]providers.FunctionDecl),

		ResourceTypeSchemaVersions: make(map[string]uint64),
	}

	if resp.Provider.Version < 0 {
		// We're not using the version numbers here yet, but we'll check
		// for validity anyway in case we start using them in future.
		return nil, fmt.Errorf("provider %s has invalid negative schema version for its configuration blocks,which is a bug in the provider ", addr)
	}

	for t, r := range resp.ResourceTypes {
		if err := r.Block.InternalValidate(); err != nil {
			return nil, fmt.Errorf("provider %s has invalid schema for managed resource type %q, which is a bug in the provider: %q", addr, t, err)
		}
		s.ResourceTypes[t] = r.Block
		s.ResourceTypeSchemaVersions[t] = uint64(r.Version)
		if r.Version < 0 {
			return nil, fmt.Errorf("provider %s has invalid negative schema version for managed resource type %q, which is a bug in the provider", addr, t)
		}
	}

	for t, d := range resp.DataSources {
		if err := d.Block.InternalValidate(); err != nil {
			return nil, fmt.Errorf("provider %s has invalid schema for data resource type %q, which is a bug in the provider: %q", addr, t, err)
		}
		s.DataSources[t] = d.Block
		if d.Version < 0 {
			// We're not using the version numbers here yet, but we'll check
			// for validity anyway in case we start using them in future.
			return nil, fmt.Errorf("provider %s has invalid negative schema version for data resource type %q, which is a bug in the provider", addr, t)
		}
	}

	for n, f := range resp.Functions {
		if !hclsyntax.ValidIdentifier(n) {
			return nil, fmt.Errorf("provider %s declares function with invalid name %q", addr, n)
		}
		// We'll also do some enforcement of parameter names, even though they
		// are only for docs/UI for now, to leave room for us to potentially
		// use them for other purposes later.
		seenParams := make(map[string]int, len(f.Parameters))
		for i, p := range f.Parameters {
			if !hclsyntax.ValidIdentifier(p.Name) {
				return nil, fmt.Errorf("provider %s function %q declares invalid name %q for parameter %d", addr, n, p.Name, i)
			}
			if prevIdx, exists := seenParams[p.Name]; exists {
				return nil, fmt.Errorf("provider %s function %q reuses name %q for both parameters %d and %d", addr, n, p.Name, prevIdx, i)
			}
			seenParams[p.Name] = i
		}
		if p := f.VariadicParameter; p != nil {
			if !hclsyntax.ValidIdentifier(p.Name) {
				return nil, fmt.Errorf("provider %s function %q declares invalid name %q for its variadic parameter", addr, n, p.Name)
			}
			if prevIdx, exists := seenParams[p.Name]; exists {
				return nil, fmt.Errorf("provider %s function %q reuses name %q for both parameter %d and its variadic parameter", addr, n, p.Name, prevIdx)
			}
		}
		s.Functions[n] = f
	}

	if resp.ProviderMeta.Block != nil {
		s.ProviderMeta = resp.ProviderMeta.Block
	}

	cp.providerSchemas[addr] = s
	return s, nil
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

	return providerSchema.Provider, nil
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

// ProviderFunctionDecls is a helper wrapper around ProviderSchema which first
// reads the schema of the given provider and then returns all of the
// functions it declares, if any.
//
// ProviderFunctionDecl will return an error if the provider schema lookup
// fails, but will return an empty set of functions if a successful response
// returns no functions, or if the provider is using an older protocol version
// which has no support for provider-contributed functions.
func (cp *contextPlugins) ProviderFunctionDecls(providerAddr addrs.Provider) (map[string]providers.FunctionDecl, error) {
	providerSchema, err := cp.ProviderSchema(providerAddr)
	if err != nil {
		return nil, err
	}

	return providerSchema.Functions, nil
}

// ProvisionerSchema uses a temporary instance of the provisioner with the
// given type name to obtain the schema for that provisioner's configuration.
//
// ProvisionerSchema memoizes results by provisioner type name, so it's fine
// to repeatedly call this method with the same name if various different
// parts of Terraform all need the same schema information.
func (cp *contextPlugins) ProvisionerSchema(typ string) (*configschema.Block, error) {
	cp.schemasLock.Lock()
	defer cp.schemasLock.Unlock()

	if schema, ok := cp.provisionerSchemas[typ]; ok {
		return schema, nil
	}

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

	cp.provisionerSchemas[typ] = resp.Provisioner
	return resp.Provisioner, nil
}
