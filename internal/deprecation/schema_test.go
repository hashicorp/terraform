// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package deprecation

import (
	"testing"

	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/zclconf/go-cty/cty"
)

func TestMarkDeprecatedValues_NilSchema(t *testing.T) {
	val := cty.ObjectVal(map[string]cty.Value{
		"foo": cty.StringVal("bar"),
	})

	result := MarkDeprecatedValues(val, nil, "origin")

	if !result.RawEquals(val) {
		t.Errorf("expected value to be unchanged when schema is nil")
	}
}

func TestMarkDeprecatedValues_NoDeprecations(t *testing.T) {
	schema := &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"foo": {
				Type:       cty.String,
				Optional:   true,
				Deprecated: false,
			},
			"bar": {
				Type:       cty.Number,
				Optional:   true,
				Deprecated: false,
			},
		},
	}

	val := cty.ObjectVal(map[string]cty.Value{
		"foo": cty.StringVal("hello"),
		"bar": cty.NumberIntVal(42),
	})

	result := MarkDeprecatedValues(val, schema, "origin")

	if result.IsMarked() {
		t.Errorf("expected value to not be marked when nothing is deprecated")
	}

	unmarkedResult, pathMarks := result.UnmarkDeepWithPaths()
	if len(pathMarks) > 0 {
		t.Errorf("expected no marks, got %d marks", len(pathMarks))
	}

	if !unmarkedResult.RawEquals(val) {
		t.Errorf("expected unmarked value to equal original value")
	}
}

func TestMarkDeprecatedValues_DeprecatedBlock(t *testing.T) {
	schema := &configschema.Block{
		Deprecated: true,
		Attributes: map[string]*configschema.Attribute{
			"foo": {
				Type:     cty.String,
				Optional: true,
			},
		},
	}

	val := cty.ObjectVal(map[string]cty.Value{
		"foo": cty.StringVal("bar"),
	})

	result := MarkDeprecatedValues(val, schema, "origin")

	if !result.IsMarked() {
		t.Fatalf("expected result to be marked")
	}

	unmarkedResult, pathMarks := result.UnmarkDeepWithPaths()
	deprecatedPaths, _ := marks.PathsWithMark(pathMarks, marks.Deprecation)

	if len(deprecatedPaths) == 0 {
		t.Fatalf("expected at least one deprecated path")
	}

	// The root value itself should be marked as deprecated
	foundRootDeprecation := false
	for _, pvm := range pathMarks {
		if len(pvm.Path) == 0 {
			for mark := range pvm.Marks {
				if _, ok := mark.(marks.DeprecationMark); ok {
					foundRootDeprecation = true
					break
				}
			}
		}
	}

	if !foundRootDeprecation {
		t.Errorf("expected root value to be marked as deprecated")
	}

	if !unmarkedResult.RawEquals(val) {
		t.Errorf("expected unmarked value to equal original value")
	}
}

func TestMarkDeprecatedValues_DeprecatedAttribute(t *testing.T) {
	schema := &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"deprecated_attr": {
				Type:       cty.String,
				Optional:   true,
				Deprecated: true,
			},
			"normal_attr": {
				Type:     cty.String,
				Optional: true,
			},
		},
	}

	val := cty.ObjectVal(map[string]cty.Value{
		"deprecated_attr": cty.StringVal("old"),
		"normal_attr":     cty.StringVal("new"),
	})

	result := MarkDeprecatedValues(val, schema, "origin")

	unmarkedResult, pathMarks := result.UnmarkDeepWithPaths()
	deprecatedPaths, _ := marks.PathsWithMark(pathMarks, marks.Deprecation)

	if len(deprecatedPaths) != 1 {
		t.Fatalf("expected exactly 1 deprecated path, got %d", len(deprecatedPaths))
	}

	expectedPath := cty.GetAttrPath("deprecated_attr")
	if !deprecatedPaths[0].Equals(expectedPath) {
		t.Errorf("expected deprecated path to be %#v, got %#v", expectedPath, deprecatedPaths[0])
	}

	if !unmarkedResult.RawEquals(val) {
		t.Errorf("expected unmarked value to equal original value")
	}
}

