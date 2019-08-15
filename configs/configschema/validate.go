package configschema

import (
	"fmt"

	"github.com/zclconf/go-cty/cty"
)

// Validate ensures a cty.Value conforms to the schema precisely.
func (b Block) Validate(val cty.Value) error {
	if err := b.validate(val); err != nil {
		return fmt.Errorf("validate error: %s", err)
	}
	return nil
}
func (b Block) validate(val cty.Value) error {
	if !val.IsKnown() {
		return nil
	}

	if !val.Type().IsObjectType() {
		return fmt.Errorf("value must be an object, got %#v", val.Type())
	}

	// get a list of all allowed object attributes
	allowedNames := map[string]struct{}{}
	for name := range b.Attributes {
		allowedNames[name] = struct{}{}
	}
	for name := range b.BlockTypes {
		allowedNames[name] = struct{}{}
	}

	valMap := map[string]cty.Value{}
	if !val.IsNull() {
		valMap = val.AsValueMap()
	}

	// verify that we don't have any unexpected attributes
	for name, attrVal := range valMap {
		if _, ok := allowedNames[name]; !ok {
			return fmt.Errorf("unexpected attribute %q: %#v", name, attrVal)
		}
	}

	for name, attr := range b.Attributes {
		attrVal, exists := valMap[name]
		// attrVal may be unknown, but we don't actually compare the value here

		if attr.Required {
			if !exists || attrVal.IsNull() {
				return fmt.Errorf("attribute %q is required", name)
			}
		}

		if !exists || attrVal.IsNull() {
			continue
		}

		if !attrVal.Type().Equals(attr.Type) {
			// the types must be exact, unless the schema allows any type
			if attr.Type != cty.DynamicPseudoType {
				return fmt.Errorf("attribute %q expected type %#v, got %#v", name, attr.Type, attrVal.Type())
			}
		}
	}

	for name, nested := range b.BlockTypes {
		blockVal := valMap[name]
		// NestingGroup cannot be null
		if nested.Nesting == NestingGroup && blockVal.IsNull() {
			return fmt.Errorf("block %q cannot be null", name)
		}

		// wait until the value is known to complete validation
		if !blockVal.IsKnown() {
			continue
		}

		// and only validate length once the value is wholly known
		if blockVal.IsWhollyKnown() {
			switch nested.Nesting {
			case NestingSingle:
				if blockVal.IsNull() && nested.MinItems == 1 {
					return fmt.Errorf("insufficient items for attribute %q; must have at least %d", name, nested.MinItems)
				}

			case NestingGroup:
				// Nested group isn't required during decode, but by this point it must have a value
				if blockVal.IsNull() {
					return fmt.Errorf("missing value for NestedGroup %q", name)
				}

			case NestingList, NestingSet, NestingMap:
				items := 0
				if !blockVal.IsNull() {
					items = blockVal.LengthInt()
				}

				if items < nested.MinItems {
					return fmt.Errorf("insufficient items for attribute %q; must have at least %d", name, nested.MinItems)
				}

				if nested.MaxItems > 0 && items > nested.MaxItems {
					return fmt.Errorf("too many items for attribute %q; cannot have more than %d", name, nested.MaxItems)
				}
			}
		}

		switch nested.Nesting {
		case NestingSingle, NestingGroup:
			if err := nested.Block.validate(blockVal); err != nil {
				return fmt.Errorf("%q: %s", name, err)
			}
		case NestingList:
			if !blockVal.Type().IsListType() {
				return fmt.Errorf("expected list for block %q, got %#v", name, blockVal)
			}

			for _, val := range blockVal.AsValueSlice() {
				if err := nested.Block.validate(val); err != nil {
					// add the block name to the error context
					return fmt.Errorf("%q: %s", name, err)
				}
			}
		case NestingSet:
			if !blockVal.Type().IsSetType() {
				return fmt.Errorf("expected set for block %q, got %#v", name, blockVal)
			}
			for _, val := range blockVal.AsValueSlice() {
				if err := nested.Block.validate(val); err != nil {
					// add the block name to the error context
					return fmt.Errorf("%q: %s", name, err)
				}
			}
		case NestingMap:
			if !blockVal.Type().IsMapType() && !blockVal.Type().IsObjectType() {
				return fmt.Errorf("expected map or object for block %q, got %#v", name, blockVal)
			}
			for key, val := range blockVal.AsValueMap() {
				if err := nested.Block.validate(val); err != nil {
					// add the block name and map key to the error context
					return fmt.Errorf("%s[%q]: %s", name, key, err)
				}
			}
		default:
			panic(fmt.Sprintf("invalid nesting mode: %s", nested.Nesting))
		}
	}
	return nil
}
