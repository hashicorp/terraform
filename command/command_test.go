package command

import (
	"path/filepath"

	"github.com/hashicorp/terraform/terraform"
)

// This is the directory where our test fixtures are.
const fixtureDir = "./test-fixtures"

func testFixturePath(name string) string {
	return filepath.Join(fixtureDir, name, "main.tf")
}

func testTFConfig(p terraform.ResourceProvider) *terraform.Config {
	return &terraform.Config{
		Providers: map[string]terraform.ResourceProviderFactory{
			"test": func() (terraform.ResourceProvider, error) {
				return p, nil
			},
		},
	}
}

func testProvider() *terraform.MockResourceProvider {
	p := new(terraform.MockResourceProvider)
	p.ResourcesReturn = []terraform.ResourceType{
		terraform.ResourceType{
			Name: "test_instance",
		},
	}

	return p
}
