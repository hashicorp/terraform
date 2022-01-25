package terraform

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/providers"
)

func simpleTestSchemas() *Schemas {
	provider := simpleMockProvider()
	provisioner := simpleMockProvisioner()

	return &Schemas{
		Providers: map[addrs.Provider]*ProviderSchema{
			addrs.NewDefaultProvider("test"): provider.ProviderSchema(),
		},
		Provisioners: map[string]*configschema.Block{
			"test": provisioner.GetSchemaResponse.Provisioner,
		},
	}
}

// schemaOnlyProvidersForTesting is a testing helper that constructs a
// plugin library that contains a set of providers that only know how to
// return schema, and will exhibit undefined behavior if used for any other
// purpose.
//
// The intended use for this is in testing components that use schemas to
// drive other behavior, such as reference analysis during graph construction,
// but that don't actually need to interact with providers otherwise.
func schemaOnlyProvidersForTesting(schemas map[addrs.Provider]*ProviderSchema) *contextPlugins {
	factories := make(map[addrs.Provider]providers.Factory, len(schemas))

	for providerAddr, schema := range schemas {

		resp := &providers.GetProviderSchemaResponse{
			Provider: providers.Schema{
				Block: schema.Provider,
			},
			ResourceTypes: make(map[string]providers.Schema),
			DataSources:   make(map[string]providers.Schema),
		}
		for t, tSchema := range schema.ResourceTypes {
			resp.ResourceTypes[t] = providers.Schema{
				Block:   tSchema,
				Version: int64(schema.ResourceTypeSchemaVersions[t]),
			}
		}
		for t, tSchema := range schema.DataSources {
			resp.DataSources[t] = providers.Schema{
				Block: tSchema,
			}
		}

		provider := &MockProvider{
			GetProviderSchemaResponse: resp,
		}

		factories[providerAddr] = func() (providers.Interface, error) {
			return provider, nil
		}
	}

	return newContextPlugins(factories, nil)
}
