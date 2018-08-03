package remote

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	tfe "github.com/hashicorp/go-tfe"
	"github.com/hashicorp/terraform/backend"
)

func (b *Remote) opPlan(stopCtx, cancelCtx context.Context, op *backend.Operation, runningOp *backend.RunningOperation) {
	log.Printf("[INFO] backend/remote: starting Plan operation")

	if op.Plan != nil {
		runningOp.Err = fmt.Errorf(strings.TrimSpace(planErrPlanNotSupported))
		return
	}

	if op.PlanOutPath != "" {
		runningOp.Err = fmt.Errorf(strings.TrimSpace(planErrOutPathNotSupported))
		return
	}

	if op.Targets != nil {
		runningOp.Err = fmt.Errorf(strings.TrimSpace(planErrTargetsNotSupported))
		return
	}

	if (op.Module == nil || op.Module.Config().Dir == "") && !op.Destroy {
		runningOp.Err = fmt.Errorf(strings.TrimSpace(planErrNoConfig))
		return
	}

	// Retrieve the workspace used to run this operation in.
	w, err := b.client.Workspaces.Read(stopCtx, b.organization, op.Workspace)
	if err != nil {
		if err != context.Canceled {
			runningOp.Err = fmt.Errorf(strings.TrimSpace(fmt.Sprintf(
				generalErr, "error retrieving workspace", err)))
		}
		return
	}

	configOptions := tfe.ConfigurationVersionCreateOptions{
		AutoQueueRuns: tfe.Bool(false),
		Speculative:   tfe.Bool(true),
	}

	cv, err := b.client.ConfigurationVersions.Create(stopCtx, w.ID, configOptions)
	if err != nil {
		if err != context.Canceled {
			runningOp.Err = fmt.Errorf(strings.TrimSpace(fmt.Sprintf(
				generalErr, "error creating configuration version", err)))
		}
		return
	}

	var configDir string
	if op.Module != nil && op.Module.Config().Dir != "" {
		configDir = op.Module.Config().Dir
	} else {
		configDir, err = ioutil.TempDir("", "tf")
		if err != nil {
			runningOp.Err = fmt.Errorf(strings.TrimSpace(fmt.Sprintf(
				generalErr, "error creating temp directory", err)))
			return
		}
		defer os.RemoveAll(configDir)
	}

	err = b.client.ConfigurationVersions.Upload(stopCtx, cv.UploadURL, configDir)
	if err != nil {
		if err != context.Canceled {
			runningOp.Err = fmt.Errorf(strings.TrimSpace(fmt.Sprintf(
				generalErr, "error uploading configuration files", err)))
		}
		return
	}

	uploaded := false
	for i := 0; i < 60 && !uploaded; i++ {
		select {
		case <-stopCtx.Done():
			return
		case <-cancelCtx.Done():
			return
		case <-time.After(500 * time.Millisecond):
			cv, err = b.client.ConfigurationVersions.Read(stopCtx, cv.ID)
			if err != nil {
				if err != context.Canceled {
					runningOp.Err = fmt.Errorf(strings.TrimSpace(fmt.Sprintf(
						generalErr, "error retrieving configuration version", err)))
				}
				return
			}

			if cv.Status == tfe.ConfigurationUploaded {
				uploaded = true
			}
		}
	}

	if !uploaded {
		runningOp.Err = fmt.Errorf(strings.TrimSpace(fmt.Sprintf(
			generalErr, "error uploading configuration files", "operation timed out")))
		return
	}

	runOptions := tfe.RunCreateOptions{
		IsDestroy:            tfe.Bool(op.Destroy),
		Message:              tfe.String("Queued manually using Terraform"),
		ConfigurationVersion: cv,
		Workspace:            w,
	}

	r, err := b.client.Runs.Create(stopCtx, runOptions)
	if err != nil {
		if err != context.Canceled {
			runningOp.Err = fmt.Errorf(strings.TrimSpace(fmt.Sprintf(
				generalErr, "error creating run", err)))
		}
		return
	}

	r, err = b.client.Runs.Read(stopCtx, r.ID)
	if err != nil {
		if err != context.Canceled {
			runningOp.Err = fmt.Errorf(strings.TrimSpace(fmt.Sprintf(
				generalErr, "error retrieving run", err)))
		}
		return
	}

	if b.CLI != nil {
		b.CLI.Output(b.Colorize().Color(strings.TrimSpace(fmt.Sprintf(
			planDefaultHeader, b.hostname, b.organization, op.Workspace, r.ID)) + "\n"))
	}

	logs, err := b.client.Plans.Logs(stopCtx, r.Plan.ID)
	if err != nil {
		if err != context.Canceled {
			runningOp.Err = fmt.Errorf(strings.TrimSpace(fmt.Sprintf(
				generalErr, "error retrieving logs", err)))
		}
		return
	}
	scanner := bufio.NewScanner(logs)

	for scanner.Scan() {
		if b.CLI != nil {
			b.CLI.Output(b.Colorize().Color(scanner.Text()))
		}
	}
	if err := scanner.Err(); err != nil {
		if err != context.Canceled && err != io.EOF {
			runningOp.Err = fmt.Errorf("Error reading logs: %v", err)
		}
		return
	}
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
To view this plan in a browser, visit:
https://%s/app/%s/%s/runs/%s[reset]

Waiting for the plan to start...
`
