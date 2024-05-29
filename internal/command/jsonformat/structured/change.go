// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package structured

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"

	"github.com/hashicorp/terraform/internal/command/jsonformat/structured/attribute_path"
	"github.com/hashicorp/terraform/internal/command/jsonplan"
	"github.com/hashicorp/terraform/internal/command/jsonstate"
	viewsjson "github.com/hashicorp/terraform/internal/command/views/json"
	"github.com/hashicorp/terraform/internal/plans"
)

// Change contains the unmarshalled generic interface{} types that are output by
// the JSON functions in the various json packages (such as jsonplan and
// jsonprovider).
//
// A Change can be converted into a computed.Diff, ready for rendering, with the
// ComputeDiffForAttribute, ComputeDiffForOutput, and ComputeDiffForBlock
// functions.
//
// The Before and After fields are actually go-cty values, but we cannot convert
// them directly because of the HCP Terraform redacted endpoint. The redacted
// endpoint turns sensitive values into strings regardless of their types.
// Because of this, we cannot just do a direct conversion using the ctyjson
// package. We would have to iterate through the schema first, find the
// sensitive values and their mapped types, update the types inside the schema
// to strings, and then go back and do the overall conversion. This isn't
// including any of the more complicated parts around what happens if something
// was sensitive before and isn't sensitive after or vice versa. This would mean
// the type would need to change between the before and after value. It is in
// fact just easier to iterate through the values as generic JSON interfaces.
type Change struct {

	// BeforeExplicit matches AfterExplicit except references the Before value.
	BeforeExplicit bool

	// AfterExplicit refers to whether the After value is explicit or
	// implicit. It is explicit if it has been specified by the user, and
	// implicit if it has been set as a consequence of other changes.
	//
	// For example, explicitly setting a value to null in a list should result
	// in After being null and AfterExplicit being true. In comparison,
	// removing an element from a list should also result in After being null
	// and AfterExplicit being false. Without the explicit information our
	// functions would not be able to tell the difference between these two
	// cases.
	AfterExplicit bool

	// Before contains the value before the proposed change.
	//
	// The type of the value should be informed by the schema and cast
	// appropriately when needed.
	Before interface{}

	// After contains the value after the proposed change.
	//
	// The type of the value should be informed by the schema and cast
	// appropriately when needed.
	After interface{}

	// Unknown describes whether the After value is known or unknown at the time
	// of the plan. In practice, this means the after value should be rendered
	// simply as `(known after apply)`.
	//
	// The concrete value could be a boolean describing whether the entirety of
	// the After value is unknown, or it could be a list or a map depending on
	// the schema describing whether specific elements or attributes within the
	// value are unknown.
	Unknown interface{}

	// BeforeSensitive matches Unknown, but references whether the Before value
	// is sensitive.
	BeforeSensitive interface{}

	// AfterSensitive matches Unknown, but references whether the After value is
	// sensitive.
	AfterSensitive interface{}

	// ReplacePaths contains a set of paths that point to attributes/elements
	// that are causing the overall resource to be replaced rather than simply
	// updated.
	ReplacePaths attribute_path.Matcher

	// RelevantAttributes contains a set of paths that point attributes/elements
	// that we should display. Any element/attribute not matched by this Matcher
	// should be skipped.
	RelevantAttributes attribute_path.Matcher

	// NonLegacySchema must only be used when rendering the change to the CLI,
	// and is otherwise ignored. This flag is set when we can be sure that the
	// change originated from a resource which is not using the legacy SDK, so
	// we don't need to hide changes between empty and null strings.
	// NonLegacySchema is only switched to true by the renderer, because that is
	// where we have most of the schema information to detect the condition.
	NonLegacySchema bool
}

// FromJsonChange unmarshals the raw []byte values in the jsonplan.Change
// structs into generic interface{} types that can be reasoned about.
func FromJsonChange(change jsonplan.Change, relevantAttributes attribute_path.Matcher) Change {
	ret := Change{
		Before:             unmarshalGeneric(change.Before),
		After:              unmarshalGeneric(change.After),
		Unknown:            unmarshalGeneric(change.AfterUnknown),
		BeforeSensitive:    unmarshalGeneric(change.BeforeSensitive),
		AfterSensitive:     unmarshalGeneric(change.AfterSensitive),
		ReplacePaths:       attribute_path.Parse(change.ReplacePaths, false),
		RelevantAttributes: relevantAttributes,
	}

	// A forget-only action (i.e. ["forget"], not ["create", "forget"])
	// should be represented as a no-op, so it does not look like we are
	// proposing to delete the resource.
	if len(change.Actions) == 1 && change.Actions[0] == "forget" {
		ret = ret.AsNoOp()
	}

	return ret
}

// FromJsonResource unmarshals the raw values in the jsonstate.Resource structs
// into generic interface{} types that can be reasoned about.
func FromJsonResource(resource jsonstate.Resource) Change {
	return Change{
		// We model resource formatting as NoOps.
		Before: unwrapAttributeValues(resource.AttributeValues),
		After:  unwrapAttributeValues(resource.AttributeValues),

		// We have some sensitive values, but we don't have any unknown values.
		Unknown:         false,
		BeforeSensitive: unmarshalGeneric(resource.SensitiveValues),
		AfterSensitive:  unmarshalGeneric(resource.SensitiveValues),

		// We don't display replacement data for resources, and all attributes
		// are relevant.
		ReplacePaths:       attribute_path.Empty(false),
		RelevantAttributes: attribute_path.AlwaysMatcher(),
	}
}

