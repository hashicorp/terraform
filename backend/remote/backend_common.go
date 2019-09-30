package remote

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"time"

	tfe "github.com/hashicorp/go-tfe"
	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/terraform"
	"github.com/hashicorp/terraform/tfdiags"
)

var (
	errApplyDiscarded   = errors.New("Apply discarded.")
	errDestroyDiscarded = errors.New("Destroy discarded.")
	errRunApproved      = errors.New("approved using the UI or API")
	errRunDiscarded     = errors.New("discarded using the UI or API")
	errRunOverridden    = errors.New("overridden using the UI or API")
)

// backoff will perform exponential backoff based on the iteration and
// limited by the provided min and max (in milliseconds) durations.
func backoff(min, max float64, iter int) time.Duration {
	backoff := math.Pow(2, float64(iter)/5) * min
	if backoff > max {
		backoff = max
	}
	return time.Duration(backoff) * time.Millisecond
}

func (b *Remote) waitForRun(stopCtx, cancelCtx context.Context, op *backend.Operation, opType string, r *tfe.Run, w *tfe.Workspace) (*tfe.Run, error) {
	started := time.Now()
	updated := started
	for i := 0; ; i++ {
		select {
		case <-stopCtx.Done():
			return r, stopCtx.Err()
		case <-cancelCtx.Done():
			return r, cancelCtx.Err()
		case <-time.After(backoff(1000, 3000, i)):
			// Timer up, show status
		}

		// Retrieve the run to get its current status.
		r, err := b.client.Runs.Read(stopCtx, r.ID)
		if err != nil {
			return r, generalError("Failed to retrieve run", err)
		}

		// Return if the run is no longer pending.
		if r.Status != tfe.RunPending && r.Status != tfe.RunConfirmed {
			if i == 0 && opType == "plan" && b.CLI != nil {
				b.CLI.Output(b.Colorize().Color(fmt.Sprintf("Waiting for the %s to start...\n", opType)))
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
			position := 0
			elapsed := ""

			// Calculate and set the elapsed time.
			if i > 0 {
				elapsed = fmt.Sprintf(
					" (%s elapsed)", current.Sub(started).Truncate(30*time.Second))
			}

			// Retrieve the workspace used to run this operation in.
			w, err = b.client.Workspaces.Read(stopCtx, b.organization, w.Name)
			if err != nil {
				return nil, generalError("Failed to retrieve workspace", err)
			}

			// If the workspace is locked the run will not be queued and we can
			// update the status without making any expensive calls.
			if w.Locked && w.CurrentRun != nil {
				cr, err := b.client.Runs.Read(stopCtx, w.CurrentRun.ID)
				if err != nil {
					return r, generalError("Failed to retrieve current run", err)
				}
				if cr.Status == tfe.RunPending {
					b.CLI.Output(b.Colorize().Color(
						"Waiting for the manually locked workspace to be unlocked..." + elapsed))
					continue
				}
			}

			// Skip checking the workspace queue when we are the current run.
			if w.CurrentRun == nil || w.CurrentRun.ID != r.ID {
				found := false
				options := tfe.RunListOptions{}
			runlist:
				for {
					rl, err := b.client.Runs.List(stopCtx, w.ID, options)
					if err != nil {
						return r, generalError("Failed to retrieve run list", err)
					}

					// Loop through all runs to calculate the workspace queue position.
					for _, item := range rl.Items {
						if !found {
							if r.ID == item.ID {
								found = true
							}
							continue
						}

						// If the run is in a final state, ignore it and continue.
						switch item.Status {
						case tfe.RunApplied, tfe.RunCanceled, tfe.RunDiscarded, tfe.RunErrored:
							continue
						case tfe.RunPlanned:
							if op.Type == backend.OperationTypePlan {
								continue
							}
						}

						// Increase the workspace queue position.
						position++

						// Stop searching when we reached the current run.
						if w.CurrentRun != nil && w.CurrentRun.ID == item.ID {
							break runlist
						}
					}

					// Exit the loop when we've seen all pages.
					if rl.CurrentPage >= rl.TotalPages {
						break
					}

					// Update the page number to get the next page.
					options.PageNumber = rl.NextPage
				}

				if position > 0 {
					b.CLI.Output(b.Colorize().Color(fmt.Sprintf(
						"Waiting for %d run(s) to finish before being queued...%s",
						position,
						elapsed,
					)))
					continue
				}
			}

			options := tfe.RunQueueOptions{}
		search:
			for {
				rq, err := b.client.Organizations.RunQueue(stopCtx, b.organization, options)
				if err != nil {
					return r, generalError("Failed to retrieve queue", err)
				}

				// Search through all queued items to find our run.
				for _, item := range rq.Items {
					if r.ID == item.ID {
						position = item.PositionInQueue
						break search
					}
				}

				// Exit the loop when we've seen all pages.
				if rq.CurrentPage >= rq.TotalPages {
					break
				}

				// Update the page number to get the next page.
				options.PageNumber = rq.NextPage
			}

			if position > 0 {
				c, err := b.client.Organizations.Capacity(stopCtx, b.organization)
				if err != nil {
					return r, generalError("Failed to retrieve capacity", err)
				}
				b.CLI.Output(b.Colorize().Color(fmt.Sprintf(
					"Waiting for %d queued run(s) to finish before starting...%s",
					position-c.Running,
					elapsed,
				)))
				continue
			}

			b.CLI.Output(b.Colorize().Color(fmt.Sprintf(
				"Waiting for the %s to start...%s", opType, elapsed)))
		}
	}
}

func (b *Remote) parseVariableValues(op *backend.Operation) (terraform.InputValues, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	result := make(terraform.InputValues)

	// Load the configuration using the caller-provided configuration loader.
	config, _, configDiags := op.ConfigLoader.LoadConfigWithSnapshot(op.ConfigDir)
	diags = diags.Append(configDiags)
	if diags.HasErrors() {
		return nil, diags
	}

	variables, varDiags := backend.ParseVariableValues(op.Variables, config.Module.Variables)
	diags = diags.Append(varDiags)
	if diags.HasErrors() {
		return nil, diags
	}

	// Save only the explicitly defined variables.
	for k, v := range variables {
		switch v.SourceType {
		case terraform.ValueFromCLIArg, terraform.ValueFromNamedFile:
			result[k] = v
		}
	}

	return result, diags
}

func (b *Remote) costEstimate(stopCtx, cancelCtx context.Context, op *backend.Operation, r *tfe.Run) error {
	if r.CostEstimate == nil {
		return nil
	}

	if b.CLI != nil {
		b.CLI.Output("\n------------------------------------------------------------------------\n")
	}

	msgPrefix := "Cost estimation"
	if b.CLI != nil {
		b.CLI.Output(b.Colorize().Color(msgPrefix + ":\n"))
	}

	started := time.Now()
	updated := started
	for i := 0; ; i++ {
		select {
		case <-stopCtx.Done():
			return stopCtx.Err()
		case <-cancelCtx.Done():
			return cancelCtx.Err()
		case <-time.After(1 * time.Second):
		}

		// Retrieve the cost estimate to get its current status.
		ce, err := b.client.CostEstimates.Read(stopCtx, r.CostEstimate.ID)
		if err != nil {
			return generalError("Failed to retrieve cost estimate", err)
		}

		switch ce.Status {
		case tfe.CostEstimateFinished:
			delta, err := strconv.ParseFloat(ce.DeltaMonthlyCost, 64)
			if err != nil {
				return generalError("Unexpected error", err)
			}

			sign := "+"
			if delta < 0 {
				sign = "-"
			}

			deltaRepr := strings.Replace(ce.DeltaMonthlyCost, "-", "", 1)

			if b.CLI != nil {
				b.CLI.Output(b.Colorize().Color(fmt.Sprintf("Resources: %d of %d estimated", ce.MatchedResourcesCount, ce.ResourcesCount)))
				b.CLI.Output(b.Colorize().Color(fmt.Sprintf("           $%s/mo %s$%s", ce.ProposedMonthlyCost, sign, deltaRepr)))

				if len(r.PolicyChecks) == 0 && r.HasChanges && op.Type == backend.OperationTypeApply {
					b.CLI.Output("\n------------------------------------------------------------------------")
				}
			}

			return nil
		case tfe.CostEstimatePending, tfe.CostEstimateQueued:
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
				b.CLI.Output(b.Colorize().Color("Waiting for cost estimate to complete..." + elapsed + "\n"))
			}
			continue
		case tfe.CostEstimateErrored:
			return fmt.Errorf(msgPrefix + " errored.")
		case tfe.CostEstimateCanceled:
			return fmt.Errorf(msgPrefix + " canceled.")
		default:
			return fmt.Errorf("Unknown or unexpected cost estimate state: %s", ce.Status)
		}
	}
	return nil
}

