package differ

import (
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/command/jsonformat/change"
)

func (v Value) computeAttributeChangeAsPrimitive(ctyType cty.Type) change.Change {
	return v.AsChange(change.Primitive(v.Before, v.After, ctyType))
}
