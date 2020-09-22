package jsonprovider

import (
	"encoding/json"

	"github.com/hashicorp/terraform/configs/configschema"
)

type attribute struct {
	AttributeType   json.RawMessage `json:"type,omitempty"`
	Description     string          `json:"description,omitempty"`
	DescriptionKind string          `json:"description_kind,omitempty"`
	Deprecated      bool            `json:"deprecated,omitempty"`
	Required        bool            `json:"required,omitempty"`
	Optional        bool            `json:"optional,omitempty"`
	Computed        bool            `json:"computed,omitempty"`
	Sensitive       bool            `json:"sensitive,omitempty"`
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
	// we're not concerned about errors because at this point the schema has
	// already been checked and re-checked.
	attrTy, _ := attr.Type.MarshalJSON()

	return &attribute{
		AttributeType:   attrTy,
		Description:     attr.Description,
		DescriptionKind: marshalStringKind(attr.DescriptionKind),
		Required:        attr.Required,
		Optional:        attr.Optional,
		Computed:        attr.Computed,
		Sensitive:       attr.Sensitive,
		Deprecated:      attr.Deprecated,
	}
}
