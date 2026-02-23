// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package mocking

import (
	"math/rand"
	"testing"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/configs/configschema"
)

// TestFillAttribute tests the FillAttribute function which handles
// configschema.Attribute values including NestedType attributes with various
// nesting modes (NestingSingle, NestingGroup, NestingList, NestingSet, NestingMap).
func TestFillAttribute(t *testing.T) {
	tcs := map[string]struct {
		in        cty.Value
		attribute *configschema.Attribute
		expected  cty.Value
	}{
		// =====================================================================
		// Plain attributes (no NestedType) - falls through to fillType
		// =====================================================================

		"plain_string_attribute": {
			in: cty.StringVal("hello"),
			attribute: &configschema.Attribute{
				Type: cty.String,
			},
			expected: cty.StringVal("hello"),
		},

		"plain_number_to_string_conversion": {
			in: cty.NumberIntVal(42),
			attribute: &configschema.Attribute{
				Type: cty.String,
			},
			expected: cty.StringVal("42"),
		},

		"plain_list_of_objects_attribute": {
			// When the attribute uses Type (not NestedType), it goes through
			// fillType directly. This is the "plain" list-of-objects case
			// that already worked before the fix.
			in: cty.TupleVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"name": cty.StringVal("first"),
				}),
			}),
			attribute: &configschema.Attribute{
				Type: cty.List(cty.Object(map[string]cty.Type{
					"name":  cty.String,
					"value": cty.String,
				})),
			},
			expected: cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"name":  cty.StringVal("first"),
					"value": cty.StringVal("ssnk9qhr"),
				}),
			}),
		},

		// =====================================================================
		// NestingSingle
		// =====================================================================

		"nesting_single_partial_attrs": {
			in: cty.ObjectVal(map[string]cty.Value{
				"name": cty.StringVal("hello"),
			}),
			attribute: &configschema.Attribute{
				NestedType: &configschema.Object{
					Nesting: configschema.NestingSingle,
					Attributes: map[string]*configschema.Attribute{
						"name":  {Type: cty.String},
						"value": {Type: cty.String},
					},
				},
			},
			expected: cty.ObjectVal(map[string]cty.Value{
				"name":  cty.StringVal("hello"),
				"value": cty.StringVal("ssnk9qhr"),
			}),
		},

		"nesting_single_all_attrs_present": {
			in: cty.ObjectVal(map[string]cty.Value{
				"name":  cty.StringVal("hello"),
				"value": cty.StringVal("world"),
			}),
			attribute: &configschema.Attribute{
				NestedType: &configschema.Object{
					Nesting: configschema.NestingSingle,
					Attributes: map[string]*configschema.Attribute{
						"name":  {Type: cty.String},
						"value": {Type: cty.String},
					},
				},
			},
			expected: cty.ObjectVal(map[string]cty.Value{
				"name":  cty.StringVal("hello"),
				"value": cty.StringVal("world"),
			}),
		},

		"nesting_single_empty_schema": {
			in: cty.EmptyObjectVal,
			attribute: &configschema.Attribute{
				NestedType: &configschema.Object{
					Nesting:    configschema.NestingSingle,
					Attributes: map[string]*configschema.Attribute{},
				},
			},
			expected: cty.EmptyObjectVal,
		},

		"nesting_single_extra_attrs_in_input_dropped": {
			// Input has attributes not in the schema; they should be dropped.
			in: cty.ObjectVal(map[string]cty.Value{
				"name":  cty.StringVal("hello"),
				"extra": cty.StringVal("should be dropped"),
			}),
			attribute: &configschema.Attribute{
				NestedType: &configschema.Object{
					Nesting: configschema.NestingSingle,
					Attributes: map[string]*configschema.Attribute{
						"name": {Type: cty.String},
					},
				},
			},
			expected: cty.ObjectVal(map[string]cty.Value{
				"name": cty.StringVal("hello"),
			}),
		},

		// =====================================================================
		// NestingGroup (behaves same as NestingSingle for fill purposes)
		// =====================================================================

		"nesting_group_partial_attrs": {
			in: cty.ObjectVal(map[string]cty.Value{
				"name": cty.StringVal("hello"),
			}),
			attribute: &configschema.Attribute{
				NestedType: &configschema.Object{
					Nesting: configschema.NestingGroup,
					Attributes: map[string]*configschema.Attribute{
						"name":  {Type: cty.String},
						"value": {Type: cty.String},
					},
				},
			},
			expected: cty.ObjectVal(map[string]cty.Value{
				"name":  cty.StringVal("hello"),
				"value": cty.StringVal("ssnk9qhr"),
			}),
		},

		// =====================================================================
		// NestingList - the main bug fix (GitHub issue #37939)
		// =====================================================================

		"nesting_list_tuple_input_partial_attrs": {
			// This is the core bug scenario: HCL produces a tuple when parsing
			// list literals in override_data values. The old code would fail
			// or return empty for this input.
			in: cty.TupleVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"name": cty.StringVal("first"),
				}),
			}),
			attribute: &configschema.Attribute{
				NestedType: &configschema.Object{
					Nesting: configschema.NestingList,
					Attributes: map[string]*configschema.Attribute{
						"name":  {Type: cty.String},
						"value": {Type: cty.String},
					},
				},
			},
			expected: cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"name":  cty.StringVal("first"),
					"value": cty.StringVal("ssnk9qhr"),
				}),
			}),
		},

		"nesting_list_list_input_partial_attrs": {
			// Input is already a proper list (not a tuple).
			in: cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"name": cty.StringVal("first"),
				}),
			}),
			attribute: &configschema.Attribute{
				NestedType: &configschema.Object{
					Nesting: configschema.NestingList,
					Attributes: map[string]*configschema.Attribute{
						"name":  {Type: cty.String},
						"value": {Type: cty.String},
					},
				},
			},
			expected: cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"name":  cty.StringVal("first"),
					"value": cty.StringVal("ssnk9qhr"),
				}),
			}),
		},

		"nesting_list_multiple_elements": {
			in: cty.TupleVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"name": cty.StringVal("first"),
				}),
				cty.ObjectVal(map[string]cty.Value{
					"name": cty.StringVal("second"),
				}),
			}),
			attribute: &configschema.Attribute{
				NestedType: &configschema.Object{
					Nesting: configschema.NestingList,
					Attributes: map[string]*configschema.Attribute{
						"name":  {Type: cty.String},
						"value": {Type: cty.String},
					},
				},
			},
			expected: cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"name":  cty.StringVal("first"),
					"value": cty.StringVal("ssnk9qhr"),
				}),
				cty.ObjectVal(map[string]cty.Value{
					"name":  cty.StringVal("second"),
					"value": cty.StringVal("amyllmyg"),
				}),
			}),
		},

		"nesting_list_all_attrs_present": {
			in: cty.TupleVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"name":  cty.StringVal("first"),
					"value": cty.StringVal("one"),
				}),
			}),
			attribute: &configschema.Attribute{
				NestedType: &configschema.Object{
					Nesting: configschema.NestingList,
					Attributes: map[string]*configschema.Attribute{
						"name":  {Type: cty.String},
						"value": {Type: cty.String},
					},
				},
			},
			expected: cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"name":  cty.StringVal("first"),
					"value": cty.StringVal("one"),
				}),
			}),
		},

		"nesting_list_extra_attrs_in_elements_dropped": {
			// Input objects have attributes not defined in the schema.
			in: cty.TupleVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"name":  cty.StringVal("first"),
					"value": cty.StringVal("one"),
					"extra": cty.StringVal("should be dropped"),
				}),
			}),
			attribute: &configschema.Attribute{
				NestedType: &configschema.Object{
					Nesting: configschema.NestingList,
					Attributes: map[string]*configschema.Attribute{
						"name":  {Type: cty.String},
						"value": {Type: cty.String},
					},
				},
			},
			expected: cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"name":  cty.StringVal("first"),
					"value": cty.StringVal("one"),
				}),
			}),
		},

		"nesting_list_set_input": {
			// A set can also be converted to a list.
			in: cty.SetVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"name":  cty.StringVal("first"),
					"value": cty.StringVal("one"),
				}),
			}),
			attribute: &configschema.Attribute{
				NestedType: &configschema.Object{
					Nesting: configschema.NestingList,
					Attributes: map[string]*configschema.Attribute{
						"name":  {Type: cty.String},
						"value": {Type: cty.String},
					},
				},
			},
			expected: cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"name":  cty.StringVal("first"),
					"value": cty.StringVal("one"),
				}),
			}),
		},

		"nesting_list_element_type_conversion": {
			// Elements have attributes that need type conversion (number→string).
			in: cty.TupleVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"name":  cty.StringVal("first"),
					"count": cty.NumberIntVal(42),
				}),
			}),
			attribute: &configschema.Attribute{
				NestedType: &configschema.Object{
					Nesting: configschema.NestingList,
					Attributes: map[string]*configschema.Attribute{
						"name":  {Type: cty.String},
						"count": {Type: cty.String},
					},
				},
			},
			expected: cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"name":  cty.StringVal("first"),
					"count": cty.StringVal("42"),
				}),
			}),
		},

		"nesting_list_three_attrs_partial": {
			// Object schema with 3 attributes, only 1 provided.
			// Attributes are filled in alphabetical order: "alpha" (provided),
			// "beta" (generated=ssnk9qhr), "gamma" (generated=amyllmyg).
			in: cty.TupleVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"alpha": cty.StringVal("provided"),
				}),
			}),
			attribute: &configschema.Attribute{
				NestedType: &configschema.Object{
					Nesting: configschema.NestingList,
					Attributes: map[string]*configschema.Attribute{
						"alpha": {Type: cty.String},
						"beta":  {Type: cty.String},
						"gamma": {Type: cty.String},
					},
				},
			},
			expected: cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"alpha": cty.StringVal("provided"),
					"beta":  cty.StringVal("ssnk9qhr"),
					"gamma": cty.StringVal("amyllmyg"),
				}),
			}),
		},

		// =====================================================================
		// NestingSet - BUG: current implementation ignores input and always
		// returns an empty set. These tests document the CURRENT behavior.
		// If NestingSet is fixed to use fillIterable (like NestingList),
		// these tests should be updated to expect filled values.
		// =====================================================================

		"nesting_set_returns_empty_ignoring_input": {
			// BUG: The current implementation returns an empty set regardless
			// of the input. Override values for NestingSet attributes are
			// silently dropped, just like the original NestingList bug.
			in: cty.SetVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"name":  cty.StringVal("first"),
					"value": cty.StringVal("one"),
				}),
			}),
			attribute: &configschema.Attribute{
				NestedType: &configschema.Object{
					Nesting: configschema.NestingSet,
					Attributes: map[string]*configschema.Attribute{
						"name":  {Type: cty.String},
						"value": {Type: cty.String},
					},
				},
			},
			// BUG: should be cty.SetVal with the input values preserved,
			// but current code returns empty.
			expected: cty.SetValEmpty(cty.Object(map[string]cty.Type{
				"name":  cty.String,
				"value": cty.String,
			})),
		},

		// =====================================================================
		// NestingMap - BUG: current implementation ignores input and always
		// returns an empty map. These tests document the CURRENT behavior.
		// If NestingMap is fixed to use fillIterable (like NestingList),
		// these tests should be updated to expect filled values.
		// =====================================================================

		"nesting_map_returns_empty_ignoring_input": {
			// BUG: The current implementation returns an empty map regardless
			// of the input. Override values for NestingMap attributes are
			// silently dropped, just like the original NestingList bug.
			in: cty.MapVal(map[string]cty.Value{
				"one": cty.ObjectVal(map[string]cty.Value{
					"name":  cty.StringVal("first"),
					"value": cty.StringVal("val1"),
				}),
			}),
			attribute: &configschema.Attribute{
				NestedType: &configschema.Object{
					Nesting: configschema.NestingMap,
					Attributes: map[string]*configschema.Attribute{
						"name":  {Type: cty.String},
						"value": {Type: cty.String},
					},
				},
			},
			// BUG: should be cty.MapVal with the input values preserved,
			// but current code returns empty.
			expected: cty.MapValEmpty(cty.Object(map[string]cty.Type{
				"name":  cty.String,
				"value": cty.String,
			})),
		},

		// =====================================================================
		// Deep nesting: NestingSingle containing NestingList
		// =====================================================================

		"nesting_single_containing_nesting_list": {
			// A single object that has a child attribute which is a NestingList.
			// The fillObject function recurses into fillAttribute for children,
			// which then calls fillIterable for the NestingList child.
			in: cty.ObjectVal(map[string]cty.Value{
				"id": cty.StringVal("myid"),
				"items": cty.TupleVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"name": cty.StringVal("item1"),
					}),
				}),
			}),
			attribute: &configschema.Attribute{
				NestedType: &configschema.Object{
					Nesting: configschema.NestingSingle,
					Attributes: map[string]*configschema.Attribute{
						"id": {Type: cty.String},
						"items": {
							NestedType: &configschema.Object{
								Nesting: configschema.NestingList,
								Attributes: map[string]*configschema.Attribute{
									"name":  {Type: cty.String},
									"value": {Type: cty.String},
								},
							},
						},
					},
				},
			},
			expected: cty.ObjectVal(map[string]cty.Value{
				"id": cty.StringVal("myid"),
				"items": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"name":  cty.StringVal("item1"),
						"value": cty.StringVal("ssnk9qhr"),
					}),
				}),
			}),
		},

		// =====================================================================
		// Deep nesting: NestingList containing NestingSingle
		// =====================================================================

		"nesting_list_containing_nesting_single": {
			// List elements contain a NestingSingle child attribute.
			// The fillIterable path goes through fillType which handles
			// the list→object recursion, but nested NestedType attributes
			// within the objects are handled purely at the cty.Type level
			// since fillType doesn't know about configschema.Attribute.
			// This works because ConfigType() produces the full type tree.
			in: cty.TupleVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"name": cty.StringVal("item1"),
					"detail": cty.ObjectVal(map[string]cty.Value{
						"key": cty.StringVal("mykey"),
					}),
				}),
			}),
			attribute: &configschema.Attribute{
				NestedType: &configschema.Object{
					Nesting: configschema.NestingList,
					Attributes: map[string]*configschema.Attribute{
						"name": {Type: cty.String},
						"detail": {
							NestedType: &configschema.Object{
								Nesting: configschema.NestingSingle,
								Attributes: map[string]*configschema.Attribute{
									"key":   {Type: cty.String},
									"value": {Type: cty.String},
								},
							},
						},
					},
				},
			},
			expected: cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"name": cty.StringVal("item1"),
					"detail": cty.ObjectVal(map[string]cty.Value{
						"key":   cty.StringVal("mykey"),
						"value": cty.StringVal("ssnk9qhr"),
					}),
				}),
			}),
		},

		// =====================================================================
		// Deep nesting: NestingList containing NestingList (3 levels)
		// =====================================================================

		"nesting_list_containing_nesting_list": {
			// Three levels: outer list → objects with inner list → objects.
			// Tests that the fillIterable→fillType recursion handles nested
			// list-of-object types correctly through pure cty.Type handling.
			in: cty.TupleVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"group": cty.StringVal("g1"),
					"members": cty.TupleVal([]cty.Value{
						cty.ObjectVal(map[string]cty.Value{
							"name": cty.StringVal("alice"),
						}),
					}),
				}),
			}),
			attribute: &configschema.Attribute{
				NestedType: &configschema.Object{
					Nesting: configschema.NestingList,
					Attributes: map[string]*configschema.Attribute{
						"group": {Type: cty.String},
						"members": {
							NestedType: &configschema.Object{
								Nesting: configschema.NestingList,
								Attributes: map[string]*configschema.Attribute{
									"name": {Type: cty.String},
									"role": {Type: cty.String},
								},
							},
						},
					},
				},
			},
			expected: cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"group": cty.StringVal("g1"),
					"members": cty.ListVal([]cty.Value{
						cty.ObjectVal(map[string]cty.Value{
							"name": cty.StringVal("alice"),
							"role": cty.StringVal("ssnk9qhr"),
						}),
					}),
				}),
			}),
		},

		// =====================================================================
		// Deep nesting: NestingSingle containing NestingSingle (2 levels)
		// =====================================================================

		"nesting_single_containing_nesting_single": {
			in: cty.ObjectVal(map[string]cty.Value{
				"outer_name": cty.StringVal("outer"),
				"inner": cty.ObjectVal(map[string]cty.Value{
					"inner_name": cty.StringVal("inner"),
				}),
			}),
			attribute: &configschema.Attribute{
				NestedType: &configschema.Object{
					Nesting: configschema.NestingSingle,
					Attributes: map[string]*configschema.Attribute{
						"outer_name": {Type: cty.String},
						"inner": {
							NestedType: &configschema.Object{
								Nesting: configschema.NestingSingle,
								Attributes: map[string]*configschema.Attribute{
									"inner_name":  {Type: cty.String},
									"inner_value": {Type: cty.String},
								},
							},
						},
					},
				},
			},
			expected: cty.ObjectVal(map[string]cty.Value{
				"outer_name": cty.StringVal("outer"),
				"inner": cty.ObjectVal(map[string]cty.Value{
					"inner_name":  cty.StringVal("inner"),
					"inner_value": cty.StringVal("ssnk9qhr"),
				}),
			}),
		},

		// =====================================================================
		// NestingSingle with mixed attribute types inside
		// =====================================================================

		"nesting_single_with_mixed_types": {
			// Object with string, number, bool, and list attributes.
			in: cty.ObjectVal(map[string]cty.Value{
				"name":   cty.StringVal("test"),
				"active": cty.True,
			}),
			attribute: &configschema.Attribute{
				NestedType: &configschema.Object{
					Nesting: configschema.NestingSingle,
					Attributes: map[string]*configschema.Attribute{
						"active": {Type: cty.Bool},
						"count":  {Type: cty.Number},
						"name":   {Type: cty.String},
					},
				},
			},
			// Sorted: active (provided), count (generated=0), name (provided)
			expected: cty.ObjectVal(map[string]cty.Value{
				"active": cty.True,
				"count":  cty.Zero,
				"name":   cty.StringVal("test"),
			}),
		},

		// =====================================================================
		// NestingList with elements that have no missing attributes
		// =====================================================================

		"nesting_list_no_missing_attrs_multiple_elements": {
			in: cty.TupleVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"name":  cty.StringVal("first"),
					"value": cty.StringVal("one"),
				}),
				cty.ObjectVal(map[string]cty.Value{
					"name":  cty.StringVal("second"),
					"value": cty.StringVal("two"),
				}),
			}),
			attribute: &configschema.Attribute{
				NestedType: &configschema.Object{
					Nesting: configschema.NestingList,
					Attributes: map[string]*configschema.Attribute{
						"name":  {Type: cty.String},
						"value": {Type: cty.String},
					},
				},
			},
			expected: cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"name":  cty.StringVal("first"),
					"value": cty.StringVal("one"),
				}),
				cty.ObjectVal(map[string]cty.Value{
					"name":  cty.StringVal("second"),
					"value": cty.StringVal("two"),
				}),
			}),
		},

		// =====================================================================
		// NestingList with single-attr objects (minimal schema)
		// =====================================================================

		"nesting_list_single_attr_schema": {
			in: cty.TupleVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{}),
			}),
			attribute: &configschema.Attribute{
				NestedType: &configschema.Object{
					Nesting: configschema.NestingList,
					Attributes: map[string]*configschema.Attribute{
						"id": {Type: cty.String},
					},
				},
			},
			expected: cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("ssnk9qhr"),
				}),
			}),
		},

		// =====================================================================
		// NestingSingle with a child that has a plain list attribute (Type not NestedType)
		// =====================================================================

		"nesting_single_with_plain_list_child": {
			in: cty.ObjectVal(map[string]cty.Value{
				"name": cty.StringVal("test"),
				"tags": cty.ListVal([]cty.Value{
					cty.StringVal("tag1"),
					cty.StringVal("tag2"),
				}),
			}),
			attribute: &configschema.Attribute{
				NestedType: &configschema.Object{
					Nesting: configschema.NestingSingle,
					Attributes: map[string]*configschema.Attribute{
						"name": {Type: cty.String},
						"tags": {Type: cty.List(cty.String)},
					},
				},
			},
			expected: cty.ObjectVal(map[string]cty.Value{
				"name": cty.StringVal("test"),
				"tags": cty.ListVal([]cty.Value{
					cty.StringVal("tag1"),
					cty.StringVal("tag2"),
				}),
			}),
		},

		// =====================================================================
		// NestingList with bool and number attribute types
		// =====================================================================

		"nesting_list_with_bool_and_number_attrs": {
			in: cty.TupleVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"name": cty.StringVal("item"),
				}),
			}),
			attribute: &configschema.Attribute{
				NestedType: &configschema.Object{
					Nesting: configschema.NestingList,
					Attributes: map[string]*configschema.Attribute{
						"active": {Type: cty.Bool},
						"count":  {Type: cty.Number},
						"name":   {Type: cty.String},
					},
				},
			},
			// Sorted: active (generated=false), count (generated=0), name (provided)
			expected: cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"active": cty.False,
					"count":  cty.Zero,
					"name":   cty.StringVal("item"),
				}),
			}),
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			testRand = rand.New(rand.NewSource(0))
			defer func() {
				testRand = nil
			}()

			actual, err := FillAttribute(tc.in, tc.attribute)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if !tc.expected.RawEquals(actual) {
				t.Errorf("\nexpected: %s\nactual:   %s", tc.expected.GoString(), actual.GoString())
			}
		})
	}
}

