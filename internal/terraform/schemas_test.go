package terraform

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
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
