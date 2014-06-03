package terraform

import (
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform/config"
)

// This is the directory where our test fixtures are.
const fixtureDir = "./test-fixtures"

func testConfig(t *testing.T, name string) *config.Config {
	c, err := config.Load(filepath.Join(fixtureDir, name, "main.tf"))
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	return c
}

func testProviderFunc(n string, rs []string) ResourceProviderFactory {
	resources := make([]ResourceType, len(rs))
	for i, v := range rs {
		resources[i] = ResourceType{
			Name: v,
		}
	}

	return func() (ResourceProvider, error) {
		result := &MockResourceProvider{
			Meta:            n,
			ResourcesReturn: resources,
		}

		return result, nil
	}
}

func testProviderName(p ResourceProvider) string {
	return p.(*MockResourceProvider).Meta.(string)
}

func testResourceMapping(tf *Terraform) map[string]ResourceProvider {
	result := make(map[string]ResourceProvider)
	for resource, provider := range tf.mapping {
		result[resource.Id()] = provider
	}

	return result
}

func TestNew(t *testing.T) {
	config := testConfig(t, "new-good")
	tfConfig := &Config{
		Config: config,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFunc("aws", []string{"aws_instance"}),
			"do":  testProviderFunc("do", []string{"do_droplet"}),
		},
	}

	tf, err := New(tfConfig)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if tf == nil {
		t.Fatal("tf should not be nil")
	}

	mapping := testResourceMapping(tf)
	if len(mapping) != 2 {
		t.Fatalf("bad: %#v", mapping)
	}
	if testProviderName(mapping["aws_instance.foo"]) != "aws" {
		t.Fatalf("bad: %#v", mapping)
	}
	if testProviderName(mapping["do_droplet.bar"]) != "do" {
		t.Fatalf("bad: %#v", mapping)
	}
}
