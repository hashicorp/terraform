package jsonprovider

import (
	"encoding/json"

	"github.com/hashicorp/terraform/configs/configschema"
)

type attribute struct {
	AttributeType json.RawMessage `json:"type,omitempty"`
	Description   string          `json:"description,omitempty"`
	Required      bool            `json:"required,omitempty"`
	Optional      bool            `json:"optional,omitempty"`
	Computed      bool            `json:"computed,omitempty"`
	Sensitive     bool            `json:"sensitive,omitempty"`
}

func marshalAttribute(attr *configschema.Attribute) *attribute {
	attrTy, _ := attr.Type.MarshalJSON()

	return &attribute{
		AttributeType: attrTy,
		Description:   attr.Description,
		Required:      attr.Required,
		Optional:      attr.Optional,
		Computed:      attr.Computed,
		Sensitive:     attr.Sensitive,
	}
}
