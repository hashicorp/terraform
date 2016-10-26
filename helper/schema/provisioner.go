package schema

import (
	"github.com/hashicorp/terraform/terraform"
)

type Provisioner struct {
	Schema map[string]*Schema
}

func (p *Provisioner) Export() (*terraform.ResourceProvisionerSchema, error) {
	result := new(terraform.ResourceProvisionerSchema)
	result.Schema = schemaMap(p.Schema).Export()
	return result, nil
}
