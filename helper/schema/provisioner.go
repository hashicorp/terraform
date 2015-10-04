package schema

import (
	"github.com/hashicorp/terraform/terraform"
)

type Provisioner struct {
	Schema map[string]*Schema
}

func (p *Provisioner) Export() (terraform.ResourceProvisionerSchema, error) {
	schema := schemaMap(p.Schema).Export()
	result := terraform.ResourceProvisionerSchema{}
	result.Schema = schema
	return result, nil
}
