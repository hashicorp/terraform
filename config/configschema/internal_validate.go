package configschema

import (
	"fmt"
	"regexp"

	"github.com/zclconf/go-cty/cty"

	multierror "github.com/hashicorp/go-multierror"
)

var validName = regexp.MustCompile(`^[a-z0-9_]+$`)

// InternalValidate returns an error if the receiving block and its child
// schema definitions have any consistencies with the documented rules for
// valid schema.
//
// This is intended to be used within unit tests to detect when a given
// schema is invalid.
func (b *Block) InternalValidate() error {
	if b == nil {
		return fmt.Errorf("top-level block schema is nil")
	}
	return b.internalValidate("", nil)

}

func (b *Block) internalValidate(prefix string, err error) error {
	for name, attrS := range b.Attributes {
		if attrS == nil {
			err = multierror.Append(err, fmt.Errorf("%s%s: attribute schema is nil", prefix, name))
			continue
		}
		if !validName.MatchString(name) {
			err = multierror.Append(err, fmt.Errorf("%s%s: name may contain only lowercase letters, digits and underscores", prefix, name))
		}
		if attrS.Optional == false && attrS.Required == false && attrS.Computed == false {
			err = multierror.Append(err, fmt.Errorf("%s%s: must set Optional, Required or Computed", prefix, name))
		}
		if attrS.Optional && attrS.Required {
			err = multierror.Append(err, fmt.Errorf("%s%s: cannot set both Optional and Required", prefix, name))
		}
		if attrS.Computed && attrS.Required {
			err = multierror.Append(err, fmt.Errorf("%s%s: cannot set both Computed and Required", prefix, name))
		}
		if attrS.Type == cty.NilType {
			err = multierror.Append(err, fmt.Errorf("%s%s: Type must be set to something other than cty.NilType", prefix, name))
		}
	}

	for name, blockS := range b.BlockTypes {
		if blockS == nil {
			err = multierror.Append(err, fmt.Errorf("%s%s: block schema is nil", prefix, name))
			continue
		}

		if _, isAttr := b.Attributes[name]; isAttr {
			err = multierror.Append(err, fmt.Errorf("%s%s: name defined as both attribute and child block type", prefix, name))
		} else if !validName.MatchString(name) {
			err = multierror.Append(err, fmt.Errorf("%s%s: name may contain only lowercase letters, digits and underscores", prefix, name))
		}

		if blockS.MinItems < 0 || blockS.MaxItems < 0 {
			err = multierror.Append(err, fmt.Errorf("%s%s: MinItems and MaxItems must both be greater than zero", prefix, name))
		}

		switch blockS.Nesting {
		case NestingSingle:
			switch {
			case blockS.MinItems != blockS.MaxItems:
				err = multierror.Append(err, fmt.Errorf("%s%s: MinItems and MaxItems must match in NestingSingle mode", prefix, name))
			case blockS.MinItems < 0 || blockS.MinItems > 1:
				err = multierror.Append(err, fmt.Errorf("%s%s: MinItems and MaxItems must be set to either 0 or 1 in NestingSingle mode", prefix, name))
			}
		case NestingList, NestingSet:
			if blockS.MinItems > blockS.MaxItems && blockS.MaxItems != 0 {
				err = multierror.Append(err, fmt.Errorf("%s%s: MinItems must be less than or equal to MaxItems in %s mode", prefix, name, blockS.Nesting))
			}
		case NestingMap:
			if blockS.MinItems != 0 || blockS.MaxItems != 0 {
				err = multierror.Append(err, fmt.Errorf("%s%s: MinItems and MaxItems must both be 0 in NestingMap mode", prefix, name))
			}
		default:
			err = multierror.Append(err, fmt.Errorf("%s%s: invalid nesting mode %s", prefix, name, blockS.Nesting))
		}

		subPrefix := prefix + name + "."
		err = blockS.Block.internalValidate(subPrefix, err)
	}

	return err
}
