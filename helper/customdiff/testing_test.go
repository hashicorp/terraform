package customdiff

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

func testProvider(s map[string]*schema.Schema, cd schema.CustomizeDiffFunc) terraform.ResourceProvider {
	return &schema.Provider{
		ResourcesMap: map[string]*schema.Resource{
			"test": {
				Schema:        s,
				CustomizeDiff: cd,
			},
		},
	}
}

func testDiff(provider terraform.ResourceProvider, old, new map[string]string) (*terraform.InstanceDiff, error) {
	newI := make(map[string]interface{}, len(new))
	for k, v := range new {
		newI[k] = v
	}

	return provider.Diff(
		&terraform.InstanceInfo{
			Id:         "test",
			Type:       "test",
			ModulePath: []string{},
		},
		&terraform.InstanceState{
			Attributes: old,
		},
		&terraform.ResourceConfig{
			Config: newI,
		},
	)
}
