package remote

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	tfe "github.com/hashicorp/go-tfe"
	"github.com/hashicorp/terraform/backend"
)

func (b *Remote) opPlan(stopCtx, cancelCtx context.Context, op *backend.Operation) (*tfe.Run, error) {
	log.Printf("[INFO] backend/remote: starting Plan operation")

	// Retrieve the workspace used to run this operation in.
	w, err := b.client.Workspaces.Read(stopCtx, b.organization, op.Workspace)
	if err != nil {
		return nil, generalError("error retrieving workspace", err)
	}

	if !w.Permissions.CanQueueRun {
		return nil, fmt.Errorf(strings.TrimSpace(fmt.Sprintf(planErrNoQueueRunRights)))
	}

	if op.ModuleDepth != defaultModuleDepth {
		return nil, fmt.Errorf(strings.TrimSpace(planErrModuleDepthNotSupported))
	}

	if op.Parallelism != defaultParallelism {
		return nil, fmt.Errorf(strings.TrimSpace(planErrParallelismNotSupported))
	}

	if op.Plan != nil {
		return nil, fmt.Errorf(strings.TrimSpace(planErrPlanNotSupported))
	}

	if op.PlanOutPath != "" {
		return nil, fmt.Errorf(strings.TrimSpace(planErrOutPathNotSupported))
	}

	if !op.PlanRefresh {
		return nil, fmt.Errorf(strings.TrimSpace(planErrNoRefreshNotSupported))
	}

	if op.Targets != nil {
		return nil, fmt.Errorf(strings.TrimSpace(planErrTargetsNotSupported))
	}

	if op.Variables != nil {
		return nil, fmt.Errorf(strings.TrimSpace(
			fmt.Sprintf(planErrVariablesNotSupported, b.hostname, b.organization, op.Workspace)))
	}

	if (op.Module == nil || op.Module.Config().Dir == "") && !op.Destroy {
		return nil, fmt.Errorf(strings.TrimSpace(planErrNoConfig))
	}

	return b.plan(stopCtx, cancelCtx, op, w)
}

