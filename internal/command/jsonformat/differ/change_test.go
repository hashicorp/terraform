package differ

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"

	"github.com/hashicorp/terraform/internal/command/jsonformat/computed/renderers"
	"github.com/hashicorp/terraform/internal/command/jsonformat/differ/replace"
	"github.com/hashicorp/terraform/internal/command/jsonprovider"
	"github.com/hashicorp/terraform/internal/plans"
)

type SetDiff struct {
	Before SetDiffEntry
	After  SetDiffEntry
}

type SetDiffEntry struct {
	SingleDiff renderers.ValidateDiffFunction
	ObjectDiff map[string]renderers.ValidateDiffFunction

	Replace bool
	Action  plans.Action
}

func (entry SetDiffEntry) Validate(obj func(attributes map[string]renderers.ValidateDiffFunction, action plans.Action, replace bool) renderers.ValidateDiffFunction) renderers.ValidateDiffFunction {
	if entry.SingleDiff != nil {
		return entry.SingleDiff
	}

	return obj(entry.ObjectDiff, entry.Action, entry.Replace)
}

func TestValue_ObjectAttributes(t *testing.T) {
	// This function holds a range of test cases creating, deleting and editing
	// objects. It is built in such a way that it can automatically test these
	// operations on objects both directly and nested, as well as within all
	// types of collections.

	tcs := map[string]struct {
		input                Change
		attributes           map[string]cty.Type
		validateSingleDiff   renderers.ValidateDiffFunction
		validateObject       renderers.ValidateDiffFunction
		validateNestedObject renderers.ValidateDiffFunction
		validateDiffs        map[string]renderers.ValidateDiffFunction
		validateReplace      bool
		validateAction       plans.Action
		// Sets break changes out differently to the other collections, so they
		// have their own entry.
		validateSetDiffs *SetDiff
	}{
		"create": {
			input: Change{
				Before: nil,
				After: map[string]interface{}{
					"attribute_one": "new",
				},
			},
			attributes: map[string]cty.Type{
				"attribute_one": cty.String,
			},
			validateDiffs: map[string]renderers.ValidateDiffFunction{
				"attribute_one": renderers.ValidatePrimitive(nil, "new", plans.Create, false),
			},
			validateAction:  plans.Create,
			validateReplace: false,
		},
		"delete": {
			input: Change{
				Before: map[string]interface{}{
					"attribute_one": "old",
				},
				After: nil,
			},
			attributes: map[string]cty.Type{
				"attribute_one": cty.String,
			},
			validateDiffs: map[string]renderers.ValidateDiffFunction{
				"attribute_one": renderers.ValidatePrimitive("old", nil, plans.Delete, false),
			},
			validateAction:  plans.Delete,
			validateReplace: false,
		},
		"create_sensitive": {
			input: Change{
				Before: nil,
				After: map[string]interface{}{
					"attribute_one": "new",
				},
				AfterSensitive: true,
			},
			attributes: map[string]cty.Type{
				"attribute_one": cty.String,
			},
			validateSingleDiff: renderers.ValidateSensitive(renderers.ValidateObject(map[string]renderers.ValidateDiffFunction{
				"attribute_one": renderers.ValidatePrimitive(nil, "new", plans.Create, false),
			}, plans.Create, false),
				false,
				true,
				plans.Create,
				false),
			validateNestedObject: renderers.ValidateSensitive(renderers.ValidateNestedObject(map[string]renderers.ValidateDiffFunction{
				"attribute_one": renderers.ValidatePrimitive(nil, "new", plans.Create, false),
			}, plans.Create, false),
				false,
				true,
				plans.Create,
				false),
		},
		"delete_sensitive": {
			input: Change{
				Before: map[string]interface{}{
					"attribute_one": "old",
				},
				BeforeSensitive: true,
				After:           nil,
			},
			attributes: map[string]cty.Type{
				"attribute_one": cty.String,
			},
			validateSingleDiff: renderers.ValidateSensitive(renderers.ValidateObject(map[string]renderers.ValidateDiffFunction{
				"attribute_one": renderers.ValidatePrimitive("old", nil, plans.Delete, false),
			}, plans.Delete, false), true, false, plans.Delete, false),
			validateNestedObject: renderers.ValidateSensitive(renderers.ValidateNestedObject(map[string]renderers.ValidateDiffFunction{
				"attribute_one": renderers.ValidatePrimitive("old", nil, plans.Delete, false),
			}, plans.Delete, false), true, false, plans.Delete, false),
		},
		"create_unknown": {
			input: Change{
				Before:  nil,
				After:   nil,
				Unknown: true,
			},
			attributes: map[string]cty.Type{
				"attribute_one": cty.String,
			},
			validateSingleDiff: renderers.ValidateUnknown(nil, plans.Create, false),
		},
		"update_unknown": {
			input: Change{
				Before: map[string]interface{}{
					"attribute_one": "old",
				},
				After:   nil,
				Unknown: true,
			},
			attributes: map[string]cty.Type{
				"attribute_one": cty.String,
			},
			validateObject: renderers.ValidateUnknown(renderers.ValidateObject(map[string]renderers.ValidateDiffFunction{
				"attribute_one": renderers.ValidatePrimitive("old", nil, plans.Delete, false),
			}, plans.Delete, false), plans.Update, false),
			validateNestedObject: renderers.ValidateUnknown(renderers.ValidateNestedObject(map[string]renderers.ValidateDiffFunction{
				"attribute_one": renderers.ValidateUnknown(renderers.ValidatePrimitive("old", nil, plans.Delete, false), plans.Update, false),
			}, plans.Update, false), plans.Update, false),
			validateSetDiffs: &SetDiff{
				Before: SetDiffEntry{
					ObjectDiff: map[string]renderers.ValidateDiffFunction{
						"attribute_one": renderers.ValidatePrimitive("old", nil, plans.Delete, false),
					},
					Action:  plans.Delete,
					Replace: false,
				},
				After: SetDiffEntry{
					SingleDiff: renderers.ValidateUnknown(nil, plans.Create, false),
				},
			},
		},
		"create_attribute": {
			input: Change{
				Before: map[string]interface{}{},
				After: map[string]interface{}{
					"attribute_one": "new",
				},
			},
			attributes: map[string]cty.Type{
				"attribute_one": cty.String,
			},
			validateDiffs: map[string]renderers.ValidateDiffFunction{
				"attribute_one": renderers.ValidatePrimitive(nil, "new", plans.Create, false),
			},
			validateAction:  plans.Update,
			validateReplace: false,
			validateSetDiffs: &SetDiff{
				Before: SetDiffEntry{
					ObjectDiff: nil,
					Action:     plans.Delete,
					Replace:    false,
				},
				After: SetDiffEntry{
					ObjectDiff: map[string]renderers.ValidateDiffFunction{
						"attribute_one": renderers.ValidatePrimitive(nil, "new", plans.Create, false),
					},
					Action:  plans.Create,
					Replace: false,
				},
			},
		},
		"create_attribute_from_explicit_null": {
			input: Change{
				Before: map[string]interface{}{
					"attribute_one": nil,
				},
				After: map[string]interface{}{
					"attribute_one": "new",
				},
			},
			attributes: map[string]cty.Type{
				"attribute_one": cty.String,
			},
			validateDiffs: map[string]renderers.ValidateDiffFunction{
				"attribute_one": renderers.ValidatePrimitive(nil, "new", plans.Create, false),
			},
			validateAction:  plans.Update,
			validateReplace: false,
			validateSetDiffs: &SetDiff{
				Before: SetDiffEntry{
					ObjectDiff: nil,
					Action:     plans.Delete,
					Replace:    false,
				},
				After: SetDiffEntry{
					ObjectDiff: map[string]renderers.ValidateDiffFunction{
						"attribute_one": renderers.ValidatePrimitive(nil, "new", plans.Create, false),
					},
					Action:  plans.Create,
					Replace: false,
				},
			},
		},
		"delete_attribute": {
			input: Change{
				Before: map[string]interface{}{
					"attribute_one": "old",
				},
				After: map[string]interface{}{},
			},
			attributes: map[string]cty.Type{
				"attribute_one": cty.String,
			},
			validateDiffs: map[string]renderers.ValidateDiffFunction{
				"attribute_one": renderers.ValidatePrimitive("old", nil, plans.Delete, false),
			},
			validateAction:  plans.Update,
			validateReplace: false,
			validateSetDiffs: &SetDiff{
				Before: SetDiffEntry{
					ObjectDiff: map[string]renderers.ValidateDiffFunction{
						"attribute_one": renderers.ValidatePrimitive("old", nil, plans.Delete, false),
					},
					Action:  plans.Delete,
					Replace: false,
				},
				After: SetDiffEntry{
					ObjectDiff: nil,
					Action:     plans.Create,
					Replace:    false,
				},
			},
		},
		"delete_attribute_to_explicit_null": {
			input: Change{
				Before: map[string]interface{}{
					"attribute_one": "old",
				},
				After: map[string]interface{}{
					"attribute_one": nil,
				},
			},
			attributes: map[string]cty.Type{
				"attribute_one": cty.String,
			},
			validateDiffs: map[string]renderers.ValidateDiffFunction{
				"attribute_one": renderers.ValidatePrimitive("old", nil, plans.Delete, false),
			},
			validateAction:  plans.Update,
			validateReplace: false,
			validateSetDiffs: &SetDiff{
				Before: SetDiffEntry{
					ObjectDiff: map[string]renderers.ValidateDiffFunction{
						"attribute_one": renderers.ValidatePrimitive("old", nil, plans.Delete, false),
					},
					Action:  plans.Delete,
					Replace: false,
				},
				After: SetDiffEntry{
					ObjectDiff: nil,
					Action:     plans.Create,
					Replace:    false,
				},
			},
		},
		"update_attribute": {
			input: Change{
				Before: map[string]interface{}{
					"attribute_one": "old",
				},
				After: map[string]interface{}{
					"attribute_one": "new",
				},
			},
			attributes: map[string]cty.Type{
				"attribute_one": cty.String,
			},
			validateDiffs: map[string]renderers.ValidateDiffFunction{
				"attribute_one": renderers.ValidatePrimitive("old", "new", plans.Update, false),
			},
			validateAction:  plans.Update,
			validateReplace: false,
			validateSetDiffs: &SetDiff{
				Before: SetDiffEntry{
					ObjectDiff: map[string]renderers.ValidateDiffFunction{
						"attribute_one": renderers.ValidatePrimitive("old", nil, plans.Delete, false),
					},
					Action:  plans.Delete,
					Replace: false,
				},
				After: SetDiffEntry{
					ObjectDiff: map[string]renderers.ValidateDiffFunction{
						"attribute_one": renderers.ValidatePrimitive(nil, "new", plans.Create, false),
					},
					Action:  plans.Create,
					Replace: false,
				},
			},
		},
		"create_sensitive_attribute": {
			input: Change{
				Before: map[string]interface{}{},
				After: map[string]interface{}{
					"attribute_one": "new",
				},
				AfterSensitive: map[string]interface{}{
					"attribute_one": true,
				},
			},
			attributes: map[string]cty.Type{
				"attribute_one": cty.String,
			},
			validateDiffs: map[string]renderers.ValidateDiffFunction{
				"attribute_one": renderers.ValidateSensitive(renderers.ValidatePrimitive(nil, "new", plans.Create, false), false, true, plans.Create, false),
			},
			validateAction:  plans.Update,
			validateReplace: false,
			validateSetDiffs: &SetDiff{
				Before: SetDiffEntry{
					ObjectDiff: nil,
					Action:     plans.Delete,
					Replace:    false,
				},
				After: SetDiffEntry{
					ObjectDiff: map[string]renderers.ValidateDiffFunction{
						"attribute_one": renderers.ValidateSensitive(renderers.ValidatePrimitive(nil, "new", plans.Create, false), false, true, plans.Create, false),
					},
					Action:  plans.Create,
					Replace: false,
				},
			},
		},
		"delete_sensitive_attribute": {
			input: Change{
				Before: map[string]interface{}{
					"attribute_one": "old",
				},
				BeforeSensitive: map[string]interface{}{
					"attribute_one": true,
				},
				After: map[string]interface{}{},
			},
			attributes: map[string]cty.Type{
				"attribute_one": cty.String,
			},
			validateDiffs: map[string]renderers.ValidateDiffFunction{
				"attribute_one": renderers.ValidateSensitive(renderers.ValidatePrimitive("old", nil, plans.Delete, false), true, false, plans.Delete, false),
			},
			validateAction:  plans.Update,
			validateReplace: false,
			validateSetDiffs: &SetDiff{
				Before: SetDiffEntry{
					ObjectDiff: map[string]renderers.ValidateDiffFunction{
						"attribute_one": renderers.ValidateSensitive(renderers.ValidatePrimitive("old", nil, plans.Delete, false), true, false, plans.Delete, false),
					},
					Action:  plans.Delete,
					Replace: false,
				},
				After: SetDiffEntry{
					ObjectDiff: nil,
					Action:     plans.Create,
					Replace:    false,
				},
			},
		},
		"update_sensitive_attribute": {
			input: Change{
				Before: map[string]interface{}{
					"attribute_one": "old",
				},
				BeforeSensitive: map[string]interface{}{
					"attribute_one": true,
				},
				After: map[string]interface{}{
					"attribute_one": "new",
				},
				AfterSensitive: map[string]interface{}{
					"attribute_one": true,
				},
			},
			attributes: map[string]cty.Type{
				"attribute_one": cty.String,
			},
			validateDiffs: map[string]renderers.ValidateDiffFunction{
				"attribute_one": renderers.ValidateSensitive(renderers.ValidatePrimitive("old", "new", plans.Update, false), true, true, plans.Update, false),
			},
			validateAction:  plans.Update,
			validateReplace: false,
			validateSetDiffs: &SetDiff{
				Before: SetDiffEntry{
					ObjectDiff: map[string]renderers.ValidateDiffFunction{
						"attribute_one": renderers.ValidateSensitive(renderers.ValidatePrimitive("old", nil, plans.Delete, false), true, false, plans.Delete, false),
					},
					Action:  plans.Delete,
					Replace: false,
				},
				After: SetDiffEntry{
					ObjectDiff: map[string]renderers.ValidateDiffFunction{
						"attribute_one": renderers.ValidateSensitive(renderers.ValidatePrimitive(nil, "new", plans.Create, false), false, true, plans.Create, false),
					},
					Action:  plans.Create,
					Replace: false,
				},
			},
		},
		"create_computed_attribute": {
			input: Change{
				Before: map[string]interface{}{},
				After:  map[string]interface{}{},
				Unknown: map[string]interface{}{
					"attribute_one": true,
				},
			},
			attributes: map[string]cty.Type{
				"attribute_one": cty.String,
			},
			validateDiffs: map[string]renderers.ValidateDiffFunction{
				"attribute_one": renderers.ValidateUnknown(nil, plans.Create, false),
			},
			validateAction:  plans.Update,
			validateReplace: false,
		},
		"update_computed_attribute": {
			input: Change{
				Before: map[string]interface{}{
					"attribute_one": "old",
				},
				After: map[string]interface{}{},
				Unknown: map[string]interface{}{
					"attribute_one": true,
				},
			},
			attributes: map[string]cty.Type{
				"attribute_one": cty.String,
			},
			validateDiffs: map[string]renderers.ValidateDiffFunction{
				"attribute_one": renderers.ValidateUnknown(
					renderers.ValidatePrimitive("old", nil, plans.Delete, false),
					plans.Update,
					false),
			},
			validateAction:  plans.Update,
			validateReplace: false,
			validateSetDiffs: &SetDiff{
				Before: SetDiffEntry{
					ObjectDiff: map[string]renderers.ValidateDiffFunction{
						"attribute_one": renderers.ValidatePrimitive("old", nil, plans.Delete, false),
					},
					Action:  plans.Delete,
					Replace: false,
				},
				After: SetDiffEntry{
					ObjectDiff: map[string]renderers.ValidateDiffFunction{
						"attribute_one": renderers.ValidateUnknown(nil, plans.Create, false),
					},
					Action:  plans.Create,
					Replace: false,
				},
			},
		},
		"ignores_unset_fields": {
			input: Change{
				Before: map[string]interface{}{},
				After:  map[string]interface{}{},
			},
			attributes: map[string]cty.Type{
				"attribute_one": cty.String,
			},
			validateDiffs:   map[string]renderers.ValidateDiffFunction{},
			validateAction:  plans.NoOp,
			validateReplace: false,
		},
		"update_replace_self": {
			input: Change{
				Before: map[string]interface{}{
					"attribute_one": "old",
				},
				After: map[string]interface{}{
					"attribute_one": "new",
				},
				ReplacePaths: replace.ForcesReplacement{
					ReplacePaths: [][]interface{}{
						{},
					},
				},
			},
			attributes: map[string]cty.Type{
				"attribute_one": cty.String,
			},
			validateDiffs: map[string]renderers.ValidateDiffFunction{
				"attribute_one": renderers.ValidatePrimitive("old", "new", plans.Update, false),
			},
			validateAction:  plans.Update,
			validateReplace: true,
			validateSetDiffs: &SetDiff{
				Before: SetDiffEntry{
					ObjectDiff: map[string]renderers.ValidateDiffFunction{
						"attribute_one": renderers.ValidatePrimitive("old", nil, plans.Delete, false),
					},
					Action:  plans.Delete,
					Replace: true,
				},
				After: SetDiffEntry{
					ObjectDiff: map[string]renderers.ValidateDiffFunction{
						"attribute_one": renderers.ValidatePrimitive(nil, "new", plans.Create, false),
					},
					Action:  plans.Create,
					Replace: false,
				},
			},
		},
		"update_replace_attribute": {
			input: Change{
				Before: map[string]interface{}{
					"attribute_one": "old",
				},
				After: map[string]interface{}{
					"attribute_one": "new",
				},
				ReplacePaths: replace.ForcesReplacement{
					ReplacePaths: [][]interface{}{
						{"attribute_one"},
					},
				},
			},
			attributes: map[string]cty.Type{
				"attribute_one": cty.String,
			},
			validateDiffs: map[string]renderers.ValidateDiffFunction{
				"attribute_one": renderers.ValidatePrimitive("old", "new", plans.Update, true),
			},
			validateAction:  plans.Update,
			validateReplace: false,
			validateSetDiffs: &SetDiff{
				Before: SetDiffEntry{
					ObjectDiff: map[string]renderers.ValidateDiffFunction{
						"attribute_one": renderers.ValidatePrimitive("old", nil, plans.Delete, true),
					},
					Action:  plans.Delete,
					Replace: false,
				},
				After: SetDiffEntry{
					ObjectDiff: map[string]renderers.ValidateDiffFunction{
						"attribute_one": renderers.ValidatePrimitive(nil, "new", plans.Create, false),
					},
					Action:  plans.Create,
					Replace: false,
				},
			},
		},
	}

	for name, tmp := range tcs {
		tc := tmp

		collectionDefaultAction := plans.Update
		if name == "ignores_unset_fields" {
			// Special case for this test, as it is the only one that doesn't
			// have the collection types return an update.
			collectionDefaultAction = plans.NoOp
		}

		t.Run(name, func(t *testing.T) {
			t.Run("object", func(t *testing.T) {
				attribute := &jsonprovider.Attribute{
					AttributeType: unmarshalType(t, cty.Object(tc.attributes)),
				}

				if tc.validateObject != nil {
					tc.validateObject(t, tc.input.ComputeDiffForAttribute(attribute))
					return
				}

				if tc.validateSingleDiff != nil {
					tc.validateSingleDiff(t, tc.input.ComputeDiffForAttribute(attribute))
					return
				}

				validate := renderers.ValidateObject(tc.validateDiffs, tc.validateAction, tc.validateReplace)
				validate(t, tc.input.ComputeDiffForAttribute(attribute))
			})

			t.Run("map", func(t *testing.T) {
				attribute := &jsonprovider.Attribute{
					AttributeType: unmarshalType(t, cty.Map(cty.Object(tc.attributes))),
				}

				input := wrapChangeInMap(tc.input)

				if tc.validateObject != nil {
					validate := renderers.ValidateMap(map[string]renderers.ValidateDiffFunction{
						"element": tc.validateObject,
					}, collectionDefaultAction, false)
					validate(t, input.ComputeDiffForAttribute(attribute))
					return
				}

				if tc.validateSingleDiff != nil {
					validate := renderers.ValidateMap(map[string]renderers.ValidateDiffFunction{
						"element": tc.validateSingleDiff,
					}, collectionDefaultAction, false)
					validate(t, input.ComputeDiffForAttribute(attribute))
					return
				}

				validate := renderers.ValidateMap(map[string]renderers.ValidateDiffFunction{
					"element": renderers.ValidateObject(tc.validateDiffs, tc.validateAction, tc.validateReplace),
				}, collectionDefaultAction, false)
				validate(t, input.ComputeDiffForAttribute(attribute))
			})

			t.Run("list", func(t *testing.T) {
				attribute := &jsonprovider.Attribute{
					AttributeType: unmarshalType(t, cty.List(cty.Object(tc.attributes))),
				}

				input := wrapChangeInSlice(tc.input)

				if tc.validateObject != nil {
					validate := renderers.ValidateList([]renderers.ValidateDiffFunction{
						tc.validateObject,
					}, collectionDefaultAction, false)
					validate(t, input.ComputeDiffForAttribute(attribute))
					return
				}

				if tc.validateSingleDiff != nil {
					validate := renderers.ValidateList([]renderers.ValidateDiffFunction{
						tc.validateSingleDiff,
					}, collectionDefaultAction, false)
					validate(t, input.ComputeDiffForAttribute(attribute))
					return
				}

				validate := renderers.ValidateList([]renderers.ValidateDiffFunction{
					renderers.ValidateObject(tc.validateDiffs, tc.validateAction, tc.validateReplace),
				}, collectionDefaultAction, false)
				validate(t, input.ComputeDiffForAttribute(attribute))
			})

			t.Run("set", func(t *testing.T) {
				attribute := &jsonprovider.Attribute{
					AttributeType: unmarshalType(t, cty.Set(cty.Object(tc.attributes))),
				}

				input := wrapChangeInSlice(tc.input)

				if tc.validateSetDiffs != nil {
					validate := renderers.ValidateSet(func() []renderers.ValidateDiffFunction {
						var ret []renderers.ValidateDiffFunction
						ret = append(ret, tc.validateSetDiffs.Before.Validate(renderers.ValidateObject))
						ret = append(ret, tc.validateSetDiffs.After.Validate(renderers.ValidateObject))
						return ret
					}(), collectionDefaultAction, false)
					validate(t, input.ComputeDiffForAttribute(attribute))
					return
				}

				if tc.validateObject != nil {
					validate := renderers.ValidateSet([]renderers.ValidateDiffFunction{
						tc.validateObject,
					}, collectionDefaultAction, false)
					validate(t, input.ComputeDiffForAttribute(attribute))
					return
				}

				if tc.validateSingleDiff != nil {
					validate := renderers.ValidateSet([]renderers.ValidateDiffFunction{
						tc.validateSingleDiff,
					}, collectionDefaultAction, false)
					validate(t, input.ComputeDiffForAttribute(attribute))
					return
				}

				validate := renderers.ValidateSet([]renderers.ValidateDiffFunction{
					renderers.ValidateObject(tc.validateDiffs, tc.validateAction, tc.validateReplace),
				}, collectionDefaultAction, false)
				validate(t, input.ComputeDiffForAttribute(attribute))
			})
		})

		t.Run(fmt.Sprintf("nested_%s", name), func(t *testing.T) {
			t.Run("object", func(t *testing.T) {
				attribute := &jsonprovider.Attribute{
					AttributeNestedType: &jsonprovider.NestedType{
						Attributes: func() map[string]*jsonprovider.Attribute {
							attributes := make(map[string]*jsonprovider.Attribute)
							for key, attribute := range tc.attributes {
								attributes[key] = &jsonprovider.Attribute{
									AttributeType: unmarshalType(t, attribute),
								}
							}
							return attributes
						}(),
						NestingMode: "single",
					},
				}

				if tc.validateNestedObject != nil {
					tc.validateNestedObject(t, tc.input.ComputeDiffForAttribute(attribute))
					return
				}

				if tc.validateSingleDiff != nil {
					tc.validateSingleDiff(t, tc.input.ComputeDiffForAttribute(attribute))
					return
				}

				validate := renderers.ValidateNestedObject(tc.validateDiffs, tc.validateAction, tc.validateReplace)
				validate(t, tc.input.ComputeDiffForAttribute(attribute))
			})

			t.Run("map", func(t *testing.T) {
				attribute := &jsonprovider.Attribute{
					AttributeNestedType: &jsonprovider.NestedType{
						Attributes: func() map[string]*jsonprovider.Attribute {
							attributes := make(map[string]*jsonprovider.Attribute)
							for key, attribute := range tc.attributes {
								attributes[key] = &jsonprovider.Attribute{
									AttributeType: unmarshalType(t, attribute),
								}
							}
							return attributes
						}(),
						NestingMode: "map",
					},
				}

				input := wrapChangeInMap(tc.input)

				if tc.validateNestedObject != nil {
					validate := renderers.ValidateMap(map[string]renderers.ValidateDiffFunction{
						"element": tc.validateNestedObject,
					}, collectionDefaultAction, false)
					validate(t, input.ComputeDiffForAttribute(attribute))
					return
				}

				if tc.validateSingleDiff != nil {
					validate := renderers.ValidateMap(map[string]renderers.ValidateDiffFunction{
						"element": tc.validateSingleDiff,
					}, collectionDefaultAction, false)
					validate(t, input.ComputeDiffForAttribute(attribute))
					return
				}

				validate := renderers.ValidateMap(map[string]renderers.ValidateDiffFunction{
					"element": renderers.ValidateNestedObject(tc.validateDiffs, tc.validateAction, tc.validateReplace),
				}, collectionDefaultAction, false)
				validate(t, input.ComputeDiffForAttribute(attribute))
			})

			t.Run("list", func(t *testing.T) {
				attribute := &jsonprovider.Attribute{
					AttributeNestedType: &jsonprovider.NestedType{
						Attributes: func() map[string]*jsonprovider.Attribute {
							attributes := make(map[string]*jsonprovider.Attribute)
							for key, attribute := range tc.attributes {
								attributes[key] = &jsonprovider.Attribute{
									AttributeType: unmarshalType(t, attribute),
								}
							}
							return attributes
						}(),
						NestingMode: "list",
					},
				}

				input := wrapChangeInSlice(tc.input)

				if tc.validateNestedObject != nil {
					validate := renderers.ValidateNestedList([]renderers.ValidateDiffFunction{
						tc.validateNestedObject,
					}, collectionDefaultAction, false)
					validate(t, input.ComputeDiffForAttribute(attribute))
					return
				}

				if tc.validateSingleDiff != nil {
					validate := renderers.ValidateNestedList([]renderers.ValidateDiffFunction{
						tc.validateSingleDiff,
					}, collectionDefaultAction, false)
					validate(t, input.ComputeDiffForAttribute(attribute))
					return
				}

				validate := renderers.ValidateNestedList([]renderers.ValidateDiffFunction{
					renderers.ValidateNestedObject(tc.validateDiffs, tc.validateAction, tc.validateReplace),
				}, collectionDefaultAction, false)
				validate(t, input.ComputeDiffForAttribute(attribute))
			})

			t.Run("set", func(t *testing.T) {
				attribute := &jsonprovider.Attribute{
					AttributeNestedType: &jsonprovider.NestedType{
						Attributes: func() map[string]*jsonprovider.Attribute {
							attributes := make(map[string]*jsonprovider.Attribute)
							for key, attribute := range tc.attributes {
								attributes[key] = &jsonprovider.Attribute{
									AttributeType: unmarshalType(t, attribute),
								}
							}
							return attributes
						}(),
						NestingMode: "set",
					},
				}

				input := wrapChangeInSlice(tc.input)

				if tc.validateSetDiffs != nil {
					validate := renderers.ValidateSet(func() []renderers.ValidateDiffFunction {
						var ret []renderers.ValidateDiffFunction
						ret = append(ret, tc.validateSetDiffs.Before.Validate(renderers.ValidateNestedObject))
						ret = append(ret, tc.validateSetDiffs.After.Validate(renderers.ValidateNestedObject))
						return ret
					}(), collectionDefaultAction, false)
					validate(t, input.ComputeDiffForAttribute(attribute))
					return
				}

				if tc.validateNestedObject != nil {
					validate := renderers.ValidateSet([]renderers.ValidateDiffFunction{
						tc.validateNestedObject,
					}, collectionDefaultAction, false)
					validate(t, input.ComputeDiffForAttribute(attribute))
					return
				}

				if tc.validateSingleDiff != nil {
					validate := renderers.ValidateSet([]renderers.ValidateDiffFunction{
						tc.validateSingleDiff,
					}, collectionDefaultAction, false)
					validate(t, input.ComputeDiffForAttribute(attribute))
					return
				}

				validate := renderers.ValidateSet([]renderers.ValidateDiffFunction{
					renderers.ValidateNestedObject(tc.validateDiffs, tc.validateAction, tc.validateReplace),
				}, collectionDefaultAction, false)
				validate(t, input.ComputeDiffForAttribute(attribute))
			})
		})
	}
}

