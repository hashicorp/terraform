package terraform

import (
	"fmt"
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
	if pc.RawConfig.Raw["foo"].(string) != "bar" {
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
	if pc.RawConfig.Raw["foo"].(string) != "bar" {
		t.Fatalf("bad: %#v", pc)
	}
	pc = testProviderConfig(tf, "aws_elb.lb")
	if pc.RawConfig.Raw["foo"].(string) != "baz" {
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

func TestNew_providerValidate(t *testing.T) {
	config := testConfig(t, "new-provider-validate")
	tfConfig := &Config{
		Config: config,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFunc("aws", []string{"aws_instance"}),
		},
	}

	tf, err := New(tfConfig)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	p := testProviderMock(testProvider(tf, "aws_instance.foo"))
	if !p.ValidateCalled {
		t.Fatal("validate should be called")
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

func TestTerraformApply(t *testing.T) {
	tf := testTerraform(t, "apply-good")

	s := &State{}
	p, err := tf.Plan(s)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := tf.Apply(p)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if len(state.Resources) < 2 {
		t.Fatalf("bad: %#v", state.Resources)
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyStr)
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}
}

func TestTerraformApply_compute(t *testing.T) {
	// This tests that computed variables are properly re-diffed
	// to get the value prior to application (Apply).
	tf := testTerraform(t, "apply-compute")

	s := &State{}
	p, err := tf.Plan(s)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	// Set meta to change behavior so that computed variables are filled
	testProviderMock(testProvider(tf, "aws_instance.foo")).Meta =
		"compute"

	state, err := tf.Apply(p)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyComputeStr)
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}
}

func TestTerraformPlan(t *testing.T) {
	tf := testTerraform(t, "plan-good")

	plan, err := tf.Plan(nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if len(plan.Diff.Resources) < 2 {
		t.Fatalf("bad: %#v", plan.Diff.Resources)
	}

	actual := strings.TrimSpace(plan.String())
	expected := strings.TrimSpace(testTerraformPlanStr)
	if actual != expected {
		t.Fatalf("bad:\n%s", actual)
	}

	p := testProviderMock(testProvider(tf, "aws_instance.foo"))
	if !p.RefreshCalled {
		t.Fatal("refresh should be called")
	}
	if p.RefreshState == nil {
		t.Fatal("refresh should have state")
	}
}

func TestTerraformPlan_nil(t *testing.T) {
	tf := testTerraform(t, "plan-nil")

	plan, err := tf.Plan(nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if len(plan.Diff.Resources) != 0 {
		t.Fatalf("bad: %#v", plan.Diff.Resources)
	}
}

func TestTerraformPlan_computed(t *testing.T) {
	tf := testTerraform(t, "plan-computed")

	plan, err := tf.Plan(nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if len(plan.Diff.Resources) < 2 {
		t.Fatalf("bad: %#v", plan.Diff.Resources)
	}

	actual := strings.TrimSpace(plan.String())
	expected := strings.TrimSpace(testTerraformPlanComputedStr)
	if actual != expected {
		t.Fatalf("bad:\n%s", actual)
	}
}

func TestTerraformPlan_providerInit(t *testing.T) {
	tf := testTerraform(t, "plan-provider-init")

	_, err := tf.Plan(nil)
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
	if p.ConfigureConfig.Raw["foo"].(string) != "2" {
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
		p := &MockResourceProvider{Meta: n}

		applyFn := func(
			s *ResourceState,
			d *ResourceDiff) (*ResourceState, error) {
			result := &ResourceState{
				ID:         "foo",
				Attributes: make(map[string]string),
			}

			if d != nil {
				for ak, ad := range d.Attributes {
					result.Attributes[ak] = ad.New
				}
			}

			return result, nil
		}

		diffFn := func(
			s *ResourceState,
			c *ResourceConfig) (*ResourceDiff, error) {
			var diff ResourceDiff
			diff.Attributes = make(map[string]*ResourceAttrDiff)
			diff.Attributes["type"] = &ResourceAttrDiff{
				Old: "",
				New: s.Type,
			}

			for k, v := range c.Raw {
				if _, ok := v.(string); !ok {
					continue
				}

				if k == "nil" {
					return nil, nil
				}

				if k == "compute" {
					attrDiff := &ResourceAttrDiff{
						Old:         "",
						New:         "",
						NewComputed: true,
					}

					// If the value of Meta turns into "compute", then we
					// fill the computed values.
					if mv, ok := p.Meta.(string); ok && mv == "compute" {
						attrDiff.NewComputed = false
						attrDiff.New = fmt.Sprintf("computed_%s", v.(string))
					}

					diff.Attributes[v.(string)] = attrDiff
					continue
				}

				// If this key is not computed, then look it up in the
				// cleaned config.
				found := false
				for _, ck := range c.ComputedKeys {
					if ck == k {
						found = true
						break
					}
				}
				if !found {
					v = c.Config[k]
				}

				attrDiff := &ResourceAttrDiff{
					Old: "",
					New: v.(string),
				}

				if strings.Contains(attrDiff.New, config.UnknownVariableValue) {
					attrDiff.NewComputed = true
				}

				diff.Attributes[k] = attrDiff
			}

			for _, k := range c.ComputedKeys {
				diff.Attributes[k] = &ResourceAttrDiff{
					Old:         "",
					NewComputed: true,
				}
			}

			return &diff, nil
		}

		refreshFn := func(s *ResourceState) (*ResourceState, error) {
			return s, nil
		}

		p.ApplyFn = applyFn
		p.DiffFn = diffFn
		p.RefreshFn = refreshFn
		p.ResourcesReturn = resources

		return p, nil
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

const testTerraformApplyStr = `
aws_instance.bar:
  ID = foo
  type = aws_instance
  foo = bar
aws_instance.foo:
  ID = foo
  type = aws_instance
  num = 2
`

const testTerraformApplyComputeStr = `
aws_instance.bar:
  ID = foo
  type = aws_instance
  foo = computed_id
aws_instance.foo:
  ID = foo
  type = aws_instance
  num = 2
  id = computed_id
`

const testTerraformPlanStr = `
UPDATE: aws_instance.bar
  foo:  "" => "2"
  type: "" => "aws_instance"
UPDATE: aws_instance.foo
  num:  "" => "2"
  type: "" => "aws_instance"
`

const testTerraformPlanComputedStr = `
UPDATE: aws_instance.bar
  foo:  "" => "<computed>"
  type: "" => "aws_instance"
UPDATE: aws_instance.foo
  id:   "" => "<computed>"
  num:  "" => "2"
  type: "" => "aws_instance"
`