func TestMarkDeprecatedValues_MultipleDeprecatedAttributes(t *testing.T) {
	schema := &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"deprecated_one": {
				Type:       cty.String,
				Optional:   true,
				Deprecated: true,
			},
			"deprecated_two": {
				Type:       cty.Number,
				Optional:   true,
				Deprecated: true,
			},
			"normal_attr": {
				Type:     cty.Bool,
				Optional: true,
			},
		},
	}

	val := cty.ObjectVal(map[string]cty.Value{
		"deprecated_one": cty.StringVal("old1"),
		"deprecated_two": cty.NumberIntVal(123),
		"normal_attr":    cty.BoolVal(true),
	})

	result := MarkDeprecatedValues(val, schema, "origin")

	unmarkedResult, pathMarks := result.UnmarkDeepWithPaths()
	deprecatedPaths, _ := marks.PathsWithMark(pathMarks, marks.Deprecation)

	if len(deprecatedPaths) != 2 {
		t.Fatalf("expected exactly 2 deprecated paths, got %d", len(deprecatedPaths))
	}

	pathSet := make(map[string]bool)
	for _, p := range deprecatedPaths {
		if len(p) == 1 {
			if getAttr, ok := p[0].(cty.GetAttrStep); ok {
				pathSet[getAttr.Name] = true
			}
		}
	}

	if !pathSet["deprecated_one"] || !pathSet["deprecated_two"] {
		t.Errorf("expected both deprecated_one and deprecated_two to be marked as deprecated")
	}

	if !unmarkedResult.RawEquals(val) {
		t.Errorf("expected unmarked value to equal original value")
	}
}

func TestMarkDeprecatedValues_NestedBlock(t *testing.T) {
	schema := &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"name": {
				Type:     cty.String,
				Optional: true,
			},
		},
		BlockTypes: map[string]*configschema.NestedBlock{
			"nested": {
				Nesting: configschema.NestingList,
				Block: configschema.Block{
					Deprecated: true,
					Attributes: map[string]*configschema.Attribute{
						"value": {
							Type:     cty.String,
							Optional: true,
						},
					},
				},
			},
		},
	}

	val := cty.ObjectVal(map[string]cty.Value{
		"name": cty.StringVal("test"),
		"nested": cty.ListVal([]cty.Value{
			cty.ObjectVal(map[string]cty.Value{
				"value": cty.StringVal("item1"),
			}),
		}),
	})

	result := MarkDeprecatedValues(val, schema, "origin")

	unmarkedResult, pathMarks := result.UnmarkDeepWithPaths()
	deprecatedPaths, _ := marks.PathsWithMark(pathMarks, marks.Deprecation)

	if len(deprecatedPaths) == 0 {
		t.Fatalf("expected at least one deprecated path for nested block")
	}

	if !unmarkedResult.RawEquals(val) {
		t.Errorf("expected unmarked value to equal original value")
	}
}

func TestMarkDeprecatedValues_NestedDeprecatedAttribute(t *testing.T) {
	schema := &configschema.Block{
		BlockTypes: map[string]*configschema.NestedBlock{
			"config": {
				Nesting: configschema.NestingList,
				Block: configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"deprecated_field": {
							Type:       cty.String,
							Optional:   true,
							Deprecated: true,
						},
						"normal_field": {
							Type:     cty.String,
							Optional: true,
						},
					},
				},
			},
		},
	}

	val := cty.ObjectVal(map[string]cty.Value{
		"config": cty.ListVal([]cty.Value{
			cty.ObjectVal(map[string]cty.Value{
				"deprecated_field": cty.StringVal("old"),
				"normal_field":     cty.StringVal("new"),
			}),
		}),
	})

	result := MarkDeprecatedValues(val, schema, "origin")

	unmarkedResult, pathMarks := result.UnmarkDeepWithPaths()
	deprecatedPaths, _ := marks.PathsWithMark(pathMarks, marks.Deprecation)

	if len(deprecatedPaths) == 0 {
		t.Fatalf("expected at least one deprecated path")
	}

	// Check that the deprecated field within the nested block is marked
	foundDeprecatedField := false
	for _, pvm := range pathMarks {
		for i, step := range pvm.Path {
			if getAttr, ok := step.(cty.GetAttrStep); ok && getAttr.Name == "deprecated_field" {
				// Check if it's inside the config list
				if i > 0 {
					foundDeprecatedField = true
					break
				}
			}
		}
	}

	if !foundDeprecatedField {
		t.Errorf("expected nested deprecated_field to be marked")
	}

	if !unmarkedResult.RawEquals(val) {
		t.Errorf("expected unmarked value to equal original value")
	}
}

