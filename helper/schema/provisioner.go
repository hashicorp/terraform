package schema

import (
	"github.com/hashicorp/terraform/terraform"
)

type Provisioner struct {
	Schema map[string]*Schema
}

func (p *Provisioner) Export() (terraform.ResourceSchemaInfo, error) {
	return schemaMap(p.Schema).Export(), nil
}
