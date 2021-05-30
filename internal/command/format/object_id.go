package format

import (
	"github.com/zclconf/go-cty/cty"
)

// ObjectValueID takes a value that is assumed to be an object representation
// of some resource instance object and attempts to heuristically find an
// attribute of it that is likely to be a unique identifier in the remote
// system that it belongs to which will be useful to the user.
//
// If such an attribute is found, its name and string value intended for
// display are returned. Both returned strings are empty if no such attribute
// exists, in which case the caller should assume that the resource instance
// address within the Terraform configuration is the best available identifier.
//
// This is only a best-effort sort of thing, relying on naming conventions in
// our resource type schemas. The result is not guaranteed to be unique, but
// should generally be suitable for display to an end-user anyway.
//
// This function will panic if the given value is not of an object type.
func ObjectValueID(obj cty.Value) (k, v string) {
	if obj.IsNull() || !obj.IsKnown() {
		return "", ""
	}

	atys := obj.Type().AttributeTypes()

	switch {

	case atys["id"] == cty.String:
		v := obj.GetAttr("id")
		if v.IsKnown() && !v.IsNull() {
			return "id", v.AsString()
		}

	case atys["name"] == cty.String:
		// "name" isn't always globally unique, but if there isn't also an
		// "id" then it _often_ is, in practice.
		v := obj.GetAttr("name")
		if v.IsKnown() && !v.IsNull() {
			return "name", v.AsString()
		}
	}

	return "", ""
}

// ObjectValueName takes a value that is assumed to be an object representation
// of some resource instance object and attempts to heuristically find an
// attribute of it that is likely to be a human-friendly name in the remote
// system that it belongs to which will be useful to the user.
//
// If such an attribute is found, its name and string value intended for
// display are returned. Both returned strings are empty if no such attribute
// exists, in which case the caller should assume that the resource instance
// address within the Terraform configuration is the best available identifier.
//
// This is only a best-effort sort of thing, relying on naming conventions in
// our resource type schemas. The result is not guaranteed to be unique, but
// should generally be suitable for display to an end-user anyway.
//
// Callers that use both ObjectValueName and ObjectValueID at the same time
// should be prepared to get the same attribute key and value from both in
// some cases, since there is overlap betweek the id-extraction and
// name-extraction heuristics.
//
// This function will panic if the given value is not of an object type.
func ObjectValueName(obj cty.Value) (k, v string) {
	if obj.IsNull() || !obj.IsKnown() {
		return "", ""
	}

	atys := obj.Type().AttributeTypes()

	switch {

	case atys["name"] == cty.String:
		v := obj.GetAttr("name")
		if v.IsKnown() && !v.IsNull() {
			return "name", v.AsString()
		}

	case atys["tags"].IsMapType() && atys["tags"].ElementType() == cty.String:
		tags := obj.GetAttr("tags")
		if tags.IsNull() || !tags.IsWhollyKnown() {
			break
		}

		switch {
		case tags.HasIndex(cty.StringVal("name")).RawEquals(cty.True):
			v := tags.Index(cty.StringVal("name"))
			if v.IsKnown() && !v.IsNull() {
				return "tags.name", v.AsString()
			}
		case tags.HasIndex(cty.StringVal("Name")).RawEquals(cty.True):
			// AWS-style naming convention
			v := tags.Index(cty.StringVal("Name"))
			if v.IsKnown() && !v.IsNull() {
				return "tags.Name", v.AsString()
			}
		}
	}

	return "", ""
}

// ObjectValueIDOrName is a convenience wrapper around both ObjectValueID
// and ObjectValueName (in that preference order) to try to extract some sort
// of human-friendly descriptive string value for an object as additional
// context about an object when it is being displayed in a compact way (where
// not all of the attributes are visible.)
//
// Just as with the two functions it wraps, it is a best-effort and may return
// two empty strings if no suitable attribute can be found for a given object.
func ObjectValueIDOrName(obj cty.Value) (k, v string) {
	k, v = ObjectValueID(obj)
	if k != "" {
		return
	}
	k, v = ObjectValueName(obj)
	return
}