func TestMarkDeprecatedValues_NonObjectTypes(t *testing.T) {
	tests := []struct {
		name   string
		schema *configschema.Block
		val    cty.Value
	}{
		{
			name: "string value",
			schema: &configschema.Block{
				Deprecated: false,
			},
			val: cty.StringVal("test"),
		},
		{
			name: "number value",
			schema: &configschema.Block{
				Deprecated: false,
			},
			val: cty.NumberIntVal(42),
		},
		{
			name: "bool value",
			schema: &configschema.Block{
				Deprecated: false,
			},
			val: cty.BoolVal(true),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MarkDeprecatedValues(tt.val, tt.schema, "origin")

			// For non-object types, the function should handle gracefully
			// and not crash
			if result.IsNull() {
				t.Errorf("result should not be null")
			}
		})
	}
}

func TestMarkDeprecatedValues_DeprecatedBlockAndAttribute(t *testing.T) {
	schema := &configschema.Block{
		Deprecated: true,
		Attributes: map[string]*configschema.Attribute{
			"deprecated_attr": {
				Type:       cty.String,
				Optional:   true,
				Deprecated: true,
			},
			"normal_attr": {
				Type:     cty.String,
				Optional: true,
			},
		},
	}

	val := cty.ObjectVal(map[string]cty.Value{
		"deprecated_attr": cty.StringVal("old"),
		"normal_attr":     cty.StringVal("new"),
	})

	result := MarkDeprecatedValues(val, schema, "origin")

	unmarkedResult, pathMarks := result.UnmarkDeepWithPaths()
	deprecatedPaths, _ := marks.PathsWithMark(pathMarks, marks.Deprecation)

	// Should have both the block itself and the deprecated attribute marked
	if len(deprecatedPaths) < 1 {
		t.Fatalf("expected at least 1 deprecated path, got %d", len(deprecatedPaths))
	}

	if !unmarkedResult.RawEquals(val) {
		t.Errorf("expected unmarked value to equal original value")
	}
}

func TestMarkDeprecatedValues_EmptyObject(t *testing.T) {
	schema := &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"foo": {
				Type:       cty.String,
				Optional:   true,
				Deprecated: true,
			},
		},
	}

	val := cty.EmptyObjectVal

	result := MarkDeprecatedValues(val, schema, "origin")

	// Should not crash on empty object
	if result.IsNull() {
		t.Errorf("result should not be null for empty object")
	}
}

func TestMarkDeprecatedValues_NullValue(t *testing.T) {
	schema := &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"foo": {
				Type:       cty.String,
				Optional:   true,
				Deprecated: true,
			},
		},
	}

	val := cty.NullVal(cty.Object(map[string]cty.Type{
		"foo": cty.String,
	}))

	result := MarkDeprecatedValues(val, schema, "origin")

	// Should handle null values gracefully
	if !result.IsNull() {
		t.Errorf("null input should remain null")
	}
}

func TestMarkDeprecatedValues_MapType(t *testing.T) {
	schema := &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"tags": {
				Type:       cty.Map(cty.String),
				Optional:   true,
				Deprecated: false,
			},
		},
	}

	val := cty.ObjectVal(map[string]cty.Value{
		"tags": cty.MapVal(map[string]cty.Value{
			"env":  cty.StringVal("prod"),
			"team": cty.StringVal("platform"),
		}),
	})

	result := MarkDeprecatedValues(val, schema, "origin")

	// Should handle map types without crashing
	unmarkedResult, _ := result.UnmarkDeepWithPaths()
	if !unmarkedResult.RawEquals(val) {
		t.Errorf("expected unmarked value to equal original value")
	}
}