func TestValue_BlockAttributesAndNestedBlocks(t *testing.T) {
	// This function tests manipulating simple attributes and blocks within
	// blocks. It automatically tests these operations within the contexts of
	// different block types.

	tcs := map[string]struct {
		before      interface{}
		after       interface{}
		block       *jsonprovider.Block
		validate    renderers.ValidateDiffFunction
		validateSet []renderers.ValidateDiffFunction
	}{
		"create_attribute": {
			before: map[string]interface{}{},
			after: map[string]interface{}{
				"attribute_one": "new",
			},
			block: &jsonprovider.Block{
				Attributes: map[string]*jsonprovider.Attribute{
					"attribute_one": {
						AttributeType: unmarshalType(t, cty.String),
					},
				},
			},
			validate: renderers.ValidateBlock(map[string]renderers.ValidateDiffFunction{
				"attribute_one": renderers.ValidatePrimitive(nil, "new", plans.Create, false),
			}, nil, nil, nil, nil, plans.Update, false),
			validateSet: []renderers.ValidateDiffFunction{
				renderers.ValidateBlock(nil, nil, nil, nil, nil, plans.Delete, false),
				renderers.ValidateBlock(map[string]renderers.ValidateDiffFunction{
					"attribute_one": renderers.ValidatePrimitive(nil, "new", plans.Create, false),
				}, nil, nil, nil, nil, plans.Create, false),
			},
		},
		"update_attribute": {
			before: map[string]interface{}{
				"attribute_one": "old",
			},
			after: map[string]interface{}{
				"attribute_one": "new",
			},
			block: &jsonprovider.Block{
				Attributes: map[string]*jsonprovider.Attribute{
					"attribute_one": {
						AttributeType: unmarshalType(t, cty.String),
					},
				},
			},
			validate: renderers.ValidateBlock(map[string]renderers.ValidateDiffFunction{
				"attribute_one": renderers.ValidatePrimitive("old", "new", plans.Update, false),
			}, nil, nil, nil, nil, plans.Update, false),
			validateSet: []renderers.ValidateDiffFunction{
				renderers.ValidateBlock(map[string]renderers.ValidateDiffFunction{
					"attribute_one": renderers.ValidatePrimitive("old", nil, plans.Delete, false),
				}, nil, nil, nil, nil, plans.Delete, false),
				renderers.ValidateBlock(map[string]renderers.ValidateDiffFunction{
					"attribute_one": renderers.ValidatePrimitive(nil, "new", plans.Create, false),
				}, nil, nil, nil, nil, plans.Create, false),
			},
		},
		"delete_attribute": {
			before: map[string]interface{}{
				"attribute_one": "old",
			},
			after: map[string]interface{}{},
			block: &jsonprovider.Block{
				Attributes: map[string]*jsonprovider.Attribute{
					"attribute_one": {
						AttributeType: unmarshalType(t, cty.String),
					},
				},
			},
			validate: renderers.ValidateBlock(map[string]renderers.ValidateDiffFunction{
				"attribute_one": renderers.ValidatePrimitive("old", nil, plans.Delete, false),
			}, nil, nil, nil, nil, plans.Update, false),
			validateSet: []renderers.ValidateDiffFunction{
				renderers.ValidateBlock(map[string]renderers.ValidateDiffFunction{
					"attribute_one": renderers.ValidatePrimitive("old", nil, plans.Delete, false),
				}, nil, nil, nil, nil, plans.Delete, false),
				renderers.ValidateBlock(nil, nil, nil, nil, nil, plans.Create, false),
			},
		},
		"create_block": {
			before: map[string]interface{}{},
			after: map[string]interface{}{
				"block_one": map[string]interface{}{
					"attribute_one": "new",
				},
			},
			block: &jsonprovider.Block{
				BlockTypes: map[string]*jsonprovider.BlockType{
					"block_one": {
						Block: &jsonprovider.Block{
							Attributes: map[string]*jsonprovider.Attribute{
								"attribute_one": {
									AttributeType: unmarshalType(t, cty.String),
								},
							},
						},
						NestingMode: "single",
					},
				},
			},
			validate: renderers.ValidateBlock(nil, map[string]renderers.ValidateDiffFunction{
				"block_one": renderers.ValidateBlock(map[string]renderers.ValidateDiffFunction{
					"attribute_one": renderers.ValidatePrimitive(nil, "new", plans.Create, false),
				}, nil, nil, nil, nil, plans.Create, false),
			}, nil, nil, nil, plans.Update, false),
			validateSet: []renderers.ValidateDiffFunction{
				renderers.ValidateBlock(nil, nil, nil, nil, nil, plans.Delete, false),
				renderers.ValidateBlock(nil, map[string]renderers.ValidateDiffFunction{
					"block_one": renderers.ValidateBlock(map[string]renderers.ValidateDiffFunction{
						"attribute_one": renderers.ValidatePrimitive(nil, "new", plans.Create, false),
					}, nil, nil, nil, nil, plans.Create, false),
				}, nil, nil, nil, plans.Create, false),
			},
		},
		"update_block": {
			before: map[string]interface{}{
				"block_one": map[string]interface{}{
					"attribute_one": "old",
				},
			},
			after: map[string]interface{}{
				"block_one": map[string]interface{}{
					"attribute_one": "new",
				},
			},
			block: &jsonprovider.Block{
				BlockTypes: map[string]*jsonprovider.BlockType{
					"block_one": {
						Block: &jsonprovider.Block{
							Attributes: map[string]*jsonprovider.Attribute{
								"attribute_one": {
									AttributeType: unmarshalType(t, cty.String),
								},
							},
						},
						NestingMode: "single",
					},
				},
			},
			validate: renderers.ValidateBlock(nil, map[string]renderers.ValidateDiffFunction{
				"block_one": renderers.ValidateBlock(map[string]renderers.ValidateDiffFunction{
					"attribute_one": renderers.ValidatePrimitive("old", "new", plans.Update, false),
				}, nil, nil, nil, nil, plans.Update, false),
			}, nil, nil, nil, plans.Update, false),
			validateSet: []renderers.ValidateDiffFunction{
				renderers.ValidateBlock(nil, map[string]renderers.ValidateDiffFunction{
					"block_one": renderers.ValidateBlock(map[string]renderers.ValidateDiffFunction{
						"attribute_one": renderers.ValidatePrimitive("old", nil, plans.Delete, false),
					}, nil, nil, nil, nil, plans.Delete, false),
				}, nil, nil, nil, plans.Delete, false),
				renderers.ValidateBlock(nil, map[string]renderers.ValidateDiffFunction{
					"block_one": renderers.ValidateBlock(map[string]renderers.ValidateDiffFunction{
						"attribute_one": renderers.ValidatePrimitive(nil, "new", plans.Create, false),
					}, nil, nil, nil, nil, plans.Create, false),
				}, nil, nil, nil, plans.Create, false),
			},
		},
		"delete_block": {
			before: map[string]interface{}{
				"block_one": map[string]interface{}{
					"attribute_one": "old",
				},
			},
			after: map[string]interface{}{},
			block: &jsonprovider.Block{
				BlockTypes: map[string]*jsonprovider.BlockType{
					"block_one": {
						Block: &jsonprovider.Block{
							Attributes: map[string]*jsonprovider.Attribute{
								"attribute_one": {
									AttributeType: unmarshalType(t, cty.String),
								},
							},
						},
						NestingMode: "single",
					},
				},
			},
			validate: renderers.ValidateBlock(nil, map[string]renderers.ValidateDiffFunction{
				"block_one": renderers.ValidateBlock(map[string]renderers.ValidateDiffFunction{
					"attribute_one": renderers.ValidatePrimitive("old", nil, plans.Delete, false),
				}, nil, nil, nil, nil, plans.Delete, false),
			}, nil, nil, nil, plans.Update, false),
			validateSet: []renderers.ValidateDiffFunction{
				renderers.ValidateBlock(nil, map[string]renderers.ValidateDiffFunction{
					"block_one": renderers.ValidateBlock(map[string]renderers.ValidateDiffFunction{
						"attribute_one": renderers.ValidatePrimitive("old", nil, plans.Delete, false),
					}, nil, nil, nil, nil, plans.Delete, false),
				}, nil, nil, nil, plans.Delete, false),
				renderers.ValidateBlock(nil, nil, nil, nil, nil, plans.Create, false),
			},
		},
	}
	for name, tmp := range tcs {
		tc := tmp

		t.Run(name, func(t *testing.T) {
			t.Run("single", func(t *testing.T) {
				input := Change{
					Before: map[string]interface{}{
						"block_type": tc.before,
					},
					After: map[string]interface{}{
						"block_type": tc.after,
					},
				}

				block := &jsonprovider.Block{
					BlockTypes: map[string]*jsonprovider.BlockType{
						"block_type": {
							Block:       tc.block,
							NestingMode: "single",
						},
					},
				}

				validate := renderers.ValidateBlock(nil, map[string]renderers.ValidateDiffFunction{
					"block_type": tc.validate,
				}, nil, nil, nil, plans.Update, false)
				validate(t, input.ComputeDiffForBlock(block))
			})
			t.Run("map", func(t *testing.T) {
				input := Change{
					Before: map[string]interface{}{
						"block_type": map[string]interface{}{
							"one": tc.before,
						},
					},
					After: map[string]interface{}{
						"block_type": map[string]interface{}{
							"one": tc.after,
						},
					},
				}

				block := &jsonprovider.Block{
					BlockTypes: map[string]*jsonprovider.BlockType{
						"block_type": {
							Block:       tc.block,
							NestingMode: "map",
						},
					},
				}

				validate := renderers.ValidateBlock(nil, nil, nil, map[string]map[string]renderers.ValidateDiffFunction{
					"block_type": {
						"one": tc.validate,
					},
				}, nil, plans.Update, false)
				validate(t, input.ComputeDiffForBlock(block))
			})
			t.Run("list", func(t *testing.T) {
				input := Change{
					Before: map[string]interface{}{
						"block_type": []interface{}{
							tc.before,
						},
					},
					After: map[string]interface{}{
						"block_type": []interface{}{
							tc.after,
						},
					},
				}

				block := &jsonprovider.Block{
					BlockTypes: map[string]*jsonprovider.BlockType{
						"block_type": {
							Block:       tc.block,
							NestingMode: "list",
						},
					},
				}

				validate := renderers.ValidateBlock(nil, nil, map[string][]renderers.ValidateDiffFunction{
					"block_type": {
						tc.validate,
					},
				}, nil, nil, plans.Update, false)
				validate(t, input.ComputeDiffForBlock(block))
			})
			t.Run("set", func(t *testing.T) {
				input := Change{
					Before: map[string]interface{}{
						"block_type": []interface{}{
							tc.before,
						},
					},
					After: map[string]interface{}{
						"block_type": []interface{}{
							tc.after,
						},
					},
				}

				block := &jsonprovider.Block{
					BlockTypes: map[string]*jsonprovider.BlockType{
						"block_type": {
							Block:       tc.block,
							NestingMode: "set",
						},
					},
				}

				validate := renderers.ValidateBlock(nil, nil, nil, nil, map[string][]renderers.ValidateDiffFunction{
					"block_type": func() []renderers.ValidateDiffFunction {
						if tc.validateSet != nil {
							return tc.validateSet
						}
						return []renderers.ValidateDiffFunction{tc.validate}
					}(),
				}, plans.Update, false)
				validate(t, input.ComputeDiffForBlock(block))
			})
		})
	}
}

