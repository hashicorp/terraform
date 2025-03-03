// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cloud

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/hashicorp/go-tfe"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	svchost "github.com/hashicorp/terraform-svchost"
	"github.com/hashicorp/terraform-svchost/disco"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/backend/backendrun"
	"github.com/hashicorp/terraform/internal/command/format"
	"github.com/hashicorp/terraform/internal/command/jsonformat"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/logging"
	"github.com/hashicorp/terraform/internal/moduletest"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/terminal"
	"github.com/hashicorp/terraform/internal/tfdiags"
	tfversion "github.com/hashicorp/terraform/version"
)

// TestSuiteRunner executes any tests found in the relevant directories in TFC.
//
// It uploads the configuration and uses go-tfe to execute a .
//
// We keep this separate from Cloud, as the tests don't execute with a
// particular workspace in mind but instead with a specific module from a
// private registry. Many things within Cloud assume the existence of a
// workspace when initialising so it isn't pratical to share this for tests.
type TestSuiteRunner struct {

	// ConfigDirectory and TestingDirectory are the paths to the directory
	// that contains our configuration and our testing files.
	ConfigDirectory  string
	TestingDirectory string

	// Config is the actual loaded config.
	Config *configs.Config

	Services *disco.Disco

	// Source is the private registry module we should be sending the tests
	// to when they execute.
	Source string

	// GlobalVariables are the variables provided by the TF_VAR_* environment
	// variables and -var and -var-file flags.
	GlobalVariables map[string]backendrun.UnparsedVariableValue

	// Stopped and Cancelled track whether the user requested the testing
	// process to be interrupted. Stopped is a nice graceful exit, we'll still
	// tidy up any state that was created and mark the tests with relevant
	// `skipped` status updates. Cancelled is a hard stop right now exit, we
	// won't attempt to clean up any state left hanging, and tests will just
	// be left showing `pending` as the status. We will still print out the
	// destroy summary diagnostics that tell the user what state has been left
	// behind and needs manual clean up.
	Stopped   bool
	Cancelled bool

	// StoppedCtx and CancelledCtx allow in progress Terraform operations to
	// respond to external calls from the test command.
	StoppedCtx   context.Context
	CancelledCtx context.Context

	// Verbose tells the runner to print out plan files during each test run.
	Verbose bool

	// OperationParallelism is the limit Terraform places on total parallel operations
	// during the plan or apply command within a single test run.
	OperationParallelism int

	// Filters restricts which test files will be executed.
	Filters []string

	// Renderer knows how to convert JSON logs retrieved from TFE back into
	// human-readable.
	//
	// If this is nil, the runner will print the raw logs directly to Streams.
	Renderer *jsonformat.Renderer

	// View and Streams provide alternate ways to output raw data to the
	// user.
	View    views.Test
	Streams *terminal.Streams

	// appName is the name of the instance this test suite runner is configured
	// against. Can be "HCP Terraform" or "Terraform Enterprise"
	appName string

	// clientOverride allows tests to specify the client instead of letting the
	// system initialise one itself.
	clientOverride *tfe.Client
}

func (runner *TestSuiteRunner) Stop() {
	runner.Stopped = true
}

func (runner *TestSuiteRunner) IsStopped() bool {
	return runner.Stopped
}

func (runner *TestSuiteRunner) Cancel() {
	runner.Cancelled = true
}

