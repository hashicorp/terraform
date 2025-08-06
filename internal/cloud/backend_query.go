// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cloud

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	tfe "github.com/hashicorp/go-tfe"
	"github.com/hashicorp/terraform/internal/backend/backendrun"
	"github.com/hashicorp/terraform/internal/command/jsonformat"
	"github.com/hashicorp/terraform/internal/genconfig"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func (b *Cloud) opQuery(stopCtx, cancelCtx context.Context, op *backendrun.Operation, w *tfe.Workspace) (*tfe.QueryRun, error) {
	log.Printf("[INFO] cloud: starting Query operation")

	var diags tfdiags.Diagnostics

	// TODO? maybe check workspace permissions if the user has the permission to run queries

	if len(op.GenerateConfigOut) > 0 {
		diags = diags.Append(genconfig.ValidateTargetFile(op.GenerateConfigOut))
	}

	if diags.HasErrors() {
		return nil, diags.Err()
	}

	return b.query(stopCtx, cancelCtx, op, w)
}

func (b *Cloud) query(stopCtx, cancelCtx context.Context, op *backendrun.Operation, w *tfe.Workspace) (*tfe.QueryRun, error) {
	if b.CLI != nil {
		header := fmt.Sprintf(queryDefaultHeader, b.appName)
		b.CLI.Output(b.Colorize().Color(strings.TrimSpace(header) + "\n"))
	}

	configOptions := tfe.ConfigurationVersionCreateOptions{
		AutoQueueRuns: tfe.Bool(false),
	}
	cv, err := b.uploadConfigurationVersion(stopCtx, cancelCtx, op, w, configOptions)
	if err != nil {
		return nil, err
	}

	queryRunOptions := tfe.QueryRunCreateOptions{
		ConfigurationVersion: cv,
		Workspace:            w,
		Source:               tfe.QueryRunSourceAPI,
	}

	// TODO variables

	r, err := b.client.QueryRuns.Create(stopCtx, queryRunOptions)
	if err != nil {
		return nil, b.generalError("Failed to create query run", err)
	}

	if b.CLI != nil {
		// TODO replace with URL from create response, once available
		b.CLI.Output(b.Colorize().Color(strings.TrimSpace(fmt.Sprintf(
			queryRunHeader, b.Hostname, b.Organization, op.Workspace, r.ID)) + "\n"))
	}

	r, err = b.waitForQueryRun(stopCtx, cancelCtx, r)
	if err != nil {
		return r, err
	}

	err = b.renderQueryRunLogs(stopCtx, op, r)
	if err != nil {
		return r, err
	}

	return r, nil
}

func (b *Cloud) renderQueryRunLogs(ctx context.Context, op *backendrun.Operation, run *tfe.QueryRun) error {
	logs, err := b.client.QueryRuns.Logs(ctx, run.ID)
	if err != nil {
		return err
	}

	if b.CLI != nil {
		reader := bufio.NewReaderSize(logs, 64*1024)

		for next := true; next; {
			var l, line []byte
			var err error

			for isPrefix := true; isPrefix; {
				l, isPrefix, err = reader.ReadLine()
				if err != nil {
					if err != io.EOF {
						return b.generalError("Failed to read logs", err)
					}
					next = false
				}

				line = append(line, l...)
			}

			if next || len(line) > 0 {
				log := &jsonformat.JSONLog{}
				if err := json.Unmarshal(line, log); err != nil {
					// If we can not parse the line as JSON, we will simply
					// print the line. This maintains backwards compatibility for
					// users who do not wish to enable structured output in their
					// workspace.
					b.CLI.Output(string(line))
					continue
				}

				// We will ignore plan output, change summary or outputs logs
				// during the plan phase.
				if log.Type == jsonformat.LogOutputs ||
					log.Type == jsonformat.LogChangeSummary ||
					log.Type == jsonformat.LogPlannedChange {
					continue
				}

				if b.renderer != nil {
					// Otherwise, we will print the log
					err := b.renderer.RenderLog(log)
					if err != nil {
						return err
					}
				}
			}
		}
	}

	// Get the run's current status and include the workspace and plan. We will check if
	// the run has errored, if structured output is enabled, and if the plan
	run, err = b.client.QueryRuns.Read(ctx, run.ID)
	if err != nil {
		return err
	}

	// TODO maybe write configuration
	// if len(op.GenerateConfigOut) > 0 {
	// 	diags := maybeWriteGeneratedConfig(redactedPlan, op.GenerateConfigOut)
	// 	if diags.HasErrors() {
	// 		return diags.Err()
	// 	}
	// }

	return nil
}
func (b *Cloud) waitForQueryRun(stopCtx, cancelCtx context.Context, r *tfe.QueryRun) (*tfe.QueryRun, error) {
	started := time.Now()
	updated := started
	for i := 0; ; i++ {
		select {
		case <-stopCtx.Done():
			return r, stopCtx.Err()
		case <-cancelCtx.Done():
			return r, cancelCtx.Err()
		case <-time.After(backoff(backoffMin, backoffMax, i)):
			// Timer up, show status
		}

		// Retrieve the run to get its current status.
		r, err := b.client.QueryRuns.Read(stopCtx, r.ID)
		if err != nil {
			return r, b.generalError("Failed to retrieve run", err)
		}

		// Return if the run is no longer pending.
		if r.Status != tfe.QueryRunPending {
			if i == 0 && b.CLI != nil {
				b.CLI.Output(b.Colorize().Color("Waiting for the query run to start...\n"))
			}
			if i > 0 && b.CLI != nil {
				// Insert a blank line to separate the ouputs.
				b.CLI.Output("")
			}
			return r, nil
		}

		// Check if 30 seconds have passed since the last update.
		current := time.Now()
		if b.CLI != nil && (i == 0 || current.Sub(updated).Seconds() > 30) {
			updated = current
			elapsed := ""

			// Calculate and set the elapsed time.
			if i > 0 {
				elapsed = fmt.Sprintf(
					" (%s elapsed)", current.Sub(started).Truncate(30*time.Second))
			}

			b.CLI.Output(b.Colorize().Color(fmt.Sprintf("Waiting for the query run to start...%s", elapsed)))
		}
	}
}

func (b *Cloud) cancelQueryRun(cancelCtx context.Context, op *backendrun.Operation, r *tfe.QueryRun) error {
	v, err := op.UIIn.Input(cancelCtx, &terraform.InputOpts{
		Id:          "cancel",
		Query:       "\nDo you want to cancel the remote operation?",
		Description: "Only 'yes' will be accepted to cancel.",
	})
	if err != nil {
		return b.generalError("Failed asking to cancel", err)
	}
	if v != "yes" {
		if b.CLI != nil {
			b.CLI.Output(b.Colorize().Color(strings.TrimSpace(operationNotCanceled)))
		}
		return nil
	}

	// Try to cancel the remote operation.
	err = b.client.QueryRuns.Cancel(cancelCtx, r.ID)
	if err != nil {
		return b.generalError("Failed to cancel query run", err)
	}
	if b.CLI != nil {
		b.CLI.Output(b.Colorize().Color(strings.TrimSpace(operationCanceled)))
	}

	return nil
}

const queryDefaultHeader = `
[reset][yellow]Running query in %s. Output will stream here. Pressing Ctrl-C
will stop streaming the logs, but will not stop the query running remotely.[reset]

Preparing the remote query run...
`

const queryRunHeader = `
[reset][yellow]To view this query run in a browser, visit:[reset]
[reset][yellow]https://%s/app/%s/%s/search/%s[reset]
`
