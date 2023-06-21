// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package cloud

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	tfe "github.com/hashicorp/go-tfe"
	version "github.com/hashicorp/go-version"

	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/cloud/cloudplan"
	"github.com/hashicorp/terraform/internal/command/jsonformat"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/genconfig"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

var planConfigurationVersionsPollInterval = 500 * time.Millisecond

func (b *Cloud) opPlan(stopCtx, cancelCtx context.Context, op *backend.Operation, w *tfe.Workspace) (*tfe.Run, error) {
	log.Printf("[INFO] cloud: starting Plan operation")

	var diags tfdiags.Diagnostics

	if !w.Permissions.CanQueueRun {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Insufficient rights to generate a plan",
			"The provided credentials have insufficient rights to generate a plan. In order "+
				"to generate plans, at least plan permissions on the workspace are required.",
		))
		return nil, diags.Err()
	}

	if b.ContextOpts != nil && b.ContextOpts.Parallelism != defaultParallelism {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Custom parallelism values are currently not supported",
			`Terraform Cloud does not support setting a custom parallelism `+
				`value at this time.`,
		))
	}

	if op.PlanFile != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Displaying a saved plan is currently not supported",
			`Terraform Cloud currently requires configuration to be present and `+
				`does not accept an existing saved plan as an argument at this time.`,
		))
	}

	if !op.HasConfig() && op.PlanMode != plans.DestroyMode {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"No configuration files found",
			`Plan requires configuration to be present. Planning without a configuration `+
				`would mark everything for destruction, which is normally not what is desired. `+
				`If you would like to destroy everything, please run plan with the "-destroy" `+
				`flag or create a single empty configuration file. Otherwise, please create `+
				`a Terraform configuration file in the path being executed and try again.`,
		))
	}

	if len(op.GenerateConfigOut) > 0 {
		diags = diags.Append(genconfig.ValidateTargetFile(op.GenerateConfigOut))
	}

	// Return if there are any errors.
	if diags.HasErrors() {
		return nil, diags.Err()
	}

	run, err := b.plan(stopCtx, cancelCtx, op, w)

	// Save plan even if the run errored, as long as it actually exists
	if run != nil && run.ID != "" && op.PlanOutPath != "" {
		bookmark := cloudplan.NewSavedPlanBookmark(run.ID, b.hostname)
		saveErr := bookmark.Save(op.PlanOutPath)
		if err == nil {
			err = saveErr
		} else {
			err = fmt.Errorf("%w\nAdditionally, an error occurred when saving the plan: %s", err, saveErr.Error())
		}
	}

	// Cloud integration currently doesn't support genconfig, but we might have
	// saved a cloud plan file.
	op.View.PlanNextStep(op.PlanOutPath, "")

	return run, err
}

