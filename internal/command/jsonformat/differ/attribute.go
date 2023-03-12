package differ

import (
	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"

	"github.com/hashicorp/terraform/internal/command/jsonformat/computed"

	"github.com/hashicorp/terraform/internal/command/jsonprovider"
)

func (change Change) ComputeDiffForAttribute(attribute *jsonprovider.Attribute) computed.Diff {
	if attribute.AttributeNestedType != nil {
		return change.computeDiffForNestedAttribute(attribute.AttributeNestedType)
	}
	return change.ComputeDiffForType(unmarshalAttribute(attribute))
}

func (change Change) computeDiffForNestedAttribute(nested *jsonprovider.NestedType) computed.Diff {
	if sensitive, ok := change.checkForSensitiveNestedAttribute(nested); ok {
		return sensitive
	}

	if computed, ok := change.checkForUnknownNestedAttribute(nested); ok {
		return computed
	}

	switch NestingMode(nested.NestingMode) {
	case nestingModeSingle, nestingModeGroup:
		return change.computeAttributeDiffAsNestedObject(nested.Attributes)
	case nestingModeMap:
		return change.computeAttributeDiffAsNestedMap(nested.Attributes)
	case nestingModeList:
		return change.computeAttributeDiffAsNestedList(nested.Attributes)
	case nestingModeSet:
		return change.computeAttributeDiffAsNestedSet(nested.Attributes)
	default:
		panic("unrecognized nesting mode: " + nested.NestingMode)
	}
}

func (change Change) ComputeDiffForType(ctype cty.Type) computed.Diff {
	if sensitive, ok := change.checkForSensitiveType(ctype); ok {
		return sensitive
	}

	if computed, ok := change.checkForUnknownType(ctype); ok {
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
		return change.ComputeDiffForOutput()
	case ctype.IsPrimitiveType():
		return change.computeAttributeDiffAsPrimitive(ctype)
	case ctype.IsObjectType():
		return change.computeAttributeDiffAsObject(ctype.AttributeTypes())
	case ctype.IsMapType():
		return change.computeAttributeDiffAsMap(ctype.ElementType())
	case ctype.IsListType():
		return change.computeAttributeDiffAsList(ctype.ElementType())
	case ctype.IsTupleType():
		return change.computeAttributeDiffAsTuple(ctype.TupleElementTypes())
	case ctype.IsSetType():
		return change.computeAttributeDiffAsSet(ctype.ElementType())
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
