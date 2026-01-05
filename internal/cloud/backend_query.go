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
	viewsjson "github.com/hashicorp/terraform/internal/command/views/json"
	"github.com/hashicorp/terraform/internal/genconfig"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"
)

func (b *Cloud) opQuery(stopCtx, cancelCtx context.Context, op *backendrun.Operation, w *tfe.Workspace) (OperationResult, error) {
	log.Printf("[INFO] cloud: starting Query operation")

	var diags tfdiags.Diagnostics

	// TODO? maybe check workspace permissions if the user has the permission to run queries

	if len(op.GenerateConfigOut) > 0 {
		diags = diags.Append(genconfig.ValidateTargetFile(op.GenerateConfigOut))
	}

	if diags.HasErrors() {
		return &QueryRunResult{}, diags.Err()
	}

	return b.query(stopCtx, cancelCtx, op, w)
}

func (b *Cloud) query(stopCtx, cancelCtx context.Context, op *backendrun.Operation, w *tfe.Workspace) (OperationResult, error) {
	if b.CLI != nil {
		header := fmt.Sprintf(queryDefaultHeader, b.appName)
		b.CLI.Output(b.Colorize().Color(strings.TrimSpace(header) + "\n"))
	}

	configOptions := tfe.ConfigurationVersionCreateOptions{
		AutoQueueRuns: tfe.Bool(false),
	}
	cv, err := b.uploadConfigurationVersion(stopCtx, cancelCtx, op, w, configOptions)
	if err != nil {
		return &QueryRunResult{}, err
	}

	queryRunOptions := tfe.QueryRunCreateOptions{
		ConfigurationVersion: cv,
		Workspace:            w,
		Source:               tfe.QueryRunSourceAPI,
	}

	runVariables, err := b.parseRunVariables(op)
	if err != nil {
		return nil, err
	}
	queryRunOptions.Variables = runVariables

	r, err := b.client.QueryRuns.Create(stopCtx, queryRunOptions)
	if err != nil {
		return &QueryRunResult{}, b.generalError("Failed to create query run", err)
	}

	if b.CLI != nil {
		// TODO replace with URL from create response, once available
		b.CLI.Output(b.Colorize().Color(strings.TrimSpace(fmt.Sprintf(
			queryRunHeader, b.Hostname, b.Organization, op.Workspace, r.ID)) + "\n"))
	}

	r, err = b.waitForQueryRun(stopCtx, cancelCtx, r)
	if err != nil {
		return &QueryRunResult{run: r, backend: b}, err
	}

	err = b.renderQueryRunLogs(stopCtx, op, r)
	if err != nil {
		return &QueryRunResult{run: r, backend: b}, err
	}

	return &QueryRunResult{run: r, backend: b}, nil
}

func (b *Cloud) renderQueryRunLogs(ctx context.Context, op *backendrun.Operation, run *tfe.QueryRun) error {
	logs, err := b.client.QueryRuns.Logs(ctx, run.ID)
	if err != nil {
		return err
	}
	configs := map[string]string{}
	wantConfig := len(op.GenerateConfigOut) > 0

	if b.CLI != nil {
		reader := bufio.NewReaderSize(logs, 64*1024)
		results := map[string][]*viewsjson.QueryResult{}

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

				if b.renderer != nil {
					// Instead of using renderer.RenderLog for individual log messages,
					// we collect all logs of a list block and output them at once.
					// This allows us to ensure all messages of a list block are grouped
					// and indented as in the PostListQuery hook.
					switch log.Type {
					case jsonformat.LogListStart:
						results[log.ListQueryStart.Address] = make([]*viewsjson.QueryResult, 0)
					case jsonformat.LogListResourceFound:
						results[log.ListQueryResult.Address] = append(results[log.ListQueryResult.Address], log.ListQueryResult)
						if wantConfig {
							configs[log.ListQueryResult.Address] +=
								fmt.Sprintf("%s\n%s\n\n", log.ListQueryResult.Config, log.ListQueryResult.ImportConfig)
						}
					case jsonformat.LogListComplete:
						addr := log.ListQueryComplete.Address

						identities := make([]string, 0, len(results[addr]))
						displayNames := make([]string, 0, len(results[addr]))
						maxIdentityLen := 0
						for _, result := range results[addr] {
							identity := formatIdentity(result.Identity)
							if len(identity) > maxIdentityLen {
								maxIdentityLen = len(identity)
							}
							identities = append(identities, identity)

							displayNames = append(displayNames, result.DisplayName)
						}

						result := strings.Builder{}
						for i, identity := range identities {
							result.WriteString(fmt.Sprintf("%s   %-*s   %s\n", addr, maxIdentityLen, identity, displayNames[i]))
						}

						if result.Len() > 0 {
							b.renderer.Streams.Println(result.String())
						}
					default:
						err := b.renderer.RenderLog(log)
						if err != nil {
							return err
						}
					}
				}
			}
		}
	}

	if wantConfig && len(configs) > 0 {
		diags := genconfig.ValidateTargetFile(op.GenerateConfigOut)
		if diags.HasErrors() {
			return diags.Err()
		}

		var writer io.Writer
		for addr, config := range configs {
			change := genconfig.Change{
				Addr:            addr,
				GeneratedConfig: config,
			}

			writer, _, diags = change.MaybeWriteConfig(writer, op.GenerateConfigOut)
			if diags.HasErrors() {
				return diags.Err()
			}
		}
	}

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

// formatIdentity formats the identity map into a string representation.
// It flattens the map into a string of key=value pairs, separated by commas.
func formatIdentity(identity map[string]json.RawMessage) string {
	ctyObj := make(map[string]cty.Value, len(identity))
	for key, value := range identity {
		ty, err := ctyjson.ImpliedType(value)
		if err != nil {
			continue
		}
		v, err := ctyjson.Unmarshal(value, ty)
		if err != nil {
			continue
		}
		ctyObj[key] = v
	}
	return tfdiags.ObjectToString(cty.ObjectVal(ctyObj))
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
