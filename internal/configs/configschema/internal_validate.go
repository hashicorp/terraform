// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package configschema

import (
	"fmt"
	"regexp"

	"github.com/zclconf/go-cty/cty"

	multierror "github.com/hashicorp/go-multierror"
)

var validName = regexp.MustCompile(`^[a-z0-9_]+$`)

// InternalValidate returns an error if the receiving block and its child schema
// definitions have any inconsistencies with the documented rules for valid
// schema.
//
// This can be used within unit tests to detect when a given schema is invalid,
// and is run when terraform loads provider schemas during NewContext.
func (b *Block) InternalValidate() error {
	if b == nil {
		return fmt.Errorf("top-level block schema is nil")
	}
	return b.internalValidate("")
}

func (b *Block) internalValidate(prefix string) error {
	var multiErr *multierror.Error

	for name, attrS := range b.Attributes {
		if attrS == nil {
			multiErr = multierror.Append(multiErr, fmt.Errorf("%s%s: attribute schema is nil", prefix, name))
			continue
		}
		multiErr = multierror.Append(multiErr, attrS.internalValidate(name, prefix))
	}

	for name, blockS := range b.BlockTypes {
		if blockS == nil {
			multiErr = multierror.Append(multiErr, fmt.Errorf("%s%s: block schema is nil", prefix, name))
			continue
		}

		if _, isAttr := b.Attributes[name]; isAttr {
			multiErr = multierror.Append(multiErr, fmt.Errorf("%s%s: name defined as both attribute and child block type", prefix, name))
		} else if !validName.MatchString(name) {
			multiErr = multierror.Append(multiErr, fmt.Errorf("%s%s: name may contain only lowercase letters, digits and underscores", prefix, name))
		}

		if blockS.MinItems < 0 || blockS.MaxItems < 0 {
			multiErr = multierror.Append(multiErr, fmt.Errorf("%s%s: MinItems and MaxItems must both be greater than zero", prefix, name))
		}

		switch blockS.Nesting {
		case NestingSingle:
			switch {
			case blockS.MinItems != blockS.MaxItems:
				multiErr = multierror.Append(multiErr, fmt.Errorf("%s%s: MinItems and MaxItems must match in NestingSingle mode", prefix, name))
			case blockS.MinItems < 0 || blockS.MinItems > 1:
				multiErr = multierror.Append(multiErr, fmt.Errorf("%s%s: MinItems and MaxItems must be set to either 0 or 1 in NestingSingle mode", prefix, name))
			}
		case NestingGroup:
			if blockS.MinItems != 0 || blockS.MaxItems != 0 {
				multiErr = multierror.Append(multiErr, fmt.Errorf("%s%s: MinItems and MaxItems cannot be used in NestingGroup mode", prefix, name))
			}
		case NestingList, NestingSet:
			if blockS.MinItems > blockS.MaxItems && blockS.MaxItems != 0 {
				multiErr = multierror.Append(multiErr, fmt.Errorf("%s%s: MinItems must be less than or equal to MaxItems in %s mode", prefix, name, blockS.Nesting))
			}
			if blockS.Nesting == NestingSet {
				ety := blockS.Block.ImpliedType()
				if ety.HasDynamicTypes() {
					// This is not permitted because the HCL (cty) set implementation
					// needs to know the exact type of set elements in order to
					// properly hash them, and so can't support mixed types.
					multiErr = multierror.Append(multiErr, fmt.Errorf("%s%s: NestingSet blocks may not contain attributes of cty.DynamicPseudoType", prefix, name))
				}
			}
		case NestingMap:
			if blockS.MinItems != 0 || blockS.MaxItems != 0 {
				multiErr = multierror.Append(multiErr, fmt.Errorf("%s%s: MinItems and MaxItems must both be 0 in NestingMap mode", prefix, name))
			}
		default:
			multiErr = multierror.Append(multiErr, fmt.Errorf("%s%s: invalid nesting mode %s", prefix, name, blockS.Nesting))
		}

		subPrefix := prefix + name + "."
		multiErr = multierror.Append(multiErr, blockS.Block.internalValidate(subPrefix))
	}

	return multiErr.ErrorOrNil()
}

// InternalValidate returns an error if the receiving attribute and its child
// schema definitions have any inconsistencies with the documented rules for
// valid schema.
func (a *Attribute) InternalValidate(name string) error {
	if a == nil {
		return fmt.Errorf("attribute schema is nil")
	}
	return a.internalValidate(name, "")
}

func (a *Attribute) internalValidate(name, prefix string) error {
	var err *multierror.Error

	/* FIXME: this validation breaks certain existing providers and cannot be enforced without coordination.
	if !validName.MatchString(name) {
		err = multierror.Append(err, fmt.Errorf("%s%s: name may contain only lowercase letters, digits and underscores", prefix, name))
	}
	*/
	if !a.Optional && !a.Required && !a.Computed {
		err = multierror.Append(err, fmt.Errorf("%s%s: must set Optional, Required or Computed", prefix, name))
	}
	if a.Optional && a.Required {
		err = multierror.Append(err, fmt.Errorf("%s%s: cannot set both Optional and Required", prefix, name))
	}
	if a.Computed && a.Required {
		err = multierror.Append(err, fmt.Errorf("%s%s: cannot set both Computed and Required", prefix, name))
	}

	if a.Type == cty.NilType && a.NestedType == nil {
		err = multierror.Append(err, fmt.Errorf("%s%s: either Type or NestedType must be defined", prefix, name))
	}

	if a.Type != cty.NilType {
		if a.NestedType != nil {
			err = multierror.Append(fmt.Errorf("%s: Type and NestedType cannot both be set", name))
		}
	}

	if a.NestedType != nil {
		switch a.NestedType.Nesting {
		case NestingSingle, NestingMap:
			// no validations to perform
		case NestingList, NestingSet:
			if a.NestedType.Nesting == NestingSet {
				ety := a.ImpliedType()
				if ety.HasDynamicTypes() {
					// This is not permitted because the HCL (cty) set implementation
					// needs to know the exact type of set elements in order to
					// properly hash them, and so can't support mixed types.
					err = multierror.Append(err, fmt.Errorf("%s%s: NestingSet blocks may not contain attributes of cty.DynamicPseudoType", prefix, name))
				}
			}
		default:
			err = multierror.Append(err, fmt.Errorf("%s%s: invalid nesting mode %s", prefix, name, a.NestedType.Nesting))
		}
		for name, attrS := range a.NestedType.Attributes {
			if attrS == nil {
				err = multierror.Append(err, fmt.Errorf("%s%s: attribute schema is nil", prefix, name))
				continue
			}
			err = multierror.Append(err, attrS.internalValidate(name, prefix))
		}
	}

	return err.ErrorOrNil()
}