func TestMarkDeprecatedValues_ListType(t *testing.T) {
	schema := &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"items": {
				Type:       cty.List(cty.String),
				Optional:   true,
				Deprecated: true,
			},
		},
	}

	val := cty.ObjectVal(map[string]cty.Value{
		"items": cty.ListVal([]cty.Value{
			cty.StringVal("one"),
			cty.StringVal("two"),
		}),
	})

	result := MarkDeprecatedValues(val, schema, "origin")

	unmarkedResult, pathMarks := result.UnmarkDeepWithPaths()
	deprecatedPaths, _ := marks.PathsWithMark(pathMarks, marks.Deprecation)

	if len(deprecatedPaths) == 0 {
		t.Fatalf("expected deprecated list attribute to be marked")
	}

	if !unmarkedResult.RawEquals(val) {
		t.Errorf("expected unmarked value to equal original value")
	}
}

func TestMarkDeprecatedValues_NestedType_NestingSingle(t *testing.T) {
	schema := &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"id": {
				Type:     cty.String,
				Optional: true,
			},
			"config": {
				NestedType: &configschema.Object{
					Nesting: configschema.NestingSingle,
					Attributes: map[string]*configschema.Attribute{
						"deprecated_field": {
							Type:       cty.String,
							Optional:   true,
							Deprecated: true,
						},
						"normal_field": {
							Type:     cty.String,
							Optional: true,
						},
					},
				},
			},
		},
	}

	val := cty.ObjectVal(map[string]cty.Value{
		"id": cty.StringVal("test-id"),
		"config": cty.ObjectVal(map[string]cty.Value{
			"deprecated_field": cty.StringVal("old"),
			"normal_field":     cty.StringVal("new"),
		}),
	})

	result := MarkDeprecatedValues(val, schema, "origin")

	unmarkedResult, pathMarks := result.UnmarkDeepWithPaths()
	deprecatedPaths, _ := marks.PathsWithMark(pathMarks, marks.Deprecation)

	if len(deprecatedPaths) == 0 {
		t.Fatalf("expected at least one deprecated path")
	}

	// Check that the deprecated field within the nested type is marked
	foundDeprecatedField := false
	for _, path := range deprecatedPaths {
		if len(path) >= 2 {
			if getAttr, ok := path[0].(cty.GetAttrStep); ok && getAttr.Name == "config" {
				if getAttr2, ok := path[1].(cty.GetAttrStep); ok && getAttr2.Name == "deprecated_field" {
					foundDeprecatedField = true
					break
				}
			}
		}
	}

	if !foundDeprecatedField {
		t.Errorf("expected config.deprecated_field to be marked as deprecated")
	}

	if !unmarkedResult.RawEquals(val) {
		t.Errorf("expected unmarked value to equal original value")
	}
}

func TestMarkDeprecatedValues_NestedType_NestingList(t *testing.T) {
	schema := &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"id": {
				Type:     cty.String,
				Optional: true,
			},
			"disks": {
				NestedType: &configschema.Object{
					Nesting: configschema.NestingList,
					Attributes: map[string]*configschema.Attribute{
						"mount_point": {
							Type:       cty.String,
							Optional:   true,
							Deprecated: true,
						},
						"size": {
							Type:     cty.String,
							Optional: true,
						},
					},
				},
			},
		},
	}

	val := cty.ObjectVal(map[string]cty.Value{
		"id": cty.StringVal("test-id"),
		"disks": cty.ListVal([]cty.Value{
			cty.ObjectVal(map[string]cty.Value{
				"mount_point": cty.StringVal("/mnt/data"),
				"size":        cty.StringVal("100GB"),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"mount_point": cty.StringVal("/mnt/backup"),
				"size":        cty.StringVal("200GB"),
			}),
		}),
	})

	result := MarkDeprecatedValues(val, schema, "origin")

	unmarkedResult, pathMarks := result.UnmarkDeepWithPaths()
	deprecatedPaths, _ := marks.PathsWithMark(pathMarks, marks.Deprecation)

	if len(deprecatedPaths) == 0 {
		t.Fatalf("expected at least one deprecated path")
	}

	// Should have deprecated marks for mount_point in both list items
	mountPointCount := 0
	for _, path := range deprecatedPaths {
		for _, step := range path {
			if getAttr, ok := step.(cty.GetAttrStep); ok && getAttr.Name == "mount_point" {
				mountPointCount++
				break
			}
		}
	}

	if mountPointCount != 2 {
		t.Errorf("expected 2 deprecated mount_point fields, got %d", mountPointCount)
	}

	if !unmarkedResult.RawEquals(val) {
		t.Errorf("expected unmarked value to equal original value")
	}
}

