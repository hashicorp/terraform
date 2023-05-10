// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package cloud

import (
	"context"
	"fmt"
	"strings"

	tfe "github.com/hashicorp/go-tfe"
)

const (
	changedPolicyEnforcementAction = "changed_policy_enforcements"
	changedTaskEnforcementAction   = "changed_task_enforcements"
	ignoredPolicySetAction         = "ignored_policy_sets"
)

func (b *Cloud) renderRunWarnings(ctx context.Context, client *tfe.Client, runId string) error {
	if b.CLI == nil {
		return nil
	}

	result, err := client.RunEvents.List(ctx, runId, nil)
	if err != nil {
		return err
	}
	if result == nil {
		return nil
	}

	// We don't have to worry about paging as the API doesn't support it yet
	for _, re := range result.Items {
		switch re.Action {
		case changedPolicyEnforcementAction, changedTaskEnforcementAction, ignoredPolicySetAction:
			if re.Description != "" {
				b.CLI.Warn(b.Colorize().Color(strings.TrimSpace(fmt.Sprintf(
					runWarningHeader, re.Description)) + "\n"))
			}
		}
	}

	return nil
}

const runWarningHeader = `
[reset][yellow]Warning:[reset] %s
`
