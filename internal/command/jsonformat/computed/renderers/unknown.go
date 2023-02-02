package renderers

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/command/jsonformat/computed"

	"github.com/hashicorp/terraform/internal/plans"
)

var _ computed.DiffRenderer = (*unknownRenderer)(nil)

func Unknown(before computed.Diff) computed.DiffRenderer {
	return &unknownRenderer{
		before: before,
	}
}

type unknownRenderer struct {
	NoWarningsRenderer

	before computed.Diff
}

func (renderer unknownRenderer) RenderHuman(diff computed.Diff, indent int, opts computed.RenderHumanOpts) string {
	if diff.Action == plans.Create {
		return "(known after apply)"
	}

	// Never render null suffix for children of unknown changes.
	opts.OverrideNullSuffix = true
	return fmt.Sprintf("%s -> (known after apply)", renderer.before.RenderHuman(indent, opts))
}