// FromJsonOutput unmarshals the raw values in the jsonstate.Output structs into
// generic interface{} types that can be reasoned about.
func FromJsonOutput(output jsonstate.Output) Change {
	return Change{
		// We model resource formatting as NoOps.
		Before: unmarshalGeneric(output.Value),
		After:  unmarshalGeneric(output.Value),

		// We have some sensitive values, but we don't have any unknown values.
		Unknown:         false,
		BeforeSensitive: output.Sensitive,
		AfterSensitive:  output.Sensitive,

		// We don't display replacement data for resources, and all attributes
		// are relevant.
		ReplacePaths:       attribute_path.Empty(false),
		RelevantAttributes: attribute_path.AlwaysMatcher(),
	}
}

// FromJsonViewsOutput unmarshals the raw values in the viewsjson.Output structs into
// generic interface{} types that can be reasoned about.
func FromJsonViewsOutput(output viewsjson.Output) Change {
	return Change{
		// We model resource formatting as NoOps.
		Before: unmarshalGeneric(output.Value),
		After:  unmarshalGeneric(output.Value),

		// We have some sensitive values, but we don't have any unknown values.
		Unknown:         false,
		BeforeSensitive: output.Sensitive,
		AfterSensitive:  output.Sensitive,

		// We don't display replacement data for resources, and all attributes
		// are relevant.
		ReplacePaths:       attribute_path.Empty(false),
		RelevantAttributes: attribute_path.AlwaysMatcher(),
	}
}

// CalculateAction does a very simple analysis to make the best guess at the
// action this change describes. For complex types such as objects, maps, lists,
// or sets it is likely more efficient to work out the action directly instead
// of relying on this function.
func (change Change) CalculateAction() plans.Action {
	if (change.Before == nil && !change.BeforeExplicit) && (change.After != nil || change.AfterExplicit) {
		return plans.Create
	}
	if (change.After == nil && !change.AfterExplicit) && (change.Before != nil || change.BeforeExplicit) {
		return plans.Delete
	}

	if reflect.DeepEqual(change.Before, change.After) && change.AfterExplicit == change.BeforeExplicit && change.IsAfterSensitive() == change.IsBeforeSensitive() {
		return plans.NoOp
	}

	return plans.Update
}

// GetDefaultActionForIteration is used to guess what the change could be for
// complex attributes (collections and objects) and blocks.
//
// You can't really tell the difference between a NoOp and an Update just by
// looking at the attribute itself as you need to inspect the children.
//
// This function returns a Delete or a Create action if the before or after
// values were null, and returns a NoOp for all other cases. It should be used
// in conjunction with compareActions to calculate the actual action based on
// the actions of the children.
func (change Change) GetDefaultActionForIteration() plans.Action {
	if change.Before == nil && change.After == nil {
		return plans.NoOp
	}

	if change.Before == nil {
		return plans.Create
	}
	if change.After == nil {
		return plans.Delete
	}
	return plans.NoOp
}

// AsNoOp returns the current change as if it is a NoOp operation.
//
// Basically it replaces all the after values with the before values.
func (change Change) AsNoOp() Change {
	return Change{
		BeforeExplicit:     change.BeforeExplicit,
		AfterExplicit:      change.BeforeExplicit,
		Before:             change.Before,
		After:              change.Before,
		Unknown:            false,
		BeforeSensitive:    change.BeforeSensitive,
		AfterSensitive:     change.BeforeSensitive,
		ReplacePaths:       change.ReplacePaths,
		RelevantAttributes: change.RelevantAttributes,
	}
}

// AsDelete returns the current change as if it is a Delete operation.
//
// Basically it replaces all the after values with nil or false.
func (change Change) AsDelete() Change {
	return Change{
		BeforeExplicit:     change.BeforeExplicit,
		AfterExplicit:      false,
		Before:             change.Before,
		After:              nil,
		Unknown:            nil,
		BeforeSensitive:    change.BeforeSensitive,
		AfterSensitive:     nil,
		ReplacePaths:       change.ReplacePaths,
		RelevantAttributes: change.RelevantAttributes,
	}
}

// AsCreate returns the current change as if it is a Create operation.
//
// Basically it replaces all the before values with nil or false.
func (change Change) AsCreate() Change {
	return Change{
		BeforeExplicit:     false,
		AfterExplicit:      change.AfterExplicit,
		Before:             nil,
		After:              change.After,
		Unknown:            change.Unknown,
		BeforeSensitive:    nil,
		AfterSensitive:     change.AfterSensitive,
		ReplacePaths:       change.ReplacePaths,
		RelevantAttributes: change.RelevantAttributes,
	}
}

func unmarshalGeneric(raw json.RawMessage) interface{} {
	if raw == nil {
		return nil
	}

	out, err := ParseJson(bytes.NewReader(raw))
	if err != nil {
		panic("unrecognized json type: " + err.Error())
	}
	return out
}

func unwrapAttributeValues(values jsonstate.AttributeValues) map[string]interface{} {
	out := make(map[string]interface{})
	for key, value := range values {
		out[key] = unmarshalGeneric(value)
	}
	return out
}

func ParseJson(reader io.Reader) (interface{}, error) {
	decoder := json.NewDecoder(reader)
	decoder.UseNumber()

	var jv interface{}
	if err := decoder.Decode(&jv); err != nil {
		return nil, err
	}

	// The JSON decoder should have consumed the entire input stream, so
	// we should be at EOF now.
	if token, err := decoder.Token(); err != io.EOF {
		return nil, fmt.Errorf("unexpected token after valid JSON: %v", token)
	}

	return jv, nil
}
