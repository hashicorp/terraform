package file

import (
	"github.com/hashicorp/terraform/terraform"
)

type ResourceProvisioner struct{}

func (p *ResourceProvisioner) Apply(s *terraform.ResourceState,
	c *terraform.ResourceConfig) (*terraform.ResourceState, error) {
	panic("not implemented")
	return s, nil
}

func (p *ResourceProvisioner) Validate(c *terraform.ResourceConfig) (ws []string, es []error) {
	return
}