func TestMarkDeprecatedValues_NestedType_NestingSet(t *testing.T) {
	// Note: The current implementation of AttributeByPath only handles GetAttrStep,
	// not IndexStep, so nested attributes within set elements cannot be individually
	// marked. This test verifies that the entire set attribute can still be marked
	// if it is deprecated.
	schema := &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"tags": {
				Deprecated: true, // Mark the entire attribute as deprecated
				NestedType: &configschema.Object{
					Nesting: configschema.NestingSet,
					Attributes: map[string]*configschema.Attribute{
						"key": {
							Type:     cty.String,
							Optional: true,
						},
						"value": {
							Type:     cty.String,
							Optional: true,
						},
					},
				},
			},
		},
	}

	val := cty.ObjectVal(map[string]cty.Value{
		"tags": cty.SetVal([]cty.Value{
			cty.ObjectVal(map[string]cty.Value{
				"key":   cty.StringVal("env"),
				"value": cty.StringVal("prod"),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"key":   cty.StringVal("team"),
				"value": cty.StringVal("platform"),
			}),
		}),
	})

	result := MarkDeprecatedValues(val, schema, "origin")

	unmarkedResult, pathMarks := result.UnmarkDeepWithPaths()
	deprecatedPaths, _ := marks.PathsWithMark(pathMarks, marks.Deprecation)

	if len(deprecatedPaths) == 0 {
		t.Fatalf("expected at least one deprecated path")
	}

	// The entire tags attribute should be marked as deprecated
	foundDeprecatedTags := false
	for _, path := range deprecatedPaths {
		if len(path) == 1 {
			if getAttr, ok := path[0].(cty.GetAttrStep); ok && getAttr.Name == "tags" {
				foundDeprecatedTags = true
				break
			}
		}
	}

	if !foundDeprecatedTags {
		t.Errorf("expected tags attribute to be marked as deprecated")
	}

	if !unmarkedResult.RawEquals(val) {
		t.Errorf("expected unmarked value to equal original value")
	}
}

func TestMarkDeprecatedValues_NestedType_NestingMap(t *testing.T) {
	schema := &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"metadata": {
				NestedType: &configschema.Object{
					Nesting: configschema.NestingMap,
					Attributes: map[string]*configschema.Attribute{
						"description": {
							Type:     cty.String,
							Optional: true,
						},
						"deprecated_label": {
							Type:       cty.String,
							Optional:   true,
							Deprecated: true,
						},
					},
				},
			},
		},
	}

	val := cty.ObjectVal(map[string]cty.Value{
		"metadata": cty.MapVal(map[string]cty.Value{
			"primary": cty.ObjectVal(map[string]cty.Value{
				"description":      cty.StringVal("Primary config"),
				"deprecated_label": cty.StringVal("old_label"),
			}),
			"secondary": cty.ObjectVal(map[string]cty.Value{
				"description":      cty.StringVal("Secondary config"),
				"deprecated_label": cty.StringVal("old_label_2"),
			}),
		}),
	})

	result := MarkDeprecatedValues(val, schema, "origin")

	unmarkedResult, pathMarks := result.UnmarkDeepWithPaths()
	deprecatedPaths, _ := marks.PathsWithMark(pathMarks, marks.Deprecation)

	if len(deprecatedPaths) == 0 {
		t.Fatalf("expected at least one deprecated path")
	}

	// Check that deprecated_label fields are marked
	deprecatedLabelCount := 0
	for _, path := range deprecatedPaths {
		for _, step := range path {
			if getAttr, ok := step.(cty.GetAttrStep); ok && getAttr.Name == "deprecated_label" {
				deprecatedLabelCount++
				break
			}
		}
	}

	if deprecatedLabelCount != 2 {
		t.Errorf("expected 2 deprecated_label fields, got %d", deprecatedLabelCount)
	}

	if !unmarkedResult.RawEquals(val) {
		t.Errorf("expected unmarked value to equal original value")
	}
}

