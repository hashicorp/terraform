package jsonprovider

import (
	"encoding/json"

	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/zclconf/go-cty/cty"
)

type attribute struct {
	AttributeType       json.RawMessage `json:"type,omitempty"`
	AttributeNestedType *nestedType     `json:"nested_type,omitempty"`
	Description         string          `json:"description,omitempty"`
	DescriptionKind     string          `json:"description_kind,omitempty"`
	Deprecated          bool            `json:"deprecated,omitempty"`
	Required            bool            `json:"required,omitempty"`
	Optional            bool            `json:"optional,omitempty"`
	Computed            bool            `json:"computed,omitempty"`
	Sensitive           bool            `json:"sensitive,omitempty"`
}

type nestedType struct {
	Attributes  map[string]*attribute `json:"attributes,omitempty"`
	NestingMode string                `json:"nesting_mode,omitempty"`
}

func marshalStringKind(sk configschema.StringKind) string {
	switch sk {
	default:
		return "plain"
	case configschema.StringMarkdown:
		return "markdown"
	}
}

func marshalAttribute(attr *configschema.Attribute) *attribute {
	ret := &attribute{
		Description:     attr.Description,
		DescriptionKind: marshalStringKind(attr.DescriptionKind),
		Required:        attr.Required,
		Optional:        attr.Optional,
		Computed:        attr.Computed,
		Sensitive:       attr.Sensitive,
		Deprecated:      attr.Deprecated,
	}

	// we're not concerned about errors because at this point the schema has
	// already been checked and re-checked.
	if attr.Type != cty.NilType {
		attrTy, _ := attr.Type.MarshalJSON()
		ret.AttributeType = attrTy
	}

	if attr.NestedType != nil {
		nestedTy := nestedType{
			NestingMode: nestingModeString(attr.NestedType.Nesting),
		}
		attrs := make(map[string]*attribute, len(attr.NestedType.Attributes))
		for k, attr := range attr.NestedType.Attributes {
			attrs[k] = marshalAttribute(attr)
		}
		nestedTy.Attributes = attrs
		ret.AttributeNestedType = &nestedTy
	}

	return ret
}
