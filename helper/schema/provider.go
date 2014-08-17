package schema

import (
	"fmt"

	"github.com/hashicorp/terraform/terraform"
)

// Provider represents a Resource provider in Terraform, and properly
// implements all of the ResourceProvider API.
//
// This is a friendlier API than the core Terraform ResourceProvider API,
// and is recommended to be used over that.
type Provider struct {
	Schema    map[string]*Schema
	Resources map[string]*Resource

	Configure ConfigureFunc
}

// ConfigureFunc is the function used to configure a Provider.
type ConfigureFunc func(*ResourceData) error

// Validate validates the provider configuration against the schema.
func (p *Provider) Validate(c *terraform.ResourceConfig) ([]string, []error) {
	return schemaMap(p.Schema).Validate(c)
}

// ValidateResource validates the resource configuration against the
// proper schema.
func (p *Provider) ValidateResource(
	t string, c *terraform.ResourceConfig) ([]string, []error) {
	r, ok := p.Resources[t]
	if !ok {
		return nil, []error{fmt.Errorf(
			"Provider doesn't support resource: %s", t)}
	}

	return r.Validate(c)
}
