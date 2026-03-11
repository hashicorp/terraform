// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package structured

import (
	"github.com/hashicorp/terraform/internal/command/jsonformat/structured/attribute_path"
)

// ChangeMap is a Change that represents a Map or an Object type, and has
// converted the relevant interfaces into maps for easier access.
type ChangeMap struct {
	// Before contains the value before the proposed change.
	Before map[string]interface{}

	// After contains the value after the proposed change.
	After map[string]interface{}

	// Unknown contains the unknown status of any elements/attributes of this
	// map/object.
	Unknown map[string]interface{}

	// BeforeSensitive contains the before sensitive status of any
	// elements/attributes of this map/object.
	BeforeSensitive map[string]interface{}

	// AfterSensitive contains the after sensitive status of any
	// elements/attributes of this map/object.
	AfterSensitive map[string]interface{}

	// ReplacePaths matches the same attributes in Change exactly.
	ReplacePaths attribute_path.Matcher

	// RelevantAttributes matches the same attributes in Change exactly.
	RelevantAttributes attribute_path.Matcher

	// this reflects the parent NonLegacyValue, so that any behavior is
	// automatically inherited into child changes.
	nonLegacySchema bool
}

// AsMap converts the Change into an object or map representation by converting
// the internal Before, After, Unknown, BeforeSensitive, and AfterSensitive
// data structures into generic maps.
func (change Change) AsMap() ChangeMap {
	return ChangeMap{
		Before:             genericToMap(change.Before),
		After:              genericToMap(change.After),
		Unknown:            genericToMap(change.Unknown),
		BeforeSensitive:    genericToMap(change.BeforeSensitive),
		AfterSensitive:     genericToMap(change.AfterSensitive),
		ReplacePaths:       change.ReplacePaths,
		RelevantAttributes: change.RelevantAttributes,
		nonLegacySchema:    change.NonLegacySchema,
	}
}

// GetChild safely packages up a Change object for the given child, handling
// all the cases where the data might be null or a static boolean.
func (m ChangeMap) GetChild(key string) Change {
	before, beforeExplicit := getFromGenericMap(m.Before, key)
	after, afterExplicit := getFromGenericMap(m.After, key)
	unknown, _ := getFromGenericMap(m.Unknown, key)
	beforeSensitive, _ := getFromGenericMap(m.BeforeSensitive, key)
	afterSensitive, _ := getFromGenericMap(m.AfterSensitive, key)

	return Change{
		BeforeExplicit:     beforeExplicit,
		AfterExplicit:      afterExplicit,
		Before:             before,
		After:              after,
		Unknown:            unknown,
		BeforeSensitive:    beforeSensitive,
		AfterSensitive:     afterSensitive,
		ReplacePaths:       m.ReplacePaths.GetChildWithKey(key),
		RelevantAttributes: m.RelevantAttributes.GetChildWithKey(key),
		NonLegacySchema:    m.nonLegacySchema,
	}
}

// ExplicitKeys returns the keys in the Before and After, as opposed to AllKeys
// which also includes keys from the additional meta structures (like the
// sensitive and unknown values).
//
// This function is useful for processing nested attributes and repeated blocks
// where the unknown and sensitive structs contain information about the actual
// attributes, while the before and after structs hold the actual nested values.
func (m ChangeMap) ExplicitKeys() []string {
	keys := make(map[string]bool)
	for before := range m.Before {
		if _, ok := keys[before]; ok {
			continue
		}
		keys[before] = true
	}
	for after := range m.After {
		if _, ok := keys[after]; ok {
			continue
		}
		keys[after] = true
	}

	var dedupedKeys []string
	for key := range keys {
		dedupedKeys = append(dedupedKeys, key)
	}
	return dedupedKeys
}

// AllKeys returns all the possible keys for this map. The keys for the map are
// potentially hidden and spread across multiple internal data structures and
// so this function conveniently packages them up.
func (m ChangeMap) AllKeys() []string {
	keys := make(map[string]bool)
	for before := range m.Before {
		if _, ok := keys[before]; ok {
			continue
		}
		keys[before] = true
	}
	for after := range m.After {
		if _, ok := keys[after]; ok {
			continue
		}
		keys[after] = true
	}
	for unknown := range m.Unknown {
		if _, ok := keys[unknown]; ok {
			continue
		}
		keys[unknown] = true
	}
	for sensitive := range m.AfterSensitive {
		if _, ok := keys[sensitive]; ok {
			continue
		}
		keys[sensitive] = true
	}
	for sensitive := range m.BeforeSensitive {
		if _, ok := keys[sensitive]; ok {
			continue
		}
		keys[sensitive] = true
	}

	var dedupedKeys []string
	for key := range keys {
		dedupedKeys = append(dedupedKeys, key)
	}
	return dedupedKeys
}

func getFromGenericMap(generic map[string]interface{}, key string) (interface{}, bool) {
	if generic == nil {
		return nil, false
	}

	if child, ok := generic[key]; ok {
		return child, ok
	}
	return nil, false
}

func genericToMap(generic interface{}) map[string]interface{} {
	if concrete, ok := generic.(map[string]interface{}); ok {
		return concrete
	}
	return nil
}
