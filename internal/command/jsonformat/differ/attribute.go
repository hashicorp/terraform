package differ

import (
	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"

	"github.com/hashicorp/terraform/internal/command/jsonformat/change"
	"github.com/hashicorp/terraform/internal/command/jsonprovider"
)

func (v Value) computeChangeForAttribute(attribute *jsonprovider.Attribute) change.Change {
	if attribute.AttributeNestedType != nil {
		return v.computeChangeForNestedAttribute(attribute.AttributeNestedType)
	}
	return v.computeChangeForType(unmarshalAttribute(attribute))
}

func (v Value) computeChangeForNestedAttribute(attribute *jsonprovider.NestedType) change.Change {
	switch attribute.NestingMode {
	case "single", "group":
		return v.computeAttributeChangeAsNestedObject(attribute.Attributes)
	case "map":
		return v.computeAttributeChangeAsNestedMap(attribute.Attributes)
	case "list":
		return v.computeAttributeChangeAsNestedList(attribute.Attributes)
	default:
		panic("unrecognized nesting mode: " + attribute.NestingMode)
	}
}

func (v Value) computeChangeForType(ctyType cty.Type) change.Change {
	switch {
	case ctyType.IsPrimitiveType():
		return v.computeAttributeChangeAsPrimitive(ctyType)
	case ctyType.IsObjectType():
		return v.computeAttributeChangeAsObject(ctyType.AttributeTypes())
	case ctyType.IsMapType():
		return v.computeAttributeChangeAsMap(ctyType.ElementType())
	case ctyType.IsListType():
		return v.computeAttributeChangeAsList(ctyType.ElementType())
	default:
		panic("unrecognized type: " + ctyType.FriendlyName())
	}
}

func unmarshalAttribute(attribute *jsonprovider.Attribute) cty.Type {
	ctyType, err := ctyjson.UnmarshalType(attribute.AttributeType)
	if err != nil {
		panic("could not unmarshal attribute type: " + err.Error())
	}
	return ctyType
}
