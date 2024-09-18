package structured

import (
	"github.com/hashicorp/terraform/internal/command/jsonformat/computed"
	"github.com/hashicorp/terraform/internal/plans"
)

type ProcessEphemeralInner func(change Change) computed.Diff
type CreateEphemeralDiff func(inner computed.Diff, beforeEphemeral, afterEphemeral bool, action plans.Action) computed.Diff

func (change Change) CheckForEphemeral(processInner ProcessEphemeralInner, createDiff CreateEphemeralDiff) (computed.Diff, bool) {
	if !change.IsBeforeEphemeral() && !change.IsAfterEphemeral() {
		return computed.Diff{}, false
	}

	value := Change{
		BeforeExplicit:     change.BeforeExplicit,
		AfterExplicit:      change.AfterExplicit,
		Before:             change.Before,
		After:              change.After,
		Unknown:            change.Unknown,
		BeforeSensitive:    change.BeforeSensitive,
		AfterSensitive:     change.AfterSensitive,
		BeforeEphemeral:    change.BeforeEphemeral,
		AfterEphemeral:     change.AfterEphemeral,
		ReplacePaths:       change.ReplacePaths,
		RelevantAttributes: change.RelevantAttributes,
	}

	// TODO: this turns create into update
	inner := processInner(value)

	return createDiff(inner, change.IsBeforeEphemeral(), change.IsAfterEphemeral(), inner.Action), true
}
