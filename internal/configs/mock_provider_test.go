package configs

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
)

func TestMockData_Merge(t *testing.T) {

	tcs := map[string]struct {
		current *MockData
		target  *MockData
		result  *MockData
	}{
		"empty_target": {
			current: &MockData{
				MockResources: map[string]*MockResource{
					"test_resource": {
						Mode:     addrs.ManagedResourceMode,
						Type:     "test_resource",
						Defaults: cty.StringVal("current"),
					},
				},
				MockDataSources: map[string]*MockResource{
					"test_data_source": {
						Mode:     addrs.ManagedResourceMode,
						Type:     "test_data_source",
						Defaults: cty.StringVal("current"),
					},
				},
				Overrides: addrs.MakeMap[addrs.Targetable, *Override](
					makeOverride(t, "test_resource.my_resource", cty.StringVal("current")),
				),
			},
			target: &MockData{
				MockResources:   map[string]*MockResource{},
				MockDataSources: map[string]*MockResource{},
				Overrides:       addrs.MakeMap[addrs.Targetable, *Override](),
			},
			result: &MockData{
				MockResources: map[string]*MockResource{
					"test_resource": {
						Mode:     addrs.ManagedResourceMode,
						Type:     "test_resource",
						Defaults: cty.StringVal("current"),
					},
				},
				MockDataSources: map[string]*MockResource{
					"test_data_source": {
						Mode:     addrs.ManagedResourceMode,
						Type:     "test_data_source",
						Defaults: cty.StringVal("current"),
					},
				},
				Overrides: addrs.MakeMap[addrs.Targetable, *Override](
					makeOverride(t, "test_resource.my_resource", cty.StringVal("current")),
				),
			},
		},
		"nil_target": {
			current: &MockData{
				MockResources: map[string]*MockResource{
					"test_resource": {
						Mode:     addrs.ManagedResourceMode,
						Type:     "test_resource",
						Defaults: cty.StringVal("current"),
					},
				},
				MockDataSources: map[string]*MockResource{
					"test_data_source": {
						Mode:     addrs.ManagedResourceMode,
						Type:     "test_data_source",
						Defaults: cty.StringVal("current"),
					},
				},
				Overrides: addrs.MakeMap[addrs.Targetable, *Override](
					makeOverride(t, "test_resource.my_resource", cty.StringVal("current")),
				),
			},
			target: nil,
			result: &MockData{
				MockResources: map[string]*MockResource{
					"test_resource": {
						Mode:     addrs.ManagedResourceMode,
						Type:     "test_resource",
						Defaults: cty.StringVal("current"),
					},
				},
				MockDataSources: map[string]*MockResource{
					"test_data_source": {
						Mode:     addrs.ManagedResourceMode,
						Type:     "test_data_source",
						Defaults: cty.StringVal("current"),
					},
				},
				Overrides: addrs.MakeMap[addrs.Targetable, *Override](
					makeOverride(t, "test_resource.my_resource", cty.StringVal("current")),
				),
			},
		},
		"all_collisions": {
			current: &MockData{
				MockResources: map[string]*MockResource{
					"test_resource": {
						Mode:     addrs.ManagedResourceMode,
						Type:     "test_resource",
						Defaults: cty.StringVal("current"),
					},
				},
				MockDataSources: map[string]*MockResource{
					"test_data_source": {
						Mode:     addrs.ManagedResourceMode,
						Type:     "test_data_source",
						Defaults: cty.StringVal("current"),
					},
				},
				Overrides: addrs.MakeMap[addrs.Targetable, *Override](
					makeOverride(t, "test_resource.my_resource", cty.StringVal("current")),
				),
			},
			target: &MockData{
				MockResources: map[string]*MockResource{
					"test_resource": {
						Mode:     addrs.ManagedResourceMode,
						Type:     "test_resource",
						Defaults: cty.StringVal("target"),
					},
				},
				MockDataSources: map[string]*MockResource{
					"test_data_source": {
						Mode:     addrs.ManagedResourceMode,
						Type:     "test_data_source",
						Defaults: cty.StringVal("target"),
					},
				},
				Overrides: addrs.MakeMap[addrs.Targetable, *Override](
					makeOverride(t, "test_resource.my_resource", cty.StringVal("target")),
				),
			},
			result: &MockData{
				MockResources: map[string]*MockResource{
					"test_resource": {
						Mode:     addrs.ManagedResourceMode,
						Type:     "test_resource",
						Defaults: cty.StringVal("current"),
					},
				},
				MockDataSources: map[string]*MockResource{
					"test_data_source": {
						Mode:     addrs.ManagedResourceMode,
						Type:     "test_data_source",
						Defaults: cty.StringVal("current"),
					},
				},
				Overrides: addrs.MakeMap[addrs.Targetable, *Override](
					makeOverride(t, "test_resource.my_resource", cty.StringVal("current")),
				),
			},
		},
		"no_collisions": {
			current: &MockData{
				MockResources: map[string]*MockResource{
					"test_resource": {
						Mode:     addrs.ManagedResourceMode,
						Type:     "test_resource",
						Defaults: cty.StringVal("current"),
					},
				},
				MockDataSources: map[string]*MockResource{
					"test_data_source": {
						Mode:     addrs.ManagedResourceMode,
						Type:     "test_data_source",
						Defaults: cty.StringVal("current"),
					},
				},
				Overrides: addrs.MakeMap[addrs.Targetable, *Override](
					makeOverride(t, "test_resource.my_resource", cty.StringVal("current")),
				),
			},
			target: &MockData{
				MockResources: map[string]*MockResource{
					"test_resource_two": {
						Mode:     addrs.ManagedResourceMode,
						Type:     "test_resource_two",
						Defaults: cty.StringVal("target"),
					},
				},
				MockDataSources: map[string]*MockResource{
					"test_data_source_two": {
						Mode:     addrs.ManagedResourceMode,
						Type:     "test_data_source_two",
						Defaults: cty.StringVal("target"),
					},
				},
				Overrides: addrs.MakeMap[addrs.Targetable, *Override](
					makeOverride(t, "test_resource.my_other_resource", cty.StringVal("target")),
				),
			},
			result: &MockData{
				MockResources: map[string]*MockResource{
					"test_resource": {
						Mode:     addrs.ManagedResourceMode,
						Type:     "test_resource",
						Defaults: cty.StringVal("current"),
					},
					"test_resource_two": {
						Mode:     addrs.ManagedResourceMode,
						Type:     "test_resource_two",
						Defaults: cty.StringVal("target"),
					},
				},
				MockDataSources: map[string]*MockResource{
					"test_data_source": {
						Mode:     addrs.ManagedResourceMode,
						Type:     "test_data_source",
						Defaults: cty.StringVal("current"),
					},
					"test_data_source_two": {
						Mode:     addrs.ManagedResourceMode,
						Type:     "test_data_source_two",
						Defaults: cty.StringVal("target"),
					},
				},
				Overrides: addrs.MakeMap[addrs.Targetable, *Override](
					makeOverride(t, "test_resource.my_resource", cty.StringVal("current")),
					makeOverride(t, "test_resource.my_other_resource", cty.StringVal("target")),
				),
			},
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			diags := tc.current.Merge(tc.target, true)
			validateMockData(t, tc.current, tc.result)

			var details []string
			for _, diag := range diags {
				details = append(details, diag.Detail)
			}
			if len(details) > 0 {
				t.Errorf("expected no diags but found [%s]", strings.Join(details, ", "))
			}

		})
	}
}

