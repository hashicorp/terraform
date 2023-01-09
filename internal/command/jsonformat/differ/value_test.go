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

func TestValue_ObjectAttributes(t *testing.T) {
	// We break these tests out into their own function, so we can automatically
	// test both objects and nested objects together.

	tcs := map[string]struct {
		input                Value
		attributes           map[string]cty.Type
		validateSingleChange change.ValidateChangeFunc
		validateObject       change.ValidateChangeFunc
		validateNestedObject change.ValidateChangeFunc
		validateChanges      map[string]change.ValidateChangeFunc
		validateReplace      bool
		validateAction       plans.Action
	}{
		"object_create": {
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
		"object_delete": {
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
		"object_create_sensitive": {
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
		"object_delete_sensitive": {
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
		"object_create_unknown": {
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
		"object_update_unknown": {
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
		},
		"object_create_attribute": {
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
		},
		"object_create_attribute_from_explicit_null": {
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
		},
		"object_delete_attribute": {
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
		},
		"object_delete_attribute_to_explicit_null": {
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
		},
		"object_update_attribute": {
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
		},
		"object_create_sensitive_attribute": {
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
		},
		"object_delete_sensitive_attribute": {
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
		},
		"object_update_sensitive_attribute": {
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
		},
		"object_create_computed_attribute": {
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
		},
		"object_update_computed_attribute": {
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
		},
		"object_ignores_unset_fields": {
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
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {

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

		t.Run(fmt.Sprintf("nested_%s", name), func(t *testing.T) {
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
	}
}

func TestValue_Attribute(t *testing.T) {
	tcs := map[string]struct {
		input          Value
		attribute      *jsonprovider.Attribute
		validateChange change.ValidateChangeFunc
	}{
		"primitive_create": {
			input: Value{
				After: "new",
			},
			attribute: &jsonprovider.Attribute{
				AttributeType: []byte("\"string\""),
			},
			validateChange: change.ValidatePrimitive(nil, strptr("\"new\""), plans.Create, false),
		},
		"primitive_delete": {
			input: Value{
				Before: "old",
			},
			attribute: &jsonprovider.Attribute{
				AttributeType: []byte("\"string\""),
			},
			validateChange: change.ValidatePrimitive(strptr("\"old\""), nil, plans.Delete, false),
		},
		"primitive_update": {
			input: Value{
				Before: "old",
				After:  "new",
			},
			attribute: &jsonprovider.Attribute{
				AttributeType: []byte("\"string\""),
			},
			validateChange: change.ValidatePrimitive(strptr("\"old\""), strptr("\"new\""), plans.Update, false),
		},
		"primitive_set_explicit_null": {
			input: Value{
				Before:        "old",
				After:         nil,
				AfterExplicit: true,
			},
			attribute: &jsonprovider.Attribute{
				AttributeType: []byte("\"string\""),
			},
			validateChange: change.ValidatePrimitive(strptr("\"old\""), nil, plans.Update, false),
		},
		"primitive_unset_explicit_null": {
			input: Value{
				BeforeExplicit: true,
				Before:         nil,
				After:          "new",
			},
			attribute: &jsonprovider.Attribute{
				AttributeType: []byte("\"string\""),
			},
			validateChange: change.ValidatePrimitive(nil, strptr("\"new\""), plans.Update, false),
		},
		"primitive_create_sensitive": {
			input: Value{
				Before:         nil,
				After:          "new",
				AfterSensitive: true,
			},
			attribute: &jsonprovider.Attribute{
				AttributeType: []byte("\"string\""),
			},
			validateChange: change.ValidateSensitive(nil, "new", false, true, plans.Create, false),
		},
		"primitive_delete_sensitive": {
			input: Value{
				Before:          "old",
				BeforeSensitive: true,
				After:           nil,
			},
			attribute: &jsonprovider.Attribute{
				AttributeType: []byte("\"string\""),
			},
			validateChange: change.ValidateSensitive("old", nil, true, false, plans.Delete, false),
		},
		"primitive_update_sensitive": {
			input: Value{
				Before:          "old",
				BeforeSensitive: true,
				After:           "new",
				AfterSensitive:  true,
			},
			attribute: &jsonprovider.Attribute{
				AttributeType: []byte("\"string\""),
			},
			validateChange: change.ValidateSensitive("old", "new", true, true, plans.Update, false),
		},
		"primitive_create_computed": {
			input: Value{
				Before:  nil,
				After:   nil,
				Unknown: true,
			},
			attribute: &jsonprovider.Attribute{
				AttributeType: []byte("\"string\""),
			},
			validateChange: change.ValidateComputed(nil, plans.Create, false),
		},
		"primitive_update_computed": {
			input: Value{
				Before:  "old",
				After:   nil,
				Unknown: true,
			},
			attribute: &jsonprovider.Attribute{
				AttributeType: []byte("\"string\""),
			},
			validateChange: change.ValidateComputed(change.ValidatePrimitive(strptr("\"old\""), nil, plans.Delete, false), plans.Update, false),
		},
	}
	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			tc.validateChange(t, tc.input.ComputeChange(tc.attribute))
		})
	}
}

func unmarshalType(t *testing.T, ctyType cty.Type) json.RawMessage {
	msg, err := ctyjson.MarshalType(ctyType)
	if err != nil {
		t.Fatalf("invalid type: %s", ctyType.FriendlyName())
	}
	return msg
}