func (runner *TestSuiteRunner) Test() (moduletest.Status, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	configDirectory, err := filepath.Abs(runner.ConfigDirectory)
	if err != nil {
		diags = diags.Append(fmt.Errorf("Failed to get absolute path of the configuration directory: %v", err))
		return moduletest.Error, diags
	}

	variables, variableDiags := ParseCloudRunTestVariables(runner.GlobalVariables)
	diags = diags.Append(variableDiags)
	if variableDiags.HasErrors() {
		// Stop early if we couldn't parse the global variables.
		return moduletest.Error, diags
	}

	addr, err := tfaddr.ParseModuleSource(runner.Source)
	if err != nil {
		if parserError, ok := err.(*tfaddr.ParserError); ok {
			diags = diags.Append(tfdiags.AttributeValue(
				tfdiags.Error,
				parserError.Summary,
				parserError.Detail,
				cty.Path{cty.GetAttrStep{Name: "source"}}))
		} else {
			diags = diags.Append(err)
		}
		return moduletest.Error, diags
	}

	if addr.Package.Host == tfaddr.DefaultModuleRegistryHost {
		// Then they've reference something from the public registry. We can't
		// run tests against that in this way yet.
		diags = diags.Append(tfdiags.AttributeValue(
			tfdiags.Error,
			"Module source points to the public registry",
			"HCP Terraform and Terraform Enterprise can only execute tests for modules held within private registries.",
			cty.Path{cty.GetAttrStep{Name: "source"}}))
		return moduletest.Error, diags
	}

	id := tfe.RegistryModuleID{
		Organization: addr.Package.Namespace,
		Name:         addr.Package.Name,
		Provider:     addr.Package.TargetSystem,
		Namespace:    addr.Package.Namespace,
		RegistryName: tfe.PrivateRegistry,
	}

	client, module, clientDiags := runner.client(addr, id)
	diags = diags.Append(clientDiags)
	if clientDiags.HasErrors() {
		return moduletest.Error, diags
	}

	configurationVersion, err := client.ConfigurationVersions.CreateForRegistryModule(runner.StoppedCtx, id)
	if err != nil {
		diags = diags.Append(runner.generalError("Failed to create configuration version", err))
		return moduletest.Error, diags
	}

	if runner.Stopped || runner.Cancelled {
		return moduletest.Error, diags
	}

	if err := client.ConfigurationVersions.Upload(runner.StoppedCtx, configurationVersion.UploadURL, configDirectory); err != nil {
		diags = diags.Append(runner.generalError("Failed to upload configuration version", err))
		return moduletest.Error, diags
	}

	if runner.Stopped || runner.Cancelled {
		return moduletest.Error, diags
	}

	// From here, we'll pass any cancellation signals into the test run instead
	// of cancelling things locally. The reason for this is we want to make sure
	// the test run tidies up any state properly. This means, we'll send the
	// cancellation signals and then still wait for and process the logs.
	//
	// This also means that all calls to HCP Terraform will use context.Background()
	// instead of the stopped or cancelled context as we want them to finish and
	// the run to be cancelled by HCP Terraform properly.

	opts := tfe.TestRunCreateOptions{
		Filters:       runner.Filters,
		TestDirectory: tfe.String(runner.TestingDirectory),
		Verbose:       tfe.Bool(runner.Verbose),
		Parallelism:   tfe.Int(runner.OperationParallelism),
		Variables: func() []*tfe.RunVariable {
			runVariables := make([]*tfe.RunVariable, 0, len(variables))
			for name, value := range variables {
				runVariables = append(runVariables, &tfe.RunVariable{
					Key:   name,
					Value: value,
				})
			}
			return runVariables
		}(),
		ConfigurationVersion: configurationVersion,
		RegistryModule:       module,
	}

	run, err := client.TestRuns.Create(context.Background(), opts)
	if err != nil {
		diags = diags.Append(runner.generalError("Failed to create test run", err))
		return moduletest.Error, diags
	}

	runningCtx, done := context.WithCancel(context.Background())

	go func() {
		defer logging.PanicHandler()
		defer done()

		// Let's wait for the test run to start separately, so we can provide
		// some nice updates while we wait.

		completed := false
		started := time.Now()
		updated := started
		for i := 0; !completed; i++ {
			run, err := client.TestRuns.Read(context.Background(), id, run.ID)
			if err != nil {
				diags = diags.Append(runner.generalError("Failed to retrieve test run", err))
				return // exit early
			}

			if run.Status != tfe.TestRunQueued {
				// We block as long as the test run is still queued.
				completed = true
				continue // We can render the logs now.
			}

			current := time.Now()
			if i == 0 || current.Sub(updated).Seconds() > 30 {
				updated = current

				// TODO: Provide better updates based on queue status etc.
				// We could look through the queue to find out exactly where the
				// test run is and give a count down. Other stuff like that.
				// For now, we'll just print a simple status updated.

				runner.View.TFCStatusUpdate(run.Status, current.Sub(started))
			}
		}

		// The test run has actually started now, so let's render the logs.

		logDiags := runner.renderLogs(client, run, id)
		diags = diags.Append(logDiags)
	}()

	// We're doing a couple of things in the wait function. Firstly, waiting
	// for the test run to actually finish. Secondly, listening for interrupt
	// signals and forwarding them onto TFC.
	waitDiags := runner.wait(runningCtx, client, run, id)
	diags = diags.Append(waitDiags)

	if diags.HasErrors() {
		return moduletest.Error, diags
	}

	// Refresh the run now we know it is finished.
	run, err = client.TestRuns.Read(context.Background(), id, run.ID)
	if err != nil {
		diags = diags.Append(runner.generalError("Failed to retrieve completed test run", err))
		return moduletest.Error, diags
	}

	if run.Status != tfe.TestRunFinished {
		// The only reason we'd get here without the run being finished properly
		// is because the run errored outside the scope of the tests, or because
		// the run was cancelled. Either way, we can just mark it has having
		// errored for the purpose of our return code.
		return moduletest.Error, diags
	}

	// Otherwise the run has finished successfully, and we can look at the
	// actual status of the test instead of the run to figure out what status we
	// should return.

	switch run.TestStatus {
	case tfe.TestError:
		return moduletest.Error, diags
	case tfe.TestFail:
		return moduletest.Fail, diags
	case tfe.TestPass:
		return moduletest.Pass, diags
	case tfe.TestPending:
		return moduletest.Pending, diags
	case tfe.TestSkip:
		return moduletest.Skip, diags
	default:
		panic("found unrecognized test status: " + run.TestStatus)
	}
}

