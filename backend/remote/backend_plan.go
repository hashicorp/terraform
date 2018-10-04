package remote

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
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

	if op.Plan != nil {
		return nil, fmt.Errorf(strings.TrimSpace(planErrPlanNotSupported))
	}

	if op.PlanOutPath != "" {
		return nil, fmt.Errorf(strings.TrimSpace(planErrOutPathNotSupported))
	}

	if op.Targets != nil {
		return nil, fmt.Errorf(strings.TrimSpace(planErrTargetsNotSupported))
	}

	if (op.Module == nil || op.Module.Config().Dir == "") && !op.Destroy {
		return nil, fmt.Errorf(strings.TrimSpace(planErrNoConfig))
	}

	// Retrieve the workspace used to run this operation in.
	w, err := b.client.Workspaces.Read(stopCtx, b.organization, op.Workspace)
	if err != nil {
		return nil, generalError("error retrieving workspace", err)
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

	r, err = b.client.Runs.Read(stopCtx, r.ID)
	if err != nil {
		return r, generalError("error retrieving run", err)
	}

	if b.CLI != nil {
		header := planDefaultHeader
		if op.Type == backend.OperationTypeApply {
			header = applyDefaultHeader
		}
		b.CLI.Output(b.Colorize().Color(strings.TrimSpace(fmt.Sprintf(
			header, b.hostname, b.organization, op.Workspace, r.ID)) + "\n"))
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

const planErrPlanNotSupported = `
Displaying a saved plan is currently not supported!

The "remote" backend currently requires configuration to be present
and does not accept an existing saved plan as an argument at this time.
`

const planErrOutPathNotSupported = `
Saving a generated plan is currently not supported!

The "remote" backend does not support saving the generated execution
plan locally at this time.
`

const planErrTargetsNotSupported = `
Resource targeting is currently not supported!

The "remote" backend does not support resource targeting at this time.
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

Waiting for the plan to start...
`

// The newline in this error is to make it look good in the CLI!
const lockTimeoutErr = `
[reset][red]Lock timeout exceeded, sending interrupt to cancel the remote operation.
[reset]
`
