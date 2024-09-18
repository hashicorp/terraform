package renderers

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/command/jsonformat/computed"
	"github.com/hashicorp/terraform/internal/plans"
)

var _ computed.DiffRenderer = (*ephemeralRenderer)(nil)

func Ephemeral(action plans.Action, beforeEphemeral, afterEphemeral bool) computed.DiffRenderer {
	return &ephemeralRenderer{
		planAction:      action,
		beforeEphemeral: beforeEphemeral,
		afterEphemeral:  afterEphemeral,
	}
}

type ephemeralRenderer struct {
	planAction      plans.Action
	beforeEphemeral bool
	afterEphemeral  bool
}

func (renderer ephemeralRenderer) RenderHuman(diff computed.Diff, indent int, opts computed.RenderHumanOpts) string {
	return fmt.Sprintf("(ephemeral value)%s%s", nullSuffix(diff.Action, opts), forcesReplacement(diff.Replace, opts))
}

func (renderer ephemeralRenderer) WarningsHuman(diff computed.Diff, indent int, opts computed.RenderHumanOpts) []string {
	if (renderer.beforeEphemeral == renderer.afterEphemeral) || renderer.planAction == plans.Create || renderer.planAction == plans.Delete {
		// Only display warnings for ephemeral values if they are changing from
		// being ephemeral or to being ephemeral and if they are not being
		// destroyed or created.
		return []string{}
	}

	var warning string
	if renderer.beforeEphemeral {
		warning = opts.Colorize.Color(fmt.Sprintf("  # [yellow]Warning[reset]: this attribute value will no longer be marked as ephemeral\n%s  # after applying this change.", formatIndent(indent)))
	} else {
		warning = opts.Colorize.Color(fmt.Sprintf("  # [yellow]Warning[reset]: this attribute value will be marked as ephemeral and will not\n%s  # display in UI output after applying this change.", formatIndent(indent)))
	}

	if renderer.planAction == plans.NoOp {
		return []string{fmt.Sprintf("%s The value is unchanged.", warning)}
	}
	return []string{warning}
}