// TestFillAttribute_Errors tests error cases for FillAttribute.
func TestFillAttribute_Errors(t *testing.T) {
	tcs := map[string]struct {
		in        cty.Value
		attribute *configschema.Attribute
		err       string
	}{
		"nesting_single_non_object_input": {
			// NestingSingle requires an object input; a string should fail.
			in: cty.StringVal("not an object"),
			attribute: &configschema.Attribute{
				NestedType: &configschema.Object{
					Nesting: configschema.NestingSingle,
					Attributes: map[string]*configschema.Attribute{
						"name": {Type: cty.String},
					},
				},
			},
			err: "incompatible types; expected object type, found string",
		},
		"nesting_group_non_object_input": {
			in: cty.NumberIntVal(42),
			attribute: &configschema.Attribute{
				NestedType: &configschema.Object{
					Nesting: configschema.NestingGroup,
					Attributes: map[string]*configschema.Attribute{
						"name": {Type: cty.String},
					},
				},
			},
			err: "incompatible types; expected object type, found number",
		},
		"nesting_list_incompatible_input": {
			// Passing a plain string where a list of objects is expected.
			in: cty.StringVal("not a list"),
			attribute: &configschema.Attribute{
				NestedType: &configschema.Object{
					Nesting: configschema.NestingList,
					Attributes: map[string]*configschema.Attribute{
						"name": {Type: cty.String},
					},
				},
			},
			err: "incompatible types; expected list of object, found string",
		},
		"nesting_list_element_type_mismatch": {
			// Tuple elements are strings, but the schema expects objects.
			in: cty.TupleVal([]cty.Value{
				cty.StringVal("not an object"),
			}),
			attribute: &configschema.Attribute{
				NestedType: &configschema.Object{
					Nesting: configschema.NestingList,
					Attributes: map[string]*configschema.Attribute{
						"name": {Type: cty.String},
					},
				},
			},
			err: "incompatible types; expected object, found string",
		},
		"plain_attribute_type_mismatch": {
			// Plain attribute with completely incompatible type.
			in: cty.ListValEmpty(cty.String),
			attribute: &configschema.Attribute{
				Type: cty.Bool,
			},
			err: "bool required, but have list of string",
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			actual, err := FillAttribute(tc.in, tc.attribute)
			if err == nil {
				t.Fatalf("expected error but got success with value: %s", actual.GoString())
			}
			if out := err.Error(); out != tc.err {
				t.Errorf("\nexpected error: %s\nactual error:   %s", tc.err, out)
			}
			if actual != cty.NilVal {
				t.Errorf("expected cty.NilVal on error but got: %s", actual.GoString())
			}
		})
	}
}

