package terraform

import (
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/config"
)

// This is the directory where our test fixtures are.
const fixtureDir = "./test-fixtures"

/*
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

func TestTerraformApply_unknownAttribute(t *testing.T) {
	tf := testTerraform(t, "apply-unknown")

	s := &State{}
	p, err := tf.Plan(s)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := tf.Apply(p)
	if err == nil {
		t.Fatal("should error")
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyUnknownAttrStr)
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}
}

func TestTerraformApply_vars(t *testing.T) {
	tf := testTerraform(t, "apply-vars")
	//tf.variables = map[string]string{"foo": "baz"}

	s := &State{}
	p, err := tf.Plan(s)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	// Explicitly set the "foo" variable
	p.Vars["foo"] = "bar"

	state, err := tf.Apply(p)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyVarsStr)
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}
}
*/

func TestTerraformPlan(t *testing.T) {
	c := testConfig(t, "plan-good")
	tf := testTerraform2(t, nil)

	plan, err := tf.Plan(c, nil, nil)
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
}

func TestTerraformPlan_nil(t *testing.T) {
	c := testConfig(t, "plan-nil")
	tf := testTerraform2(t, nil)

	plan, err := tf.Plan(c, nil, nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if len(plan.Diff.Resources) != 0 {
		t.Fatalf("bad: %#v", plan.Diff.Resources)
	}
}

func TestTerraformPlan_computed(t *testing.T) {
	c := testConfig(t, "plan-computed")
	tf := testTerraform2(t, nil)

	plan, err := tf.Plan(c, nil, nil)
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

func TestTerraformRefresh(t *testing.T) {
	rpAWS := new(MockResourceProvider)
	rpAWS.ResourcesReturn = []ResourceType{
		ResourceType{Name: "aws_instance"},
	}

	c := testConfig(t, "refresh-basic")
	tf := testTerraform2(t, &Config{
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(rpAWS),
		},
	})

	rpAWS.RefreshReturn = &ResourceState{
		ID: "foo",
	}

	s, err := tf.Refresh(c, nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if !rpAWS.RefreshCalled {
		t.Fatal("refresh should be called")
	}
	if rpAWS.RefreshState != nil {
		t.Fatalf("bad: %#v", rpAWS.RefreshState)
	}
	if !reflect.DeepEqual(s.Resources["aws_instance.web"], rpAWS.RefreshReturn) {
		t.Fatalf("bad: %#v", s.Resources)
	}
}

func TestTerraformRefresh_state(t *testing.T) {
	rpAWS := new(MockResourceProvider)
	rpAWS.ResourcesReturn = []ResourceType{
		ResourceType{Name: "aws_instance"},
	}

	c := testConfig(t, "refresh-basic")
	tf := testTerraform2(t, &Config{
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(rpAWS),
		},
	})

	rpAWS.RefreshReturn = &ResourceState{
		ID: "foo",
	}

	state := &State{
		Resources: map[string]*ResourceState{
			"aws_instance.web": &ResourceState{
				ID: "bar",
			},
		},
	}

	s, err := tf.Refresh(c, state)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if !rpAWS.RefreshCalled {
		t.Fatal("refresh should be called")
	}
	if !reflect.DeepEqual(rpAWS.RefreshState, state.Resources["aws_instance.web"]) {
		t.Fatalf("bad: %#v", rpAWS.RefreshState)
	}
	if !reflect.DeepEqual(s.Resources["aws_instance.web"], rpAWS.RefreshReturn) {
		t.Fatalf("bad: %#v", s.Resources)
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
				result = result.MergeDiff(d)
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
			if _, ok := s.Attributes["nil"]; ok {
				return nil, nil
			}

			return s, nil
		}

		p.ApplyFn = applyFn
		p.DiffFn = diffFn
		p.RefreshFn = refreshFn
		p.ResourcesReturn = resources

		return p, nil
	}
}

func testProviderFuncFixed(rp ResourceProvider) ResourceProviderFactory {
	return func() (ResourceProvider, error) {
		return rp, nil
	}
}

func testProvider(tf *Terraform, n string) ResourceProvider {
	/*
		for r, tp := range tf.mapping {
			if r.Id() == n {
				return tp.Provider
			}
		}
	*/

	return nil
}

func testProviderMock(p ResourceProvider) *MockResourceProvider {
	return p.(*MockResourceProvider)
}

func testProviderConfig(tf *Terraform, n string) *config.ProviderConfig {
	/*
		for r, tp := range tf.mapping {
			if r.Id() == n {
				return tp.Config
			}
		}
	*/

	return nil
}

func testProviderName(t *testing.T, tf *Terraform, n string) string {
	var p ResourceProvider
	/*
		for r, tp := range tf.mapping {
			if r.Id() == n {
				p = tp.Provider
				break
			}
		}
	*/

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

func testTerraform2(t *testing.T, c *Config) *Terraform {
	if c == nil {
		c = &Config{
			Providers: map[string]ResourceProviderFactory{
				"aws": testProviderFunc("aws", []string{"aws_instance"}),
				"do":  testProviderFunc("do", []string{"do_droplet"}),
			},
		}
	}

	tf, err := New(c)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if tf == nil {
		t.Fatal("tf should not be nil")
	}

	return tf
}

func testTerraformProvider(tf *Terraform, n string) *terraformProvider {
	/*
		for r, tp := range tf.mapping {
			if r.Id() == n {
				return tp
			}
		}
	*/

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

const testTerraformApplyUnknownAttrStr = `
aws_instance.foo:
  ID = foo
  type = aws_instance
  num = 2
`

const testTerraformApplyVarsStr = `
aws_instance.bar:
  ID = foo
  type = aws_instance
  foo = bar
aws_instance.foo:
  ID = foo
  type = aws_instance
  num = 2
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
