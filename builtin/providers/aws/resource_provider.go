package aws

import (
	"github.com/hashicorp/terraform/terraform"
)

type ResourceProvider struct {
}

func (p *ResourceProvider) Validate(c *terraform.ResourceConfig) ([]string, []error) {
	errs := c.CheckSet([]string{
		"access_key",
		"secret_key",
	})

	return nil, errs
}

func (p *ResourceProvider) Configure(*terraform.ResourceConfig) error {
	return nil
}

func (p *ResourceProvider) Diff(
	s *terraform.ResourceState,
	c *terraform.ResourceConfig) (*terraform.ResourceDiff, error) {
	return nil, nil
}

func (p *ResourceProvider) Resources() []terraform.ResourceType {
	return []terraform.ResourceType{
		terraform.ResourceType{
			Name: "aws_instance",
		},
	}
}
