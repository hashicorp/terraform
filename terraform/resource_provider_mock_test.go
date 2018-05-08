package terraform

import (
	"testing"

	"github.com/hashicorp/terraform/config/configschema"
)

func TestMockResourceProvider_impl(t *testing.T) {
	var _ ResourceProvider = new(MockResourceProvider)
	var _ ResourceProviderCloser = new(MockResourceProvider)
}

// testProviderComponentFactory creates a componentFactory that contains only
// a single given.
func testProviderComponentFactory(name string, provider ResourceProvider) *basicComponentFactory {
	return &basicComponentFactory{
		providers: map[string]ResourceProviderFactory{
			name: ResourceProviderFactoryFixed(provider),
		},
	}
}

// mockProviderWithConfigSchema is a test helper to concisely create a mock
// provider with the given schema for its own configuration.
func mockProviderWithConfigSchema(schema *configschema.Block) *MockResourceProvider {
	return &MockResourceProvider{
		GetSchemaReturn: &ProviderSchema{
			Provider: schema,
		},
	}
}

// mockProviderWithResourceTypeSchema is a test helper to concisely create a mock
// provider with a schema containing a single resource type.
func mockProviderWithResourceTypeSchema(name string, schema *configschema.Block) *MockResourceProvider {
	return &MockResourceProvider{
		GetSchemaReturn: &ProviderSchema{
			ResourceTypes: map[string]*configschema.Block{
				name: schema,
			},
		},
	}
}

// mockProviderWithDataSourceSchema is a test helper to concisely create a mock
// provider with a schema containing a single data source.
func mockProviderWithDataSourceSchema(name string, schema *configschema.Block) *MockResourceProvider {
	return &MockResourceProvider{
		GetSchemaReturn: &ProviderSchema{
			DataSources: map[string]*configschema.Block{
				name: schema,
			},
		},
	}
}