func TestMarkDeprecatedValues_NestedType_DeprecatedNestedAttribute(t *testing.T) {
	schema := &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"deprecated_config": {
				Deprecated: true,
				NestedType: &configschema.Object{
					Nesting: configschema.NestingSingle,
					Attributes: map[string]*configschema.Attribute{
						"field": {
							Type:     cty.String,
							Optional: true,
						},
					},
				},
			},
		},
	}

	val := cty.ObjectVal(map[string]cty.Value{
		"deprecated_config": cty.ObjectVal(map[string]cty.Value{
			"field": cty.StringVal("value"),
		}),
	})

	result := MarkDeprecatedValues(val, schema, "origin")

	unmarkedResult, pathMarks := result.UnmarkDeepWithPaths()
	deprecatedPaths, _ := marks.PathsWithMark(pathMarks, marks.Deprecation)

	if len(deprecatedPaths) != 1 {
		t.Fatalf("expected at least one deprecated path")
	}

	// The entire deprecated_config attribute should be marked
	foundDeprecatedConfig := false
	for _, path := range deprecatedPaths {
		if len(path) == 1 {
			if getAttr, ok := path[0].(cty.GetAttrStep); ok && getAttr.Name == "deprecated_config" {
				foundDeprecatedConfig = true
				break
			}
		}
	}

	if !foundDeprecatedConfig {
		t.Errorf("expected deprecated_config to be marked as deprecated")
	}

	if !unmarkedResult.RawEquals(val) {
		t.Errorf("expected unmarked value to equal original value")
	}
}

func TestMarkDeprecatedValues_NestedType_MultipleDeprecatedFields(t *testing.T) {
	schema := &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"connection": {
				NestedType: &configschema.Object{
					Nesting: configschema.NestingSingle,
					Attributes: map[string]*configschema.Attribute{
						"deprecated_host": {
							Type:       cty.String,
							Optional:   true,
							Deprecated: true,
						},
						"deprecated_port": {
							Type:       cty.Number,
							Optional:   true,
							Deprecated: true,
						},
						"username": {
							Type:     cty.String,
							Optional: true,
						},
					},
				},
			},
		},
	}

	val := cty.ObjectVal(map[string]cty.Value{
		"connection": cty.ObjectVal(map[string]cty.Value{
			"deprecated_host": cty.StringVal("example.com"),
			"deprecated_port": cty.NumberIntVal(8080),
			"username":        cty.StringVal("admin"),
		}),
	})

	result := MarkDeprecatedValues(val, schema, "origin")

	unmarkedResult, pathMarks := result.UnmarkDeepWithPaths()
	deprecatedPaths, _ := marks.PathsWithMark(pathMarks, marks.Deprecation)

	if len(deprecatedPaths) != 2 {
		t.Fatalf("expected exactly 2 deprecated paths, got %d", len(deprecatedPaths))
	}

	// Check that both deprecated_host and deprecated_port are marked
	pathSet := make(map[string]bool)
	for _, path := range deprecatedPaths {
		if len(path) == 2 {
			if getAttr, ok := path[1].(cty.GetAttrStep); ok {
				pathSet[getAttr.Name] = true
			}
		}
	}

	if !pathSet["deprecated_host"] || !pathSet["deprecated_port"] {
		t.Errorf("expected both deprecated_host and deprecated_port to be marked as deprecated")
	}

	if !unmarkedResult.RawEquals(val) {
		t.Errorf("expected unmarked value to equal original value")
	}
}

