package differ

import (
	"fmt"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/command/jsonformat/change"
)

func strptr(str string) *string {
	return &str
}

func (v Value) computeAttributeChangeAsPrimitive(ctype cty.Type) change.Change {
	return v.asChange(change.Primitive(formatAsPrimitive(v.Before, ctype), formatAsPrimitive(v.After, ctype)))
}

func formatAsPrimitive(value interface{}, ctyType cty.Type) *string {
	if value == nil {
		return nil
	}

	switch {
	case ctyType == cty.String:
		return strptr(fmt.Sprintf("\"%s\"", value))
	case ctyType == cty.Bool:
		if value.(bool) {
			return strptr("true")
		}
		return strptr("false")
	case ctyType == cty.Number:
		return strptr(fmt.Sprintf("%g", value))
	default:
		panic("unrecognized primitive type: " + ctyType.FriendlyName())
	}
}