func TestValue_Outputs(t *testing.T) {
	tcs := map[string]struct {
		input        Change
		validateDiff renderers.ValidateDiffFunction
	}{
		"primitive_create": {
			input: Change{
				Before: nil,
				After:  "new",
			},
			validateDiff: renderers.ValidatePrimitive(nil, "new", plans.Create, false),
		},
		"object_create": {
			input: Change{
				Before: nil,
				After: map[string]interface{}{
					"element_one": "new_one",
					"element_two": "new_two",
				},
			},
			validateDiff: renderers.ValidateObject(map[string]renderers.ValidateDiffFunction{
				"element_one": renderers.ValidatePrimitive(nil, "new_one", plans.Create, false),
				"element_two": renderers.ValidatePrimitive(nil, "new_two", plans.Create, false),
			}, plans.Create, false),
		},
		"list_create": {
			input: Change{
				Before: nil,
				After: []interface{}{
					"new_one",
					"new_two",
				},
			},
			validateDiff: renderers.ValidateList([]renderers.ValidateDiffFunction{
				renderers.ValidatePrimitive(nil, "new_one", plans.Create, false),
				renderers.ValidatePrimitive(nil, "new_two", plans.Create, false),
			}, plans.Create, false),
		},
		"primitive_update": {
			input: Change{
				Before: "old",
				After:  "new",
			},
			validateDiff: renderers.ValidatePrimitive("old", "new", plans.Update, false),
		},
		"object_update": {
			input: Change{
				Before: map[string]interface{}{
					"element_one": "old_one",
					"element_two": "old_two",
				},
				After: map[string]interface{}{
					"element_one": "new_one",
					"element_two": "new_two",
				},
			},
			validateDiff: renderers.ValidateObject(map[string]renderers.ValidateDiffFunction{
				"element_one": renderers.ValidatePrimitive("old_one", "new_one", plans.Update, false),
				"element_two": renderers.ValidatePrimitive("old_two", "new_two", plans.Update, false),
			}, plans.Update, false),
		},
		"list_update": {
			input: Change{
				Before: []interface{}{
					"old_one",
					"old_two",
				},
				After: []interface{}{
					"new_one",
					"new_two",
				},
			},
			validateDiff: renderers.ValidateList([]renderers.ValidateDiffFunction{
				renderers.ValidatePrimitive("old_one", nil, plans.Delete, false),
				renderers.ValidatePrimitive("old_two", nil, plans.Delete, false),
				renderers.ValidatePrimitive(nil, "new_one", plans.Create, false),
				renderers.ValidatePrimitive(nil, "new_two", plans.Create, false),
			}, plans.Update, false),
		},
		"primitive_delete": {
			input: Change{
				Before: "old",
				After:  nil,
			},
			validateDiff: renderers.ValidatePrimitive("old", nil, plans.Delete, false),
		},
		"object_delete": {
			input: Change{
				Before: map[string]interface{}{
					"element_one": "old_one",
					"element_two": "old_two",
				},
				After: nil,
			},
			validateDiff: renderers.ValidateObject(map[string]renderers.ValidateDiffFunction{
				"element_one": renderers.ValidatePrimitive("old_one", nil, plans.Delete, false),
				"element_two": renderers.ValidatePrimitive("old_two", nil, plans.Delete, false),
			}, plans.Delete, false),
		},
		"list_delete": {
			input: Change{
				Before: []interface{}{
					"old_one",
					"old_two",
				},
				After: nil,
			},
			validateDiff: renderers.ValidateList([]renderers.ValidateDiffFunction{
				renderers.ValidatePrimitive("old_one", nil, plans.Delete, false),
				renderers.ValidatePrimitive("old_two", nil, plans.Delete, false),
			}, plans.Delete, false),
		},
		"primitive_to_list": {
			input: Change{
				Before: "old",
				After: []interface{}{
					"new_one",
					"new_two",
				},
			},
			validateDiff: renderers.ValidateTypeChange(
				renderers.ValidatePrimitive("old", nil, plans.Delete, false),
				renderers.ValidateList([]renderers.ValidateDiffFunction{
					renderers.ValidatePrimitive(nil, "new_one", plans.Create, false),
					renderers.ValidatePrimitive(nil, "new_two", plans.Create, false),
				}, plans.Create, false), plans.Update, false),
		},
		"primitive_to_object": {
			input: Change{
				Before: "old",
				After: map[string]interface{}{
					"element_one": "new_one",
					"element_two": "new_two",
				},
			},
			validateDiff: renderers.ValidateTypeChange(
				renderers.ValidatePrimitive("old", nil, plans.Delete, false),
				renderers.ValidateObject(map[string]renderers.ValidateDiffFunction{
					"element_one": renderers.ValidatePrimitive(nil, "new_one", plans.Create, false),
					"element_two": renderers.ValidatePrimitive(nil, "new_two", plans.Create, false),
				}, plans.Create, false), plans.Update, false),
		},
		"list_to_primitive": {
			input: Change{
				Before: []interface{}{
					"old_one",
					"old_two",
				},
				After: "new",
			},
			validateDiff: renderers.ValidateTypeChange(
				renderers.ValidateList([]renderers.ValidateDiffFunction{
					renderers.ValidatePrimitive("old_one", nil, plans.Delete, false),
					renderers.ValidatePrimitive("old_two", nil, plans.Delete, false),
				}, plans.Delete, false),
				renderers.ValidatePrimitive(nil, "new", plans.Create, false),
				plans.Update, false),
		},
		"list_to_object": {
			input: Change{
				Before: []interface{}{
					"old_one",
					"old_two",
				},
				After: map[string]interface{}{
					"element_one": "new_one",
					"element_two": "new_two",
				},
			},
			validateDiff: renderers.ValidateTypeChange(
				renderers.ValidateList([]renderers.ValidateDiffFunction{
					renderers.ValidatePrimitive("old_one", nil, plans.Delete, false),
					renderers.ValidatePrimitive("old_two", nil, plans.Delete, false),
				}, plans.Delete, false),
				renderers.ValidateObject(map[string]renderers.ValidateDiffFunction{
					"element_one": renderers.ValidatePrimitive(nil, "new_one", plans.Create, false),
					"element_two": renderers.ValidatePrimitive(nil, "new_two", plans.Create, false),
				}, plans.Create, false), plans.Update, false),
		},
		"object_to_primitive": {
			input: Change{
				Before: map[string]interface{}{
					"element_one": "old_one",
					"element_two": "old_two",
				},
				After: "new",
			},
			validateDiff: renderers.ValidateTypeChange(
				renderers.ValidateObject(map[string]renderers.ValidateDiffFunction{
					"element_one": renderers.ValidatePrimitive("old_one", nil, plans.Delete, false),
					"element_two": renderers.ValidatePrimitive("old_two", nil, plans.Delete, false),
				}, plans.Delete, false),
				renderers.ValidatePrimitive(nil, "new", plans.Create, false),
				plans.Update, false),
		},
		"object_to_list": {
			input: Change{
				Before: map[string]interface{}{
					"element_one": "old_one",
					"element_two": "old_two",
				},
				After: []interface{}{
					"new_one",
					"new_two",
				},
			},
			validateDiff: renderers.ValidateTypeChange(
				renderers.ValidateObject(map[string]renderers.ValidateDiffFunction{
					"element_one": renderers.ValidatePrimitive("old_one", nil, plans.Delete, false),
					"element_two": renderers.ValidatePrimitive("old_two", nil, plans.Delete, false),
				}, plans.Delete, false),
				renderers.ValidateList([]renderers.ValidateDiffFunction{
					renderers.ValidatePrimitive(nil, "new_one", plans.Create, false),
					renderers.ValidatePrimitive(nil, "new_two", plans.Create, false),
				}, plans.Create, false), plans.Update, false),
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			tc.validateDiff(t, tc.input.ComputeDiffForOutput())
		})
	}
}

