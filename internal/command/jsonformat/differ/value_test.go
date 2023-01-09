package differ

import (
	"testing"

	"github.com/hashicorp/terraform/internal/command/jsonformat/change"
	"github.com/hashicorp/terraform/internal/command/jsonprovider"
	"github.com/hashicorp/terraform/internal/plans"
)

func TestValue_Attribute(t *testing.T) {
	tcs := map[string]struct {
		input           Value
		attribute       *jsonprovider.Attribute
		expectedAction  plans.Action
		expectedReplace bool
		validateChange  change.ValidateChangeFunc
	}{
		"primitive_create": {
			input: Value{
				After: "new",
			},
			attribute: &jsonprovider.Attribute{
				AttributeType: []byte("\"string\""),
			},
			expectedAction: plans.Create,
			validateChange: change.ValidatePrimitive(nil, strptr("\"new\"")),
		},
		"primitive_delete": {
			input: Value{
				Before: "old",
			},
			attribute: &jsonprovider.Attribute{
				AttributeType: []byte("\"string\""),
			},
			expectedAction: plans.Delete,
			validateChange: change.ValidatePrimitive(strptr("\"old\""), nil),
		},
		"primitive_update": {
			input: Value{
				Before: "old",
				After:  "new",
			},
			attribute: &jsonprovider.Attribute{
				AttributeType: []byte("\"string\""),
			},
			expectedAction: plans.Update,
			validateChange: change.ValidatePrimitive(strptr("\"old\""), strptr("\"new\"")),
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
			expectedAction: plans.Update,
			validateChange: change.ValidatePrimitive(strptr("\"old\""), nil),
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
			expectedAction: plans.Update,
			validateChange: change.ValidatePrimitive(nil, strptr("\"new\"")),
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
			expectedAction: plans.Create,
			validateChange: change.ValidateSensitive(nil, "new", false, true),
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
			expectedAction: plans.Delete,
			validateChange: change.ValidateSensitive("old", nil, true, false),
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
			expectedAction: plans.Update,
			validateChange: change.ValidateSensitive("old", "new", true, true),
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
			expectedAction: plans.Create,
			validateChange: change.ValidateComputed(nil),
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
			expectedAction: plans.Update,
			validateChange: change.ValidateComputed(change.ValidatePrimitive(strptr("\"old\""), nil)),
		},
	}
	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			change.ValidateChange(
				t,
				tc.validateChange,
				tc.input.ComputeChangeForAttribute(tc.attribute),
				tc.expectedAction,
				tc.expectedReplace)
		})
	}
}