func TestMockData_MergeWithCollisions(t *testing.T) {

	tcs := map[string]struct {
		current *MockData
		target  *MockData
		result  *MockData
		diags   []string
	}{
		"empty_target": {
			current: &MockData{
				MockResources: map[string]*MockResource{
					"test_resource": {
						Mode:     addrs.ManagedResourceMode,
						Type:     "test_resource",
						Defaults: cty.StringVal("current"),
					},
				},
				MockDataSources: map[string]*MockResource{
					"test_data_source": {
						Mode:     addrs.ManagedResourceMode,
						Type:     "test_data_source",
						Defaults: cty.StringVal("current"),
					},
				},
				Overrides: addrs.MakeMap[addrs.Targetable, *Override](
					makeOverride(t, "test_resource.my_resource", cty.StringVal("current")),
				),
			},
			target: &MockData{
				MockResources:   map[string]*MockResource{},
				MockDataSources: map[string]*MockResource{},
				Overrides:       addrs.MakeMap[addrs.Targetable, *Override](),
			},
			result: &MockData{
				MockResources: map[string]*MockResource{
					"test_resource": {
						Mode:     addrs.ManagedResourceMode,
						Type:     "test_resource",
						Defaults: cty.StringVal("current"),
					},
				},
				MockDataSources: map[string]*MockResource{
					"test_data_source": {
						Mode:     addrs.ManagedResourceMode,
						Type:     "test_data_source",
						Defaults: cty.StringVal("current"),
					},
				},
				Overrides: addrs.MakeMap[addrs.Targetable, *Override](
					makeOverride(t, "test_resource.my_resource", cty.StringVal("current")),
				),
			},
		},
		"nil_target": {
			current: &MockData{
				MockResources: map[string]*MockResource{
					"test_resource": {
						Mode:     addrs.ManagedResourceMode,
						Type:     "test_resource",
						Defaults: cty.StringVal("current"),
					},
				},
				MockDataSources: map[string]*MockResource{
					"test_data_source": {
						Mode:     addrs.ManagedResourceMode,
						Type:     "test_data_source",
						Defaults: cty.StringVal("current"),
					},
				},
				Overrides: addrs.MakeMap[addrs.Targetable, *Override](
					makeOverride(t, "test_resource.my_resource", cty.StringVal("current")),
				),
			},
			target: nil,
			result: &MockData{
				MockResources: map[string]*MockResource{
					"test_resource": {
						Mode:     addrs.ManagedResourceMode,
						Type:     "test_resource",
						Defaults: cty.StringVal("current"),
					},
				},
				MockDataSources: map[string]*MockResource{
					"test_data_source": {
						Mode:     addrs.ManagedResourceMode,
						Type:     "test_data_source",
						Defaults: cty.StringVal("current"),
					},
				},
				Overrides: addrs.MakeMap[addrs.Targetable, *Override](
					makeOverride(t, "test_resource.my_resource", cty.StringVal("current")),
				),
			},
		},
		"all_collisions": {
			current: &MockData{
				MockResources: map[string]*MockResource{
					"test_resource": {
						Mode:     addrs.ManagedResourceMode,
						Type:     "test_resource",
						Defaults: cty.StringVal("current"),
					},
				},
				MockDataSources: map[string]*MockResource{
					"test_data_source": {
						Mode:     addrs.ManagedResourceMode,
						Type:     "test_data_source",
						Defaults: cty.StringVal("current"),
					},
				},
				Overrides: addrs.MakeMap[addrs.Targetable, *Override](
					makeOverride(t, "test_resource.my_resource", cty.StringVal("current")),
				),
			},
			target: &MockData{
				MockResources: map[string]*MockResource{
					"test_resource": {
						Mode:     addrs.ManagedResourceMode,
						Type:     "test_resource",
						Defaults: cty.StringVal("target"),
					},
				},
				MockDataSources: map[string]*MockResource{
					"test_data_source": {
						Mode:     addrs.ManagedResourceMode,
						Type:     "test_data_source",
						Defaults: cty.StringVal("target"),
					},
				},
				Overrides: addrs.MakeMap[addrs.Targetable, *Override](
					makeOverride(t, "test_resource.my_resource", cty.StringVal("target")),
				),
			},
			result: &MockData{
				MockResources: map[string]*MockResource{
					"test_resource": {
						Mode:     addrs.ManagedResourceMode,
						Type:     "test_resource",
						Defaults: cty.StringVal("current"),
					},
				},
				MockDataSources: map[string]*MockResource{
					"test_data_source": {
						Mode:     addrs.ManagedResourceMode,
						Type:     "test_data_source",
						Defaults: cty.StringVal("current"),
					},
				},
				Overrides: addrs.MakeMap[addrs.Targetable, *Override](
					makeOverride(t, "test_resource.my_resource", cty.StringVal("current")),
				),
			},
			diags: []string{
				"A mock_resource \"test_resource\" block already exists at :0,0-0.",
				"A mock_data \"test_data_source\" block already exists at :0,0-0.",
				"An override block for test_resource.my_resource already exists at :0,0-0.",
			},
		},
		"no_collisions": {
			current: &MockData{
				MockResources: map[string]*MockResource{
					"test_resource": {
						Mode:     addrs.ManagedResourceMode,
						Type:     "test_resource",
						Defaults: cty.StringVal("current"),
					},
				},
				MockDataSources: map[string]*MockResource{
					"test_data_source": {
						Mode:     addrs.ManagedResourceMode,
						Type:     "test_data_source",
						Defaults: cty.StringVal("current"),
					},
				},
				Overrides: addrs.MakeMap[addrs.Targetable, *Override](
					makeOverride(t, "test_resource.my_resource", cty.StringVal("current")),
				),
			},
			target: &MockData{
				MockResources: map[string]*MockResource{
					"test_resource_two": {
						Mode:     addrs.ManagedResourceMode,
						Type:     "test_resource_two",
						Defaults: cty.StringVal("target"),
					},
				},
				MockDataSources: map[string]*MockResource{
					"test_data_source_two": {
						Mode:     addrs.ManagedResourceMode,
						Type:     "test_data_source_two",
						Defaults: cty.StringVal("target"),
					},
				},
				Overrides: addrs.MakeMap[addrs.Targetable, *Override](
					makeOverride(t, "test_resource.my_other_resource", cty.StringVal("target")),
				),
			},
			result: &MockData{
				MockResources: map[string]*MockResource{
					"test_resource": {
						Mode:     addrs.ManagedResourceMode,
						Type:     "test_resource",
						Defaults: cty.StringVal("current"),
					},
					"test_resource_two": {
						Mode:     addrs.ManagedResourceMode,
						Type:     "test_resource_two",
						Defaults: cty.StringVal("target"),
					},
				},
				MockDataSources: map[string]*MockResource{
					"test_data_source": {
						Mode:     addrs.ManagedResourceMode,
						Type:     "test_data_source",
						Defaults: cty.StringVal("current"),
					},
					"test_data_source_two": {
						Mode:     addrs.ManagedResourceMode,
						Type:     "test_data_source_two",
						Defaults: cty.StringVal("target"),
					},
				},
				Overrides: addrs.MakeMap[addrs.Targetable, *Override](
					makeOverride(t, "test_resource.my_resource", cty.StringVal("current")),
					makeOverride(t, "test_resource.my_other_resource", cty.StringVal("target")),
				),
			},
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			diags := tc.current.Merge(tc.target, false)
			validateMockData(t, tc.current, tc.result)

			var details []string
			for _, diag := range diags {
				details = append(details, diag.Detail)
			}
			if diff := cmp.Diff(tc.diags, details); len(diff) > 0 {
				t.Error(diff)
			}

		})
	}
}

