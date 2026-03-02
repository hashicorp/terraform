// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package cloud

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	tfe "github.com/hashicorp/go-tfe"
	version "github.com/hashicorp/go-version"

	"github.com/hashicorp/terraform/internal/backend/backendrun"
	"github.com/hashicorp/terraform/internal/cloud/cloudplan"
	"github.com/hashicorp/terraform/internal/command/jsonformat"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/genconfig"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

var planConfigurationVersionsPollInterval = 500 * time.Millisecond

func (b *Cloud) opPlan(stopCtx, cancelCtx context.Context, op *backendrun.Operation, w *tfe.Workspace) (OperationResult, error) {
	log.Printf("[INFO] cloud: starting Plan operation")

	var diags tfdiags.Diagnostics

	if !w.Permissions.CanQueueRun {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Insufficient rights to generate a plan",
			"The provided credentials have insufficient rights to generate a plan. In order "+
				"to generate plans, at least plan permissions on the workspace are required.",
		))
		return &RunResult{}, diags.Err()
	}

	if w.VCSRepo != nil && op.PlanOutPath != "" {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Saved plans not allowed for workspaces with a VCS connection",
			"A workspace that is connected to a VCS requires the VCS-driven workflow "+
				"to ensure that the VCS remains the single source of truth.",
		))
		return &RunResult{}, diags.Err()
	}

	if b.ContextOpts != nil && b.ContextOpts.Parallelism != defaultParallelism {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Custom parallelism values are currently not supported",
			fmt.Sprintf("%s does not support setting a custom parallelism ", b.appName)+
				"value at this time.",
		))
	}

	if op.PlanFile != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Displaying a saved plan is currently not supported",
			fmt.Sprintf("%s currently requires configuration to be present and ", b.appName)+
				"does not accept an existing saved plan as an argument at this time.",
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
		return &RunResult{}, diags.Err()
	}

	// If the run errored, exit before checking whether to save a plan file
	run, err := b.plan(stopCtx, cancelCtx, op, w)
	if err != nil {
		return &RunResult{}, err
	}

	// Save plan file if -out <FILE> was specified
	if op.PlanOutPath != "" {
		bookmark := cloudplan.NewSavedPlanBookmark(run.ID, b.Hostname)
		err = bookmark.Save(op.PlanOutPath)
		if err != nil {
			return &RunResult{}, err
		}
	}

	// Everything succeded, so display next steps
	op.View.PlanNextStep(op.PlanOutPath, op.GenerateConfigOut)

	return &RunResult{run: run, backend: b}, nil
}

