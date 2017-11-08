package terraform

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform/flatmap"
	tfversion "github.com/hashicorp/terraform/version"
)

func TestNewContextRequiredVersion(t *testing.T) {
	cases := []struct {
		Name    string
		Module  string
		Version string
		Value   string
		Err     bool
	}{
		{
			"no requirement",
			"",
			"0.1.0",
			"",
			false,
		},

		{
			"doesn't match",
			"",
			"0.1.0",
			"> 0.6.0",
			true,
		},

		{
			"matches",
			"",
			"0.7.0",
			"> 0.6.0",
			false,
		},

		{
			"module matches",
			"context-required-version-module",
			"0.5.0",
			"",
			false,
		},

		{
			"module doesn't match",
			"context-required-version-module",
			"0.4.0",
			"",
			true,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d-%s", i, tc.Name), func(t *testing.T) {
			// Reset the version for the tests
			old := tfversion.SemVer
			tfversion.SemVer = version.Must(version.NewVersion(tc.Version))
			defer func() { tfversion.SemVer = old }()

			name := "context-required-version"
			if tc.Module != "" {
				name = tc.Module
			}
			mod := testModule(t, name)
			if tc.Value != "" {
				mod.Config().Terraform.RequiredVersion = tc.Value
			}
			_, err := NewContext(&ContextOpts{
				Module: mod,
			})
			if (err != nil) != tc.Err {
				t.Fatalf("err: %s", err)
			}
			if err != nil {
				return
			}
		})
	}
}

func TestNewContextState(t *testing.T) {
	cases := map[string]struct {
		Input *ContextOpts
		Err   bool
	}{
		"empty TFVersion": {
			&ContextOpts{
				State: &State{},
			},
			false,
		},

		"past TFVersion": {
			&ContextOpts{
				State: &State{TFVersion: "0.1.2"},
			},
			false,
		},

		"equal TFVersion": {
			&ContextOpts{
				State: &State{TFVersion: tfversion.Version},
			},
			false,
		},

		"future TFVersion": {
			&ContextOpts{
				State: &State{TFVersion: "99.99.99"},
			},
			true,
		},

		"future TFVersion, allowed": {
			&ContextOpts{
				State:              &State{TFVersion: "99.99.99"},
				StateFutureAllowed: true,
			},
			false,
		},
	}

	for k, tc := range cases {
		ctx, err := NewContext(tc.Input)
		if (err != nil) != tc.Err {
			t.Fatalf("%s: err: %s", k, err)
		}
		if err != nil {
			continue
		}

		// Version should always be set to our current
		if ctx.state.TFVersion != tfversion.Version {
			t.Fatalf("%s: state not set to current version", k)
		}
	}
}

func testContext2(t *testing.T, opts *ContextOpts) *Context {
	t.Helper()
	// Enable the shadow graph
	opts.Shadow = true

	ctx, err := NewContext(opts)
	if err != nil {
		t.Fatalf("failed to create test context\n\n%s\n", err)
	}

	return ctx
}

func testDataApplyFn(
	info *InstanceInfo,
	d *InstanceDiff) (*InstanceState, error) {
	return testApplyFn(info, new(InstanceState), d)
}

func testDataDiffFn(
	info *InstanceInfo,
	c *ResourceConfig) (*InstanceDiff, error) {
	return testDiffFn(info, new(InstanceState), c)
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
	diff := new(InstanceDiff)
	diff.Attributes = make(map[string]*ResourceAttrDiff)

	if s != nil {
		diff.DestroyTainted = s.Tainted
	}

	for k, v := range c.Raw {
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

		for k, attrDiff := range testFlatAttrDiffs(k, v) {
			if k == "require_new" {
				attrDiff.RequiresNew = true
			}
			if _, ok := c.Raw["__"+k+"_requires_new"]; ok {
				attrDiff.RequiresNew = true
			}

			if attr, ok := s.Attributes[k]; ok {
				attrDiff.Old = attr
			}

			diff.Attributes[k] = attrDiff
		}
	}

	for _, k := range c.ComputedKeys {
		diff.Attributes[k] = &ResourceAttrDiff{
			Old:         "",
			NewComputed: true,
		}
	}

	// If we recreate this resource because it's tainted, we keep all attrs
	if !diff.RequiresNew() {
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
	}

	if !diff.Empty() {
		diff.Attributes["type"] = &ResourceAttrDiff{
			Old: "",
			New: info.Type,
		}
	}

	return diff, nil
}

// generate ResourceAttrDiffs for nested data structures in tests
func testFlatAttrDiffs(k string, i interface{}) map[string]*ResourceAttrDiff {
	diffs := make(map[string]*ResourceAttrDiff)
	// check for strings and empty containers first
	switch t := i.(type) {
	case string:
		diffs[k] = &ResourceAttrDiff{New: t}
		return diffs
	case map[string]interface{}:
		if len(t) == 0 {
			diffs[k] = &ResourceAttrDiff{New: ""}
			return diffs
		}
	case []interface{}:
		if len(t) == 0 {
			diffs[k] = &ResourceAttrDiff{New: ""}
			return diffs
		}
	}

	flat := flatmap.Flatten(map[string]interface{}{k: i})

	for k, v := range flat {
		attrDiff := &ResourceAttrDiff{
			Old: "",
			New: v,
		}
		diffs[k] = attrDiff
	}

	return diffs
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
	t.Helper()
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
	t.Helper()
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
aws_instance.web: (tainted)
  ID = bar
  provider = provider.aws

module.child:
  aws_instance.web:
    ID = new
    provider = provider.aws
`

const testContextRefreshOutputStr = `
aws_instance.web:
  ID = foo
  provider = provider.aws
  foo = bar

Outputs:

foo = bar
`

const testContextRefreshOutputPartialStr = `
<no state>
`

const testContextRefreshTaintedStr = `
aws_instance.web: (tainted)
  ID = foo
  provider = provider.aws
`