func (b *Remote) checkPolicy(stopCtx, cancelCtx context.Context, op *backend.Operation, r *tfe.Run) error {
	if b.CLI != nil {
		b.CLI.Output("\n------------------------------------------------------------------------\n")
	}
	for i, pc := range r.PolicyChecks {
		logs, err := b.client.PolicyChecks.Logs(stopCtx, pc.ID)
		if err != nil {
			return generalError("Failed to retrieve policy check logs", err)
		}
		reader := bufio.NewReaderSize(logs, 64*1024)

		// Retrieve the policy check to get its current status.
		pc, err := b.client.PolicyChecks.Read(stopCtx, pc.ID)
		if err != nil {
			return generalError("Failed to retrieve policy check", err)
		}

		var msgPrefix string
		switch pc.Scope {
		case tfe.PolicyScopeOrganization:
			msgPrefix = "Organization policy check"
		case tfe.PolicyScopeWorkspace:
			msgPrefix = "Workspace policy check"
		default:
			msgPrefix = fmt.Sprintf("Unknown policy check (%s)", pc.Scope)
		}

		if b.CLI != nil {
			b.CLI.Output(b.Colorize().Color(msgPrefix + ":\n"))
		}

		if b.CLI != nil {
			for next := true; next; {
				var l, line []byte

				for isPrefix := true; isPrefix; {
					l, isPrefix, err = reader.ReadLine()
					if err != nil {
						if err != io.EOF {
							return generalError("Failed to read logs", err)
						}
						next = false
					}
					line = append(line, l...)
				}

				if next || len(line) > 0 {
					b.CLI.Output(b.Colorize().Color(string(line)))
				}
			}
		}

		switch pc.Status {
		case tfe.PolicyPasses:
			if (r.HasChanges && op.Type == backend.OperationTypeApply || i < len(r.PolicyChecks)-1) && b.CLI != nil {
				b.CLI.Output("\n------------------------------------------------------------------------")
			}
			continue
		case tfe.PolicyErrored:
			return fmt.Errorf(msgPrefix + " errored.")
		case tfe.PolicyHardFailed:
			return fmt.Errorf(msgPrefix + " hard failed.")
		case tfe.PolicySoftFailed:
			if op.Type == backend.OperationTypePlan || op.UIOut == nil || op.UIIn == nil ||
				op.AutoApprove || !pc.Actions.IsOverridable || !pc.Permissions.CanOverride {
				return fmt.Errorf(msgPrefix + " soft failed.")
			}
		default:
			return fmt.Errorf("Unknown or unexpected policy state: %s", pc.Status)
		}

		opts := &terraform.InputOpts{
			Id:          "override",
			Query:       "\nDo you want to override the soft failed policy check?",
			Description: "Only 'override' will be accepted to override.",
		}

		err = b.confirm(stopCtx, op, opts, r, "override")
		if err != nil && err != errRunOverridden {
			return err
		}

		if err != errRunOverridden {
			if _, err = b.client.PolicyChecks.Override(stopCtx, pc.ID); err != nil {
				return generalError("Failed to override policy check", err)
			}
		}

		if b.CLI != nil {
			b.CLI.Output("------------------------------------------------------------------------")
		}
	}

	return nil
}

