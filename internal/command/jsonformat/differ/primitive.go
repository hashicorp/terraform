package differ

import (
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/command/jsonformat/computed"

	"github.com/hashicorp/terraform/internal/command/jsonformat/computed/renderers"
)

func (change Change) computeAttributeDiffAsPrimitive(ctype cty.Type) computed.Diff {
	return change.asDiff(renderers.Primitive(change.Before, change.After, ctype))
}