func TestMarkDeprecatedValues_NestedType_EmptyList(t *testing.T) {
	schema := &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"items": {
				NestedType: &configschema.Object{
					Nesting: configschema.NestingList,
					Attributes: map[string]*configschema.Attribute{
						"deprecated_field": {
							Type:       cty.String,
							Optional:   true,
							Deprecated: true,
						},
					},
				},
			},
		},
	}

	val := cty.ObjectVal(map[string]cty.Value{
		"items": cty.ListValEmpty(cty.Object(map[string]cty.Type{
			"deprecated_field": cty.String,
		})),
	})

	result := MarkDeprecatedValues(val, schema, "origin")

	// Should handle empty lists without crashing
	if result.IsNull() {
		t.Errorf("result should not be null for empty list")
	}

	unmarkedResult, _ := result.UnmarkDeepWithPaths()
	if !unmarkedResult.RawEquals(val) {
		t.Errorf("expected unmarked value to equal original value")
	}
}

func TestMarkDeprecatedValues_NestedType_NullNestedValue(t *testing.T) {
	schema := &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"config": {
				NestedType: &configschema.Object{
					Nesting: configschema.NestingSingle,
					Attributes: map[string]*configschema.Attribute{
						"deprecated_field": {
							Type:       cty.String,
							Optional:   true,
							Deprecated: true,
						},
					},
				},
			},
		},
	}

	val := cty.ObjectVal(map[string]cty.Value{
		"config": cty.NullVal(cty.Object(map[string]cty.Type{
			"deprecated_field": cty.String,
		})),
	})

	result := MarkDeprecatedValues(val, schema, "origin")

	// Should handle null nested values gracefully
	if result.IsNull() {
		t.Errorf("result should not be null")
	}

	unmarkedResult, _ := result.UnmarkDeepWithPaths()
	if !unmarkedResult.RawEquals(val) {
		t.Errorf("expected unmarked value to equal original value")
	}
}

func TestMarkDeprecatedValues_NestedType_MixedWithBlockTypes(t *testing.T) {
	schema := &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"nested_attr": {
				NestedType: &configschema.Object{
					Nesting: configschema.NestingSingle,
					Attributes: map[string]*configschema.Attribute{
						"deprecated_field": {
							Type:       cty.String,
							Optional:   true,
							Deprecated: true,
						},
					},
				},
			},
		},
		BlockTypes: map[string]*configschema.NestedBlock{
			"nested_block": {
				Nesting: configschema.NestingSingle,
				Block: configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"deprecated_block_attr": {
							Type:       cty.String,
							Optional:   true,
							Deprecated: true,
						},
					},
				},
			},
		},
	}

	val := cty.ObjectVal(map[string]cty.Value{
		"nested_attr": cty.ObjectVal(map[string]cty.Value{
			"deprecated_field": cty.StringVal("attr_value"),
		}),
		"nested_block": cty.ObjectVal(map[string]cty.Value{
			"deprecated_block_attr": cty.StringVal("block_value"),
		}),
	})

	result := MarkDeprecatedValues(val, schema, "origin")

	unmarkedResult, pathMarks := result.UnmarkDeepWithPaths()
	deprecatedPaths, _ := marks.PathsWithMark(pathMarks, marks.Deprecation)

	if len(deprecatedPaths) != 2 {
		t.Fatalf("expected exactly 2 deprecated paths (one from NestedType, one from BlockType), got %d", len(deprecatedPaths))
	}

	// Check that both deprecated fields are marked
	pathSet := make(map[string]bool)
	for _, path := range deprecatedPaths {
		if len(path) >= 2 {
			if getAttr, ok := path[1].(cty.GetAttrStep); ok {
				pathSet[getAttr.Name] = true
			}
		}
	}

	if !pathSet["deprecated_field"] || !pathSet["deprecated_block_attr"] {
		t.Errorf("expected both deprecated_field and deprecated_block_attr to be marked")
	}

	if !unmarkedResult.RawEquals(val) {
		t.Errorf("expected unmarked value to equal original value")
	}
}
