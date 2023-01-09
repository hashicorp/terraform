package differ

import (
	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"

	"github.com/hashicorp/terraform/internal/command/jsonformat/change"
	"github.com/hashicorp/terraform/internal/command/jsonprovider"
)

func (v Value) ComputeChangeForAttribute(attribute *jsonprovider.Attribute) change.Change {
	return v.ComputeChangeForType(unmarshalAttribute(attribute))
}

func (v Value) ComputeChangeForType(ctyType cty.Type) change.Change {
	switch {
	case ctyType.IsPrimitiveType():
		return v.computeAttributeChangeAsPrimitive(ctyType)
	default:
		panic("not implemented")
	}
}

func unmarshalAttribute(attribute *jsonprovider.Attribute) cty.Type {
	if attribute.AttributeNestedType != nil {
		children := make(map[string]cty.Type)
		for key, child := range attribute.AttributeNestedType.Attributes {
			children[key] = unmarshalAttribute(child)
		}
		return cty.Object(children)
	}

	ctyType, err := ctyjson.UnmarshalType(attribute.AttributeType)
	if err != nil {
		panic("could not unmarshal attribute type: " + err.Error())
	}
	return ctyType
}
