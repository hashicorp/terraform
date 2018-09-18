package terraform

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hil"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/config/hcl2shim"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/configs/configload"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/flatmap"
	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/plans/planfile"
	"github.com/hashicorp/terraform/providers"
	"github.com/hashicorp/terraform/provisioners"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/states/statefile"
	"github.com/hashicorp/terraform/tfdiags"
	tfversion "github.com/hashicorp/terraform/version"
)

var (
	equateEmpty   = cmpopts.EquateEmpty()
	typeComparer  = cmp.Comparer(cty.Type.Equals)
	valueComparer = cmp.Comparer(cty.Value.RawEquals)
	valueTrans    = cmp.Transformer("hcl2shim", hcl2shim.ConfigValueFromHCL2)
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
				constraint, err := version.NewConstraint(tc.Value)
				if err != nil {
					t.Fatalf("can't parse %q as version constraint", tc.Value)
				}
				mod.Module.CoreVersionConstraints = append(mod.Module.CoreVersionConstraints, configs.VersionConstraint{
					Required: constraint,
				})
			}
			_, diags := NewContext(&ContextOpts{
				Config: mod,
			})
			if diags.HasErrors() != tc.Err {
				t.Fatalf("err: %s", diags.Err())
			}
		})
	}
}

func testContext2(t *testing.T, opts *ContextOpts) *Context {
	t.Helper()

	ctx, diags := NewContext(opts)
	if diags.HasErrors() {
		t.Fatalf("failed to create test context\n\n%s\n", diags.Err())
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

	// find the OLD id, which is probably in the ID field for now, but eventually
	// ID should only be in one place.
	id := s.ID
	if id == "" {
		id = s.Attributes["id"]
	}
	if idAttr, ok := d.Attributes["id"]; ok && !idAttr.NewComputed {
		id = idAttr.New
	}

	if id == "" || id == config.UnknownVariableValue {
		id = "foo"
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
			if v == hil.UnknownValue || v == "unknown" {
				// compute wasn't set in the config, so don't use these
				// computed values from the schema.
				delete(c.Raw, k)
				delete(c.Raw, "compute_value")

				// we need to remove this from the list of ComputedKeys too,
				// since it would get re-added to the diff further down
				newComputed := make([]string, 0, len(c.ComputedKeys))
				for _, ck := range c.ComputedKeys {
					if ck == "compute" || ck == "compute_value" {
						continue
					}
					newComputed = append(newComputed, ck)
				}
				c.ComputedKeys = newComputed

				if v == "unknown" {
					diff.Attributes["unknown"] = &ResourceAttrDiff{
						Old:         "",
						New:         "",
						NewComputed: true,
					}

					c.ComputedKeys = append(c.ComputedKeys, "unknown")
				}

				continue
			}

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
			// we need to ignore 'id' for now, since it's always inferred to be
			// computed.
			if k == "id" {
				continue
			}

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
		if k == "id" {
			continue
		}
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
		if s != nil && s.Attributes != nil {
			diff.Attributes["type"].Old = s.Attributes["type"]
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

	// The legacy flatmap-based diff producing done by helper/schema would
	// additionally insert a k+".%" key here recording the length of the map,
	// which is for some reason not also done by flatmap.Flatten. To make our
	// mock shims helper/schema-compatible, we'll just fake that up here.
	switch t := i.(type) {
	case map[string]interface{}:
		attrDiff := &ResourceAttrDiff{
			Old: "",
			New: strconv.Itoa(len(t)),
		}
		diffs[k+".%"] = attrDiff
	}

	return diffs
}

func testProvider(prefix string) *MockProvider {
	p := new(MockProvider)
	p.ReadResourceFn = func(req providers.ReadResourceRequest) providers.ReadResourceResponse {
		return providers.ReadResourceResponse{NewState: req.PriorState}
	}

	p.GetSchemaReturn = testProviderSchema(prefix)

	return p
}

func testProvisioner() *MockProvisioner {
	p := new(MockProvisioner)
	p.GetSchemaResponse = provisioners.GetSchemaResponse{
		Provisioner: &configschema.Block{
			Attributes: map[string]*configschema.Attribute{
				"command": {
					Type:     cty.String,
					Optional: true,
				},
				"order": {
					Type:     cty.String,
					Optional: true,
				},
				"when": {
					Type:     cty.String,
					Optional: true,
				},
			},
		},
	}
	return p
}

func checkStateString(t *testing.T, state *states.State, expected string) {
	t.Helper()
	actual := strings.TrimSpace(state.String())
	expected = strings.TrimSpace(expected)

	if actual != expected {
		t.Fatalf("incorrect state\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

func resourceState(resourceType, resourceID string) *ResourceState {
	providerResource := strings.Split(resourceType, "_")
	return &ResourceState{
		Type: resourceType,
		Primary: &InstanceState{
			ID: resourceID,
			Attributes: map[string]string{
				"id": resourceID,
			},
		},
		Provider: "provider." + providerResource[0],
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

func testProviderSchema(name string) *ProviderSchema {
	return &ProviderSchema{
		Provider: &configschema.Block{
			Attributes: map[string]*configschema.Attribute{
				"region": {
					Type:     cty.String,
					Optional: true,
				},
				"foo": {
					Type:     cty.String,
					Optional: true,
				},
				"value": {
					Type:     cty.String,
					Optional: true,
				},
				"root": {
					Type:     cty.Number,
					Optional: true,
				},
			},
		},
		ResourceTypes: map[string]*configschema.Block{
			name + "_instance": {
				Attributes: map[string]*configschema.Attribute{
					"id": {
						Type:     cty.String,
						Computed: true,
					},
					"ami": {
						Type:     cty.String,
						Optional: true,
					},
					"dep": {
						Type:     cty.String,
						Optional: true,
					},
					"num": {
						Type:     cty.Number,
						Optional: true,
					},
					"require_new": {
						Type:     cty.String,
						Optional: true,
					},
					"var": {
						Type:     cty.String,
						Optional: true,
					},
					"foo": {
						Type:     cty.String,
						Optional: true,
					},
					"bar": {
						Type:     cty.String,
						Optional: true,
					},
					"compute": {
						Type:     cty.String,
						Optional: true,
						Computed: false,
					},
					"compute_value": {
						Type:     cty.String,
						Optional: true,
						Computed: true,
					},
					"value": {
						Type:     cty.String,
						Optional: true,
					},
					"output": {
						Type:     cty.String,
						Optional: true,
					},
					"write": {
						Type:     cty.String,
						Optional: true,
					},
					"instance": {
						Type:     cty.String,
						Optional: true,
					},
					"vpc_id": {
						Type:     cty.String,
						Optional: true,
					},
					"type": {
						Type:     cty.String,
						Computed: true,
					},

					// Generated by testDiffFn if compute = "unknown" is set in the test config
					"unknown": {
						Type:     cty.String,
						Computed: true,
					},
				},
			},
			name + "_eip": {
				Attributes: map[string]*configschema.Attribute{
					"id": {
						Type:     cty.String,
						Computed: true,
					},
					"instance": {
						Type:     cty.String,
						Optional: true,
					},
				},
			},
			name + "_resource": {
				Attributes: map[string]*configschema.Attribute{
					"id": {
						Type:     cty.String,
						Computed: true,
					},
					"value": {
						Type:     cty.String,
						Optional: true,
					},
					"random": {
						Type:     cty.String,
						Optional: true,
					},
				},
			},
			name + "_ami_list": {
				Attributes: map[string]*configschema.Attribute{
					"id": {
						Type:     cty.String,
						Optional: true,
					},
					"ids": {
						Type:     cty.List(cty.String),
						Optional: true,
					},
				},
			},
			name + "_remote_state": {
				Attributes: map[string]*configschema.Attribute{
					"id": {
						Type:     cty.String,
						Optional: true,
					},
					"foo": {
						Type:     cty.String,
						Optional: true,
					},
					"output": {
						Type:     cty.Map(cty.String),
						Computed: true,
					},
				},
			},
			name + "_file": {
				Attributes: map[string]*configschema.Attribute{
					"id": {
						Type:     cty.String,
						Optional: true,
					},
					"template": {
						Type:     cty.String,
						Optional: true,
					},
					"rendered": {
						Type:     cty.String,
						Computed: true,
					},
					"__template_requires_new": {
						Type:     cty.String,
						Optional: true,
					},
				},
			},
		},
		DataSources: map[string]*configschema.Block{
			name + "_data_source": {
				Attributes: map[string]*configschema.Attribute{
					"id": {
						Type:     cty.String,
						Optional: true,
					},
					"foo": {
						Type:     cty.String,
						Optional: true,
					},
				},
			},
			name + "_remote_state": {
				Attributes: map[string]*configschema.Attribute{
					"id": {
						Type:     cty.String,
						Optional: true,
					},
					"foo": {
						Type:     cty.String,
						Optional: true,
					},
					"output": {
						Type:     cty.Map(cty.String),
						Optional: true,
					},
				},
			},
			name + "_file": {
				Attributes: map[string]*configschema.Attribute{
					"id": {
						Type:     cty.String,
						Optional: true,
					},
					"template": {
						Type:     cty.String,
						Optional: true,
					},
					"rendered": {
						Type:     cty.String,
						Computed: true,
					},
				},
			},
		},
	}

}

// contextForPlanViaFile is a helper that creates a temporary plan file, then
// reads it back in again and produces a ContextOpts object containing the
// planned changes, prior state and config from the plan file.
//
// This is intended for testing the separated plan/apply workflow in a more
// convenient way than spelling out all of these steps every time. Normally
// only the command and backend packages need to deal with such things, but
// our context tests try to exercise lots of stuff at once and so having them
// round-trip things through on-disk files is often an important part of
// fully representing an old bug in a regression test.
func contextOptsForPlanViaFile(configSnap *configload.Snapshot, state *states.State, plan *plans.Plan) (*ContextOpts, error) {
	dir, err := ioutil.TempDir("", "terraform-contextForPlanViaFile")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)

	// We'll just create a dummy statefile.File here because we're not going
	// to run through any of the codepaths that care about Lineage/Serial/etc
	// here anyway.
	stateFile := &statefile.File{
		State: state,
	}

	filename := filepath.Join(dir, "tfplan")
	err = planfile.Create(filename, configSnap, stateFile, plan)
	if err != nil {
		return nil, err
	}

	pr, err := planfile.Open(filename)
	if err != nil {
		return nil, err
	}

	config, diags := pr.ReadConfig()
	if diags.HasErrors() {
		return nil, diags.Err()
	}

	stateFile, err = pr.ReadStateFile()
	if err != nil {
		return nil, err
	}

	plan, err = pr.ReadPlan()
	if err != nil {
		return nil, err
	}

	vars := make(InputValues)
	for name, vv := range plan.VariableValues {
		val, err := vv.Decode(cty.DynamicPseudoType)
		if err != nil {
			return nil, fmt.Errorf("can't decode value for variable %q: %s", name, err)
		}
		vars[name] = &InputValue{
			Value:      val,
			SourceType: ValueFromPlan,
		}
	}

	return &ContextOpts{
		Config:          config,
		State:           stateFile.State,
		Changes:         plan.Changes,
		Variables:       vars,
		Targets:         plan.TargetAddrs,
		ProviderSHA256s: plan.ProviderSHA256s,
	}, nil
}

// shimLegacyState is a helper that takes the legacy state type and
// converts it to the new state type.
//
// This is implemented as a state file upgrade, so it will not preserve
// parts of the state structure that are not included in a serialized state,
// such as the resolved results of any local values, outputs in non-root
// modules, etc.
func shimLegacyState(legacy *State) (*states.State, error) {
	if legacy == nil {
		return nil, nil
	}
	var buf bytes.Buffer
	err := WriteState(legacy, &buf)
	if err != nil {
		return nil, err
	}
	f, err := statefile.Read(&buf)
	if err != nil {
		return nil, err
	}
	return f.State, err
}

// mustShimLegacyState is a wrapper around ShimLegacyState that panics if
// the conversion does not succeed. This is primarily intended for tests where
// the given legacy state is an object constructed within the test.
func mustShimLegacyState(legacy *State) *states.State {
	ret, err := shimLegacyState(legacy)
	if err != nil {
		panic(err)
	}
	return ret
}

// legacyPlanComparisonString produces a string representation of the changes
// from a plan and a given state togther, as was formerly produced by the
// String method of terraform.Plan.
//
// This is here only for compatibility with existing tests that predate our
// new plan and state types, and should not be used in new tests. Instead, use
// a library like "cmp" to do a deep equality check and diff on the two
// data structures.
func legacyPlanComparisonString(state *states.State, changes *plans.Changes) string {
	return fmt.Sprintf(
		"DIFF:\n\n%s\n\nSTATE:\n\n%s",
		legacyDiffComparisonString(changes),
		state.String(),
	)
}

// legacyDiffComparisonString produces a string representation of the changes
// from a planned changes object, as was formerly produced by the String method
// of terraform.Diff.
//
// This is here only for compatibility with existing tests that predate our
// new plan types, and should not be used in new tests. Instead, use a library
// like "cmp" to do a deep equality check and diff on the two data structures.
func legacyDiffComparisonString(changes *plans.Changes) string {
	// The old string representation of a plan was grouped by module, but
	// our new plan structure is not grouped in that way and so we'll need
	// to preprocess it in order to produce that grouping.
	type ResourceChanges struct {
		Current *plans.ResourceInstanceChangeSrc
		Deposed map[states.DeposedKey]*plans.ResourceInstanceChangeSrc
	}
	byModule := map[string]map[string]*ResourceChanges{}
	resourceKeys := map[string][]string{}
	var moduleKeys []string
	for _, rc := range changes.Resources {
		moduleKey := rc.Addr.Module.String()
		if _, exists := byModule[moduleKey]; !exists {
			moduleKeys = append(moduleKeys, moduleKey)
			byModule[moduleKey] = make(map[string]*ResourceChanges)
		}
		resourceKey := rc.Addr.Resource.String()
		if _, exists := byModule[moduleKey][resourceKey]; !exists {
			resourceKeys[moduleKey] = append(resourceKeys[moduleKey], resourceKey)
			byModule[moduleKey][resourceKey] = &ResourceChanges{
				Deposed: make(map[states.DeposedKey]*plans.ResourceInstanceChangeSrc),
			}
		}

		if rc.DeposedKey == states.NotDeposed {
			byModule[moduleKey][resourceKey].Current = rc
		} else {
			byModule[moduleKey][resourceKey].Deposed[rc.DeposedKey] = rc
		}
	}
	sort.Strings(moduleKeys)
	for _, ks := range resourceKeys {
		sort.Strings(ks)
	}

	var buf bytes.Buffer

	for _, moduleKey := range moduleKeys {
		rcs := byModule[moduleKey]
		var mBuf bytes.Buffer

		for _, resourceKey := range resourceKeys[moduleKey] {
			rc := rcs[resourceKey]

			crud := "UPDATE"
			if rc.Current != nil {
				switch rc.Current.Action {
				case plans.Replace:
					crud = "DESTROY/CREATE"
				case plans.Delete:
					crud = "DESTROY"
				case plans.Create:
					crud = "CREATE"
				}
			} else {
				// We must be working on a deposed object then, in which
				// case destroying is the only possible action.
				crud = "DESTROY"
			}

			extra := ""
			if rc.Current == nil && len(rc.Deposed) > 0 {
				extra = " (deposed only)"
			}

			fmt.Fprintf(
				&mBuf, "%s: %s%s\n",
				crud, resourceKey, extra,
			)

			attrNames := map[string]bool{}
			var oldAttrs map[string]string
			var newAttrs map[string]string
			if before := rc.Current.Before; before != nil {
				ty, err := before.ImpliedType()
				if err == nil {
					val, err := before.Decode(ty)
					if err == nil {
						oldAttrs = hcl2shim.FlatmapValueFromHCL2(val)
						for k := range oldAttrs {
							attrNames[k] = true
						}
					}
				}
			}
			if after := rc.Current.After; after != nil {
				ty, err := after.ImpliedType()
				if err == nil {
					val, err := after.Decode(ty)
					if err == nil {
						newAttrs = hcl2shim.FlatmapValueFromHCL2(val)
						for k := range newAttrs {
							attrNames[k] = true
						}
					}
				}
			}
			if oldAttrs == nil {
				oldAttrs = make(map[string]string)
			}
			if newAttrs == nil {
				newAttrs = make(map[string]string)
			}

			attrNamesOrder := make([]string, 0, len(attrNames))
			keyLen := 0
			for n := range attrNames {
				attrNamesOrder = append(attrNamesOrder, n)
				if len(n) > keyLen {
					keyLen = len(n)
				}
			}

			for _, attrK := range attrNamesOrder {
				v := newAttrs[attrK]
				u := oldAttrs[attrK]

				if v == config.UnknownVariableValue {
					v = "<computed>"
				}
				// NOTE: we don't support <sensitive> here because we would
				// need schema to do that. Excluding sensitive values
				// is now done at the UI layer, and so should not be tested
				// at the core layer.

				updateMsg := ""
				// TODO: Mark " (forces new resource)" in updateMsg when appropriate.

				fmt.Fprintf(
					&mBuf, "  %s:%s %#v => %#v%s\n",
					attrK,
					strings.Repeat(" ", keyLen-len(attrK)),
					u, v,
					updateMsg,
				)
			}
		}

		if moduleKey == "" { // root module
			buf.Write(mBuf.Bytes())
			buf.WriteByte('\n')
			continue
		}

		fmt.Fprintf(&buf, "%s:\n", moduleKey)
		s := bufio.NewScanner(&mBuf)
		for s.Scan() {
			buf.WriteString(fmt.Sprintf("  %s\n", s.Text()))
		}
	}

	return buf.String()
}

// assertNoDiagnostics fails the test in progress (using t.Fatal) if the given
// diagnostics is non-empty.
func assertNoDiagnostics(t *testing.T, diags tfdiags.Diagnostics) {
	t.Helper()
	if len(diags) == 0 {
		return
	}
	logDiagnostics(t, diags)
	t.FailNow()
}

// assertNoDiagnostics fails the test in progress (using t.Fatal) if the given
// diagnostics has any errors.
func assertNoErrors(t *testing.T, diags tfdiags.Diagnostics) {
	t.Helper()
	if !diags.HasErrors() {
		return
	}
	logDiagnostics(t, diags)
	t.FailNow()
}

// logDiagnostics is a test helper that logs the given diagnostics to to the
// given testing.T using t.Log, in a way that is hopefully useful in debugging
// a test. It does not generate any errors or fail the test. See
// assertNoDiagnostics and assertNoErrors for more specific helpers that can
// also fail the test.
func logDiagnostics(t *testing.T, diags tfdiags.Diagnostics) {
	t.Helper()
	for _, diag := range diags {
		desc := diag.Description()
		rng := diag.Source()

		var severity string
		switch diag.Severity() {
		case tfdiags.Error:
			severity = "ERROR"
		case tfdiags.Warning:
			severity = "WARN"
		default:
			severity = "???" // should never happen
		}

		if subj := rng.Subject; subj != nil {
			if desc.Detail == "" {
				t.Logf("[%s@%s] %s", severity, subj.StartString(), desc.Summary)
			} else {
				t.Logf("[%s@%s] %s: %s", severity, subj.StartString(), desc.Summary, desc.Detail)
			}
		} else {
			if desc.Detail == "" {
				t.Logf("[%s] %s", severity, desc.Summary)
			} else {
				t.Logf("[%s] %s: %s", severity, desc.Summary, desc.Detail)
			}
		}
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