// discover the TFC/E API service URL
func discoverTfeURL(hostname svchost.Hostname, services *disco.Disco) (*url.URL, error) {
	host, err := services.Discover(hostname)
	if err != nil {
		var serviceDiscoErr *disco.ErrServiceDiscoveryNetworkRequest

		switch {
		case errors.As(err, &serviceDiscoErr):
			err = fmt.Errorf("a network issue prevented cloud configuration; %w", err)
			return nil, err
		default:
			return nil, err
		}
	}

	return host.ServiceURL(tfeServiceID)
}

func (runner *TestSuiteRunner) client(addr tfaddr.Module, id tfe.RegistryModuleID) (*tfe.Client, *tfe.RegistryModule, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	var client *tfe.Client
	if runner.clientOverride != nil {
		client = runner.clientOverride
	} else {
		service, err := discoverTfeURL(addr.Package.Host, runner.Services)
		if err != nil {
			diags = diags.Append(tfdiags.AttributeValue(
				tfdiags.Error,
				strings.ToUpper(err.Error()[:1])+err.Error()[1:],
				"", // no description is needed here, the error is clear
				cty.Path{cty.GetAttrStep{Name: "hostname"}},
			))
			return nil, nil, diags
		}

		token, err := cliConfigToken(addr.Package.Host, runner.Services)
		if err != nil {
			diags = diags.Append(tfdiags.AttributeValue(
				tfdiags.Error,
				strings.ToUpper(err.Error()[:1])+err.Error()[1:],
				"", // no description is needed here, the error is clear
				cty.Path{cty.GetAttrStep{Name: "hostname"}},
			))
			return nil, nil, diags
		}

		if token == "" {
			hostname := addr.Package.Host.ForDisplay()

			loginCommand := "terraform login"
			if hostname != defaultHostname {
				loginCommand = loginCommand + " " + hostname
			}
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Required token could not be found",
				fmt.Sprintf(
					"Run the following command to generate a token for %s:\n    %s",
					hostname,
					loginCommand,
				),
			))
			return nil, nil, diags
		}

		cfg := &tfe.Config{
			Address:      service.String(),
			BasePath:     service.Path,
			Token:        token,
			Headers:      make(http.Header),
			RetryLogHook: runner.View.TFCRetryHook,
		}

		// Set the version header to the current version.
		cfg.Headers.Set(tfversion.Header, tfversion.Version)
		cfg.Headers.Set(headerSourceKey, headerSourceValue)

		if client, err = tfe.NewClient(cfg); err != nil {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Failed to create the HCP Terraform or Terraform Enterprise client",
				fmt.Sprintf(
					`Encountered an unexpected error while creating the `+
						`HCP Terraform or Terraform Enterprise client: %s.`, err,
				),
			))
			return nil, nil, diags
		}
	}

	module, err := client.RegistryModules.Read(runner.StoppedCtx, id)
	if err != nil {
		// Then the module doesn't exist, and we can't run tests against it.
		if err == tfe.ErrResourceNotFound {
			err = fmt.Errorf("module %q was not found.\n\nPlease ensure that the organization and hostname are correct and that your API token for %s is valid.", addr.ForDisplay(), addr.Package.Host.ForDisplay())
		}
		diags = diags.Append(tfdiags.AttributeValue(
			tfdiags.Error,
			fmt.Sprintf("Failed to read module %q", addr.ForDisplay()),
			fmt.Sprintf("Encountered an unexpected error while the module: %s", err),
			cty.Path{cty.GetAttrStep{Name: "source"}}))
		return client, nil, diags
	}

	// Enable retries for server errors.
	client.RetryServerErrors(true)

	runner.appName = client.AppName()
	if isValidAppName(runner.appName) {
		runner.appName = "HCP Terraform"
	}

	// Aaaaand I'm done.
	return client, module, diags
}

