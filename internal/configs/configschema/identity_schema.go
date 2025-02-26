// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package configschema

import (
	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/zclconf/go-cty/cty"
)

// IdentityAttributes represents a set of attributes in an identity schema
type IdentityAttributes map[string]*IdentityAttribute

// replace by object or maybe block?

// IdentityAttribute represents a single attribute in an identity schema
type IdentityAttribute struct {
	// Type is a type specification that the attribute's value must conform to.
	Type cty.Type

	Description string

	RequiredForImport bool
	OptionalForImport bool
}

func (ia *IdentityAttributes) ImpliedType() cty.Type {
	return ia.specType().WithoutOptionalAttributesDeep()
}

func (ia *IdentityAttributes) specType() cty.Type {
	if ia == nil {
		return cty.EmptyObject
	}

	return hcldec.ImpliedType(ia.DecoderSpec())
}

func (ia *IdentityAttributes) DecoderSpec() hcldec.Spec {
	ret := hcldec.ObjectSpec{}
	if ia == nil {
		return ret
	}

	for name, attrS := range *ia {
		ret[name] = attrS.decoderSpec(name)
	}

	return ret
}

func (ia *IdentityAttribute) decoderSpec(name string) hcldec.Spec {
	if ia == nil || ia.Type == cty.NilType {
		panic("Invalid attribute schema: schema is nil.")
	}

	ret := &hcldec.AttrSpec{Name: name}
	ret.Type = ia.Type
	// When dealing with IdentityAttribute we expect every attribute to be required.
	// This is generally true for all communication between providers and Terraform.
	// For import, we allow the user to only specify a subset of the attributes, where
	// RequiredForImport attributes are required and OptionalForImport attributes are optional.
	// The validation for this will rely on a separate spec.
	ret.Required = true

	return ret
}
