package aws

import (
	"github.com/hashicorp/terraform/terraform"
)

type ResourceProvider struct {
}

func (p *ResourceProvider) Configure(map[string]interface{}) ([]string, error) {
	return nil, nil
}

func (p *ResourceProvider) Resources() []terraform.ResourceType {
	return nil
}