func (b *Cloud) plan(stopCtx, cancelCtx context.Context, op *backend.Operation, w *tfe.Workspace) (*tfe.Run, error) {
	if b.CLI != nil {
		header := planDefaultHeader
		if op.Type == backend.OperationTypeApply || op.Type == backend.OperationTypeRefresh {
			header = applyDefaultHeader
		}
		b.CLI.Output(b.Colorize().Color(strings.TrimSpace(header) + "\n"))
	}

	// Plan-only means they ran terraform plan without -out.
	planOnly := op.Type == backend.OperationTypePlan && op.PlanOutPath == ""

	configOptions := tfe.ConfigurationVersionCreateOptions{
		AutoQueueRuns: tfe.Bool(false),
		Speculative:   tfe.Bool(planOnly),
	}

	cv, err := b.client.ConfigurationVersions.Create(stopCtx, w.ID, configOptions)
	if err != nil {
		return nil, generalError("Failed to create configuration version", err)
	}

	var configDir string
	if op.ConfigDir != "" {
		// De-normalize the configuration directory path.
		configDir, err = filepath.Abs(op.ConfigDir)
		if err != nil {
			return nil, generalError(
				"Failed to get absolute path of the configuration directory: %v", err)
		}

		// Make sure to take the working directory into account by removing
		// the working directory from the current path. This will result in
		// a path that points to the expected root of the workspace.
		configDir = filepath.Clean(strings.TrimSuffix(
			filepath.Clean(configDir),
			filepath.Clean(w.WorkingDirectory),
		))

		// If the workspace has a subdirectory as its working directory then
		// our configDir will be some parent directory of the current working
		// directory. Users are likely to find that surprising, so we'll
		// produce an explicit message about it to be transparent about what
		// we are doing and why.
		if w.WorkingDirectory != "" && filepath.Base(configDir) != w.WorkingDirectory {
			if b.CLI != nil {
				b.CLI.Output(fmt.Sprintf(strings.TrimSpace(`
The remote workspace is configured to work with configuration at
%s relative to the target repository.

Terraform will upload the contents of the following directory,
excluding files or directories as defined by a .terraformignore file
at %s/.terraformignore (if it is present),
in order to capture the filesystem context the remote workspace expects:
    %s
`), w.WorkingDirectory, configDir, configDir) + "\n")
			}
		}

	} else {
		// We did a check earlier to make sure we either have a config dir,
		// or the plan is run with -destroy. So this else clause will only
		// be executed when we are destroying and doesn't need the config.
		configDir, err = ioutil.TempDir("", "tf")
		if err != nil {
			return nil, generalError("Failed to create temporary directory", err)
		}
		defer os.RemoveAll(configDir)

		// Make sure the configured working directory exists.
		err = os.MkdirAll(filepath.Join(configDir, w.WorkingDirectory), 0700)
		if err != nil {
			return nil, generalError(
				"Failed to create temporary working directory", err)
		}
	}

	err = b.client.ConfigurationVersions.Upload(stopCtx, cv.UploadURL, configDir)
	if err != nil {
		return nil, generalError("Failed to upload configuration files", err)
	}

	uploaded := false
	for i := 0; i < 60 && !uploaded; i++ {
		select {
		case <-stopCtx.Done():
			return nil, context.Canceled
		case <-cancelCtx.Done():
			return nil, context.Canceled
		case <-time.After(planConfigurationVersionsPollInterval):
			cv, err = b.client.ConfigurationVersions.Read(stopCtx, cv.ID)
			if err != nil {
				return nil, generalError("Failed to retrieve configuration version", err)
			}

			if cv.Status == tfe.ConfigurationUploaded {
				uploaded = true
			}
		}
	}

	if !uploaded {
		return nil, generalError(
			"Failed to upload configuration files", errors.New("operation timed out"))
	}

	runOptions := tfe.RunCreateOptions{
		ConfigurationVersion: cv,
		Refresh:              tfe.Bool(op.PlanRefresh),
		Workspace:            w,
		AutoApply:            tfe.Bool(op.AutoApprove),
		SavePlan:             tfe.Bool(op.PlanOutPath != ""),
	}

	switch op.PlanMode {
	case plans.NormalMode:
		// okay, but we don't need to do anything special for this
	case plans.RefreshOnlyMode:
		runOptions.RefreshOnly = tfe.Bool(true)
	case plans.DestroyMode:
		runOptions.IsDestroy = tfe.Bool(true)
	default:
		// Shouldn't get here because we should update this for each new
		// plan mode we add, mapping it to the corresponding RunCreateOptions
		// field.
		return nil, generalError(
			"Invalid plan mode",
			fmt.Errorf("Terraform Cloud doesn't support %s", op.PlanMode),
		)
	}

	if len(op.Targets) != 0 {
		runOptions.TargetAddrs = make([]string, 0, len(op.Targets))
		for _, addr := range op.Targets {
			runOptions.TargetAddrs = append(runOptions.TargetAddrs, addr.String())
		}
	}

	if len(op.ForceReplace) != 0 {
		runOptions.ReplaceAddrs = make([]string, 0, len(op.ForceReplace))
		for _, addr := range op.ForceReplace {
			runOptions.ReplaceAddrs = append(runOptions.ReplaceAddrs, addr.String())
		}
	}

	config, _, configDiags := op.ConfigLoader.LoadConfigWithSnapshot(op.ConfigDir)
	if configDiags.HasErrors() {
		return nil, fmt.Errorf("error loading config with snapshot: %w", configDiags.Errs()[0])
	}

	variables, varDiags := ParseCloudRunVariables(op.Variables, config.Module.Variables)

	if varDiags.HasErrors() {
		return nil, varDiags.Err()
	}

	runVariables := make([]*tfe.RunVariable, 0, len(variables))
	for name, value := range variables {
		runVariables = append(runVariables, &tfe.RunVariable{
			Key:   name,
			Value: value,
		})
	}
	runOptions.Variables = runVariables

	if len(op.GenerateConfigOut) > 0 {
		runOptions.AllowConfigGeneration = tfe.Bool(true)
	}

	r, err := b.client.Runs.Create(stopCtx, runOptions)
	if err != nil {
		return r, generalError("Failed to create run", err)
	}

	// When the lock timeout is set, if the run is still pending and
	// cancellable after that period, we attempt to cancel it.
	if lockTimeout := op.StateLocker.Timeout(); lockTimeout > 0 {
		go func() {
			select {
			case <-stopCtx.Done():
				return
			case <-cancelCtx.Done():
				return
			case <-time.After(lockTimeout):
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
		b.CLI.Output(b.Colorize().Color(strings.TrimSpace(fmt.Sprintf(
			runHeader, b.hostname, b.organization, op.Workspace, r.ID)) + "\n"))
	}

	// Render any warnings that were raised during run creation
	if err := b.renderRunWarnings(stopCtx, b.client, r.ID); err != nil {
		return r, err
	}

	// Retrieve the run to get task stages.
	// Task Stages are calculated upfront so we only need to call this once for the run.
	taskStages, err := b.runTaskStages(stopCtx, b.client, r.ID)
	if err != nil {
		return r, err
	}

	if stage, ok := taskStages[tfe.PrePlan]; ok {
		if err := b.waitTaskStage(stopCtx, cancelCtx, op, r, stage.ID, "Pre-plan Tasks"); err != nil {
			return r, err
		}
	}

	r, err = b.waitForRun(stopCtx, cancelCtx, op, "plan", r, w)
	if err != nil {
		return r, err
	}

	err = b.renderPlanLogs(stopCtx, op, r)
	if err != nil {
		return r, err
	}

	// Retrieve the run to get its current status.
	r, err = b.client.Runs.Read(stopCtx, r.ID)
	if err != nil {
		return r, generalError("Failed to retrieve run", err)
	}

	// If the run is canceled or errored, we still continue to the
	// cost-estimation and policy check phases to ensure we render any
	// results available. In the case of a hard-failed policy check, the
	// status of the run will be "errored", but there is still policy
	// information which should be shown.

	if stage, ok := taskStages[tfe.PostPlan]; ok {
		if err := b.waitTaskStage(stopCtx, cancelCtx, op, r, stage.ID, "Post-plan Tasks"); err != nil {
			return r, err
		}
	}

	// Show any cost estimation output.
	if r.CostEstimate != nil {
		err = b.costEstimate(stopCtx, cancelCtx, op, r)
		if err != nil {
			return r, err
		}
	}

	// Check any configured sentinel policies.
	if len(r.PolicyChecks) > 0 {
		err = b.checkPolicy(stopCtx, cancelCtx, op, r)
		if err != nil {
			return r, err
		}
	}

	return r, nil
}

// AssertImportCompatible errors if the user is attempting to use configuration-
// driven import and the version of the agent or API is too low to support it.
func (b *Cloud) AssertImportCompatible(config *configs.Config) error {
	// Check TFC_RUN_ID is populated, indicating we are running in a remote TFC
	// execution environment.
	if len(config.Module.Import) > 0 && os.Getenv("TFC_RUN_ID") != "" {
		// First, check the remote API version is high enough.
		currentAPIVersion, err := version.NewVersion(b.client.RemoteAPIVersion())
		if err != nil {
			return fmt.Errorf("Error parsing remote API version. To proceed, please remove any import blocks from your config. Please report the following error to the Terraform team: %s", err)
		}
		desiredAPIVersion, _ := version.NewVersion("2.6")
		if currentAPIVersion.LessThan(desiredAPIVersion) {
			return fmt.Errorf("Import blocks are not supported in this version of Terraform Enterprise. Please remove any import blocks from your config or upgrade Terraform Enterprise.")
		}

		// Second, check the agent version is high enough.
		agentEnv, isSet := os.LookupEnv("TFC_AGENT_VERSION")
		if !isSet {
			return fmt.Errorf("Error reading TFC agent version. To proceed, please remove any import blocks from your config. Please report the following error to the Terraform team: TFC_AGENT_VERSION not present.")
		}
		currentAgentVersion, err := version.NewVersion(agentEnv)
		if err != nil {
			return fmt.Errorf("Error parsing TFC agent version. To proceed, please remove any import blocks from your config. Please report the following error to the Terraform team: %s", err)
		}
		desiredAgentVersion, _ := version.NewVersion("1.10")
		if currentAgentVersion.LessThan(desiredAgentVersion) {
			return fmt.Errorf("Import blocks are not supported in this version of the Terraform Cloud Agent. You are using agent version %s, but this feature requires version %s. Please remove any import blocks from your config or upgrade your agent.", currentAgentVersion, desiredAgentVersion)
		}
	}
	return nil
}

// renderPlanLogs reads the streamed plan JSON logs and calls the JSON Plan renderer (jsonformat.RenderPlan) to
// render the plan output. The plan output is fetched from the redacted output endpoint.
func (b *Cloud) renderPlanLogs(ctx context.Context, op *backend.Operation, run *tfe.Run) error {
	logs, err := b.client.Plans.Logs(ctx, run.Plan.ID)
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
						return generalError("Failed to read logs", err)
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
	run, err = b.client.Runs.ReadWithOptions(ctx, run.ID, &tfe.RunReadOptions{
		Include: []tfe.RunIncludeOpt{tfe.RunWorkspace, tfe.RunPlan},
	})
	if err != nil {
		return err
	}

	// If the run was errored, canceled, or discarded we will not resume the rest
	// of this logic and attempt to render the plan, except in certain special circumstances
	// where the plan errored but successfully generated configuration during an
	// import operation. In that case, we need to keep going so we can load the JSON plan
	// and use it to write the generated config to the specified output file.
	shouldGenerateConfig := shouldGenerateConfig(op.GenerateConfigOut, run)
	shouldRenderPlan := shouldRenderPlan(run)
	if !shouldRenderPlan && !shouldGenerateConfig {
		// We won't return an error here since we need to resume the logic that
		// follows after rendering the logs (run tasks, cost estimation, etc.)
		return nil
	}

	// Fetch the redacted JSON plan if we need it for either rendering the plan
	// or writing out generated configuration.
	var redactedPlan *jsonformat.Plan
	renderSRO, err := b.shouldRenderStructuredRunOutput(run)
	if err != nil {
		return err
	}
	if renderSRO || shouldGenerateConfig {
		redactedPlan, err = readRedactedPlan(ctx, b.client.BaseURL(), b.token, run.Plan.ID)
		if err != nil {
			return generalError("Failed to read JSON plan", err)
		}
	}

	// Write any generated config before rendering the plan, so we can stop in case of errors
	if shouldGenerateConfig {
		diags := maybeWriteGeneratedConfig(redactedPlan, op.GenerateConfigOut)
		if diags.HasErrors() {
			return diags.Err()
		}
	}

	// Only generate the human readable output from the plan if structured run output is
	// enabled. Otherwise we risk duplicate plan output since plan output may also be
	// shown in the streamed logs.
	if shouldRenderPlan && renderSRO {
		b.renderer.RenderHumanPlan(*redactedPlan, op.PlanMode)
	}

	return nil
}

// maybeWriteGeneratedConfig attempts to write any generated configuration from the JSON plan
// to the specified output file, if generated configuration exists and the correct flag was
// passed to the plan command.
func maybeWriteGeneratedConfig(plan *jsonformat.Plan, out string) (diags tfdiags.Diagnostics) {
	if genconfig.ShouldWriteConfig(out) {
		diags := genconfig.ValidateTargetFile(out)
		if diags.HasErrors() {
			return diags
		}

		var writer io.Writer
		for _, c := range plan.ResourceChanges {
			change := genconfig.Change{
				Addr:            c.Address,
				GeneratedConfig: c.Change.GeneratedConfig,
			}
			if c.Change.Importing != nil {
				change.ImportID = c.Change.Importing.ID
			}

			var moreDiags tfdiags.Diagnostics
			writer, _, moreDiags = change.MaybeWriteConfig(writer, out)
			if moreDiags.HasErrors() {
				return diags.Append(moreDiags)
			}
		}
	}

	return diags
}

// shouldRenderStructuredRunOutput ensures the remote workspace has structured
// run output enabled and, if using Terraform Enterprise, ensures it is a release
// that supports enabling SRO for CLI-driven runs. The plan output will have
// already been rendered when the logs were read if this wasn't the case.
func (b *Cloud) shouldRenderStructuredRunOutput(run *tfe.Run) (bool, error) {
	if b.renderer == nil || !run.Workspace.StructuredRunOutputEnabled {
		return false, nil
	}

	// If the cloud backend is configured against TFC, we only require that
	// the workspace has structured run output enabled.
	if b.client.IsCloud() && run.Workspace.StructuredRunOutputEnabled {
		return true, nil
	}

	// If the cloud backend is configured against TFE, ensure the release version
	// supports enabling SRO for CLI runs.
	if b.client.IsEnterprise() {
		tfeVersion := b.client.RemoteTFEVersion()
		if tfeVersion != "" {
			v := strings.Split(tfeVersion[1:], "-")
			releaseDate, err := strconv.Atoi(v[0])
			if err != nil {
				return false, err
			}

			// Any release older than 202302-1 will not support enabling SRO for
			// CLI-driven runs
			if releaseDate < 202302 {
				return false, nil
			} else if run.Workspace.StructuredRunOutputEnabled {
				return true, nil
			}
		}
	}

	// Version of TFE is unknowable
	return false, nil
}

func shouldRenderPlan(run *tfe.Run) bool {
	return !(run.Status == tfe.RunErrored || run.Status == tfe.RunCanceled ||
		run.Status == tfe.RunDiscarded)
}

func shouldGenerateConfig(out string, run *tfe.Run) bool {
	return (run.Plan.Status == tfe.PlanErrored || run.Plan.Status == tfe.PlanFinished) &&
		run.Plan.GeneratedConfiguration && len(out) > 0
}

const planDefaultHeader = `
[reset][yellow]Running plan in Terraform Cloud. Output will stream here. Pressing Ctrl-C
will stop streaming the logs, but will not stop the plan running remotely.[reset]

Preparing the remote plan...
`

const runHeader = `
[reset][yellow]To view this run in a browser, visit:
https://%s/app/%s/%s/runs/%s[reset]
`

// The newline in this error is to make it look good in the CLI!
const lockTimeoutErr = `
[reset][red]Lock timeout exceeded, sending interrupt to cancel the remote operation.
[reset]
`
