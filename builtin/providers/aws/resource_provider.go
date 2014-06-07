package aws

import (
	"github.com/hashicorp/terraform/terraform"
)

type ResourceProvider struct {
}

func (p *ResourceProvider) Configure(map[string]interface{}) error {
	return nil
}

func (p *ResourceProvider) Diff(
	s *terraform.ResourceState,
	c map[string]interface{}) (*terraform.ResourceDiff, error) {
	return nil, nil
}

func (p *ResourceProvider) Resources() []terraform.ResourceType {
	return nil
}
