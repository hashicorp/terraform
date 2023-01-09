package differ

import (
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/command/jsonformat/change"
)

func (v Value) computeAttributeChangeAsPrimitive(ctype cty.Type) change.Change {
	return v.asChange(change.Primitive(v.Before, v.After, ctype))
}