func (b *Remote) plan(stopCtx, cancelCtx context.Context, op *backend.Operation, w *tfe.Workspace) (*tfe.Run, error) {
	configOptions := tfe.ConfigurationVersionCreateOptions{
		AutoQueueRuns: tfe.Bool(false),
		Speculative:   tfe.Bool(op.Type == backend.OperationTypePlan),
	}

	cv, err := b.client.ConfigurationVersions.Create(stopCtx, w.ID, configOptions)
	if err != nil {
		return nil, generalError("error creating configuration version", err)
	}

	var configDir string
	if op.Module != nil && op.Module.Config().Dir != "" {
		// Make sure to take the working directory into account by removing
		// the working directory from the current path. This will result in
		// a path that points to the expected root of the workspace.
		configDir = filepath.Clean(strings.TrimSuffix(
			filepath.Clean(op.Module.Config().Dir),
			filepath.Clean(w.WorkingDirectory),
		))
	} else {
		// We did a check earlier to make sure we either have a config dir,
		// or the plan is run with -destroy. So this else clause will only
		// be executed when we are destroying and doesn't need the config.
		configDir, err = ioutil.TempDir("", "tf")
		if err != nil {
			return nil, generalError("error creating temporary directory", err)
		}
		defer os.RemoveAll(configDir)

		// Make sure the configured working directory exists.
		err = os.MkdirAll(filepath.Join(configDir, w.WorkingDirectory), 0700)
		if err != nil {
			return nil, generalError(
				"error creating temporary working directory", err)
		}
	}

	err = b.client.ConfigurationVersions.Upload(stopCtx, cv.UploadURL, configDir)
	if err != nil {
		return nil, generalError("error uploading configuration files", err)
	}

	uploaded := false
	for i := 0; i < 60 && !uploaded; i++ {
		select {
		case <-stopCtx.Done():
			return nil, context.Canceled
		case <-cancelCtx.Done():
			return nil, context.Canceled
		case <-time.After(500 * time.Millisecond):
			cv, err = b.client.ConfigurationVersions.Read(stopCtx, cv.ID)
			if err != nil {
				return nil, generalError("error retrieving configuration version", err)
			}

			if cv.Status == tfe.ConfigurationUploaded {
				uploaded = true
			}
		}
	}

	if !uploaded {
		return nil, generalError(
			"error uploading configuration files", errors.New("operation timed out"))
	}

	runOptions := tfe.RunCreateOptions{
		IsDestroy:            tfe.Bool(op.Destroy),
		Message:              tfe.String("Queued manually using Terraform"),
		ConfigurationVersion: cv,
		Workspace:            w,
	}

	r, err := b.client.Runs.Create(stopCtx, runOptions)
	if err != nil {
		return r, generalError("error creating run", err)
	}

	// When the lock timeout is set,
	if op.StateLockTimeout > 0 {
		go func() {
			select {
			case <-stopCtx.Done():
				return
			case <-cancelCtx.Done():
				return
			case <-time.After(op.StateLockTimeout):
				// Retrieve the run to get its current status.
				r, err := b.client.Runs.Read(cancelCtx, r.ID)
				if err != nil {
					log.Printf("[ERROR] error reading run: %v", err)
					return
				}

				if r.Status == tfe.RunPending && r.Actions.IsCancelable {
					if b.CLI != nil {
						b.CLI.Output(b.Colorize().Color(strings.TrimSpace(lockTimeoutErr)))
					}

					// We abuse the auto aprove flag to indicate that we do not
					// want to ask if the remote operation should be canceled.
					op.AutoApprove = true

					p, err := os.FindProcess(os.Getpid())
					if err != nil {
						log.Printf("[ERROR] error searching process ID: %v", err)
						return
					}
					p.Signal(syscall.SIGINT)
				}
			}
		}()
	}

	if b.CLI != nil {
		header := planDefaultHeader
		if op.Type == backend.OperationTypeApply {
			header = applyDefaultHeader
		}
		b.CLI.Output(b.Colorize().Color(strings.TrimSpace(fmt.Sprintf(
			header, b.hostname, b.organization, op.Workspace, r.ID)) + "\n"))
	}

	r, err = b.waitForRun(stopCtx, cancelCtx, op, "plan", r, w)
	if err != nil {
		return r, err
	}

	if b.CLI != nil {
		// Insert a blank line to separate the ouputs.
		b.CLI.Output("")
	}

	logs, err := b.client.Plans.Logs(stopCtx, r.Plan.ID)
	if err != nil {
		return r, generalError("error retrieving logs", err)
	}
	scanner := bufio.NewScanner(logs)

	for scanner.Scan() {
		if b.CLI != nil {
			b.CLI.Output(b.Colorize().Color(scanner.Text()))
		}
	}
	if err := scanner.Err(); err != nil {
		return r, generalError("error reading logs", err)
	}

	return r, nil
}

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
			return r, generalError("error retrieving run", err)
		}

		// Return if the run is no longer pending.
		if r.Status != tfe.RunPending && r.Status != tfe.RunConfirmed {
			if i == 0 && b.CLI != nil {
				b.CLI.Output(b.Colorize().Color(fmt.Sprintf("Waiting for the %s to start...", opType)))
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
				return nil, generalError("error retrieving workspace", err)
			}

			// If the workspace is locked the run will not be queued and we can
			// update the status without making any expensive calls.
			if w.Locked && w.CurrentRun != nil {
				cr, err := b.client.Runs.Read(stopCtx, w.CurrentRun.ID)
				if err != nil {
					return r, generalError("error retrieving current run", err)
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
						return r, generalError("error retrieving run list", err)
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
					return r, generalError("error retrieving queue", err)
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
					return r, generalError("error retrieving capacity", err)
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

const planErrNoQueueRunRights = `
Insufficient rights to generate a plan!

[reset][yellow]The provided credentials have insufficient rights to generate a plan. In order
to generate plans, at least plan permissions on the workspace are required.[reset]
`

const planErrModuleDepthNotSupported = `
Custom module depths are currently not supported!

The "remote" backend does not support setting a custom module
depth at this time.
`

const planErrParallelismNotSupported = `
Custom parallelism values are currently not supported!

The "remote" backend does not support setting a custom parallelism
value at this time.
`

const planErrPlanNotSupported = `
Displaying a saved plan is currently not supported!

The "remote" backend currently requires configuration to be present and
does not accept an existing saved plan as an argument at this time.
`

const planErrOutPathNotSupported = `
Saving a generated plan is currently not supported!

The "remote" backend does not support saving the generated execution
plan locally at this time.
`

const planErrNoRefreshNotSupported = `
Planning without refresh is currently not supported!

Currently the "remote" backend will always do an in-memory refresh of
the Terraform state prior to generating the plan.
`

const planErrTargetsNotSupported = `
Resource targeting is currently not supported!

The "remote" backend does not support resource targeting at this time.
`

const planErrVariablesNotSupported = `
Run variables are currently not supported!

The "remote" backend does not support setting run variables at this time.
Currently the only to way to pass variables to the remote backend is by
creating a '*.auto.tfvars' variables file. This file will automatically
be loaded by the "remote" backend when the workspace is configured to use
Terraform v0.10.0 or later.

Additionally you can also set variables on the workspace in the web UI:
https://%s/app/%s/%s/variables
`

const planErrNoConfig = `
No configuration files found!

Plan requires configuration to be present. Planning without a configuration
would mark everything for destruction, which is normally not what is desired.
If you would like to destroy everything, please run plan with the "-destroy"
flag or create a single empty configuration file. Otherwise, please create
a Terraform configuration file in the path being executed and try again.
`

const planDefaultHeader = `
[reset][yellow]Running plan in the remote backend. Output will stream here. Pressing Ctrl-C
will stop streaming the logs, but will not stop the plan running remotely.
To view this run in a browser, visit:
https://%s/app/%s/%s/runs/%s[reset]
`

// The newline in this error is to make it look good in the CLI!
const lockTimeoutErr = `
[reset][red]Lock timeout exceeded, sending interrupt to cancel the remote operation.
[reset]
`
