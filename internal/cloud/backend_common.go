// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cloud

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	tfe "github.com/hashicorp/go-tfe"
	"github.com/hashicorp/jsonapi"
	"github.com/hashicorp/terraform/internal/backend/backendrun"
	"github.com/hashicorp/terraform/internal/command/jsonformat"
	"github.com/hashicorp/terraform/internal/logging"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/terraform"
)

var (
	backoffMin = 1000.0
	backoffMax = 3000.0

	runPollInterval = 3 * time.Second
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

func (b *Cloud) waitForRun(stopCtx, cancelCtx context.Context, op *backendrun.Operation, opType string, r *tfe.Run, w *tfe.Workspace) (*tfe.Run, error) {
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
		r, err := b.client.Runs.Read(stopCtx, r.ID)
		if err != nil {
			return r, b.generalError("Failed to retrieve run", err)
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
			w, err = b.client.Workspaces.Read(stopCtx, b.Organization, w.Name)
			if err != nil {
				return nil, b.generalError("Failed to retrieve workspace", err)
			}

			// If the workspace is locked the run will not be queued and we can
			// update the status without making any expensive calls.
			if w.Locked && w.CurrentRun != nil {
				cr, err := b.client.Runs.Read(stopCtx, w.CurrentRun.ID)
				if err != nil {
					return r, b.generalError("Failed to retrieve current run", err)
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
				options := &tfe.RunListOptions{}
			runlist:
				for {
					rl, err := b.client.Runs.List(stopCtx, w.ID, options)
					if err != nil {
						return r, b.generalError("Failed to retrieve run list", err)
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
							if op.Type == backendrun.OperationTypePlan {
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

			options := tfe.ReadRunQueueOptions{}
		search:
			for {
				rq, err := b.client.Organizations.ReadRunQueue(stopCtx, b.Organization, options)
				if err != nil {
					return r, b.generalError("Failed to retrieve queue", err)
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
				c, err := b.client.Organizations.ReadCapacity(stopCtx, b.Organization)
				if err != nil {
					return r, b.generalError("Failed to retrieve capacity", err)
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

func (b *Cloud) waitTaskStage(stopCtx, cancelCtx context.Context, op *backendrun.Operation, r *tfe.Run, stageID string, outputTitle string) error {
	integration := &IntegrationContext{
		B:             b,
		StopContext:   stopCtx,
		CancelContext: cancelCtx,
		Op:            op,
		Run:           r,
	}
	return b.runTaskStage(integration, integration.BeginOutput(outputTitle), stageID)
}

func (b *Cloud) costEstimate(stopCtx, cancelCtx context.Context, op *backendrun.Operation, r *tfe.Run) error {
	if r.CostEstimate == nil {
		return nil
	}

	msgPrefix := "Cost Estimation"
	started := time.Now()
	updated := started
	for i := 0; ; i++ {
		select {
		case <-stopCtx.Done():
			return stopCtx.Err()
		case <-cancelCtx.Done():
			return cancelCtx.Err()
		case <-time.After(backoff(backoffMin, backoffMax, i)):
		}

		// Retrieve the cost estimate to get its current status.
		ce, err := b.client.CostEstimates.Read(stopCtx, r.CostEstimate.ID)
		if err != nil {
			return b.generalError("Failed to retrieve cost estimate", err)
		}

		// If the run is canceled or errored, but the cost-estimate still has
		// no result, there is nothing further to render.
		if ce.Status != tfe.CostEstimateFinished {
			if r.Status == tfe.RunCanceled || r.Status == tfe.RunErrored {
				return nil
			}
		}

		// checking if i == 0 so as to avoid printing this starting horizontal-rule
		// every retry, and that it only prints it on the first (i=0) attempt.
		if b.CLI != nil && i == 0 {
			b.CLI.Output("\n------------------------------------------------------------------------\n")
		}

		switch ce.Status {
		case tfe.CostEstimateFinished:
			delta, err := strconv.ParseFloat(ce.DeltaMonthlyCost, 64)
			if err != nil {
				return b.generalError("Unexpected error", err)
			}

			sign := "+"
			if delta < 0 {
				sign = "-"
			}

			deltaRepr := strings.Replace(ce.DeltaMonthlyCost, "-", "", 1)

			if b.CLI != nil {
				b.CLI.Output(b.Colorize().Color("[bold]" + msgPrefix + ":\n"))
				b.CLI.Output(b.Colorize().Color(fmt.Sprintf("Resources: %d of %d estimated", ce.MatchedResourcesCount, ce.ResourcesCount)))
				b.CLI.Output(b.Colorize().Color(fmt.Sprintf("           $%s/mo %s$%s", ce.ProposedMonthlyCost, sign, deltaRepr)))

				if len(r.PolicyChecks) == 0 && r.HasChanges && op.Type == backendrun.OperationTypeApply {
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
				b.CLI.Output(b.Colorize().Color("[bold]" + msgPrefix + ":\n"))
				b.CLI.Output(b.Colorize().Color("Waiting for cost estimate to complete..." + elapsed + "\n"))
			}
			continue
		case tfe.CostEstimateSkippedDueToTargeting:
			b.CLI.Output(b.Colorize().Color("[bold]" + msgPrefix + ":\n"))
			b.CLI.Output("Not available for this plan, because it was created with the -target option.")
			b.CLI.Output("\n------------------------------------------------------------------------")
			return nil
		case tfe.CostEstimateErrored:
			b.CLI.Output(msgPrefix + " errored.\n")
			b.CLI.Output("\n------------------------------------------------------------------------")
			return nil
		case tfe.CostEstimateCanceled:
			return fmt.Errorf("%s canceled.", msgPrefix)
		default:
			return fmt.Errorf("Unknown or unexpected cost estimate state: %s", ce.Status)
		}
	}
}

func (b *Cloud) checkPolicy(stopCtx, cancelCtx context.Context, op *backendrun.Operation, r *tfe.Run) error {
	if b.CLI != nil {
		b.CLI.Output("\n------------------------------------------------------------------------\n")
	}
	for i, pc := range r.PolicyChecks {
		// Read the policy check logs. This is a blocking call that will only
		// return once the policy check is complete.
		logs, err := b.client.PolicyChecks.Logs(stopCtx, pc.ID)
		if err != nil {
			return b.generalError("Failed to retrieve policy check logs", err)
		}
		reader := bufio.NewReaderSize(logs, 64*1024)

		// Retrieve the policy check to get its current status.
		pc, err := b.client.PolicyChecks.Read(stopCtx, pc.ID)
		if err != nil {
			return b.generalError("Failed to retrieve policy check", err)
		}

		// If the run is canceled or errored, but the policy check still has
		// no result, there is nothing further to render.
		if r.Status == tfe.RunCanceled || r.Status == tfe.RunErrored {
			switch pc.Status {
			case tfe.PolicyPending, tfe.PolicyQueued, tfe.PolicyUnreachable:
				continue
			}
		}

		var msgPrefix string
		switch pc.Scope {
		case tfe.PolicyScopeOrganization:
			msgPrefix = "Organization Policy Check"
		case tfe.PolicyScopeWorkspace:
			msgPrefix = "Workspace Policy Check"
		default:
			msgPrefix = fmt.Sprintf("Unknown policy check (%s)", pc.Scope)
		}

		if b.CLI != nil {
			b.CLI.Output(b.Colorize().Color("[bold]" + msgPrefix + ":\n"))
		}

		if b.CLI != nil {
			for next := true; next; {
				var l, line []byte

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
					b.CLI.Output(b.Colorize().Color(string(line)))
				}
			}
		}

		switch pc.Status {
		case tfe.PolicyPasses:
			if (r.HasChanges && op.Type == backendrun.OperationTypeApply || i < len(r.PolicyChecks)-1) && b.CLI != nil {
				b.CLI.Output("\n------------------------------------------------------------------------")
			}
			continue
		case tfe.PolicyErrored:
			return fmt.Errorf("%s errored.", msgPrefix)
		case tfe.PolicyHardFailed:
			return fmt.Errorf("%s hard failed.", msgPrefix)
		case tfe.PolicySoftFailed:
			runURL := fmt.Sprintf(runHeaderErr, b.Hostname, b.Organization, op.Workspace, r.ID)

			if op.Type == backendrun.OperationTypePlan || op.UIOut == nil || op.UIIn == nil ||
				!pc.Actions.IsOverridable || !pc.Permissions.CanOverride {
				return fmt.Errorf("%s soft failed.\n%s", msgPrefix, runURL)
			}

			if op.AutoApprove {
				if _, err = b.client.PolicyChecks.Override(stopCtx, pc.ID); err != nil {
					return b.generalError(fmt.Sprintf("Failed to override policy check.\n%s", runURL), err)
				}
			} else if !b.input {
				return errPolicyOverrideNeedsUIConfirmation
			} else {
				opts := &terraform.InputOpts{
					Id:          "override",
					Query:       "\nDo you want to override the soft failed policy check?",
					Description: "Only 'override' will be accepted to override.",
				}
				err = b.confirm(stopCtx, op, opts, r, "override")
				if err != nil && err != errRunOverridden {
					return fmt.Errorf("Failed to override: %w\n%s\n", err, runURL)
				}

				if err != errRunOverridden {
					if _, err = b.client.PolicyChecks.Override(stopCtx, pc.ID); err != nil {
						return b.generalError(fmt.Sprintf("Failed to override policy check.\n%s", runURL), err)
					}
				} else {
					runURL := fmt.Sprintf(runHeader, b.Hostname, b.Organization, op.Workspace, r.ID)
					b.CLI.Output(fmt.Sprintf("The run needs to be manually overridden or discarded.\n%s\n", runURL))
				}
			}

			if b.CLI != nil {
				b.CLI.Output("------------------------------------------------------------------------")
			}
		default:
			return fmt.Errorf("Unknown or unexpected policy state: %s", pc.Status)
		}
	}

	return nil
}

func (b *Cloud) confirm(stopCtx context.Context, op *backendrun.Operation, opts *terraform.InputOpts, r *tfe.Run, keyword string) error {
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
			case <-time.After(runPollInterval):
				// Retrieve the run again to get its current status.
				r, err := b.client.Runs.Read(stopCtx, r.ID)
				if err != nil {
					result <- b.generalError("Failed to retrieve run", err)
					return
				}

				switch keyword {
				case "override":
					if r.Status != tfe.RunPolicyOverride && r.Status != tfe.RunPostPlanAwaitingDecision {
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
						err = errApplyDiscarded
						if op.PlanMode == plans.DestroyMode {
							err = errDestroyDiscarded
						}
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
				return b.generalError("Failed to retrieve run", err)
			}

			// Make sure we discard the run if possible.
			if r.Actions.IsDiscardable {
				err = b.client.Runs.Discard(stopCtx, r.ID, tfe.RunDiscardOptions{})
				if err != nil {
					if op.PlanMode == plans.DestroyMode {
						return b.generalError("Failed to discard destroy", err)
					}
					return b.generalError("Failed to discard apply", err)
				}
			}

			// Even if the run was discarded successfully, we still
			// return an error as the apply command was canceled.
			if op.PlanMode == plans.DestroyMode {
				return errDestroyDiscarded
			}
			return errApplyDiscarded
		}

		return nil
	}()

	return <-result
}

// This method will fetch the redacted plan output as a byte slice, mirroring
// the behavior of the similar client.Plans.ReadJSONOutput method.
//
// Note: Apologies for the lengthy definition, this is a result of not being
// able to mock receiver methods
var readRedactedPlan func(context.Context, url.URL, string, string) ([]byte, error) = func(ctx context.Context, baseURL url.URL, token string, planID string) ([]byte, error) {
	client := retryablehttp.NewClient()
	client.RetryMax = 10
	client.RetryWaitMin = 100 * time.Millisecond
	client.RetryWaitMax = 400 * time.Millisecond
	client.Logger = logging.HCLogger()

	u, err := baseURL.Parse(fmt.Sprintf(
		"plans/%s/json-output-redacted", url.QueryEscape(planID)))
	if err != nil {
		return nil, err
	}

	req, err := retryablehttp.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err = checkResponseCode(resp); err != nil {
		return nil, err
	}

	return io.ReadAll(resp.Body)
}

// decodeRedactedPlan unmarshals a downloaded redacted plan into a struct the
// jsonformat.Renderer expects.
func decodeRedactedPlan(jsonBytes []byte) (*jsonformat.Plan, error) {
	r := bytes.NewReader(jsonBytes)
	p := &jsonformat.Plan{}
	if err := json.NewDecoder(r).Decode(p); err != nil {
		return nil, err
	}
	return p, nil
}

func checkResponseCode(r *http.Response) error {
	if r.StatusCode >= 200 && r.StatusCode <= 299 {
		return nil
	}

	var errs []string
	var err error

	switch r.StatusCode {
	case 401:
		return tfe.ErrUnauthorized
	case 404:
		return tfe.ErrResourceNotFound
	}

	errs, err = decodeErrorPayload(r)
	if err != nil {
		return err
	}

	return errors.New(strings.Join(errs, "\n"))
}

func decodeErrorPayload(r *http.Response) ([]string, error) {
	// Decode the error payload.
	var errs []string
	errPayload := &jsonapi.ErrorsPayload{}
	err := json.NewDecoder(r.Body).Decode(errPayload)
	if err != nil || len(errPayload.Errors) == 0 {
		return errs, errors.New(r.Status)
	}

	// Parse and format the errors.
	for _, e := range errPayload.Errors {
		if e.Detail == "" {
			errs = append(errs, e.Title)
		} else {
			errs = append(errs, fmt.Sprintf("%s\n\n%s", e.Title, e.Detail))
		}
	}

	return errs, nil
}

func isValidAppName(name string) bool {
	return name == "HCP Terraform" || name == "Terraform Enterprise"
}
