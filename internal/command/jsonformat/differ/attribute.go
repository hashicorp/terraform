package differ

import (
	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"

	"github.com/hashicorp/terraform/internal/command/jsonformat/change"
	"github.com/hashicorp/terraform/internal/command/jsonprovider"
)

func (v Value) ComputeChangeForAttribute(attribute *jsonprovider.Attribute) change.Change {
	if attribute.AttributeNestedType != nil {
		return v.computeChangeForNestedAttribute(attribute.AttributeNestedType)
	}
	return v.computeChangeForType(unmarshalAttribute(attribute))
}

func (v Value) computeChangeForNestedAttribute(nested *jsonprovider.NestedType) change.Change {
	if sensitive, ok := v.checkForSensitiveNestedAttribute(nested); ok {
		return sensitive
	}

	if computed, ok := v.checkForComputedNestedAttribute(nested); ok {
		return computed
	}

	switch NestingMode(nested.NestingMode) {
	case nestingModeSingle, nestingModeGroup:
		return v.computeAttributeChangeAsNestedObject(nested.Attributes)
	case nestingModeMap:
		return v.computeAttributeChangeAsNestedMap(nested.Attributes)
	case nestingModeList:
		return v.computeAttributeChangeAsNestedList(nested.Attributes)
	case nestingModeSet:
		return v.computeAttributeChangeAsNestedSet(nested.Attributes)
	default:
		panic("unrecognized nesting mode: " + nested.NestingMode)
	}
}

func (v Value) computeChangeForType(ctype cty.Type) change.Change {
	if sensitive, ok := v.checkForSensitiveType(ctype); ok {
		return sensitive
	}

	if computed, ok := v.checkForComputedType(ctype); ok {
		return computed
	}

	switch {
	case ctype == cty.NilType, ctype == cty.DynamicPseudoType:
		// Forward nil or dynamic types over to be processed as outputs.
		// There is nothing particularly special about the way outputs are
		// processed that make this unsafe, we could just as easily call this
		// function computeChangeForDynamicValues(), but external callers will
		// only be in this situation when processing outputs so this function
		// is named for their benefit.
		return v.ComputeChangeForOutput()
	case ctype.IsPrimitiveType():
		return v.computeAttributeChangeAsPrimitive(ctype)
	case ctype.IsObjectType():
		return v.computeAttributeChangeAsObject(ctype.AttributeTypes())
	case ctype.IsMapType():
		return v.computeAttributeChangeAsMap(ctype.ElementType())
	case ctype.IsListType():
		return v.computeAttributeChangeAsList(ctype.ElementType())
	case ctype.IsTupleType():
		return v.computeAttributeChangeAsTuple(ctype.TupleElementTypes())
	case ctype.IsSetType():
		return v.computeAttributeChangeAsSet(ctype.ElementType())
	default:
		panic("unrecognized type: " + ctype.FriendlyName())
	}
}

func unmarshalAttribute(attribute *jsonprovider.Attribute) cty.Type {
	ctyType, err := ctyjson.UnmarshalType(attribute.AttributeType)
	if err != nil {
		panic("could not unmarshal attribute type: " + err.Error())
	}
	return ctyType
}
