package terraform

import (
	"fmt"
	"path/filepath"
	"reflect"
	"sync"
	"testing"

	"github.com/hashicorp/terraform/config"
)

// This is the directory where our test fixtures are.
const fixtureDir = "./test-fixtures"

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
	if rpAWS.RefreshState.ID != "" {
		t.Fatalf("bad: %#v", rpAWS.RefreshState)
	}
	if !reflect.DeepEqual(s.Resources["aws_instance.web"], rpAWS.RefreshReturn) {
		t.Fatalf("bad: %#v", s.Resources)
	}

	for _, r := range s.Resources {
		if r.Type == "" {
			t.Fatalf("no type: %#v", r)
		}
	}
}

func TestTerraformRefresh_hook(t *testing.T) {
	rpAWS := new(MockResourceProvider)
	rpAWS.ResourcesReturn = []ResourceType{
		ResourceType{Name: "aws_instance"},
	}

	h := new(MockHook)

	c := testConfig(t, "refresh-basic")
	tf := testTerraform2(t, &Config{
		Hooks: []Hook{h},
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(rpAWS),
		},
	})

	if _, err := tf.Refresh(c, nil); err != nil {
		t.Fatalf("err: %s", err)
	}
	if !h.PreRefreshCalled {
		t.Fatal("should be called")
	}
	if h.PreRefreshState.Type != "aws_instance" {
		t.Fatalf("bad: %#v", h.PreRefreshState)
	}
	if !h.PostRefreshCalled {
		t.Fatal("should be called")
	}
	if h.PostRefreshState.Type != "aws_instance" {
		t.Fatalf("bad: %#v", h.PostRefreshState)
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
			if d.Destroy {
				return nil, nil
			}

			id := "foo"
			if idAttr, ok := d.Attributes["id"]; ok && !idAttr.NewComputed {
				id = idAttr.New
			}

			result := &ResourceState{
				ID: id,
			}

			if d != nil {
				result = result.MergeDiff(d)
			}

			if depAttr, ok := d.Attributes["dep"]; ok {
				result.Dependencies = []ResourceDependency{
					ResourceDependency{
						ID: depAttr.New,
					},
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

				// This key is used for other purposes
				if k == "compute_value" {
					continue
				}

				if k == "compute" {
					attrDiff := &ResourceAttrDiff{
						Old:         "",
						New:         "",
						NewComputed: true,
					}

					if cv, ok := c.Config["compute_value"]; ok {
						if cv.(string) == "1" {
							attrDiff.NewComputed = false
							attrDiff.New = fmt.Sprintf("computed_%s", v.(string))
						}
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

func testProviderMock(p ResourceProvider) *MockResourceProvider {
	return p.(*MockResourceProvider)
}

func testTerraform2(t *testing.T, c *Config) *Terraform {
	if c == nil {
		c = new(Config)
	}

	if c.Providers == nil {
		c.Providers = map[string]ResourceProviderFactory{
			"aws": testProviderFunc("aws", []string{"aws_instance"}),
			"do":  testProviderFunc("do", []string{"do_droplet"}),
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

// HookRecordApplyOrder is a test hook that records the order of applies
// by recording the PreApply event.
type HookRecordApplyOrder struct {
	NilHook

	Active bool

	IDs    []string
	States []*ResourceState
	Diffs  []*ResourceDiff

	l sync.Mutex
}

func (h *HookRecordApplyOrder) PreApply(
	id string,
	s *ResourceState,
	d *ResourceDiff) (HookAction, error) {
	if h.Active {
		h.l.Lock()
		defer h.l.Unlock()

		h.IDs = append(h.IDs, id)
		h.Diffs = append(h.Diffs, d)
		h.States = append(h.States, s)
	}

	return HookActionContinue, nil
}

// Below are all the constant strings that are the expected output for
// various tests.

const testTerraformApplyStr = `
aws_instance.bar:
  ID = foo
  foo = bar
  type = aws_instance
aws_instance.foo:
  ID = foo
  num = 2
  type = aws_instance
`

const testTerraformApplyCancelStr = `
aws_instance.foo:
  ID = foo
  num = 2
`

const testTerraformApplyComputeStr = `
aws_instance.bar:
  ID = foo
  foo = computed_dynamical
  type = aws_instance
aws_instance.foo:
  ID = foo
  dynamical = computed_dynamical
  num = 2
  type = aws_instance
`

const testTerraformApplyDestroyStr = `
<no state>
`

const testTerraformApplyErrorStr = `
aws_instance.foo:
  ID = foo
  num = 2
`

const testTerraformApplyErrorPartialStr = `
aws_instance.bar:
  ID = bar
aws_instance.foo:
  ID = foo
  num = 2
`

const testTerraformApplyUnknownAttrStr = `
aws_instance.foo:
  ID = foo
  num = 2
  type = aws_instance
`

const testTerraformApplyVarsStr = `
aws_instance.bar:
  ID = foo
  foo = bar
  type = aws_instance
aws_instance.foo:
  ID = foo
  num = 2
  type = aws_instance
`

const testTerraformPlanStr = `
DIFF:

UPDATE: aws_instance.bar
  foo:  "" => "2"
  type: "" => "aws_instance"
UPDATE: aws_instance.foo
  num:  "" => "2"
  type: "" => "aws_instance"

STATE:

<no state>
`

const testTerraformPlanComputedStr = `
DIFF:

UPDATE: aws_instance.bar
  foo:  "" => "<computed>"
  type: "" => "aws_instance"
UPDATE: aws_instance.foo
  id:   "" => "<computed>"
  num:  "" => "2"
  type: "" => "aws_instance"

STATE:

<no state>
`

const testTerraformPlanDestroyStr = `
DIFF:

DESTROY: aws_instance.one
DESTROY: aws_instance.two

STATE:

aws_instance.one:
  ID = bar
aws_instance.two:
  ID = baz
`

const testTerraformPlanOrphanStr = `
DIFF:

DESTROY: aws_instance.baz
UPDATE: aws_instance.foo
  num:  "" => "2"
  type: "" => "aws_instance"

STATE:

aws_instance.baz:
  ID = bar
`

const testTerraformPlanStateStr = `
DIFF:

UPDATE: aws_instance.bar
  foo:  "" => "2"
  type: "" => "aws_instance"
UPDATE: aws_instance.foo
  num:  "" => "2"
  type: "" => ""

STATE:

aws_instance.foo:
  ID = bar
`
