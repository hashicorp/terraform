package terraform

import (
	"github.com/hashicorp/terraform/configs/configschema"
)

func simpleTestSchemas() *Schemas {
	provider := simpleMockProvider()
	provisioner := simpleMockProvisioner()
	return &Schemas{
		Providers: map[string]*ProviderSchema{
			"test": provider.GetSchemaReturn,
		},
		Provisioners: map[string]*configschema.Block{
			"test": provisioner.GetSchemaResponse.Provisioner,
		},
	}
}
