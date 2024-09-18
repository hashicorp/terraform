package differ

import (
	"github.com/hashicorp/terraform/internal/command/jsonformat/computed"
	"github.com/hashicorp/terraform/internal/command/jsonformat/computed/renderers"
	"github.com/hashicorp/terraform/internal/command/jsonformat/structured"
)

type CreateEphemeralRenderer func(computed.Diff, bool, bool) computed.DiffRenderer

func checkForEphemeralType(change structured.Change) (computed.Diff, bool) {
	if !change.IsBeforeEphemeral() && !change.IsAfterEphemeral() {
		return computed.Diff{}, false
	}

	return computed.NewDiff(renderers.Ephemeral(change.PlanAction, change.IsBeforeEphemeral(), change.IsAfterEphemeral()), change.CalculateAction(), change.ReplacePaths.Matches()), true
}
