package differ

import (
	"github.com/hashicorp/terraform/internal/command/jsonformat/change"
	"github.com/hashicorp/terraform/internal/command/jsonprovider"
)

func (v Value) ComputeChangeForBlock(block *jsonprovider.Block) change.Change {
	panic("not implemented")
}