func (runner *TestSuiteRunner) wait(ctx context.Context, client *tfe.Client, run *tfe.TestRun, moduleId tfe.RegistryModuleID) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	handleCancelled := func() {
		if err := client.TestRuns.Cancel(context.Background(), moduleId, run.ID); err != nil {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Could not cancel the test run",
				fmt.Sprintf("Terraform could not cancel the test run, you will have to navigate to the %s console and cancel the test run manually.\n\nThe error message received when cancelling the test run was %s", client.AppName(), err)))
			return
		}

		// At this point we've requested a force cancel, and we know that
		// Terraform locally is just going to quit after some amount of time so
		// we'll just wait for that to happen or for HCP Terraform to finish, whichever
		// happens first.
		<-ctx.Done()
	}

	handleStopped := func() {
		if err := client.TestRuns.Cancel(context.Background(), moduleId, run.ID); err != nil {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Could not stop the test run",
				fmt.Sprintf("Terraform could not stop the test run, you will have to navigate to the %s console and cancel the test run manually.\n\nThe error message received when stopping the test run was %s", client.AppName(), err)))
			return
		}

		// We've request a cancel, we're happy to just wait for HCP Terraform to cancel
		// the run appropriately.
		select {
		case <-runner.CancelledCtx.Done():
			// We got more pushy, let's force cancel.
			handleCancelled()
		case <-ctx.Done():
			// It finished normally after we request the cancel. Do nothing.
		}
	}

	select {
	case <-runner.StoppedCtx.Done():
		// The StoppedCtx is passed in from the command package, which is
		// listening for interrupts from the user. After the first interrupt the
		// StoppedCtx is triggered.
		handleStopped()
	case <-ctx.Done():
		// The remote run finished normally! Do nothing.
	}

	return diags
}