func TestFillType(t *testing.T) {
	tcs := map[string]struct {
		in  cty.Value
		out cty.Value
	}{
		"object_to_object": {
			in: cty.ObjectVal(map[string]cty.Value{
				"id": cty.StringVal("hello"),
			}),
			out: cty.ObjectVal(map[string]cty.Value{
				"id":    cty.StringVal("hello"),
				"value": cty.StringVal("ssnk9qhr"),
			}),
		},
		"map_to_object": {
			in: cty.MapVal(map[string]cty.Value{
				"id": cty.StringVal("hello"),
			}),
			out: cty.ObjectVal(map[string]cty.Value{
				"id":    cty.StringVal("hello"),
				"value": cty.StringVal("ssnk9qhr"),
			}),
		},
		"list_to_list": {
			in: cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{}),
				cty.ObjectVal(map[string]cty.Value{}),
			}),
			out: cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("ssnk9qhr"),
				}),
				cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("amyllmyg"),
				}),
			}),
		},
		"tuple_to_list": {
			in: cty.TupleVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{}),
				cty.ObjectVal(map[string]cty.Value{}),
			}),
			out: cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("ssnk9qhr"),
				}),
				cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("amyllmyg"),
				}),
			}),
		},
		"set_to_list": {
			in: cty.SetVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"value": cty.StringVal("ssnk9qhr"),
				}),
				cty.ObjectVal(map[string]cty.Value{
					"value": cty.StringVal("amyllmyg"),
				}),
			}),
			out: cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"id":    cty.StringVal("ssnk9qhr"),
					"value": cty.StringVal("amyllmyg"),
				}),
				cty.ObjectVal(map[string]cty.Value{
					"id":    cty.StringVal("amyllmyg"),
					"value": cty.StringVal("ssnk9qhr"),
				}),
			}),
		},
		"list_to_set": {
			in: cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{}),
				cty.ObjectVal(map[string]cty.Value{}),
			}),
			out: cty.SetVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("ssnk9qhr"),
				}),
				cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("amyllmyg"),
				}),
			}),
		},
		"tuple_to_set": {
			in: cty.TupleVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{}),
				cty.ObjectVal(map[string]cty.Value{}),
			}),
			out: cty.SetVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("ssnk9qhr"),
				}),
				cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("amyllmyg"),
				}),
			}),
		},
		"set_to_set": {
			in: cty.SetVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"value": cty.StringVal("ssnk9qhr"),
				}),
				cty.ObjectVal(map[string]cty.Value{
					"value": cty.StringVal("amyllmyg"),
				}),
			}),
			out: cty.SetVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"id":    cty.StringVal("ssnk9qhr"),
					"value": cty.StringVal("amyllmyg"),
				}),
				cty.ObjectVal(map[string]cty.Value{
					"id":    cty.StringVal("amyllmyg"),
					"value": cty.StringVal("ssnk9qhr"),
				}),
			}),
		},
		"tuple_to_tuple": {
			in: cty.TupleVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{}),
				cty.ObjectVal(map[string]cty.Value{}),
			}),
			out: cty.TupleVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("ssnk9qhr"),
				}),
				cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("amyllmyg"),
				}),
			}),
		},
		"map_to_map": {
			in: cty.MapVal(map[string]cty.Value{
				"one": cty.ObjectVal(map[string]cty.Value{}),
				"two": cty.ObjectVal(map[string]cty.Value{}),
			}),
			out: cty.MapVal(map[string]cty.Value{
				"one": cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("ssnk9qhr"),
				}),
				"two": cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("amyllmyg"),
				}),
			}),
		},
		"object_to_map": {
			in: cty.ObjectVal(map[string]cty.Value{
				"one": cty.ObjectVal(map[string]cty.Value{}),
				"two": cty.ObjectVal(map[string]cty.Value{}),
			}),
			out: cty.MapVal(map[string]cty.Value{
				"one": cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("ssnk9qhr"),
				}),
				"two": cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("amyllmyg"),
				}),
			}),
		},
		"additional_attributes": {
			in: cty.ObjectVal(map[string]cty.Value{
				"one": cty.StringVal("hello"),
				"two": cty.StringVal("world"),
			}),
			out: cty.ObjectVal(map[string]cty.Value{
				"one":   cty.StringVal("hello"),
				"three": cty.StringVal("ssnk9qhr"),
			}),
		},
		// This is just a sort of safety check to validate it falls through to
		// normal conversions for everything we don't handle.
		"normal_conversion": {
			in: cty.MapVal(map[string]cty.Value{
				"key_one": cty.StringVal("value_one"),
				"key_two": cty.StringVal("value_two"),
			}),
			out: cty.ObjectVal(map[string]cty.Value{
				"key_one": cty.StringVal("value_one"),
				"key_two": cty.StringVal("value_two"),
			}),
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {

			// Let's have predictable test outcomes.
			testRand = rand.New(rand.NewSource(0))
			defer func() {
				testRand = nil
			}()

			actual, err := FillType(tc.in, tc.out.Type())
			if err != nil {
				t.Fatal(err)
			}

			expected := tc.out
			if !expected.RawEquals(actual) {
				t.Errorf("expected:%s\nactual:   %s", expected.GoString(), actual.GoString())
			}
		})
	}
}

func TestFillType_Errors(t *testing.T) {

	tcs := map[string]struct {
		in     cty.Value
		target cty.Type
		err    string
	}{
		"error_diff_tuple_types": {
			in: cty.TupleVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{}),
				cty.StringVal("not an object"),
			}),
			target: cty.List(cty.EmptyObject),
			err:    "incompatible types; expected object, found string",
		},
		"error_diff_object_types": {
			in: cty.ObjectVal(map[string]cty.Value{
				"object": cty.ObjectVal(map[string]cty.Value{}),
				"string": cty.StringVal("not an object"),
			}),
			target: cty.Map(cty.EmptyObject),
			err:    "incompatible types; expected object, found string",
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			actual, err := FillType(tc.in, tc.target)
			if err == nil {
				t.Fatal("should have errored")
			}

			if out := err.Error(); out != tc.err {
				t.Errorf("\nexpected: %s\nactual:   %s", tc.err, out)
			}

			if actual != cty.NilVal {
				t.Fatal("should have errored")
			}
		})
	}

}