func TestValue_PrimitiveAttributes(t *testing.T) {
	// This function tests manipulating primitives: creating, deleting and
	// updating. It also automatically tests these operations within the
	// contexts of collections.

	tcs := map[string]struct {
		input              Change
		attribute          cty.Type
		validateDiff       renderers.ValidateDiffFunction
		validateSliceDiffs []renderers.ValidateDiffFunction // Lists are special in some cases.
	}{
		"primitive_create": {
			input: Change{
				After: "new",
			},
			attribute:    cty.String,
			validateDiff: renderers.ValidatePrimitive(nil, "new", plans.Create, false),
		},
		"primitive_delete": {
			input: Change{
				Before: "old",
			},
			attribute:    cty.String,
			validateDiff: renderers.ValidatePrimitive("old", nil, plans.Delete, false),
		},
		"primitive_update": {
			input: Change{
				Before: "old",
				After:  "new",
			},
			attribute:    cty.String,
			validateDiff: renderers.ValidatePrimitive("old", "new", plans.Update, false),
			validateSliceDiffs: []renderers.ValidateDiffFunction{
				renderers.ValidatePrimitive("old", nil, plans.Delete, false),
				renderers.ValidatePrimitive(nil, "new", plans.Create, false),
			},
		},
		"primitive_set_explicit_null": {
			input: Change{
				Before:        "old",
				After:         nil,
				AfterExplicit: true,
			},
			attribute:    cty.String,
			validateDiff: renderers.ValidatePrimitive("old", nil, plans.Update, false),
			validateSliceDiffs: []renderers.ValidateDiffFunction{
				renderers.ValidatePrimitive("old", nil, plans.Delete, false),
				renderers.ValidatePrimitive(nil, nil, plans.Create, false),
			},
		},
		"primitive_unset_explicit_null": {
			input: Change{
				BeforeExplicit: true,
				Before:         nil,
				After:          "new",
			},
			attribute:    cty.String,
			validateDiff: renderers.ValidatePrimitive(nil, "new", plans.Update, false),
			validateSliceDiffs: []renderers.ValidateDiffFunction{
				renderers.ValidatePrimitive(nil, nil, plans.Delete, false),
				renderers.ValidatePrimitive(nil, "new", plans.Create, false),
			},
		},
		"primitive_create_sensitive": {
			input: Change{
				Before:         nil,
				After:          "new",
				AfterSensitive: true,
			},
			attribute:    cty.String,
			validateDiff: renderers.ValidateSensitive(renderers.ValidatePrimitive(nil, "new", plans.Create, false), false, true, plans.Create, false),
		},
		"primitive_delete_sensitive": {
			input: Change{
				Before:          "old",
				BeforeSensitive: true,
				After:           nil,
			},
			attribute:    cty.String,
			validateDiff: renderers.ValidateSensitive(renderers.ValidatePrimitive("old", nil, plans.Delete, false), true, false, plans.Delete, false),
		},
		"primitive_update_sensitive": {
			input: Change{
				Before:          "old",
				BeforeSensitive: true,
				After:           "new",
				AfterSensitive:  true,
			},
			attribute:    cty.String,
			validateDiff: renderers.ValidateSensitive(renderers.ValidatePrimitive("old", "new", plans.Update, false), true, true, plans.Update, false),
			validateSliceDiffs: []renderers.ValidateDiffFunction{
				renderers.ValidateSensitive(renderers.ValidatePrimitive("old", nil, plans.Delete, false), true, false, plans.Delete, false),
				renderers.ValidateSensitive(renderers.ValidatePrimitive(nil, "new", plans.Create, false), false, true, plans.Create, false),
			},
		},
		"primitive_create_computed": {
			input: Change{
				Before:  nil,
				After:   nil,
				Unknown: true,
			},
			attribute:    cty.String,
			validateDiff: renderers.ValidateUnknown(nil, plans.Create, false),
		},
		"primitive_update_computed": {
			input: Change{
				Before:  "old",
				After:   nil,
				Unknown: true,
			},
			attribute:    cty.String,
			validateDiff: renderers.ValidateUnknown(renderers.ValidatePrimitive("old", nil, plans.Delete, false), plans.Update, false),
			validateSliceDiffs: []renderers.ValidateDiffFunction{
				renderers.ValidatePrimitive("old", nil, plans.Delete, false),
				renderers.ValidateUnknown(nil, plans.Create, false),
			},
		},
		"primitive_update_replace": {
			input: Change{
				Before: "old",
				After:  "new",
				ReplacePaths: replace.ForcesReplacement{
					ReplacePaths: [][]interface{}{
						{}, // An empty path suggests replace should be true.
					},
				},
			},
			attribute:    cty.String,
			validateDiff: renderers.ValidatePrimitive("old", "new", plans.Update, true),
			validateSliceDiffs: []renderers.ValidateDiffFunction{
				renderers.ValidatePrimitive("old", nil, plans.Delete, true),
				renderers.ValidatePrimitive(nil, "new", plans.Create, false),
			},
		},
		"noop": {
			input: Change{
				Before: "old",
				After:  "old",
			},
			attribute:    cty.String,
			validateDiff: renderers.ValidatePrimitive("old", "old", plans.NoOp, false),
		},
		"dynamic": {
			input: Change{
				Before: "old",
				After:  "new",
			},
			attribute:    cty.DynamicPseudoType,
			validateDiff: renderers.ValidatePrimitive("old", "new", plans.Update, false),
			validateSliceDiffs: []renderers.ValidateDiffFunction{
				renderers.ValidatePrimitive("old", nil, plans.Delete, false),
				renderers.ValidatePrimitive(nil, "new", plans.Create, false),
			},
		},
		"dynamic_type_change": {
			input: Change{
				Before: "old",
				After:  4.0,
			},
			attribute: cty.DynamicPseudoType,
			validateDiff: renderers.ValidateTypeChange(
				renderers.ValidatePrimitive("old", nil, plans.Delete, false),
				renderers.ValidatePrimitive(nil, 4.0, plans.Create, false),
				plans.Update, false),
			validateSliceDiffs: []renderers.ValidateDiffFunction{
				renderers.ValidatePrimitive("old", nil, plans.Delete, false),
				renderers.ValidatePrimitive(nil, 4.0, plans.Create, false),
			},
		},
	}
	for name, tmp := range tcs {
		tc := tmp

		defaultCollectionsAction := plans.Update
		if name == "noop" {
			defaultCollectionsAction = plans.NoOp
		}

		t.Run(name, func(t *testing.T) {
			t.Run("direct", func(t *testing.T) {
				tc.validateDiff(t, tc.input.ComputeDiffForAttribute(&jsonprovider.Attribute{
					AttributeType: unmarshalType(t, tc.attribute),
				}))
			})

			t.Run("map", func(t *testing.T) {
				input := wrapChangeInMap(tc.input)
				attribute := &jsonprovider.Attribute{
					AttributeType: unmarshalType(t, cty.Map(tc.attribute)),
				}

				validate := renderers.ValidateMap(map[string]renderers.ValidateDiffFunction{
					"element": tc.validateDiff,
				}, defaultCollectionsAction, false)
				validate(t, input.ComputeDiffForAttribute(attribute))
			})

			t.Run("list", func(t *testing.T) {
				input := wrapChangeInSlice(tc.input)
				attribute := &jsonprovider.Attribute{
					AttributeType: unmarshalType(t, cty.List(tc.attribute)),
				}

				if tc.validateSliceDiffs != nil {
					validate := renderers.ValidateList(tc.validateSliceDiffs, defaultCollectionsAction, false)
					validate(t, input.ComputeDiffForAttribute(attribute))
					return
				}

				validate := renderers.ValidateList([]renderers.ValidateDiffFunction{
					tc.validateDiff,
				}, defaultCollectionsAction, false)
				validate(t, input.ComputeDiffForAttribute(attribute))
			})

			t.Run("set", func(t *testing.T) {
				input := wrapChangeInSlice(tc.input)
				attribute := &jsonprovider.Attribute{
					AttributeType: unmarshalType(t, cty.Set(tc.attribute)),
				}

				if tc.validateSliceDiffs != nil {
					validate := renderers.ValidateSet(tc.validateSliceDiffs, defaultCollectionsAction, false)
					validate(t, input.ComputeDiffForAttribute(attribute))
					return
				}

				validate := renderers.ValidateSet([]renderers.ValidateDiffFunction{
					tc.validateDiff,
				}, defaultCollectionsAction, false)
				validate(t, input.ComputeDiffForAttribute(attribute))
			})
		})
	}
}

