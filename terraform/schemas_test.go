package terraform

import (
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/internal/addrs"
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
