package terraform

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/config"
)

// This is the directory where our test fixtures are.
const fixtureDir = "./test-fixtures"

func TestNew(t *testing.T) {
	configVal := testConfig(t, "new-good")
	tfConfig := &Config{
		Config: configVal,
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

	if len(tf.mapping) != 2 {
		t.Fatalf("bad: %#v", tf.mapping)
	}
	if testProviderName(t, tf, "aws_instance.foo") != "aws" {
		t.Fatalf("bad: %#v", tf.mapping)
	}
	if testProviderName(t, tf, "do_droplet.bar") != "do" {
		t.Fatalf("bad: %#v", tf.mapping)
	}

	var pc *config.ProviderConfig

	pc = testProviderConfig(tf, "do_droplet.bar")
	if pc != nil {
		t.Fatalf("bad: %#v", pc)
	}

	pc = testProviderConfig(tf, "aws_instance.foo")
	if pc.Config["foo"].(string) != "bar" {
		t.Fatalf("bad: %#v", pc)
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

func TestNew_providerConfigCache(t *testing.T) {
	configVal := testConfig(t, "new-pc-cache")
	tfConfig := &Config{
		Config: configVal,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFunc(
				"aws", []string{"aws_elb", "aws_instance"}),
			"do": testProviderFunc("do", []string{"do_droplet"}),
		},
	}

	tf, err := New(tfConfig)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if tf == nil {
		t.Fatal("tf should not be nil")
	}

	if testProviderName(t, tf, "aws_instance.foo") != "aws" {
		t.Fatalf("bad: %#v", tf.mapping)
	}
	if testProviderName(t, tf, "aws_elb.lb") != "aws" {
		t.Fatalf("bad: %#v", tf.mapping)
	}
	if testProviderName(t, tf, "do_droplet.bar") != "do" {
		t.Fatalf("bad: %#v", tf.mapping)
	}

	if testProvider(tf, "aws_instance.foo") !=
		testProvider(tf, "aws_instance.bar") {
		t.Fatalf("bad equality")
	}
	if testProvider(tf, "aws_instance.foo") ==
		testProvider(tf, "aws_elb.lb") {
		t.Fatal("should not be equal")
	}

	var pc *config.ProviderConfig
	pc = testProviderConfig(tf, "do_droplet.bar")
	if pc != nil {
		t.Fatalf("bad: %#v", pc)
	}
	pc = testProviderConfig(tf, "aws_instance.foo")
	if pc.Config["foo"].(string) != "bar" {
		t.Fatalf("bad: %#v", pc)
	}
	pc = testProviderConfig(tf, "aws_elb.lb")
	if pc.Config["foo"].(string) != "baz" {
		t.Fatalf("bad: %#v", pc)
	}

	if testProviderConfig(tf, "aws_instance.foo") !=
		testProviderConfig(tf, "aws_instance.bar") {
		t.Fatal("should be same")
	}
	if testProviderConfig(tf, "aws_instance.foo") ==
		testProviderConfig(tf, "aws_elb.lb") {
		t.Fatal("should be different")
	}

	// Finally, verify some internals here that we're using the
	// IDENTICAL *terraformProvider pointer for matching types
	if testTerraformProvider(tf, "aws_instance.foo") !=
		testTerraformProvider(tf, "aws_instance.bar") {
		t.Fatal("should be same")
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

	actual := strings.TrimSpace(diff.String())
	expected := strings.TrimSpace(testTerraformDiffStr)
	if actual != expected {
		t.Fatalf("bad:\n%s", actual)
	}
}

func TestTerraformDiff_nil(t *testing.T) {
	tf := testTerraform(t, "diff-nil")

	diff, err := tf.Diff(nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if len(diff.Resources) != 0 {
		t.Fatalf("bad: %#v", diff.Resources)
	}
}

func TestTerraformDiff_computed(t *testing.T) {
	tf := testTerraform(t, "diff-computed")

	diff, err := tf.Diff(nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if len(diff.Resources) < 2 {
		t.Fatalf("bad: %#v", diff.Resources)
	}

	actual := strings.TrimSpace(diff.String())
	expected := strings.TrimSpace(testTerraformDiffComputedStr)
	if actual != expected {
		t.Fatalf("bad:\n%s", actual)
	}
}

func TestTerraformDiff_providerInit(t *testing.T) {
	tf := testTerraform(t, "diff-provider-init")

	_, err := tf.Diff(nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	p := testProviderMock(testProvider(tf, "do_droplet.bar"))
	if p == nil {
		t.Fatal("should have provider")
	}
	if !p.ConfigureCalled {
		t.Fatal("configure should be called")
	}
	if p.ConfigureConfig["foo"].(string) != "2" {
		t.Fatalf("bad: %#v", p.ConfigureConfig)
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
		diffFn := func(
			_ *ResourceState,
			c map[string]interface{}) (*ResourceDiff, error) {
			var diff ResourceDiff
			diff.Attributes = make(map[string]*ResourceAttrDiff)
			for k, v := range c {
				if _, ok := v.(string); !ok {
					continue
				}

				if k == "nil" {
					return nil, nil
				}

				if k == "compute" {
					diff.Attributes[v.(string)] = &ResourceAttrDiff{
						Old:         "",
						New:         "",
						NewComputed: true,
					}
					continue
				}

				attrDiff := &ResourceAttrDiff{
					Old: "",
					New: v.(string),
				}

				if strings.Contains(attrDiff.New, ComputedPlaceholder) {
					attrDiff.NewComputed = true
				}

				diff.Attributes[k] = attrDiff
			}

			return &diff, nil
		}

		result := &MockResourceProvider{
			Meta:            n,
			DiffFn:          diffFn,
			ResourcesReturn: resources,
		}

		return result, nil
	}
}

func testProvider(tf *Terraform, n string) ResourceProvider {
	for r, tp := range tf.mapping {
		if r.Id() == n {
			return tp.Provider
		}
	}

	return nil
}

func testProviderMock(p ResourceProvider) *MockResourceProvider {
	return p.(*MockResourceProvider)
}

func testProviderConfig(tf *Terraform, n string) *config.ProviderConfig {
	for r, tp := range tf.mapping {
		if r.Id() == n {
			return tp.Config
		}
	}

	return nil
}

func testProviderName(t *testing.T, tf *Terraform, n string) string {
	var p ResourceProvider
	for r, tp := range tf.mapping {
		if r.Id() == n {
			p = tp.Provider
			break
		}
	}

	if p == nil {
		t.Fatalf("resource not found: %s", n)
	}

	return testProviderMock(p).Meta.(string)
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

func testTerraformProvider(tf *Terraform, n string) *terraformProvider {
	for r, tp := range tf.mapping {
		if r.Id() == n {
			return tp
		}
	}

	return nil
}

const testTerraformDiffStr = `
UPDATE: aws_instance.bar
  foo: "" => "2"
UPDATE: aws_instance.foo
  num: "" => "2"
`

const testTerraformDiffComputedStr = `
UPDATE: aws_instance.bar
  foo: "" => "<computed>"
UPDATE: aws_instance.foo
  id:  "" => "<computed>"
  num: "" => "2"
`