func (b *Remote) confirm(stopCtx context.Context, op *backend.Operation, opts *terraform.InputOpts, r *tfe.Run, keyword string) error {
	doneCtx, cancel := context.WithCancel(stopCtx)
	result := make(chan error, 2)

	go func() {
		// Make sure we cancel doneCtx before we return
		// so the input command is also canceled.
		defer cancel()

		for {
			select {
			case <-doneCtx.Done():
				return
			case <-stopCtx.Done():
				return
			case <-time.After(3 * time.Second):
				// Retrieve the run again to get its current status.
				r, err := b.client.Runs.Read(stopCtx, r.ID)
				if err != nil {
					result <- generalError("Failed to retrieve run", err)
					return
				}

				switch keyword {
				case "override":
					if r.Status != tfe.RunPolicyOverride {
						if r.Status == tfe.RunDiscarded {
							err = errRunDiscarded
						} else {
							err = errRunOverridden
						}
					}
				case "yes":
					if !r.Actions.IsConfirmable {
						if r.Status == tfe.RunDiscarded {
							err = errRunDiscarded
						} else {
							err = errRunApproved
						}
					}
				}

				if err != nil {
					if b.CLI != nil {
						b.CLI.Output(b.Colorize().Color(
							fmt.Sprintf("[reset][yellow]%s[reset]", err.Error())))
					}

					if err == errRunDiscarded {
						if op.Destroy {
							err = errDestroyDiscarded
						}
						err = errApplyDiscarded
					}

					result <- err
					return
				}
			}
		}
	}()

	result <- func() error {
		v, err := op.UIIn.Input(doneCtx, opts)
		if err != nil && err != context.Canceled && stopCtx.Err() != context.Canceled {
			return fmt.Errorf("Error asking %s: %v", opts.Id, err)
		}

		// We return the error of our parent channel as we don't
		// care about the error of the doneCtx which is only used
		// within this function. So if the doneCtx was canceled
		// because stopCtx was canceled, this will properly return
		// a context.Canceled error and otherwise it returns nil.
		if doneCtx.Err() == context.Canceled || stopCtx.Err() == context.Canceled {
			return stopCtx.Err()
		}

		// Make sure we cancel the context here so the loop that
		// checks for external changes to the run is ended before
		// we start to make changes ourselves.
		cancel()

		if v != keyword {
			// Retrieve the run again to get its current status.
			r, err = b.client.Runs.Read(stopCtx, r.ID)
			if err != nil {
				return generalError("Failed to retrieve run", err)
			}

			// Make sure we discard the run if possible.
			if r.Actions.IsDiscardable {
				err = b.client.Runs.Discard(stopCtx, r.ID, tfe.RunDiscardOptions{})
				if err != nil {
					if op.Destroy {
						return generalError("Failed to discard destroy", err)
					}
					return generalError("Failed to discard apply", err)
				}
			}

			// Even if the run was discarded successfully, we still
			// return an error as the apply command was canceled.
			if op.Destroy {
				return errDestroyDiscarded
			}
			return errApplyDiscarded
		}

		return nil
	}()

	return <-result
}
