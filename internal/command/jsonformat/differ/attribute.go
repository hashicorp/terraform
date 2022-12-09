package differ

import (
	"github.com/hashicorp/terraform/internal/command/jsonformat/change"
	"github.com/hashicorp/terraform/internal/command/jsonprovider"
)

func (v Value) ComputeChangeForAttribute(attribute *jsonprovider.Attribute) change.Change {
	panic("not implemented")
}