func TestValue_CollectionAttributes(t *testing.T) {
	// This function tests creating and deleting collections. Note, it does not
	// generally cover editing collections except in special cases as editing
	// collections is handled automatically by other functions.
	tcs := map[string]struct {
		input        Change
		attribute    *jsonprovider.Attribute
		validateDiff renderers.ValidateDiffFunction
	}{
		"map_create_empty": {
			input: Change{
				Before: nil,
				After:  map[string]interface{}{},
			},
			attribute: &jsonprovider.Attribute{
				AttributeType: unmarshalType(t, cty.Map(cty.String)),
			},
			validateDiff: renderers.ValidateMap(nil, plans.Create, false),
		},
		"map_create_populated": {
			input: Change{
				Before: nil,
				After: map[string]interface{}{
					"element_one": "one",
					"element_two": "two",
				},
			},
			attribute: &jsonprovider.Attribute{
				AttributeType: unmarshalType(t, cty.Map(cty.String)),
			},
			validateDiff: renderers.ValidateMap(map[string]renderers.ValidateDiffFunction{
				"element_one": renderers.ValidatePrimitive(nil, "one", plans.Create, false),
				"element_two": renderers.ValidatePrimitive(nil, "two", plans.Create, false),
			}, plans.Create, false),
		},
		"map_delete_empty": {
			input: Change{
				Before: map[string]interface{}{},
				After:  nil,
			},
			attribute: &jsonprovider.Attribute{
				AttributeType: unmarshalType(t, cty.Map(cty.String)),
			},
			validateDiff: renderers.ValidateMap(nil, plans.Delete, false),
		},
		"map_delete_populated": {
			input: Change{
				Before: map[string]interface{}{
					"element_one": "one",
					"element_two": "two",
				},
				After: nil,
			},
			attribute: &jsonprovider.Attribute{
				AttributeType: unmarshalType(t, cty.Map(cty.String)),
			},
			validateDiff: renderers.ValidateMap(map[string]renderers.ValidateDiffFunction{
				"element_one": renderers.ValidatePrimitive("one", nil, plans.Delete, false),
				"element_two": renderers.ValidatePrimitive("two", nil, plans.Delete, false),
			}, plans.Delete, false),
		},
		"map_create_sensitive": {
			input: Change{
				Before:         nil,
				After:          map[string]interface{}{},
				AfterSensitive: true,
			},
			attribute: &jsonprovider.Attribute{
				AttributeType: unmarshalType(t, cty.Map(cty.String)),
			},
			validateDiff: renderers.ValidateSensitive(renderers.ValidateMap(nil, plans.Create, false), false, true, plans.Create, false),
		},
		"map_update_sensitive": {
			input: Change{
				Before: map[string]interface{}{
					"element": "one",
				},
				BeforeSensitive: true,
				After:           map[string]interface{}{},
				AfterSensitive:  true,
			},
			attribute: &jsonprovider.Attribute{
				AttributeType: unmarshalType(t, cty.Map(cty.String)),
			},
			validateDiff: renderers.ValidateSensitive(renderers.ValidateMap(map[string]renderers.ValidateDiffFunction{
				"element": renderers.ValidatePrimitive("one", nil, plans.Delete, false),
			}, plans.Update, false), true, true, plans.Update, false),
		},
		"map_delete_sensitive": {
			input: Change{
				Before:          map[string]interface{}{},
				BeforeSensitive: true,
				After:           nil,
			},
			attribute: &jsonprovider.Attribute{
				AttributeType: unmarshalType(t, cty.Map(cty.String)),
			},
			validateDiff: renderers.ValidateSensitive(renderers.ValidateMap(nil, plans.Delete, false), true, false, plans.Delete, false),
		},
		"map_create_unknown": {
			input: Change{
				Before:  nil,
				After:   map[string]interface{}{},
				Unknown: true,
			},
			attribute: &jsonprovider.Attribute{
				AttributeType: unmarshalType(t, cty.Map(cty.String)),
			},
			validateDiff: renderers.ValidateUnknown(nil, plans.Create, false),
		},
		"map_update_unknown": {
			input: Change{
				Before: map[string]interface{}{},
				After: map[string]interface{}{
					"element": "one",
				},
				Unknown: true,
			},
			attribute: &jsonprovider.Attribute{
				AttributeType: unmarshalType(t, cty.Map(cty.String)),
			},
			validateDiff: renderers.ValidateUnknown(renderers.ValidateMap(nil, plans.Delete, false), plans.Update, false),
		},
		"list_create_empty": {
			input: Change{
				Before: nil,
				After:  []interface{}{},
			},
			attribute: &jsonprovider.Attribute{
				AttributeType: unmarshalType(t, cty.List(cty.String)),
			},
			validateDiff: renderers.ValidateList(nil, plans.Create, false),
		},
		"list_create_populated": {
			input: Change{
				Before: nil,
				After:  []interface{}{"one", "two"},
			},
			attribute: &jsonprovider.Attribute{
				AttributeType: unmarshalType(t, cty.List(cty.String)),
			},
			validateDiff: renderers.ValidateList([]renderers.ValidateDiffFunction{
				renderers.ValidatePrimitive(nil, "one", plans.Create, false),
				renderers.ValidatePrimitive(nil, "two", plans.Create, false),
			}, plans.Create, false),
		},
		"list_delete_empty": {
			input: Change{
				Before: []interface{}{},
				After:  nil,
			},
			attribute: &jsonprovider.Attribute{
				AttributeType: unmarshalType(t, cty.List(cty.String)),
			},
			validateDiff: renderers.ValidateList(nil, plans.Delete, false),
		},
		"list_delete_populated": {
			input: Change{
				Before: []interface{}{"one", "two"},
				After:  nil,
			},
			attribute: &jsonprovider.Attribute{
				AttributeType: unmarshalType(t, cty.List(cty.String)),
			},
			validateDiff: renderers.ValidateList([]renderers.ValidateDiffFunction{
				renderers.ValidatePrimitive("one", nil, plans.Delete, false),
				renderers.ValidatePrimitive("two", nil, plans.Delete, false),
			}, plans.Delete, false),
		},
		"list_create_sensitive": {
			input: Change{
				Before:         nil,
				After:          []interface{}{},
				AfterSensitive: true,
			},
			attribute: &jsonprovider.Attribute{
				AttributeType: unmarshalType(t, cty.List(cty.String)),
			},
			validateDiff: renderers.ValidateSensitive(renderers.ValidateList(nil, plans.Create, false), false, true, plans.Create, false),
		},
		"list_update_sensitive": {
			input: Change{
				Before:          []interface{}{"one"},
				BeforeSensitive: true,
				After:           []interface{}{},
				AfterSensitive:  true,
			},
			attribute: &jsonprovider.Attribute{
				AttributeType: unmarshalType(t, cty.List(cty.String)),
			},
			validateDiff: renderers.ValidateSensitive(renderers.ValidateList([]renderers.ValidateDiffFunction{
				renderers.ValidatePrimitive("one", nil, plans.Delete, false),
			}, plans.Update, false), true, true, plans.Update, false),
		},
		"list_delete_sensitive": {
			input: Change{
				Before:          []interface{}{},
				BeforeSensitive: true,
				After:           nil,
			},
			attribute: &jsonprovider.Attribute{
				AttributeType: unmarshalType(t, cty.List(cty.String)),
			},
			validateDiff: renderers.ValidateSensitive(renderers.ValidateList(nil, plans.Delete, false), true, false, plans.Delete, false),
		},
		"list_create_unknown": {
			input: Change{
				Before:  nil,
				After:   []interface{}{},
				Unknown: true,
			},
			attribute: &jsonprovider.Attribute{
				AttributeType: unmarshalType(t, cty.List(cty.String)),
			},
			validateDiff: renderers.ValidateUnknown(nil, plans.Create, false),
		},
		"list_update_unknown": {
			input: Change{
				Before:  []interface{}{},
				After:   []interface{}{"one"},
				Unknown: true,
			},
			attribute: &jsonprovider.Attribute{
				AttributeType: unmarshalType(t, cty.List(cty.String)),
			},
			validateDiff: renderers.ValidateUnknown(renderers.ValidateList(nil, plans.Delete, false), plans.Update, false),
		},
		"set_create_empty": {
			input: Change{
				Before: nil,
				After:  []interface{}{},
			},
			attribute: &jsonprovider.Attribute{
				AttributeType: unmarshalType(t, cty.Set(cty.String)),
			},
			validateDiff: renderers.ValidateSet(nil, plans.Create, false),
		},
		"set_create_populated": {
			input: Change{
				Before: nil,
				After:  []interface{}{"one", "two"},
			},
			attribute: &jsonprovider.Attribute{
				AttributeType: unmarshalType(t, cty.Set(cty.String)),
			},
			validateDiff: renderers.ValidateSet([]renderers.ValidateDiffFunction{
				renderers.ValidatePrimitive(nil, "one", plans.Create, false),
				renderers.ValidatePrimitive(nil, "two", plans.Create, false),
			}, plans.Create, false),
		},
		"set_delete_empty": {
			input: Change{
				Before: []interface{}{},
				After:  nil,
			},
			attribute: &jsonprovider.Attribute{
				AttributeType: unmarshalType(t, cty.Set(cty.String)),
			},
			validateDiff: renderers.ValidateSet(nil, plans.Delete, false),
		},
		"set_delete_populated": {
			input: Change{
				Before: []interface{}{"one", "two"},
				After:  nil,
			},
			attribute: &jsonprovider.Attribute{
				AttributeType: unmarshalType(t, cty.Set(cty.String)),
			},
			validateDiff: renderers.ValidateSet([]renderers.ValidateDiffFunction{
				renderers.ValidatePrimitive("one", nil, plans.Delete, false),
				renderers.ValidatePrimitive("two", nil, plans.Delete, false),
			}, plans.Delete, false),
		},
		"set_create_sensitive": {
			input: Change{
				Before:         nil,
				After:          []interface{}{},
				AfterSensitive: true,
			},
			attribute: &jsonprovider.Attribute{
				AttributeType: unmarshalType(t, cty.Set(cty.String)),
			},
			validateDiff: renderers.ValidateSensitive(renderers.ValidateSet(nil, plans.Create, false), false, true, plans.Create, false),
		},
		"set_update_sensitive": {
			input: Change{
				Before:          []interface{}{"one"},
				BeforeSensitive: true,
				After:           []interface{}{},
				AfterSensitive:  true,
			},
			attribute: &jsonprovider.Attribute{
				AttributeType: unmarshalType(t, cty.Set(cty.String)),
			},
			validateDiff: renderers.ValidateSensitive(renderers.ValidateSet([]renderers.ValidateDiffFunction{
				renderers.ValidatePrimitive("one", nil, plans.Delete, false),
			}, plans.Update, false), true, true, plans.Update, false),
		},
		"set_delete_sensitive": {
			input: Change{
				Before:          []interface{}{},
				BeforeSensitive: true,
				After:           nil,
			},
			attribute: &jsonprovider.Attribute{
				AttributeType: unmarshalType(t, cty.Set(cty.String)),
			},
			validateDiff: renderers.ValidateSensitive(renderers.ValidateSet(nil, plans.Delete, false), true, false, plans.Delete, false),
		},
		"set_create_unknown": {
			input: Change{
				Before:  nil,
				After:   []interface{}{},
				Unknown: true,
			},
			attribute: &jsonprovider.Attribute{
				AttributeType: unmarshalType(t, cty.Set(cty.String)),
			},
			validateDiff: renderers.ValidateUnknown(nil, plans.Create, false),
		},
		"set_update_unknown": {
			input: Change{
				Before:  []interface{}{},
				After:   []interface{}{"one"},
				Unknown: true,
			},
			attribute: &jsonprovider.Attribute{
				AttributeType: unmarshalType(t, cty.Set(cty.String)),
			},
			validateDiff: renderers.ValidateUnknown(renderers.ValidateSet(nil, plans.Delete, false), plans.Update, false),
		},
		"tuple_primitive": {
			input: Change{
				Before: []interface{}{
					"one",
					2.0,
					"three",
				},
				After: []interface{}{
					"one",
					4.0,
					"three",
				},
			},
			attribute: &jsonprovider.Attribute{
				AttributeType: unmarshalType(t, cty.Tuple([]cty.Type{cty.String, cty.Number, cty.String})),
			},
			validateDiff: renderers.ValidateList([]renderers.ValidateDiffFunction{
				renderers.ValidatePrimitive("one", "one", plans.NoOp, false),
				renderers.ValidatePrimitive(2.0, 4.0, plans.Update, false),
				renderers.ValidatePrimitive("three", "three", plans.NoOp, false),
			}, plans.Update, false),
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			tc.validateDiff(t, tc.input.ComputeDiffForAttribute(tc.attribute))
		})
	}
}