func (b *Cloud) plan(stopCtx, cancelCtx context.Context, op *backendrun.Operation, w *tfe.Workspace) (*tfe.Run, error) {
	if b.CLI != nil {
		header := fmt.Sprintf(planDefaultHeader, b.appName)
		if op.Type == backendrun.OperationTypeApply || op.Type == backendrun.OperationTypeRefresh {
			header = fmt.Sprintf(applyDefaultHeader, b.appName)
		}
		b.CLI.Output(b.Colorize().Color(strings.TrimSpace(header) + "\n"))
	}

	// Plan-only means they ran terraform plan without -out.
	provisional := op.PlanOutPath != ""
	planOnly := op.Type == backendrun.OperationTypePlan && !provisional

	configOptions := tfe.ConfigurationVersionCreateOptions{
		AutoQueueRuns: tfe.Bool(false),
		Speculative:   tfe.Bool(planOnly),
		Provisional:   tfe.Bool(provisional),
	}

	cv, err := b.uploadConfigurationVersion(stopCtx, cancelCtx, op, w, configOptions)
	if err != nil {
		return nil, err
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
		return nil, b.generalError(
			"Invalid plan mode",
			fmt.Errorf("%s doesn't support %s", b.appName, op.PlanMode),
		)
	}

	if len(op.Targets) != 0 {
		runOptions.TargetAddrs = make([]string, 0, len(op.Targets))
		for _, addr := range op.Targets {
			runOptions.TargetAddrs = append(runOptions.TargetAddrs, addr.String())
		}
	}

	if len(op.ActionTargets) != 0 {
		if len(op.ActionTargets) > 1 {
			// For now, we only support a single action from the command line.
			// We've future proofed the API and inputs so we can send multiple
			// but versions of Terraform will enforce this both here, and
			// on the other side.
			//
			// It shouldn't actually be possible to reach here anyway - we're
			// validating at the point the flag is read that it only has a
			// single entry. But, we'll check again to be safe.
			return nil, b.generalError("Invalid arguments",
				errors.New("at most 1 action can be invoked per operation"))
		}

		for _, target := range op.ActionTargets {
			runOptions.InvokeActionAddrs = append(runOptions.InvokeActionAddrs, target.String())
		}

	}

	if len(op.ForceReplace) != 0 {
		runOptions.ReplaceAddrs = make([]string, 0, len(op.ForceReplace))
		for _, addr := range op.ForceReplace {
			runOptions.ReplaceAddrs = append(runOptions.ReplaceAddrs, addr.String())
		}
	}

	runVariables, err := b.parseRunVariables(op)
	if err != nil {
		return nil, err
	}
	runOptions.Variables = runVariables

	if len(op.GenerateConfigOut) > 0 {
		runOptions.AllowConfigGeneration = tfe.Bool(true)
	}

	r, err := b.client.Runs.Create(stopCtx, runOptions)
	if err != nil {
		return r, b.generalError("Failed to create run", err)
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
			runHeader, b.Hostname, b.Organization, op.Workspace, r.ID)) + "\n"))
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
		return r, b.generalError("Failed to retrieve run", err)
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
			return fmt.Errorf("Error reading HCP Terraform Agent version. To proceed, please remove any import blocks from your config. Please report the following error to the Terraform team: TFC_AGENT_VERSION not present.")
		}
		currentAgentVersion, err := version.NewVersion(agentEnv)
		if err != nil {
			return fmt.Errorf("Error parsing HCP Terraform Agent version. To proceed, please remove any import blocks from your config. Please report the following error to the Terraform team: %s", err)
		}
		desiredAgentVersion, _ := version.NewVersion("1.10")
		if currentAgentVersion.LessThan(desiredAgentVersion) {
			return fmt.Errorf("Import blocks are not supported in this version of the HCP Terraform Agent. You are using agent version %s, but this feature requires version %s. Please remove any import blocks from your config or upgrade your agent.", currentAgentVersion, desiredAgentVersion)
		}
	}
	return nil
}

// renderPlanLogs reads the streamed plan JSON logs and calls the JSON Plan renderer (jsonformat.RenderPlan) to
// render the plan output. The plan output is fetched from the redacted output endpoint.
func (b *Cloud) renderPlanLogs(ctx context.Context, op *backendrun.Operation, run *tfe.Run) error {
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
		jsonBytes, err := readRedactedPlan(ctx, b.client.BaseURL(), b.Token, run.Plan.ID)
		if err != nil {
			return b.generalError("Failed to read JSON plan", err)
		}
		redactedPlan, err = decodeRedactedPlan(jsonBytes)
		if err != nil {
			return b.generalError("Failed to decode JSON plan", err)
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
[reset][yellow]Running plan in %s. Output will stream here. Pressing Ctrl-C
will stop streaming the logs, but will not stop the plan running remotely.[reset]

Preparing the remote plan...
`

const runHeader = `
[reset][yellow]To view this run in a browser, visit:[reset]
[reset][yellow]https://%s/app/%s/%s/runs/%s[reset]
`

const runHeaderErr = `
To view this run in the browser, visit:
https://%s/app/%s/%s/runs/%s
`

// The newline in this error is to make it look good in the CLI!
const lockTimeoutErr = `
[reset][red]Lock timeout exceeded, sending interrupt to cancel the remote operation.
[reset]
`
