package differ

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"

	"github.com/hashicorp/terraform/internal/command/jsonformat/change"
	"github.com/hashicorp/terraform/internal/command/jsonprovider"
	"github.com/hashicorp/terraform/internal/plans"
)

type SetChange struct {
	Before SetChangeEntry
	After  SetChangeEntry
}

type SetChangeEntry struct {
	SingleChange change.ValidateChangeFunc
	ObjectChange map[string]change.ValidateChangeFunc

	Replace bool
	Action  plans.Action
}

func (entry SetChangeEntry) Validate(obj func(attributes map[string]change.ValidateChangeFunc, action plans.Action, replace bool) change.ValidateChangeFunc) change.ValidateChangeFunc {
	if entry.SingleChange != nil {
		return entry.SingleChange
	}

	return obj(entry.ObjectChange, entry.Action, entry.Replace)
}

func TestValue_ObjectAttributes(t *testing.T) {
	// This function holds a range of test cases creating, deleting and editing
	// objects. It is built in such a way that it can automatically test these
	// operations on objects both directly and nested, as well as within all
	// types of collections.

	tcs := map[string]struct {
		input                Value
		attributes           map[string]cty.Type
		validateSingleChange change.ValidateChangeFunc
		validateObject       change.ValidateChangeFunc
		validateNestedObject change.ValidateChangeFunc
		validateChanges      map[string]change.ValidateChangeFunc
		validateReplace      bool
		validateAction       plans.Action
		// Sets break changes out differently to the other collections, so they
		// have their own entry.
		validateSetChanges *SetChange
	}{
		"create": {
			input: Value{
				Before: nil,
				After: map[string]interface{}{
					"attribute_one": "new",
				},
			},
			attributes: map[string]cty.Type{
				"attribute_one": cty.String,
			},
			validateChanges: map[string]change.ValidateChangeFunc{
				"attribute_one": change.ValidatePrimitive(nil, strptr("\"new\""), plans.Create, false),
			},
			validateAction:  plans.Create,
			validateReplace: false,
		},
		"delete": {
			input: Value{
				Before: map[string]interface{}{
					"attribute_one": "old",
				},
				After: nil,
			},
			attributes: map[string]cty.Type{
				"attribute_one": cty.String,
			},
			validateChanges: map[string]change.ValidateChangeFunc{
				"attribute_one": change.ValidatePrimitive(strptr("\"old\""), nil, plans.Delete, false),
			},
			validateAction:  plans.Delete,
			validateReplace: false,
		},
		"create_sensitive": {
			input: Value{
				Before: nil,
				After: map[string]interface{}{
					"attribute_one": "new",
				},
				AfterSensitive: true,
			},
			attributes: map[string]cty.Type{
				"attribute_one": cty.String,
			},
			validateSingleChange: change.ValidateSensitive(nil, map[string]interface{}{
				"attribute_one": "new",
			}, false, true, plans.Create, false),
		},
		"delete_sensitive": {
			input: Value{
				Before: map[string]interface{}{
					"attribute_one": "old",
				},
				BeforeSensitive: true,
				After:           nil,
			},
			attributes: map[string]cty.Type{
				"attribute_one": cty.String,
			},
			validateSingleChange: change.ValidateSensitive(map[string]interface{}{
				"attribute_one": "old",
			}, nil, true, false, plans.Delete, false),
		},
		"create_unknown": {
			input: Value{
				Before:  nil,
				After:   nil,
				Unknown: true,
			},
			attributes: map[string]cty.Type{
				"attribute_one": cty.String,
			},
			validateSingleChange: change.ValidateComputed(nil, plans.Create, false),
		},
		"update_unknown": {
			input: Value{
				Before: map[string]interface{}{
					"attribute_one": "old",
				},
				After:   nil,
				Unknown: true,
			},
			attributes: map[string]cty.Type{
				"attribute_one": cty.String,
			},
			validateObject: change.ValidateComputed(change.ValidateObject(map[string]change.ValidateChangeFunc{
				"attribute_one": change.ValidatePrimitive(strptr("\"old\""), nil, plans.Delete, false),
			}, plans.Delete, false), plans.Update, false),
			validateNestedObject: change.ValidateComputed(change.ValidateNestedObject(map[string]change.ValidateChangeFunc{
				"attribute_one": change.ValidatePrimitive(strptr("\"old\""), nil, plans.Delete, false),
			}, plans.Delete, false), plans.Update, false),
			validateSetChanges: &SetChange{
				Before: SetChangeEntry{
					ObjectChange: map[string]change.ValidateChangeFunc{
						"attribute_one": change.ValidatePrimitive(strptr("\"old\""), nil, plans.Delete, false),
					},
					Action:  plans.Delete,
					Replace: false,
				},
				After: SetChangeEntry{
					SingleChange: change.ValidateComputed(nil, plans.Create, false),
				},
			},
		},
		"create_attribute": {
			input: Value{
				Before: map[string]interface{}{},
				After: map[string]interface{}{
					"attribute_one": "new",
				},
			},
			attributes: map[string]cty.Type{
				"attribute_one": cty.String,
			},
			validateChanges: map[string]change.ValidateChangeFunc{
				"attribute_one": change.ValidatePrimitive(nil, strptr("\"new\""), plans.Create, false),
			},
			validateAction:  plans.Update,
			validateReplace: false,
			validateSetChanges: &SetChange{
				Before: SetChangeEntry{
					ObjectChange: nil,
					Action:       plans.Delete,
					Replace:      false,
				},
				After: SetChangeEntry{
					ObjectChange: map[string]change.ValidateChangeFunc{
						"attribute_one": change.ValidatePrimitive(nil, strptr("\"new\""), plans.Create, false),
					},
					Action:  plans.Create,
					Replace: false,
				},
			},
		},
		"create_attribute_from_explicit_null": {
			input: Value{
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
			validateChanges: map[string]change.ValidateChangeFunc{
				"attribute_one": change.ValidatePrimitive(nil, strptr("\"new\""), plans.Create, false),
			},
			validateAction:  plans.Update,
			validateReplace: false,
			validateSetChanges: &SetChange{
				Before: SetChangeEntry{
					ObjectChange: nil,
					Action:       plans.Delete,
					Replace:      false,
				},
				After: SetChangeEntry{
					ObjectChange: map[string]change.ValidateChangeFunc{
						"attribute_one": change.ValidatePrimitive(nil, strptr("\"new\""), plans.Create, false),
					},
					Action:  plans.Create,
					Replace: false,
				},
			},
		},
		"delete_attribute": {
			input: Value{
				Before: map[string]interface{}{
					"attribute_one": "old",
				},
				After: map[string]interface{}{},
			},
			attributes: map[string]cty.Type{
				"attribute_one": cty.String,
			},
			validateChanges: map[string]change.ValidateChangeFunc{
				"attribute_one": change.ValidatePrimitive(strptr("\"old\""), nil, plans.Delete, false),
			},
			validateAction:  plans.Update,
			validateReplace: false,
			validateSetChanges: &SetChange{
				Before: SetChangeEntry{
					ObjectChange: map[string]change.ValidateChangeFunc{
						"attribute_one": change.ValidatePrimitive(strptr("\"old\""), nil, plans.Delete, false),
					},
					Action:  plans.Delete,
					Replace: false,
				},
				After: SetChangeEntry{
					ObjectChange: nil,
					Action:       plans.Create,
					Replace:      false,
				},
			},
		},
		"delete_attribute_to_explicit_null": {
			input: Value{
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
			validateChanges: map[string]change.ValidateChangeFunc{
				"attribute_one": change.ValidatePrimitive(strptr("\"old\""), nil, plans.Delete, false),
			},
			validateAction:  plans.Update,
			validateReplace: false,
			validateSetChanges: &SetChange{
				Before: SetChangeEntry{
					ObjectChange: map[string]change.ValidateChangeFunc{
						"attribute_one": change.ValidatePrimitive(strptr("\"old\""), nil, plans.Delete, false),
					},
					Action:  plans.Delete,
					Replace: false,
				},
				After: SetChangeEntry{
					ObjectChange: nil,
					Action:       plans.Create,
					Replace:      false,
				},
			},
		},
		"update_attribute": {
			input: Value{
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
			validateChanges: map[string]change.ValidateChangeFunc{
				"attribute_one": change.ValidatePrimitive(strptr("\"old\""), strptr("\"new\""), plans.Update, false),
			},
			validateAction:  plans.Update,
			validateReplace: false,
			validateSetChanges: &SetChange{
				Before: SetChangeEntry{
					ObjectChange: map[string]change.ValidateChangeFunc{
						"attribute_one": change.ValidatePrimitive(strptr("\"old\""), nil, plans.Delete, false),
					},
					Action:  plans.Delete,
					Replace: false,
				},
				After: SetChangeEntry{
					ObjectChange: map[string]change.ValidateChangeFunc{
						"attribute_one": change.ValidatePrimitive(nil, strptr("\"new\""), plans.Create, false),
					},
					Action:  plans.Create,
					Replace: false,
				},
			},
		},
		"create_sensitive_attribute": {
			input: Value{
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
			validateChanges: map[string]change.ValidateChangeFunc{
				"attribute_one": change.ValidateSensitive(nil, "new", false, true, plans.Create, false),
			},
			validateAction:  plans.Update,
			validateReplace: false,
			validateSetChanges: &SetChange{
				Before: SetChangeEntry{
					ObjectChange: nil,
					Action:       plans.Delete,
					Replace:      false,
				},
				After: SetChangeEntry{
					ObjectChange: map[string]change.ValidateChangeFunc{
						"attribute_one": change.ValidateSensitive(nil, "new", false, true, plans.Create, false),
					},
					Action:  plans.Create,
					Replace: false,
				},
			},
		},
		"delete_sensitive_attribute": {
			input: Value{
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
			validateChanges: map[string]change.ValidateChangeFunc{
				"attribute_one": change.ValidateSensitive("old", nil, true, false, plans.Delete, false),
			},
			validateAction:  plans.Update,
			validateReplace: false,
			validateSetChanges: &SetChange{
				Before: SetChangeEntry{
					ObjectChange: map[string]change.ValidateChangeFunc{
						"attribute_one": change.ValidateSensitive("old", nil, true, false, plans.Delete, false),
					},
					Action:  plans.Delete,
					Replace: false,
				},
				After: SetChangeEntry{
					ObjectChange: nil,
					Action:       plans.Create,
					Replace:      false,
				},
			},
		},
		"update_sensitive_attribute": {
			input: Value{
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
			validateChanges: map[string]change.ValidateChangeFunc{
				"attribute_one": change.ValidateSensitive("old", "new", true, true, plans.Update, false),
			},
			validateAction:  plans.Update,
			validateReplace: false,
			validateSetChanges: &SetChange{
				Before: SetChangeEntry{
					ObjectChange: map[string]change.ValidateChangeFunc{
						"attribute_one": change.ValidateSensitive("old", nil, true, false, plans.Delete, false),
					},
					Action:  plans.Delete,
					Replace: false,
				},
				After: SetChangeEntry{
					ObjectChange: map[string]change.ValidateChangeFunc{
						"attribute_one": change.ValidateSensitive(nil, "new", false, true, plans.Create, false),
					},
					Action:  plans.Create,
					Replace: false,
				},
			},
		},
		"create_computed_attribute": {
			input: Value{
				Before: map[string]interface{}{},
				After:  map[string]interface{}{},
				Unknown: map[string]interface{}{
					"attribute_one": true,
				},
			},
			attributes: map[string]cty.Type{
				"attribute_one": cty.String,
			},
			validateChanges: map[string]change.ValidateChangeFunc{
				"attribute_one": change.ValidateComputed(nil, plans.Create, false),
			},
			validateAction:  plans.Update,
			validateReplace: false,
			validateSetChanges: &SetChange{
				Before: SetChangeEntry{
					ObjectChange: nil,
					Action:       plans.Delete,
					Replace:      false,
				},
				After: SetChangeEntry{
					ObjectChange: map[string]change.ValidateChangeFunc{
						"attribute_one": change.ValidateComputed(nil, plans.Create, false),
					},
					Action:  plans.Create,
					Replace: false,
				},
			},
		},
		"update_computed_attribute": {
			input: Value{
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
			validateChanges: map[string]change.ValidateChangeFunc{
				"attribute_one": change.ValidateComputed(
					change.ValidatePrimitive(strptr("\"old\""), nil, plans.Delete, false),
					plans.Update,
					false),
			},
			validateAction:  plans.Update,
			validateReplace: false,
			validateSetChanges: &SetChange{
				Before: SetChangeEntry{
					ObjectChange: map[string]change.ValidateChangeFunc{
						"attribute_one": change.ValidatePrimitive(strptr("\"old\""), nil, plans.Delete, false),
					},
					Action:  plans.Delete,
					Replace: false,
				},
				After: SetChangeEntry{
					ObjectChange: map[string]change.ValidateChangeFunc{
						"attribute_one": change.ValidateComputed(nil, plans.Create, false),
					},
					Action:  plans.Create,
					Replace: false,
				},
			},
		},
		"ignores_unset_fields": {
			input: Value{
				Before: map[string]interface{}{},
				After:  map[string]interface{}{},
			},
			attributes: map[string]cty.Type{
				"attribute_one": cty.String,
			},
			validateChanges: map[string]change.ValidateChangeFunc{},
			validateAction:  plans.NoOp,
			validateReplace: false,
		},
		"update_replace_self": {
			input: Value{
				Before: map[string]interface{}{
					"attribute_one": "old",
				},
				After: map[string]interface{}{
					"attribute_one": "new",
				},
				ReplacePaths: []interface{}{
					[]interface{}{},
				},
			},
			attributes: map[string]cty.Type{
				"attribute_one": cty.String,
			},
			validateChanges: map[string]change.ValidateChangeFunc{
				"attribute_one": change.ValidatePrimitive(strptr("\"old\""), strptr("\"new\""), plans.Update, false),
			},
			validateAction:  plans.Update,
			validateReplace: true,
			validateSetChanges: &SetChange{
				Before: SetChangeEntry{
					ObjectChange: map[string]change.ValidateChangeFunc{
						"attribute_one": change.ValidatePrimitive(strptr("\"old\""), nil, plans.Delete, false),
					},
					Action:  plans.Delete,
					Replace: true,
				},
				After: SetChangeEntry{
					ObjectChange: map[string]change.ValidateChangeFunc{
						"attribute_one": change.ValidatePrimitive(nil, strptr("\"new\""), plans.Create, false),
					},
					Action:  plans.Create,
					Replace: false,
				},
			},
		},
		"update_replace_attribute": {
			input: Value{
				Before: map[string]interface{}{
					"attribute_one": "old",
				},
				After: map[string]interface{}{
					"attribute_one": "new",
				},
				ReplacePaths: []interface{}{
					[]interface{}{"attribute_one"},
				},
			},
			attributes: map[string]cty.Type{
				"attribute_one": cty.String,
			},
			validateChanges: map[string]change.ValidateChangeFunc{
				"attribute_one": change.ValidatePrimitive(strptr("\"old\""), strptr("\"new\""), plans.Update, true),
			},
			validateAction:  plans.Update,
			validateReplace: false,
			validateSetChanges: &SetChange{
				Before: SetChangeEntry{
					ObjectChange: map[string]change.ValidateChangeFunc{
						"attribute_one": change.ValidatePrimitive(strptr("\"old\""), nil, plans.Delete, true),
					},
					Action:  plans.Delete,
					Replace: false,
				},
				After: SetChangeEntry{
					ObjectChange: map[string]change.ValidateChangeFunc{
						"attribute_one": change.ValidatePrimitive(nil, strptr("\"new\""), plans.Create, false),
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
					tc.validateObject(t, tc.input.ComputeChange(attribute))
					return
				}

				if tc.validateSingleChange != nil {
					tc.validateSingleChange(t, tc.input.ComputeChange(attribute))
					return
				}

				validate := change.ValidateObject(tc.validateChanges, tc.validateAction, tc.validateReplace)
				validate(t, tc.input.ComputeChange(attribute))
			})

			t.Run("map", func(t *testing.T) {
				attribute := &jsonprovider.Attribute{
					AttributeType: unmarshalType(t, cty.Map(cty.Object(tc.attributes))),
				}

				input := wrapValueInMap(tc.input)

				if tc.validateObject != nil {
					validate := change.ValidateMap(map[string]change.ValidateChangeFunc{
						"element": tc.validateObject,
					}, collectionDefaultAction, false)
					validate(t, input.ComputeChange(attribute))
					return
				}

				if tc.validateSingleChange != nil {
					validate := change.ValidateMap(map[string]change.ValidateChangeFunc{
						"element": tc.validateSingleChange,
					}, collectionDefaultAction, false)
					validate(t, input.ComputeChange(attribute))
					return
				}

				validate := change.ValidateMap(map[string]change.ValidateChangeFunc{
					"element": change.ValidateObject(tc.validateChanges, tc.validateAction, tc.validateReplace),
				}, collectionDefaultAction, false)
				validate(t, input.ComputeChange(attribute))
			})

			t.Run("list", func(t *testing.T) {
				attribute := &jsonprovider.Attribute{
					AttributeType: unmarshalType(t, cty.List(cty.Object(tc.attributes))),
				}

				input := wrapValueInSlice(tc.input)

				if tc.validateObject != nil {
					validate := change.ValidateList([]change.ValidateChangeFunc{
						tc.validateObject,
					}, collectionDefaultAction, false)
					validate(t, input.ComputeChange(attribute))
					return
				}

				if tc.validateSingleChange != nil {
					validate := change.ValidateList([]change.ValidateChangeFunc{
						tc.validateSingleChange,
					}, collectionDefaultAction, false)
					validate(t, input.ComputeChange(attribute))
					return
				}

				validate := change.ValidateList([]change.ValidateChangeFunc{
					change.ValidateObject(tc.validateChanges, tc.validateAction, tc.validateReplace),
				}, collectionDefaultAction, false)
				validate(t, input.ComputeChange(attribute))
			})

			t.Run("set", func(t *testing.T) {
				attribute := &jsonprovider.Attribute{
					AttributeType: unmarshalType(t, cty.Set(cty.Object(tc.attributes))),
				}

				input := wrapValueInSlice(tc.input)

				if tc.validateSetChanges != nil {
					validate := change.ValidateSet(func() []change.ValidateChangeFunc {
						var ret []change.ValidateChangeFunc
						ret = append(ret, tc.validateSetChanges.Before.Validate(change.ValidateObject))
						ret = append(ret, tc.validateSetChanges.After.Validate(change.ValidateObject))
						return ret
					}(), collectionDefaultAction, false)
					validate(t, input.ComputeChange(attribute))
					return
				}

				if tc.validateObject != nil {
					validate := change.ValidateSet([]change.ValidateChangeFunc{
						tc.validateObject,
					}, collectionDefaultAction, false)
					validate(t, input.ComputeChange(attribute))
					return
				}

				if tc.validateSingleChange != nil {
					validate := change.ValidateSet([]change.ValidateChangeFunc{
						tc.validateSingleChange,
					}, collectionDefaultAction, false)
					validate(t, input.ComputeChange(attribute))
					return
				}

				validate := change.ValidateSet([]change.ValidateChangeFunc{
					change.ValidateObject(tc.validateChanges, tc.validateAction, tc.validateReplace),
				}, collectionDefaultAction, false)
				validate(t, input.ComputeChange(attribute))
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
					tc.validateNestedObject(t, tc.input.ComputeChange(attribute))
					return
				}

				if tc.validateSingleChange != nil {
					tc.validateSingleChange(t, tc.input.ComputeChange(attribute))
					return
				}

				validate := change.ValidateNestedObject(tc.validateChanges, tc.validateAction, tc.validateReplace)
				validate(t, tc.input.ComputeChange(attribute))
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

				input := wrapValueInMap(tc.input)

				if tc.validateNestedObject != nil {
					validate := change.ValidateMap(map[string]change.ValidateChangeFunc{
						"element": tc.validateNestedObject,
					}, collectionDefaultAction, false)
					validate(t, input.ComputeChange(attribute))
					return
				}

				if tc.validateSingleChange != nil {
					validate := change.ValidateMap(map[string]change.ValidateChangeFunc{
						"element": tc.validateSingleChange,
					}, collectionDefaultAction, false)
					validate(t, input.ComputeChange(attribute))
					return
				}

				validate := change.ValidateMap(map[string]change.ValidateChangeFunc{
					"element": change.ValidateNestedObject(tc.validateChanges, tc.validateAction, tc.validateReplace),
				}, collectionDefaultAction, false)
				validate(t, input.ComputeChange(attribute))
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

				input := wrapValueInSlice(tc.input)

				if tc.validateNestedObject != nil {
					validate := change.ValidateNestedList([]change.ValidateChangeFunc{
						tc.validateNestedObject,
					}, collectionDefaultAction, false)
					validate(t, input.ComputeChange(attribute))
					return
				}

				if tc.validateSingleChange != nil {
					validate := change.ValidateNestedList([]change.ValidateChangeFunc{
						tc.validateSingleChange,
					}, collectionDefaultAction, false)
					validate(t, input.ComputeChange(attribute))
					return
				}

				validate := change.ValidateNestedList([]change.ValidateChangeFunc{
					change.ValidateNestedObject(tc.validateChanges, tc.validateAction, tc.validateReplace),
				}, collectionDefaultAction, false)
				validate(t, input.ComputeChange(attribute))
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

				input := wrapValueInSlice(tc.input)

				if tc.validateSetChanges != nil {
					validate := change.ValidateSet(func() []change.ValidateChangeFunc {
						var ret []change.ValidateChangeFunc
						ret = append(ret, tc.validateSetChanges.Before.Validate(change.ValidateNestedObject))
						ret = append(ret, tc.validateSetChanges.After.Validate(change.ValidateNestedObject))
						return ret
					}(), collectionDefaultAction, false)
					validate(t, input.ComputeChange(attribute))
					return
				}

				if tc.validateNestedObject != nil {
					validate := change.ValidateSet([]change.ValidateChangeFunc{
						tc.validateNestedObject,
					}, collectionDefaultAction, false)
					validate(t, input.ComputeChange(attribute))
					return
				}

				if tc.validateSingleChange != nil {
					validate := change.ValidateSet([]change.ValidateChangeFunc{
						tc.validateSingleChange,
					}, collectionDefaultAction, false)
					validate(t, input.ComputeChange(attribute))
					return
				}

				validate := change.ValidateSet([]change.ValidateChangeFunc{
					change.ValidateNestedObject(tc.validateChanges, tc.validateAction, tc.validateReplace),
				}, collectionDefaultAction, false)
				validate(t, input.ComputeChange(attribute))
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
		validate    change.ValidateChangeFunc
		validateSet []change.ValidateChangeFunc
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
			validate: change.ValidateBlock(map[string]change.ValidateChangeFunc{
				"attribute_one": change.ValidatePrimitive(nil, strptr("\"new\""), plans.Create, false),
			}, nil, plans.Update, false),
			validateSet: []change.ValidateChangeFunc{
				change.ValidateBlock(nil, nil, plans.Delete, false),
				change.ValidateBlock(map[string]change.ValidateChangeFunc{
					"attribute_one": change.ValidatePrimitive(nil, strptr("\"new\""), plans.Create, false),
				}, nil, plans.Create, false),
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
			validate: change.ValidateBlock(map[string]change.ValidateChangeFunc{
				"attribute_one": change.ValidatePrimitive(strptr("\"old\""), strptr("\"new\""), plans.Update, false),
			}, nil, plans.Update, false),
			validateSet: []change.ValidateChangeFunc{
				change.ValidateBlock(map[string]change.ValidateChangeFunc{
					"attribute_one": change.ValidatePrimitive(strptr("\"old\""), nil, plans.Delete, false),
				}, nil, plans.Delete, false),
				change.ValidateBlock(map[string]change.ValidateChangeFunc{
					"attribute_one": change.ValidatePrimitive(nil, strptr("\"new\""), plans.Create, false),
				}, nil, plans.Create, false),
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
			validate: change.ValidateBlock(map[string]change.ValidateChangeFunc{
				"attribute_one": change.ValidatePrimitive(strptr("\"old\""), nil, plans.Delete, false),
			}, nil, plans.Update, false),
			validateSet: []change.ValidateChangeFunc{
				change.ValidateBlock(map[string]change.ValidateChangeFunc{
					"attribute_one": change.ValidatePrimitive(strptr("\"old\""), nil, plans.Delete, false),
				}, nil, plans.Delete, false),
				change.ValidateBlock(nil, nil, plans.Create, false),
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
			validate: change.ValidateBlock(nil, map[string][]change.ValidateChangeFunc{
				"block_one": {
					change.ValidateBlock(map[string]change.ValidateChangeFunc{
						"attribute_one": change.ValidatePrimitive(nil, strptr("\"new\""), plans.Create, false),
					}, nil, plans.Create, false),
				},
			}, plans.Update, false),
			validateSet: []change.ValidateChangeFunc{
				change.ValidateBlock(nil, nil, plans.Delete, false),
				change.ValidateBlock(nil, map[string][]change.ValidateChangeFunc{
					"block_one": {
						change.ValidateBlock(map[string]change.ValidateChangeFunc{
							"attribute_one": change.ValidatePrimitive(nil, strptr("\"new\""), plans.Create, false),
						}, nil, plans.Create, false),
					},
				}, plans.Create, false),
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
			validate: change.ValidateBlock(nil, map[string][]change.ValidateChangeFunc{
				"block_one": {
					change.ValidateBlock(map[string]change.ValidateChangeFunc{
						"attribute_one": change.ValidatePrimitive(strptr("\"old\""), strptr("\"new\""), plans.Update, false),
					}, nil, plans.Update, false),
				},
			}, plans.Update, false),
			validateSet: []change.ValidateChangeFunc{
				change.ValidateBlock(nil, map[string][]change.ValidateChangeFunc{
					"block_one": {
						change.ValidateBlock(map[string]change.ValidateChangeFunc{
							"attribute_one": change.ValidatePrimitive(strptr("\"old\""), nil, plans.Delete, false),
						}, nil, plans.Delete, false),
					},
				}, plans.Delete, false),
				change.ValidateBlock(nil, map[string][]change.ValidateChangeFunc{
					"block_one": {
						change.ValidateBlock(map[string]change.ValidateChangeFunc{
							"attribute_one": change.ValidatePrimitive(nil, strptr("\"new\""), plans.Create, false),
						}, nil, plans.Create, false),
					},
				}, plans.Create, false),
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
			validate: change.ValidateBlock(nil, map[string][]change.ValidateChangeFunc{
				"block_one": {
					change.ValidateBlock(map[string]change.ValidateChangeFunc{
						"attribute_one": change.ValidatePrimitive(strptr("\"old\""), nil, plans.Delete, false),
					}, nil, plans.Delete, false),
				},
			}, plans.Update, false),
			validateSet: []change.ValidateChangeFunc{
				change.ValidateBlock(nil, map[string][]change.ValidateChangeFunc{
					"block_one": {
						change.ValidateBlock(map[string]change.ValidateChangeFunc{
							"attribute_one": change.ValidatePrimitive(strptr("\"old\""), nil, plans.Delete, false),
						}, nil, plans.Delete, false),
					},
				}, plans.Delete, false),
				change.ValidateBlock(nil, nil, plans.Create, false),
			},
		},
	}
	for name, tmp := range tcs {
		tc := tmp

		t.Run(name, func(t *testing.T) {
			t.Run("single", func(t *testing.T) {
				input := Value{
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

				validate := change.ValidateBlock(nil, map[string][]change.ValidateChangeFunc{
					"block_type": {
						tc.validate,
					},
				}, plans.Update, false)
				validate(t, input.ComputeChange(block))
			})
			t.Run("map", func(t *testing.T) {
				input := Value{
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

				validate := change.ValidateBlock(nil, map[string][]change.ValidateChangeFunc{
					"block_type": {
						tc.validate,
					},
				}, plans.Update, false)
				validate(t, input.ComputeChange(block))
			})
			t.Run("list", func(t *testing.T) {
				input := Value{
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

				validate := change.ValidateBlock(nil, map[string][]change.ValidateChangeFunc{
					"block_type": {
						tc.validate,
					},
				}, plans.Update, false)
				validate(t, input.ComputeChange(block))
			})
			t.Run("set", func(t *testing.T) {
				input := Value{
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

				validate := change.ValidateBlock(nil, map[string][]change.ValidateChangeFunc{
					"block_type": func() []change.ValidateChangeFunc {
						if tc.validateSet != nil {
							return tc.validateSet
						}
						return []change.ValidateChangeFunc{tc.validate}
					}(),
				}, plans.Update, false)
				validate(t, input.ComputeChange(block))
			})
		})
	}
}

func TestValue_Outputs(t *testing.T) {
	tcs := map[string]struct {
		input          Value
		validateChange change.ValidateChangeFunc
	}{
		"primitive_create": {
			input: Value{
				Before: nil,
				After:  "new",
			},
			validateChange: change.ValidatePrimitive(nil, strptr("\"new\""), plans.Create, false),
		},
		"map_create": {
			input: Value{
				Before: nil,
				After: map[string]interface{}{
					"element_one": "new_one",
					"element_two": "new_two",
				},
			},
			validateChange: change.ValidateMap(map[string]change.ValidateChangeFunc{
				"element_one": change.ValidatePrimitive(nil, strptr("\"new_one\""), plans.Create, false),
				"element_two": change.ValidatePrimitive(nil, strptr("\"new_two\""), plans.Create, false),
			}, plans.Create, false),
		},
		"list_create": {
			input: Value{
				Before: nil,
				After: []interface{}{
					"new_one",
					"new_two",
				},
			},
			validateChange: change.ValidateList([]change.ValidateChangeFunc{
				change.ValidatePrimitive(nil, strptr("\"new_one\""), plans.Create, false),
				change.ValidatePrimitive(nil, strptr("\"new_two\""), plans.Create, false),
			}, plans.Create, false),
		},
		"primitive_update": {
			input: Value{
				Before: "old",
				After:  "new",
			},
			validateChange: change.ValidatePrimitive(strptr("\"old\""), strptr("\"new\""), plans.Update, false),
		},
		"map_update": {
			input: Value{
				Before: map[string]interface{}{
					"element_one": "old_one",
					"element_two": "old_two",
				},
				After: map[string]interface{}{
					"element_one": "new_one",
					"element_two": "new_two",
				},
			},
			validateChange: change.ValidateMap(map[string]change.ValidateChangeFunc{
				"element_one": change.ValidatePrimitive(strptr("\"old_one\""), strptr("\"new_one\""), plans.Update, false),
				"element_two": change.ValidatePrimitive(strptr("\"old_two\""), strptr("\"new_two\""), plans.Update, false),
			}, plans.Update, false),
		},
		"list_update": {
			input: Value{
				Before: []interface{}{
					"old_one",
					"old_two",
				},
				After: []interface{}{
					"new_one",
					"new_two",
				},
			},
			validateChange: change.ValidateList([]change.ValidateChangeFunc{
				change.ValidatePrimitive(strptr("\"old_one\""), nil, plans.Delete, false),
				change.ValidatePrimitive(strptr("\"old_two\""), nil, plans.Delete, false),
				change.ValidatePrimitive(nil, strptr("\"new_one\""), plans.Create, false),
				change.ValidatePrimitive(nil, strptr("\"new_two\""), plans.Create, false),
			}, plans.Update, false),
		},
		"primitive_delete": {
			input: Value{
				Before: "old",
				After:  nil,
			},
			validateChange: change.ValidatePrimitive(strptr("\"old\""), nil, plans.Delete, false),
		},
		"map_delete": {
			input: Value{
				Before: map[string]interface{}{
					"element_one": "old_one",
					"element_two": "old_two",
				},
				After: nil,
			},
			validateChange: change.ValidateMap(map[string]change.ValidateChangeFunc{
				"element_one": change.ValidatePrimitive(strptr("\"old_one\""), nil, plans.Delete, false),
				"element_two": change.ValidatePrimitive(strptr("\"old_two\""), nil, plans.Delete, false),
			}, plans.Delete, false),
		},
		"list_delete": {
			input: Value{
				Before: []interface{}{
					"old_one",
					"old_two",
				},
				After: nil,
			},
			validateChange: change.ValidateList([]change.ValidateChangeFunc{
				change.ValidatePrimitive(strptr("\"old_one\""), nil, plans.Delete, false),
				change.ValidatePrimitive(strptr("\"old_two\""), nil, plans.Delete, false),
			}, plans.Delete, false),
		},
		"primitive_to_list": {
			input: Value{
				Before: "old",
				After: []interface{}{
					"new_one",
					"new_two",
				},
			},
			validateChange: change.ValidateTypeChange(
				change.ValidatePrimitive(strptr("\"old\""), nil, plans.Delete, false),
				change.ValidateList([]change.ValidateChangeFunc{
					change.ValidatePrimitive(nil, strptr("\"new_one\""), plans.Create, false),
					change.ValidatePrimitive(nil, strptr("\"new_two\""), plans.Create, false),
				}, plans.Create, false), plans.Update, false),
		},
		"primitive_to_map": {
			input: Value{
				Before: "old",
				After: map[string]interface{}{
					"element_one": "new_one",
					"element_two": "new_two",
				},
			},
			validateChange: change.ValidateTypeChange(
				change.ValidatePrimitive(strptr("\"old\""), nil, plans.Delete, false),
				change.ValidateMap(map[string]change.ValidateChangeFunc{
					"element_one": change.ValidatePrimitive(nil, strptr("\"new_one\""), plans.Create, false),
					"element_two": change.ValidatePrimitive(nil, strptr("\"new_two\""), plans.Create, false),
				}, plans.Create, false), plans.Update, false),
		},
		"list_to_primitive": {
			input: Value{
				Before: []interface{}{
					"old_one",
					"old_two",
				},
				After: "new",
			},
			validateChange: change.ValidateTypeChange(
				change.ValidateList([]change.ValidateChangeFunc{
					change.ValidatePrimitive(strptr("\"old_one\""), nil, plans.Delete, false),
					change.ValidatePrimitive(strptr("\"old_two\""), nil, plans.Delete, false),
				}, plans.Delete, false),
				change.ValidatePrimitive(nil, strptr("\"new\""), plans.Create, false),
				plans.Update, false),
		},
		"list_to_map": {
			input: Value{
				Before: []interface{}{
					"old_one",
					"old_two",
				},
				After: map[string]interface{}{
					"element_one": "new_one",
					"element_two": "new_two",
				},
			},
			validateChange: change.ValidateTypeChange(
				change.ValidateList([]change.ValidateChangeFunc{
					change.ValidatePrimitive(strptr("\"old_one\""), nil, plans.Delete, false),
					change.ValidatePrimitive(strptr("\"old_two\""), nil, plans.Delete, false),
				}, plans.Delete, false),
				change.ValidateMap(map[string]change.ValidateChangeFunc{
					"element_one": change.ValidatePrimitive(nil, strptr("\"new_one\""), plans.Create, false),
					"element_two": change.ValidatePrimitive(nil, strptr("\"new_two\""), plans.Create, false),
				}, plans.Create, false), plans.Update, false),
		},
		"map_to_primitive": {
			input: Value{
				Before: map[string]interface{}{
					"element_one": "old_one",
					"element_two": "old_two",
				},
				After: "new",
			},
			validateChange: change.ValidateTypeChange(
				change.ValidateMap(map[string]change.ValidateChangeFunc{
					"element_one": change.ValidatePrimitive(strptr("\"old_one\""), nil, plans.Delete, false),
					"element_two": change.ValidatePrimitive(strptr("\"old_two\""), nil, plans.Delete, false),
				}, plans.Delete, false),
				change.ValidatePrimitive(nil, strptr("\"new\""), plans.Create, false),
				plans.Update, false),
		},
		"map_to_list": {
			input: Value{
				Before: map[string]interface{}{
					"element_one": "old_one",
					"element_two": "old_two",
				},
				After: []interface{}{
					"new_one",
					"new_two",
				},
			},
			validateChange: change.ValidateTypeChange(
				change.ValidateMap(map[string]change.ValidateChangeFunc{
					"element_one": change.ValidatePrimitive(strptr("\"old_one\""), nil, plans.Delete, false),
					"element_two": change.ValidatePrimitive(strptr("\"old_two\""), nil, plans.Delete, false),
				}, plans.Delete, false),
				change.ValidateList([]change.ValidateChangeFunc{
					change.ValidatePrimitive(nil, strptr("\"new_one\""), plans.Create, false),
					change.ValidatePrimitive(nil, strptr("\"new_two\""), plans.Create, false),
				}, plans.Create, false), plans.Update, false),
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			tc.validateChange(t, tc.input.ComputeChange(cty.NilType))
		})
	}
}

func TestValue_PrimitiveAttributes(t *testing.T) {
	// This function tests manipulating primitives: creating, deleting and
	// updating. It also automatically tests these operations within the
	// contexts of collections.

	tcs := map[string]struct {
		input                Value
		attribute            cty.Type
		validateChange       change.ValidateChangeFunc
		validateSliceChanges []change.ValidateChangeFunc // Lists are special in some cases.
	}{
		"primitive_create": {
			input: Value{
				After: "new",
			},
			attribute:      cty.String,
			validateChange: change.ValidatePrimitive(nil, strptr("\"new\""), plans.Create, false),
		},
		"primitive_delete": {
			input: Value{
				Before: "old",
			},
			attribute:      cty.String,
			validateChange: change.ValidatePrimitive(strptr("\"old\""), nil, plans.Delete, false),
		},
		"primitive_update": {
			input: Value{
				Before: "old",
				After:  "new",
			},
			attribute:      cty.String,
			validateChange: change.ValidatePrimitive(strptr("\"old\""), strptr("\"new\""), plans.Update, false),
			validateSliceChanges: []change.ValidateChangeFunc{
				change.ValidatePrimitive(strptr("\"old\""), nil, plans.Delete, false),
				change.ValidatePrimitive(nil, strptr("\"new\""), plans.Create, false),
			},
		},
		"primitive_set_explicit_null": {
			input: Value{
				Before:        "old",
				After:         nil,
				AfterExplicit: true,
			},
			attribute:      cty.String,
			validateChange: change.ValidatePrimitive(strptr("\"old\""), nil, plans.Update, false),
			validateSliceChanges: []change.ValidateChangeFunc{
				change.ValidatePrimitive(strptr("\"old\""), nil, plans.Delete, false),
				change.ValidatePrimitive(nil, nil, plans.Create, false),
			},
		},
		"primitive_unset_explicit_null": {
			input: Value{
				BeforeExplicit: true,
				Before:         nil,
				After:          "new",
			},
			attribute:      cty.String,
			validateChange: change.ValidatePrimitive(nil, strptr("\"new\""), plans.Update, false),
			validateSliceChanges: []change.ValidateChangeFunc{
				change.ValidatePrimitive(nil, nil, plans.Delete, false),
				change.ValidatePrimitive(nil, strptr("\"new\""), plans.Create, false),
			},
		},
		"primitive_create_sensitive": {
			input: Value{
				Before:         nil,
				After:          "new",
				AfterSensitive: true,
			},
			attribute:      cty.String,
			validateChange: change.ValidateSensitive(nil, "new", false, true, plans.Create, false),
		},
		"primitive_delete_sensitive": {
			input: Value{
				Before:          "old",
				BeforeSensitive: true,
				After:           nil,
			},
			attribute:      cty.String,
			validateChange: change.ValidateSensitive("old", nil, true, false, plans.Delete, false),
		},
		"primitive_update_sensitive": {
			input: Value{
				Before:          "old",
				BeforeSensitive: true,
				After:           "new",
				AfterSensitive:  true,
			},
			attribute:      cty.String,
			validateChange: change.ValidateSensitive("old", "new", true, true, plans.Update, false),
			validateSliceChanges: []change.ValidateChangeFunc{
				change.ValidateSensitive("old", nil, true, false, plans.Delete, false),
				change.ValidateSensitive(nil, "new", false, true, plans.Create, false),
			},
		},
		"primitive_create_computed": {
			input: Value{
				Before:  nil,
				After:   nil,
				Unknown: true,
			},
			attribute:      cty.String,
			validateChange: change.ValidateComputed(nil, plans.Create, false),
		},
		"primitive_update_computed": {
			input: Value{
				Before:  "old",
				After:   nil,
				Unknown: true,
			},
			attribute:      cty.String,
			validateChange: change.ValidateComputed(change.ValidatePrimitive(strptr("\"old\""), nil, plans.Delete, false), plans.Update, false),
			validateSliceChanges: []change.ValidateChangeFunc{
				change.ValidatePrimitive(strptr("\"old\""), nil, plans.Delete, false),
				change.ValidateComputed(nil, plans.Create, false),
			},
		},
		"primitive_update_replace": {
			input: Value{
				Before: "old",
				After:  "new",
				ReplacePaths: []interface{}{
					[]interface{}{}, // An empty path suggests this attribute should be true.
				},
			},
			attribute:      cty.String,
			validateChange: change.ValidatePrimitive(strptr("\"old\""), strptr("\"new\""), plans.Update, true),
			validateSliceChanges: []change.ValidateChangeFunc{
				change.ValidatePrimitive(strptr("\"old\""), nil, plans.Delete, true),
				change.ValidatePrimitive(nil, strptr("\"new\""), plans.Create, false),
			},
		},
		"noop": {
			input: Value{
				Before: "old",
				After:  "old",
			},
			attribute:      cty.String,
			validateChange: change.ValidatePrimitive(strptr("\"old\""), strptr("\"old\""), plans.NoOp, false),
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
				tc.validateChange(t, tc.input.ComputeChange(&jsonprovider.Attribute{
					AttributeType: unmarshalType(t, tc.attribute),
				}))
			})

			t.Run("map", func(t *testing.T) {
				input := wrapValueInMap(tc.input)
				attribute := &jsonprovider.Attribute{
					AttributeType: unmarshalType(t, cty.Map(tc.attribute)),
				}

				validate := change.ValidateMap(map[string]change.ValidateChangeFunc{
					"element": tc.validateChange,
				}, defaultCollectionsAction, false)
				validate(t, input.ComputeChange(attribute))
			})

			t.Run("list", func(t *testing.T) {
				input := wrapValueInSlice(tc.input)
				attribute := &jsonprovider.Attribute{
					AttributeType: unmarshalType(t, cty.List(tc.attribute)),
				}

				if tc.validateSliceChanges != nil {
					validate := change.ValidateList(tc.validateSliceChanges, defaultCollectionsAction, false)
					validate(t, input.ComputeChange(attribute))
					return
				}

				validate := change.ValidateList([]change.ValidateChangeFunc{
					tc.validateChange,
				}, defaultCollectionsAction, false)
				validate(t, input.ComputeChange(attribute))
			})

			t.Run("set", func(t *testing.T) {
				input := wrapValueInSlice(tc.input)
				attribute := &jsonprovider.Attribute{
					AttributeType: unmarshalType(t, cty.Set(tc.attribute)),
				}

				if tc.validateSliceChanges != nil {
					validate := change.ValidateSet(tc.validateSliceChanges, defaultCollectionsAction, false)
					validate(t, input.ComputeChange(attribute))
					return
				}

				validate := change.ValidateSet([]change.ValidateChangeFunc{
					tc.validateChange,
				}, defaultCollectionsAction, false)
				validate(t, input.ComputeChange(attribute))
			})
		})
	}
}

func TestValue_CollectionAttributes(t *testing.T) {
	// This function tests creating and deleting collections. Note, it does not
	// generally cover editing collections except in special cases as editing
	// collections is handled automatically by other functions.
	tcs := map[string]struct {
		input          Value
		attribute      *jsonprovider.Attribute
		validateChange change.ValidateChangeFunc
	}{
		"map_create_empty": {
			input: Value{
				Before: nil,
				After:  map[string]interface{}{},
			},
			attribute: &jsonprovider.Attribute{
				AttributeType: unmarshalType(t, cty.Map(cty.String)),
			},
			validateChange: change.ValidateMap(nil, plans.Create, false),
		},
		"map_create_populated": {
			input: Value{
				Before: nil,
				After: map[string]interface{}{
					"element_one": "one",
					"element_two": "two",
				},
			},
			attribute: &jsonprovider.Attribute{
				AttributeType: unmarshalType(t, cty.Map(cty.String)),
			},
			validateChange: change.ValidateMap(map[string]change.ValidateChangeFunc{
				"element_one": change.ValidatePrimitive(nil, strptr("\"one\""), plans.Create, false),
				"element_two": change.ValidatePrimitive(nil, strptr("\"two\""), plans.Create, false),
			}, plans.Create, false),
		},
		"map_delete_empty": {
			input: Value{
				Before: map[string]interface{}{},
				After:  nil,
			},
			attribute: &jsonprovider.Attribute{
				AttributeType: unmarshalType(t, cty.Map(cty.String)),
			},
			validateChange: change.ValidateMap(nil, plans.Delete, false),
		},
		"map_delete_populated": {
			input: Value{
				Before: map[string]interface{}{
					"element_one": "one",
					"element_two": "two",
				},
				After: nil,
			},
			attribute: &jsonprovider.Attribute{
				AttributeType: unmarshalType(t, cty.Map(cty.String)),
			},
			validateChange: change.ValidateMap(map[string]change.ValidateChangeFunc{
				"element_one": change.ValidatePrimitive(strptr("\"one\""), nil, plans.Delete, false),
				"element_two": change.ValidatePrimitive(strptr("\"two\""), nil, plans.Delete, false),
			}, plans.Delete, false),
		},
		"map_create_sensitive": {
			input: Value{
				Before:         nil,
				After:          map[string]interface{}{},
				AfterSensitive: true,
			},
			attribute: &jsonprovider.Attribute{
				AttributeType: unmarshalType(t, cty.Map(cty.String)),
			},
			validateChange: change.ValidateSensitive(nil, map[string]interface{}{}, false, true, plans.Create, false),
		},
		"map_update_sensitive": {
			input: Value{
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
			validateChange: change.ValidateSensitive(map[string]interface{}{"element": "one"}, map[string]interface{}{}, true, true, plans.Update, false),
		},
		"map_delete_sensitive": {
			input: Value{
				Before:          map[string]interface{}{},
				BeforeSensitive: true,
				After:           nil,
			},
			attribute: &jsonprovider.Attribute{
				AttributeType: unmarshalType(t, cty.Map(cty.String)),
			},
			validateChange: change.ValidateSensitive(map[string]interface{}{}, nil, true, false, plans.Delete, false),
		},
		"map_create_unknown": {
			input: Value{
				Before:  nil,
				After:   map[string]interface{}{},
				Unknown: true,
			},
			attribute: &jsonprovider.Attribute{
				AttributeType: unmarshalType(t, cty.Map(cty.String)),
			},
			validateChange: change.ValidateComputed(nil, plans.Create, false),
		},
		"map_update_unknown": {
			input: Value{
				Before: map[string]interface{}{},
				After: map[string]interface{}{
					"element": "one",
				},
				Unknown: true,
			},
			attribute: &jsonprovider.Attribute{
				AttributeType: unmarshalType(t, cty.Map(cty.String)),
			},
			validateChange: change.ValidateComputed(change.ValidateMap(nil, plans.Delete, false), plans.Update, false),
		},
		"list_create_empty": {
			input: Value{
				Before: nil,
				After:  []interface{}{},
			},
			attribute: &jsonprovider.Attribute{
				AttributeType: unmarshalType(t, cty.List(cty.String)),
			},
			validateChange: change.ValidateList(nil, plans.Create, false),
		},
		"list_create_populated": {
			input: Value{
				Before: nil,
				After:  []interface{}{"one", "two"},
			},
			attribute: &jsonprovider.Attribute{
				AttributeType: unmarshalType(t, cty.List(cty.String)),
			},
			validateChange: change.ValidateList([]change.ValidateChangeFunc{
				change.ValidatePrimitive(nil, strptr("\"one\""), plans.Create, false),
				change.ValidatePrimitive(nil, strptr("\"two\""), plans.Create, false),
			}, plans.Create, false),
		},
		"list_delete_empty": {
			input: Value{
				Before: []interface{}{},
				After:  nil,
			},
			attribute: &jsonprovider.Attribute{
				AttributeType: unmarshalType(t, cty.List(cty.String)),
			},
			validateChange: change.ValidateList(nil, plans.Delete, false),
		},
		"list_delete_populated": {
			input: Value{
				Before: []interface{}{"one", "two"},
				After:  nil,
			},
			attribute: &jsonprovider.Attribute{
				AttributeType: unmarshalType(t, cty.List(cty.String)),
			},
			validateChange: change.ValidateList([]change.ValidateChangeFunc{
				change.ValidatePrimitive(strptr("\"one\""), nil, plans.Delete, false),
				change.ValidatePrimitive(strptr("\"two\""), nil, plans.Delete, false),
			}, plans.Delete, false),
		},
		"list_create_sensitive": {
			input: Value{
				Before:         nil,
				After:          []interface{}{},
				AfterSensitive: true,
			},
			attribute: &jsonprovider.Attribute{
				AttributeType: unmarshalType(t, cty.List(cty.String)),
			},
			validateChange: change.ValidateSensitive(nil, []interface{}{}, false, true, plans.Create, false),
		},
		"list_update_sensitive": {
			input: Value{
				Before:          []interface{}{"one"},
				BeforeSensitive: true,
				After:           []interface{}{},
				AfterSensitive:  true,
			},
			attribute: &jsonprovider.Attribute{
				AttributeType: unmarshalType(t, cty.List(cty.String)),
			},
			validateChange: change.ValidateSensitive([]interface{}{"one"}, []interface{}{}, true, true, plans.Update, false),
		},
		"list_delete_sensitive": {
			input: Value{
				Before:          []interface{}{},
				BeforeSensitive: true,
				After:           nil,
			},
			attribute: &jsonprovider.Attribute{
				AttributeType: unmarshalType(t, cty.List(cty.String)),
			},
			validateChange: change.ValidateSensitive([]interface{}{}, nil, true, false, plans.Delete, false),
		},
		"list_create_unknown": {
			input: Value{
				Before:  nil,
				After:   []interface{}{},
				Unknown: true,
			},
			attribute: &jsonprovider.Attribute{
				AttributeType: unmarshalType(t, cty.List(cty.String)),
			},
			validateChange: change.ValidateComputed(nil, plans.Create, false),
		},
		"list_update_unknown": {
			input: Value{
				Before:  []interface{}{},
				After:   []interface{}{"one"},
				Unknown: true,
			},
			attribute: &jsonprovider.Attribute{
				AttributeType: unmarshalType(t, cty.List(cty.String)),
			},
			validateChange: change.ValidateComputed(change.ValidateList(nil, plans.Delete, false), plans.Update, false),
		},
		"set_create_empty": {
			input: Value{
				Before: nil,
				After:  []interface{}{},
			},
			attribute: &jsonprovider.Attribute{
				AttributeType: unmarshalType(t, cty.Set(cty.String)),
			},
			validateChange: change.ValidateSet(nil, plans.Create, false),
		},
		"set_create_populated": {
			input: Value{
				Before: nil,
				After:  []interface{}{"one", "two"},
			},
			attribute: &jsonprovider.Attribute{
				AttributeType: unmarshalType(t, cty.Set(cty.String)),
			},
			validateChange: change.ValidateSet([]change.ValidateChangeFunc{
				change.ValidatePrimitive(nil, strptr("\"one\""), plans.Create, false),
				change.ValidatePrimitive(nil, strptr("\"two\""), plans.Create, false),
			}, plans.Create, false),
		},
		"set_delete_empty": {
			input: Value{
				Before: []interface{}{},
				After:  nil,
			},
			attribute: &jsonprovider.Attribute{
				AttributeType: unmarshalType(t, cty.Set(cty.String)),
			},
			validateChange: change.ValidateSet(nil, plans.Delete, false),
		},
		"set_delete_populated": {
			input: Value{
				Before: []interface{}{"one", "two"},
				After:  nil,
			},
			attribute: &jsonprovider.Attribute{
				AttributeType: unmarshalType(t, cty.Set(cty.String)),
			},
			validateChange: change.ValidateSet([]change.ValidateChangeFunc{
				change.ValidatePrimitive(strptr("\"one\""), nil, plans.Delete, false),
				change.ValidatePrimitive(strptr("\"two\""), nil, plans.Delete, false),
			}, plans.Delete, false),
		},
		"set_create_sensitive": {
			input: Value{
				Before:         nil,
				After:          []interface{}{},
				AfterSensitive: true,
			},
			attribute: &jsonprovider.Attribute{
				AttributeType: unmarshalType(t, cty.Set(cty.String)),
			},
			validateChange: change.ValidateSensitive(nil, []interface{}{}, false, true, plans.Create, false),
		},
		"set_update_sensitive": {
			input: Value{
				Before:          []interface{}{"one"},
				BeforeSensitive: true,
				After:           []interface{}{},
				AfterSensitive:  true,
			},
			attribute: &jsonprovider.Attribute{
				AttributeType: unmarshalType(t, cty.Set(cty.String)),
			},
			validateChange: change.ValidateSensitive([]interface{}{"one"}, []interface{}{}, true, true, plans.Update, false),
		},
		"set_delete_sensitive": {
			input: Value{
				Before:          []interface{}{},
				BeforeSensitive: true,
				After:           nil,
			},
			attribute: &jsonprovider.Attribute{
				AttributeType: unmarshalType(t, cty.Set(cty.String)),
			},
			validateChange: change.ValidateSensitive([]interface{}{}, nil, true, false, plans.Delete, false),
		},
		"set_create_unknown": {
			input: Value{
				Before:  nil,
				After:   []interface{}{},
				Unknown: true,
			},
			attribute: &jsonprovider.Attribute{
				AttributeType: unmarshalType(t, cty.Set(cty.String)),
			},
			validateChange: change.ValidateComputed(nil, plans.Create, false),
		},
		"set_update_unknown": {
			input: Value{
				Before:  []interface{}{},
				After:   []interface{}{"one"},
				Unknown: true,
			},
			attribute: &jsonprovider.Attribute{
				AttributeType: unmarshalType(t, cty.Set(cty.String)),
			},
			validateChange: change.ValidateComputed(change.ValidateSet(nil, plans.Delete, false), plans.Update, false),
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			tc.validateChange(t, tc.input.ComputeChange(tc.attribute))
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

// wrapValueInSlice does the same as wrapValueInMap, except it wraps it into a
// slice internally.
func wrapValueInSlice(input Value) Value {
	return wrapValue(input, float64(0), func(value interface{}, unknown interface{}, explicit bool) interface{} {
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

// wrapValueInMap access a single Value and returns a new Value that represents
// a map with a single element. That single element is the input value.
func wrapValueInMap(input Value) Value {
	return wrapValue(input, "element", func(value interface{}, unknown interface{}, explicit bool) interface{} {
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

func wrapValue(input Value, step interface{}, wrap func(interface{}, interface{}, bool) interface{}) Value {
	return Value{
		Before:          wrap(input.Before, nil, input.BeforeExplicit),
		After:           wrap(input.After, input.Unknown, input.AfterExplicit),
		Unknown:         wrap(input.Unknown, nil, false),
		BeforeSensitive: wrap(input.BeforeSensitive, nil, false),
		AfterSensitive:  wrap(input.AfterSensitive, nil, false),
		ReplacePaths: func() []interface{} {
			var ret []interface{}
			for _, path := range input.ReplacePaths {
				old := path.([]interface{})
				var updated []interface{}
				updated = append(updated, step)
				updated = append(updated, old...)
				ret = append(ret, updated)
			}
			return ret
		}(),
	}
}
