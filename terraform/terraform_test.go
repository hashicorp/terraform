package terraform

import (
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform/config"
)

// This is the directory where our test fixtures are.
const fixtureDir = "./test-fixtures"

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

func TestNew_graphCycle(t *testing.T) {
	config := testConfig(t, "new-graph-cycle")
	tfConfig := &Config{
		Config: config,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFunc("aws", []string{"aws_instance"}),
		},
	}

	tf, err := New(tfConfig)
	if err == nil {
		t.Fatal("should error")
	}
	if tf != nil {
		t.Fatalf("should not return tf")
	}
}

func TestNew_variables(t *testing.T) {
	config := testConfig(t, "new-variables")
	tfConfig := &Config{
		Config: config,
	}

	// Missing
	tfConfig.Variables = map[string]string{
		"bar": "baz",
	}
	tf, err := New(tfConfig)
	if err == nil {
		t.Fatal("should error")
	}
	if tf != nil {
		t.Fatalf("should not return tf")
	}

	// Good
	tfConfig.Variables = map[string]string{
		"foo": "bar",
	}
	tf, err = New(tfConfig)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if tf == nil {
		t.Fatal("tf should not be nil")
	}

	// Good
	tfConfig.Variables = map[string]string{
		"foo": "bar",
		"bar": "baz",
	}
	tf, err = New(tfConfig)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if tf == nil {
		t.Fatal("tf should not be nil")
	}
}

func TestTerraformDiff(t *testing.T) {
	tf := testTerraform(t, "diff-good")

	diff, err := tf.Diff(nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if len(diff.Resources) < 2 {
		t.Fatalf("bad: %#v", diff.Resources)
	}
}

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
		var diff ResourceDiff
		diff.Attributes = map[string]*ResourceAttrDiff{
			n: &ResourceAttrDiff{
				Old: "foo",
				New: "bar",
			},
		}

		result := &MockResourceProvider{
			Meta:            n,
			DiffReturn:      diff,
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

func testTerraform(t *testing.T, name string) *Terraform {
	config := testConfig(t, name)
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

	return tf
}
