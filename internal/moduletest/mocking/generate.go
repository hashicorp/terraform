// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package mocking

import (
	"fmt"
	"math/rand"
	"sort"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/configs/configschema"
)

var (
	// testRand and chars are used to generate random strings for the computed
	// values.
	//
	// If testRand is null, then the global random is used. This allows us to
	// seed tests for repeatable results.
	testRand *rand.Rand
	chars    = []rune("abcdefghijklmnopqrstuvwxyz0123456789")
)

// GenerateValueForAttribute accepts a configschema.Attribute and returns a
// valid value for that attribute.
func GenerateValueForAttribute(attribute *configschema.Attribute) cty.Value {
	if attribute.NestedType != nil {
		switch attribute.NestedType.Nesting {
		case configschema.NestingSingle, configschema.NestingGroup:
			var names []string
			for name := range attribute.NestedType.Attributes {
				names = append(names, name)
			}
			if len(names) == 0 {
				return cty.EmptyObjectVal
			}

			// Make the order we iterate through the attributes deterministic. We
			// are generating random strings in here so it's worth making the
			// operation repeatable.
			sort.Strings(names)

			children := make(map[string]cty.Value)
			for _, name := range names {
				children[name] = GenerateValueForAttribute(attribute.NestedType.Attributes[name])
			}
			return cty.ObjectVal(children)
		case configschema.NestingSet:
			return cty.SetValEmpty(attribute.ImpliedType().ElementType())
		case configschema.NestingList:
			return cty.ListValEmpty(attribute.ImpliedType().ElementType())
		case configschema.NestingMap:
			return cty.MapValEmpty(attribute.ImpliedType().ElementType())
		default:
			panic(fmt.Errorf("unknown nesting mode: %d", attribute.NestedType.Nesting))
		}
	}

	return GenerateValueForType(attribute.Type)
}

// GenerateValueForType accepts a cty.Type and returns a valid value for that
// type.
func GenerateValueForType(target cty.Type) cty.Value {
	switch {
	case target.IsPrimitiveType():
		switch target {
		case cty.String:
			return cty.StringVal(str(8))
		case cty.Number:
			return cty.Zero
		case cty.Bool:
			return cty.False
		default:
			panic(fmt.Errorf("unknown primitive type: %s", target.FriendlyName()))
		}
	case target.IsListType():
		return cty.ListValEmpty(target.ElementType())
	case target.IsSetType():
		return cty.SetValEmpty(target.ElementType())
	case target.IsMapType():
		return cty.MapValEmpty(target.ElementType())
	case target.IsObjectType():
		var attributes []string
		for attribute := range target.AttributeTypes() {
			attributes = append(attributes, attribute)
		}
		if len(attributes) == 0 {
			return cty.EmptyObjectVal
		}

		// Make the order we iterate through the attributes deterministic. We
		// are generating random strings in here so it's worth making the
		// operation repeatable.
		sort.Strings(attributes)

		children := make(map[string]cty.Value)
		for _, attribute := range attributes {
			children[attribute] = GenerateValueForType(target.AttributeType(attribute))
		}
		return cty.ObjectVal(children)
	case target == cty.DynamicPseudoType:
		// For dynamic types, we cannot generate a value that is guaranteed to
		// be valid. Instead, we return a null value. This means users will get
		// an error saying that the value is null, but it's better than an error
		// saying that the type is wrong which will be confusing.
		return cty.NullVal(cty.DynamicPseudoType)
	default:
		panic(fmt.Errorf("unknown complex type: %s", target.FriendlyName()))
	}
}

func str(n int) string {
	b := make([]rune, n)
	for i := range b {
		if testRand != nil {
			b[i] = chars[testRand.Intn(len(chars))]
		} else {
			b[i] = chars[rand.Intn(len(chars))]
		}
	}
	return string(b)
}
