package differ

import (
	"encoding/json"
	"github.com/hashicorp/terraform/internal/command/jsonplan"
)

// Value contains the unmarshalled generic interface{} types that are output by
// the JSON structured run output functions in the various json packages (such
// as jsonplan and jsonprovider).
//
// A Value can be converted into a change.Change, ready for rendering, with the
// ComputeChangeForAttribute, ComputeChangeForOutput, and ComputeChangeForBlock
// functions.
type Value struct {
	// BeforeExplicit refers to whether the Before value is explicit or
	// implicit. It is explicit if it has been specified by the user, and
	// implicit if it has been set as a consequence of other changes.
	//
	// For example, explicitly setting a value to null in a list should result
	// in Before being null and BeforeExplicit being true. In comparison,
	// removing an element from a list should also result in Before being null
	// and BeforeExplicit being false. Without the explicit information our
	// functions would not be able to tell the difference between these two
	// cases.
	BeforeExplicit bool

	// AfterExplicit matches BeforeExplicit except references the After value.
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

	// ReplacePaths generally contains nested slices that describe paths to
	// elements or attributes that are causing the overall resource to be
	// replaced.
	ReplacePaths interface{}
}

// ValueFromJsonChange unmarshals the raw []byte values in the jsonplan.Change
// structs into generic interface{} types that can be reasoned about.
func ValueFromJsonChange(change jsonplan.Change) Value {
	return Value{
		Before:          unmarshalGeneric(change.Before),
		After:           unmarshalGeneric(change.After),
		Unknown:         unmarshalGeneric(change.AfterUnknown),
		BeforeSensitive: unmarshalGeneric(change.BeforeSensitive),
		AfterSensitive:  unmarshalGeneric(change.AfterSensitive),
		ReplacePaths:    unmarshalGeneric(change.ReplacePaths),
	}
}

func unmarshalGeneric(raw json.RawMessage) interface{} {
	if raw == nil {
		return nil
	}

	var out interface{}
	if err := json.Unmarshal(raw, &out); err != nil {
		panic("unrecognized json type: " + err.Error())
	}
	return out
}
