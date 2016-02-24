package terraform

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func testContext2(t *testing.T, opts *ContextOpts) *Context {
	return NewContext(opts)
}

func testApplyFn(
	info *InstanceInfo,
	s *InstanceState,
	d *InstanceDiff) (*InstanceState, error) {
	if d.Destroy {
		return nil, nil
	}

	id := "foo"
	if idAttr, ok := d.Attributes["id"]; ok && !idAttr.NewComputed {
		id = idAttr.New
	}

	result := &InstanceState{
		ID:         id,
		Attributes: make(map[string]string),
	}

	// Copy all the prior attributes
	for k, v := range s.Attributes {
		result.Attributes[k] = v
	}

	if d != nil {
		result = result.MergeDiff(d)
	}
	return result, nil
}

func testDiffFn(
	info *InstanceInfo,
	s *InstanceState,
	c *ResourceConfig) (*InstanceDiff, error) {
	var diff InstanceDiff
	diff.Attributes = make(map[string]*ResourceAttrDiff)

	for k, v := range c.Raw {
		if _, ok := v.(string); !ok {
			continue
		}

		// Ignore __-prefixed keys since they're used for magic
		if k[0] == '_' && k[1] == '_' {
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

		if k == "require_new" {
			attrDiff.RequiresNew = true
		}
		if _, ok := c.Raw["__"+k+"_requires_new"]; ok {
			attrDiff.RequiresNew = true
		}
		diff.Attributes[k] = attrDiff
	}

	for _, k := range c.ComputedKeys {
		diff.Attributes[k] = &ResourceAttrDiff{
			Old:         "",
			NewComputed: true,
		}
	}

	for k, v := range diff.Attributes {
		if v.NewComputed {
			continue
		}

		old, ok := s.Attributes[k]
		if !ok {
			continue
		}
		if old == v.New {
			delete(diff.Attributes, k)
		}
	}

	if !diff.Empty() {
		diff.Attributes["type"] = &ResourceAttrDiff{
			Old: "",
			New: info.Type,
		}
	}

	return &diff, nil
}

func testProvider(prefix string) *MockResourceProvider {
	p := new(MockResourceProvider)
	p.RefreshFn = func(info *InstanceInfo, s *InstanceState) (*InstanceState, error) {
		return s, nil
	}
	p.ResourcesReturn = []ResourceType{
		ResourceType{
			Name: fmt.Sprintf("%s_instance", prefix),
		},
	}

	return p
}

func testProvisioner() *MockResourceProvisioner {
	p := new(MockResourceProvisioner)
	return p
}

func checkStateString(t *testing.T, state *State, expected string) {
	actual := strings.TrimSpace(state.String())
	expected = strings.TrimSpace(expected)

	if actual != expected {
		t.Fatalf("state does not match! actual:\n%s\n\nexpected:\n%s", actual, expected)
	}
}

func resourceState(resourceType, resourceID string) *ResourceState {
	return &ResourceState{
		Type: resourceType,
		Primary: &InstanceState{
			ID: resourceID,
		},
	}
}

// Test helper that gives a function 3 seconds to finish, assumes deadlock and
// fails test if it does not.
func testCheckDeadlock(t *testing.T, f func()) {
	timeout := make(chan bool, 1)
	done := make(chan bool, 1)
	go func() {
		time.Sleep(3 * time.Second)
		timeout <- true
	}()
	go func(f func(), done chan bool) {
		defer func() { done <- true }()
		f()
	}(f, done)
	select {
	case <-timeout:
		t.Fatalf("timed out! probably deadlock")
	case <-done:
		// ok
	}
}

const testContextGraph = `
root: root
aws_instance.bar
  aws_instance.bar -> provider.aws
aws_instance.foo
  aws_instance.foo -> provider.aws
provider.aws
root
  root -> aws_instance.bar
  root -> aws_instance.foo
`

const testContextRefreshModuleStr = `
aws_instance.web: (1 tainted)
  ID = <not created>
  Tainted ID 1 = bar

module.child:
  aws_instance.web:
    ID = new
`

const testContextRefreshOutputStr = `
aws_instance.web:
  ID = foo
  foo = bar

Outputs:

foo = bar
`

const testContextRefreshOutputPartialStr = `
<no state>
`

const testContextRefreshTaintedStr = `
aws_instance.web: (1 tainted)
  ID = <not created>
  Tainted ID 1 = foo
`