func validateMockData(t *testing.T, actual, expected *MockData) {

	// Validate mock resources.

	for key, actual := range actual.MockResources {
		expected, exists := expected.MockResources[key]
		if !exists {
			t.Errorf("actual mock resources contained %s but expected mock resources did not", key)
			continue
		}

		validateValues(t, key, actual.Defaults, expected.Defaults)
	}

	for key := range expected.MockResources {
		_, exists := actual.MockResources[key]
		if !exists {
			t.Errorf("expected mock resources contained %s but actual mock resources did not", key)
		}
	}

	// Validate mock data sources.

	for key, actual := range actual.MockDataSources {
		expected, exists := expected.MockDataSources[key]
		if !exists {
			t.Errorf("actual mock data sources contained %s but expected mock data sources did not", key)
			continue
		}

		validateValues(t, key, actual.Defaults, expected.Defaults)
	}

	for key := range expected.MockDataSources {
		_, exists := actual.MockDataSources[key]
		if !exists {
			t.Errorf("expected mock data sources contained %s but actual mock data sources did not", key)
		}
	}

	// Validate the overrides.

	for _, elem := range actual.Overrides.Elems {
		key, actual := elem.Key, elem.Value

		expected, exists := expected.Overrides.GetOk(key)
		if !exists {
			t.Errorf("actual overrides contained %s but expected overrides did not", key)
			continue
		}

		validateValues(t, key.String(), actual.Values, expected.Values)
	}

	for _, elem := range expected.Overrides.Elems {
		key := elem.Key

		if actual.Overrides.Has(key) {
			continue
		}

		t.Errorf("expected overrides contained %s but actual overrides did not", key)
	}
}

func validateValues(t *testing.T, key string, actual, expected cty.Value) {
	if !actual.RawEquals(expected) {
		t.Errorf("for %s\n\tactual: %s\n\texpected: %s", key, actual, expected)
	}
}

func makeOverride(t *testing.T, target string, values cty.Value) addrs.MapElem[addrs.Targetable, *Override] {
	addr, diags := addrs.ParseTargetStr(target)
	if diags.HasErrors() {
		t.Fatalf("failed to parse target: %s", diags)
	}

	return addrs.MapElem[addrs.Targetable, *Override]{
		Key: addr.Subject,
		Value: &Override{
			Target: addr,
			Values: values,
		},
	}
}