// unmarshalType converts a cty.Type into a json.RawMessage understood by the
// schema. It also lets the testing framework handle any errors to keep the API
// clean.
func unmarshalType(t *testing.T, ctyType cty.Type) json.RawMessage {
	msg, err := ctyjson.MarshalType(ctyType)
	if err != nil {
		t.Fatalf("invalid type: %s", ctyType.FriendlyName())
	}
	return msg
}

// wrapChangeInSlice does the same as wrapChangeInMap, except it wraps it into a
// slice internally.
func wrapChangeInSlice(input Change) Change {
	return wrapChange(input, float64(0), func(value interface{}, unknown interface{}, explicit bool) interface{} {
		switch value.(type) {
		case nil:
			if set, ok := unknown.(bool); (set && ok) || explicit {
				return []interface{}{nil}

			}
			return []interface{}{}
		default:
			return []interface{}{value}
		}
	})
}

// wrapChangeInMap access a single Change and returns a new Change that represents
// a map with a single element. That single element is the input value.
func wrapChangeInMap(input Change) Change {
	return wrapChange(input, "element", func(value interface{}, unknown interface{}, explicit bool) interface{} {
		switch value.(type) {
		case nil:
			if set, ok := unknown.(bool); (set && ok) || explicit {
				return map[string]interface{}{
					"element": nil,
				}
			}
			return map[string]interface{}{}
		default:
			return map[string]interface{}{
				"element": value,
			}
		}
	})
}

func wrapChange(input Change, step interface{}, wrap func(interface{}, interface{}, bool) interface{}) Change {

	replacePaths := replace.ForcesReplacement{}
	for _, path := range input.ReplacePaths.ReplacePaths {
		var updated []interface{}
		updated = append(updated, step)
		updated = append(updated, path...)
		replacePaths.ReplacePaths = append(replacePaths.ReplacePaths, updated)
	}

	return Change{
		Before:          wrap(input.Before, nil, input.BeforeExplicit),
		After:           wrap(input.After, input.Unknown, input.AfterExplicit),
		Unknown:         wrap(input.Unknown, nil, false),
		BeforeSensitive: wrap(input.BeforeSensitive, nil, false),
		AfterSensitive:  wrap(input.AfterSensitive, nil, false),
		ReplacePaths:    replacePaths,
	}
}