func (runner *TestSuiteRunner) renderLogs(client *tfe.Client, run *tfe.TestRun, moduleId tfe.RegistryModuleID) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	logs, err := client.TestRuns.Logs(context.Background(), moduleId, run.ID)
	if err != nil {
		diags = diags.Append(runner.generalError("Failed to retrieve logs", err))
		return diags
	}

	reader := bufio.NewReaderSize(logs, 64*1024)

	for next := true; next; {
		var l, line []byte
		var err error

		for isPrefix := true; isPrefix; {
			l, isPrefix, err = reader.ReadLine()
			if err != nil {
				if err != io.EOF {
					diags = diags.Append(runner.generalError("Failed to read logs", err))
					return diags
				}
				next = false
			}

			line = append(line, l...)
		}

		if next || len(line) > 0 {

			if runner.Renderer != nil {
				log := jsonformat.JSONLog{}
				if err := json.Unmarshal(line, &log); err != nil {
					runner.Streams.Println(string(line)) // Just print the raw line so the user can still try and interpret the information.
					continue
				}

				// Most of the log types can be rendered with just the
				// information they contain. We just pass these straight into
				// the renderer. Others, however, need additional context that
				// isn't available within the renderer so we process them first.

				switch log.Type {
				case jsonformat.LogTestInterrupt:
					interrupt := log.TestFatalInterrupt

					runner.Streams.Eprintln(format.WordWrap(log.Message, runner.Streams.Stderr.Columns()))
					if len(interrupt.State) > 0 {
						runner.Streams.Eprint(format.WordWrap("\nTerraform has already created the following resources from the module under test:\n", runner.Streams.Stderr.Columns()))
						for _, resource := range interrupt.State {
							if len(resource.DeposedKey) > 0 {
								runner.Streams.Eprintf(" - %s (%s)\n", resource.Instance, resource.DeposedKey)
							} else {
								runner.Streams.Eprintf(" - %s\n", resource.Instance)
							}
						}
					}

					if len(interrupt.States) > 0 {
						for run, resources := range interrupt.States {
							runner.Streams.Eprint(format.WordWrap(fmt.Sprintf("\nTerraform has already created the following resources for %q:\n", run), runner.Streams.Stderr.Columns()))

							for _, resource := range resources {
								if len(resource.DeposedKey) > 0 {
									runner.Streams.Eprintf(" - %s (%s)\n", resource.Instance, resource.DeposedKey)
								} else {
									runner.Streams.Eprintf(" - %s\n", resource.Instance)
								}
							}
						}
					}

					if len(interrupt.Planned) > 0 {
						module := "the module under test"
						for _, run := range runner.Config.Module.Tests[log.TestFile].Runs {
							if run.Name == log.TestRun && run.ConfigUnderTest != nil {
								module = fmt.Sprintf("%q", run.Module.Source.String())
							}
						}

						runner.Streams.Eprint(format.WordWrap(fmt.Sprintf("\nTerraform was in the process of creating the following resources for %q from %s, and they may not have been destroyed:\n", log.TestRun, module), runner.Streams.Stderr.Columns()))
						for _, resource := range interrupt.Planned {
							runner.Streams.Eprintf("  - %s\n", resource)
						}
					}

				case jsonformat.LogTestPlan:
					var uimode plans.Mode
					for _, run := range runner.Config.Module.Tests[log.TestFile].Runs {
						if run.Name == log.TestRun {
							switch run.Options.Mode {
							case configs.RefreshOnlyTestMode:
								uimode = plans.RefreshOnlyMode
							case configs.NormalTestMode:
								uimode = plans.NormalMode
							}

							// Don't keep searching the runs.
							break
						}
					}
					runner.Renderer.RenderHumanPlan(*log.TestPlan, uimode)

				case jsonformat.LogTestState:
					runner.Renderer.RenderHumanState(*log.TestState)

				default:
					// For all the rest we can just hand over to the renderer
					// to handle directly.
					if err := runner.Renderer.RenderLog(&log); err != nil {
						runner.Streams.Println(string(line)) // Just print the raw line so the can still try and interpret the information.
						continue
					}
				}

			} else {
				runner.Streams.Println(string(line)) // If the renderer is null, it means the user just wants to see the raw JSON outputs anyway.
			}
		}
	}

	return diags
}

func (runner *TestSuiteRunner) generalError(msg string, err error) error {
	var diags tfdiags.Diagnostics

	if urlErr, ok := err.(*url.Error); ok {
		err = urlErr.Err
	}

	switch err {
	case context.Canceled:
		return err
	case tfe.ErrResourceNotFound:
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			fmt.Sprintf("%s: %v", msg, err),
			fmt.Sprintf("For security, %s return '404 Not Found' responses for resources\n", runner.appName)+
				"for resources that a user doesn't have access to, in addition to resources that\n"+
				"do not exist. If the resource does exist, please check the permissions of the provided token.",
		))
		return diags.Err()
	default:
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			fmt.Sprintf("%s: %v", msg, err),
			fmt.Sprintf(`%s returned an unexpected error. Sometimes `, runner.appName)+
				`this is caused by network connection problems, in which case you could retry `+
				`the command. If the issue persists please open a support ticket to get help `+
				`resolving the problem.`,
		))
		return diags.Err()
	}
}
