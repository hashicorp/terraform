// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cloud

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/go-tfe"

	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/jsonformat"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/configs/configload"
	"github.com/hashicorp/terraform/internal/terminal"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestTest(t *testing.T) {

	streams, done := terminal.StreamsForTesting(t)
	view := views.NewTest(arguments.ViewHuman, views.NewView(streams))

	colorize := mockColorize()
	colorize.Disable = true

	mock := NewMockClient()
	client := &tfe.Client{
		ConfigurationVersions: mock.ConfigurationVersions,
		Organizations:         mock.Organizations,
		RegistryModules:       mock.RegistryModules,
		TestRuns:              mock.TestRuns,
	}

	if _, err := client.Organizations.Create(context.Background(), tfe.OrganizationCreateOptions{
		Name: tfe.String("organisation"),
	}); err != nil {
		t.Fatalf("failed to create organisation: %v", err)
	}

	if _, err := client.RegistryModules.Create(context.Background(), "organisation", tfe.RegistryModuleCreateOptions{
		Name:         tfe.String("name"),
		Provider:     tfe.String("provider"),
		RegistryName: "app.terraform.io",
		Namespace:    "organisation",
	}); err != nil {
		t.Fatalf("failed to create registry module: %v", err)
	}

	runner := TestSuiteRunner{
		// Configuration data.
		ConfigDirectory:  "testdata/test",
		TestingDirectory: "tests",
		Config:           nil, // We don't need this for this test.
		Source:           "app.terraform.io/organisation/name/provider",

		// Cancellation controls, we won't be doing any cancellations in this
		// test.
		Stopped:      false,
		Cancelled:    false,
		StoppedCtx:   context.Background(),
		CancelledCtx: context.Background(),

		// Test Options, empty for this test.
		GlobalVariables: nil,
		Verbose:         false,
		Filters:         nil,

		// Outputs
		Renderer: &jsonformat.Renderer{
			Streams:             streams,
			Colorize:            colorize,
			RunningInAutomation: false,
		},
		View:    view,
		Streams: streams,

		// Networking
		Services:       nil, // Don't need this when the client is overridden.
		clientOverride: client,
	}

	_, diags := runner.Test()
	if len(diags) > 0 {
		t.Errorf("found diags and expected none: %s", diags.ErrWithWarnings())
	}

	output := done(t)
	actual := output.All()
	expected := `main.tftest.hcl... in progress
  defaults... pass
  overrides... pass
main.tftest.hcl... tearing down
main.tftest.hcl... pass

Success! 2 passed, 0 failed.
`

	if diff := cmp.Diff(expected, actual); len(diff) > 0 {
		t.Errorf("expected:\n%s\nactual:\n%s\ndiff:\n%s", expected, actual, diff)
	}
}

func TestTest_Parallelism(t *testing.T) {

	streams, _ := terminal.StreamsForTesting(t)
	view := views.NewTest(arguments.ViewHuman, views.NewView(streams))

	colorize := mockColorize()
	colorize.Disable = true

	mock := NewMockClient()
	client := &tfe.Client{
		ConfigurationVersions: mock.ConfigurationVersions,
		Organizations:         mock.Organizations,
		RegistryModules:       mock.RegistryModules,
		TestRuns:              mock.TestRuns,
	}

	if _, err := client.Organizations.Create(context.Background(), tfe.OrganizationCreateOptions{
		Name: tfe.String("organisation"),
	}); err != nil {
		t.Fatalf("failed to create organisation: %v", err)
	}

	if _, err := client.RegistryModules.Create(context.Background(), "organisation", tfe.RegistryModuleCreateOptions{
		Name:         tfe.String("name"),
		Provider:     tfe.String("provider"),
		RegistryName: "app.terraform.io",
		Namespace:    "organisation",
	}); err != nil {
		t.Fatalf("failed to create registry module: %v", err)
	}

	runner := TestSuiteRunner{
		// Configuration data.
		ConfigDirectory:  "testdata/test",
		TestingDirectory: "tests",
		Config:           nil, // We don't need this for this test.
		Source:           "app.terraform.io/organisation/name/provider",

		// Cancellation controls, we won't be doing any cancellations in this
		// test.
		Stopped:      false,
		Cancelled:    false,
		StoppedCtx:   context.Background(),
		CancelledCtx: context.Background(),

		// Test Options, empty for this test.
		GlobalVariables:      nil,
		Verbose:              false,
		OperationParallelism: 4,
		Filters:              nil,

		// Outputs
		Renderer: &jsonformat.Renderer{
			Streams:             streams,
			Colorize:            colorize,
			RunningInAutomation: false,
		},
		View:    view,
		Streams: streams,

		// Networking
		Services:       nil, // Don't need this when the client is overridden.
		clientOverride: client,
	}

	_, diags := runner.Test()
	if len(diags) > 0 {
		t.Errorf("found diags and expected none: %s", diags.ErrWithWarnings())
	}

	if mock.TestRuns.parallelism != 4 {
		t.Errorf("expected parallelism to be 4 but was %d", mock.TestRuns.parallelism)
	}
}

func TestTest_JSON(t *testing.T) {

	streams, done := terminal.StreamsForTesting(t)
	view := views.NewTest(arguments.ViewHuman, views.NewView(streams))

	colorize := mockColorize()
	colorize.Disable = true

	mock := NewMockClient()
	client := &tfe.Client{
		ConfigurationVersions: mock.ConfigurationVersions,
		Organizations:         mock.Organizations,
		RegistryModules:       mock.RegistryModules,
		TestRuns:              mock.TestRuns,
	}

	if _, err := client.Organizations.Create(context.Background(), tfe.OrganizationCreateOptions{
		Name: tfe.String("organisation"),
	}); err != nil {
		t.Fatalf("failed to create organisation: %v", err)
	}

	if _, err := client.RegistryModules.Create(context.Background(), "organisation", tfe.RegistryModuleCreateOptions{
		Name:         tfe.String("name"),
		Provider:     tfe.String("provider"),
		RegistryName: "app.terraform.io",
		Namespace:    "organisation",
	}); err != nil {
		t.Fatalf("failed to create registry module: %v", err)
	}

	runner := TestSuiteRunner{
		// Configuration data.
		ConfigDirectory:  "testdata/test",
		TestingDirectory: "tests",
		Config:           nil, // We don't need this for this test.
		Source:           "app.terraform.io/organisation/name/provider",

		// Cancellation controls, we won't be doing any cancellations in this
		// test.
		Stopped:      false,
		Cancelled:    false,
		StoppedCtx:   context.Background(),
		CancelledCtx: context.Background(),

		// Test Options, empty for this test.
		GlobalVariables: nil,
		Verbose:         false,
		Filters:         nil,

		// Outputs
		Renderer: nil, // This should force the logs to render as JSON.
		View:     view,
		Streams:  streams,

		// Networking
		Services:       nil, // Don't need this when the client is overridden.
		clientOverride: client,
	}

	_, diags := runner.Test()
	if len(diags) > 0 {
		t.Errorf("found diags and expected none: %s", diags.ErrWithWarnings())
	}

	output := done(t)
	actual := output.All()
	expected := `{"@level":"info","@message":"Terraform 1.6.0-dev","@module":"terraform.ui","@timestamp":"2023-09-12T08:29:27.257413+02:00","terraform":"1.6.0-dev","type":"version","ui":"1.2"}
{"@level":"info","@message":"Found 1 file and 2 run blocks","@module":"terraform.ui","@timestamp":"2023-09-12T08:29:27.268731+02:00","test_abstract":{"main.tftest.hcl":["defaults","overrides"]},"type":"test_abstract"}
{"@level":"info","@message":"main.tftest.hcl... in progress","@module":"terraform.ui","@testfile":"main.tftest.hcl","@timestamp":"2023-09-12T08:29:27.268889+02:00","test_file":{"path":"main.tftest.hcl","progress":"starting"},"type":"test_file"}
{"@level":"info","@message":"  \"defaults\"... pass","@module":"terraform.ui","@testfile":"main.tftest.hcl","@testrun":"defaults","@timestamp":"2023-09-12T08:29:27.710541+02:00","test_run":{"path":"main.tftest.hcl","run":"defaults","progress":"complete","status":"pass"},"type":"test_run"}
{"@level":"info","@message":"  \"overrides\"... pass","@module":"terraform.ui","@testfile":"main.tftest.hcl","@testrun":"overrides","@timestamp":"2023-09-12T08:29:27.833351+02:00","test_run":{"path":"main.tftest.hcl","run":"overrides","progress":"complete","status":"pass"},"type":"test_run"}
{"@level":"info","@message":"main.tftest.hcl... tearing down","@module":"terraform.ui","@testfile":"main.tftest.hcl","@timestamp":"2023-09-12T08:29:27.833375+02:00","test_file":{"path":"main.tftest.hcl","progress":"teardown"},"type":"test_file"}
{"@level":"info","@message":"main.tftest.hcl... pass","@module":"terraform.ui","@testfile":"main.tftest.hcl","@timestamp":"2023-09-12T08:29:27.956488+02:00","test_file":{"path":"main.tftest.hcl","progress":"complete","status":"pass"},"type":"test_file"}
{"@level":"info","@message":"Success! 2 passed, 0 failed.","@module":"terraform.ui","@timestamp":"2023-09-12T08:29:27.956510+02:00","test_summary":{"status":"pass","passed":2,"failed":0,"errored":0,"skipped":0},"type":"test_summary"}
`

	if diff := cmp.Diff(expected, actual); len(diff) > 0 {
		t.Errorf("expected:\n%s\nactual:\n%s\ndiff:\n%s", expected, actual, diff)
	}
}

func TestTest_Verbose(t *testing.T) {

	directory := "testdata/test-verbose"

	loader, close := configload.NewLoaderForTests(t)
	defer close()

	config, configDiags := loader.LoadConfigWithTests(directory, "tests")
	if configDiags.HasErrors() {
		t.Fatalf("failed to load config: %v", configDiags.Error())
	}

	streams, done := terminal.StreamsForTesting(t)
	view := views.NewTest(arguments.ViewHuman, views.NewView(streams))

	colorize := mockColorize()
	colorize.Disable = true

	mock := NewMockClient()
	client := &tfe.Client{
		ConfigurationVersions: mock.ConfigurationVersions,
		Organizations:         mock.Organizations,
		RegistryModules:       mock.RegistryModules,
		TestRuns:              mock.TestRuns,
	}

	if _, err := client.Organizations.Create(context.Background(), tfe.OrganizationCreateOptions{
		Name: tfe.String("organisation"),
	}); err != nil {
		t.Fatalf("failed to create organisation: %v", err)
	}

	if _, err := client.RegistryModules.Create(context.Background(), "organisation", tfe.RegistryModuleCreateOptions{
		Name:         tfe.String("name"),
		Provider:     tfe.String("provider"),
		RegistryName: "app.terraform.io",
		Namespace:    "organisation",
	}); err != nil {
		t.Fatalf("failed to create registry module: %v", err)
	}

	runner := TestSuiteRunner{
		// Configuration data.
		ConfigDirectory:  directory,
		TestingDirectory: "tests",
		Config:           config,
		Source:           "app.terraform.io/organisation/name/provider",

		// Cancellation controls, we won't be doing any cancellations in this
		// test.
		Stopped:      false,
		Cancelled:    false,
		StoppedCtx:   context.Background(),
		CancelledCtx: context.Background(),

		// The test options don't actually matter, as we just retrieve whatever
		// is set in the log file.
		GlobalVariables: nil,
		Verbose:         false,
		Filters:         nil,

		// Outputs
		Renderer: &jsonformat.Renderer{
			Streams:             streams,
			Colorize:            colorize,
			RunningInAutomation: false,
		},
		View:    view,
		Streams: streams,

		// Networking
		Services:       nil, // Don't need this when the client is overridden.
		clientOverride: client,
	}

	_, diags := runner.Test()
	if len(diags) > 0 {
		t.Errorf("found diags and expected none: %s", diags.ErrWithWarnings())
	}

	output := done(t)
	actual := output.All()
	expected := `main.tftest.hcl... in progress
  defaults... pass

Changes to Outputs:
  + input = "Hello, world!"

You can apply this plan to save these new output values to the Terraform
state, without changing any real infrastructure.
╷
│ Warning: Deprecated
│
│   with data.null_data_source.values,
│   on main.tf line 7, in data "null_data_source" "values":
│    7: data "null_data_source" "values" {
│
│ The null_data_source was historically used to construct intermediate values
│ to re-use elsewhere in configuration, the same can now be achieved using
│ locals
╵
╷
│ Warning: Deprecated
│
│   with data.null_data_source.values,
│   on main.tf line 7, in data "null_data_source" "values":
│    7: data "null_data_source" "values" {
│
│ The null_data_source was historically used to construct intermediate values
│ to re-use elsewhere in configuration, the same can now be achieved using
│ locals
╵
  overrides... pass
# data.null_data_source.values:
data "null_data_source" "values" {
    has_computed_default = "default"
    id                   = "static"
    inputs               = {
        "data" = "Hello, universe!"
    }
    outputs              = {
        "data" = "Hello, universe!"
    }
    random               = "8484833523059069761"
}


Outputs:

input = "Hello, universe!"
╷
│ Warning: Deprecated
│
│   with data.null_data_source.values,
│   on main.tf line 7, in data "null_data_source" "values":
│    7: data "null_data_source" "values" {
│
│ The null_data_source was historically used to construct intermediate values
│ to re-use elsewhere in configuration, the same can now be achieved using
│ locals
╵
╷
│ Warning: Deprecated
│
│   with data.null_data_source.values,
│   on main.tf line 7, in data "null_data_source" "values":
│    7: data "null_data_source" "values" {
│
│ The null_data_source was historically used to construct intermediate values
│ to re-use elsewhere in configuration, the same can now be achieved using
│ locals
╵
main.tftest.hcl... tearing down
main.tftest.hcl... pass

Success! 2 passed, 0 failed.
`

	if diff := cmp.Diff(expected, actual); len(diff) > 0 {
		t.Errorf("expected:\n%s\nactual:\n%s\ndiff:\n%s", expected, actual, diff)
	}
}

func TestTest_Cancel(t *testing.T) {

	streams, outputFn := terminal.StreamsForTesting(t)
	view := views.NewTest(arguments.ViewHuman, views.NewView(streams))

	colorize := mockColorize()
	colorize.Disable = true

	mock := NewMockClient()
	client := &tfe.Client{
		ConfigurationVersions: mock.ConfigurationVersions,
		Organizations:         mock.Organizations,
		RegistryModules:       mock.RegistryModules,
		TestRuns:              mock.TestRuns,
	}

	if _, err := client.Organizations.Create(context.Background(), tfe.OrganizationCreateOptions{
		Name: tfe.String("organisation"),
	}); err != nil {
		t.Fatalf("failed to create organisation: %v", err)
	}

	module, err := client.RegistryModules.Create(context.Background(), "organisation", tfe.RegistryModuleCreateOptions{
		Name:         tfe.String("name"),
		Provider:     tfe.String("provider"),
		RegistryName: "app.terraform.io",
		Namespace:    "organisation",
	})
	if err != nil {
		t.Fatalf("failed to create registry module: %v", err)
	}

	doneContext, done := context.WithCancel(context.Background())
	stopContext, stop := context.WithCancel(context.Background())

	runner := TestSuiteRunner{
		// Configuration data.
		ConfigDirectory:  "testdata/test-cancel",
		TestingDirectory: "tests",
		Config:           nil, // We don't need this for this test.
		Source:           "app.terraform.io/organisation/name/provider",

		// Cancellation controls, we won't be doing any cancellations in this
		// test.
		Stopped:      false,
		Cancelled:    false,
		StoppedCtx:   stopContext,
		CancelledCtx: context.Background(),

		// Test Options, empty for this test.
		GlobalVariables: nil,
		Verbose:         false,
		Filters:         nil,

		// Outputs
		Renderer: &jsonformat.Renderer{
			Streams:             streams,
			Colorize:            colorize,
			RunningInAutomation: false,
		},
		View:    view,
		Streams: streams,

		// Networking
		Services:       nil, // Don't need this when the client is overridden.
		clientOverride: client,
	}

	// We're only going to be able to finish this if the cancellation calls
	// are done correctly.
	mock.TestRuns.targetCancels = 1

	var diags tfdiags.Diagnostics
	go func() {
		defer done()
		_, diags = runner.Test()
	}()

	stop() // immediately cancel

	// Wait for finish!
	<-doneContext.Done()

	if len(diags) > 0 {
		t.Errorf("found diags and expected none: %s", diags.ErrWithWarnings())
	}

	output := outputFn(t)
	actual := output.All()
	expected := `main.tftest.hcl... in progress

Interrupt received.
Please wait for Terraform to exit or data loss may occur.
Gracefully shutting down...

  defaults... pass
  overrides... skip
main.tftest.hcl... tearing down
main.tftest.hcl... pass

Success! 1 passed, 0 failed, 1 skipped.
`

	if diff := cmp.Diff(expected, actual); len(diff) > 0 {
		t.Errorf("expected:\n%s\nactual:\n%s\ndiff:\n%s", expected, actual, diff)
	}

	// We want to make sure the cancel signal actually made it through.
	// Luckily we can access the test runs directly in the mock client.
	tr := mock.TestRuns.modules[module.ID][0]
	if tr.Status != tfe.TestRunCanceled {
		t.Errorf("expected test run to have been cancelled but was %s", tr.Status)
	}

	if mock.TestRuns.cancels != 1 {
		t.Errorf("incorrect number of cancels, expected 1 but was %d", mock.TestRuns.cancels)
	}
}

// TestTest_DelayedCancel just makes sure that if we trigger the cancellation
// during the log reading stage then it still cancels properly.
func TestTest_DelayedCancel(t *testing.T) {

	streams, outputFn := terminal.StreamsForTesting(t)
	view := views.NewTest(arguments.ViewHuman, views.NewView(streams))

	colorize := mockColorize()
	colorize.Disable = true

	mock := NewMockClient()
	client := &tfe.Client{
		ConfigurationVersions: mock.ConfigurationVersions,
		Organizations:         mock.Organizations,
		RegistryModules:       mock.RegistryModules,
		TestRuns:              mock.TestRuns,
	}

	if _, err := client.Organizations.Create(context.Background(), tfe.OrganizationCreateOptions{
		Name: tfe.String("organisation"),
	}); err != nil {
		t.Fatalf("failed to create organisation: %v", err)
	}

	module, err := client.RegistryModules.Create(context.Background(), "organisation", tfe.RegistryModuleCreateOptions{
		Name:         tfe.String("name"),
		Provider:     tfe.String("provider"),
		RegistryName: "app.terraform.io",
		Namespace:    "organisation",
	})
	if err != nil {
		t.Fatalf("failed to create registry module: %v", err)
	}

	doneContext, done := context.WithCancel(context.Background())
	stopContext, stop := context.WithCancel(context.Background())

	mock.TestRuns.delayedCancel = stop

	// We're only going to be able to finish this if the cancellation calls
	// are done correctly.
	mock.TestRuns.targetCancels = 1

	runner := TestSuiteRunner{
		// Configuration data.
		ConfigDirectory:  "testdata/test-cancel",
		TestingDirectory: "tests",
		Config:           nil, // We don't need this for this test.
		Source:           "app.terraform.io/organisation/name/provider",

		// Cancellation controls, we won't be doing any cancellations in this
		// test.
		Stopped:      false,
		Cancelled:    false,
		StoppedCtx:   stopContext,
		CancelledCtx: context.Background(),

		// Test Options, empty for this test.
		GlobalVariables: nil,
		Verbose:         false,
		Filters:         nil,

		// Outputs
		Renderer: &jsonformat.Renderer{
			Streams:             streams,
			Colorize:            colorize,
			RunningInAutomation: false,
		},
		View:    view,
		Streams: streams,

		// Networking
		Services:       nil, // Don't need this when the client is overridden.
		clientOverride: client,
	}

	var diags tfdiags.Diagnostics
	go func() {
		defer done()
		_, diags = runner.Test()
	}()

	// Wait for finish!
	<-doneContext.Done()

	if len(diags) > 0 {
		t.Errorf("found diags and expected none: %s", diags.ErrWithWarnings())
	}

	output := outputFn(t)
	actual := output.All()
	expected := `main.tftest.hcl... in progress

Interrupt received.
Please wait for Terraform to exit or data loss may occur.
Gracefully shutting down...

  defaults... pass
  overrides... skip
main.tftest.hcl... tearing down
main.tftest.hcl... pass

Success! 1 passed, 0 failed, 1 skipped.
`

	if diff := cmp.Diff(expected, actual); len(diff) > 0 {
		t.Errorf("expected:\n%s\nactual:\n%s\ndiff:\n%s", expected, actual, diff)
	}

	// We want to make sure the cancel signal actually made it through.
	// Luckily we can access the test runs directly in the mock client.
	tr := mock.TestRuns.modules[module.ID][0]
	if tr.Status != tfe.TestRunCanceled {
		t.Errorf("expected test run to have been cancelled but was %s", tr.Status)
	}
}

func TestTest_ForceCancel(t *testing.T) {

	loader, close := configload.NewLoaderForTests(t)
	defer close()

	config, configDiags := loader.LoadConfigWithTests("testdata/test-force-cancel", "tests")
	if configDiags.HasErrors() {
		t.Fatalf("failed to load config: %v", configDiags.Error())
	}

	streams, outputFn := terminal.StreamsForTesting(t)
	view := views.NewTest(arguments.ViewHuman, views.NewView(streams))

	colorize := mockColorize()
	colorize.Disable = true

	mock := NewMockClient()
	client := &tfe.Client{
		ConfigurationVersions: mock.ConfigurationVersions,
		Organizations:         mock.Organizations,
		RegistryModules:       mock.RegistryModules,
		TestRuns:              mock.TestRuns,
	}

	if _, err := client.Organizations.Create(context.Background(), tfe.OrganizationCreateOptions{
		Name: tfe.String("organisation"),
	}); err != nil {
		t.Fatalf("failed to create organisation: %v", err)
	}

	module, err := client.RegistryModules.Create(context.Background(), "organisation", tfe.RegistryModuleCreateOptions{
		Name:         tfe.String("name"),
		Provider:     tfe.String("provider"),
		RegistryName: "app.terraform.io",
		Namespace:    "organisation",
	})
	if err != nil {
		t.Fatalf("failed to create registry module: %v", err)
	}

	doneContext, done := context.WithCancel(context.Background())
	stopContext, stop := context.WithCancel(context.Background())
	cancelContext, cancel := context.WithCancel(context.Background())

	runner := TestSuiteRunner{
		// Configuration data.
		ConfigDirectory:  "testdata/test-force-cancel",
		TestingDirectory: "tests",
		Config:           config,
		Source:           "app.terraform.io/organisation/name/provider",

		// Cancellation controls, we won't be doing any cancellations in this
		// test.
		Stopped:      false,
		Cancelled:    false,
		StoppedCtx:   stopContext,
		CancelledCtx: cancelContext,

		// Test Options, empty for this test.
		GlobalVariables: nil,
		Verbose:         false,
		Filters:         nil,

		// Outputs
		Renderer: &jsonformat.Renderer{
			Streams:             streams,
			Colorize:            colorize,
			RunningInAutomation: false,
		},
		View:    view,
		Streams: streams,

		// Networking
		Services:       nil, // Don't need this when the client is overridden.
		clientOverride: client,
	}

	// We're only going to be able to finish this if the cancellation calls
	// are done correctly.
	mock.TestRuns.targetCancels = 2

	var diags tfdiags.Diagnostics
	go func() {
		defer done()
		_, diags = runner.Test()
	}()

	stop()
	cancel()

	// Wait for finish!
	<-doneContext.Done()

	if len(diags) > 0 {
		t.Errorf("found diags and expected none: %s", diags.ErrWithWarnings())
	}

	output := outputFn(t)

	expectedErr := `Terraform was interrupted during test execution, and may not have performed
the expected cleanup operations.

Terraform was in the process of creating the following resources for
"overrides" from the module under test, and they may not have been destroyed:
  - time_sleep.wait_5_seconds
  - tfcoremock_simple_resource.resource
`
	actualErr := output.Stderr()
	if diff := cmp.Diff(expectedErr, actualErr); len(diff) > 0 {
		t.Errorf("expected:\n%s\nactual:\n%s\ndiff:\n%s", expectedErr, actualErr, diff)
	}

	actualOut := output.Stdout()
	expectedOut := `main.tftest.hcl... in progress
  defaults... pass

Interrupt received.
Please wait for Terraform to exit or data loss may occur.
Gracefully shutting down...


Two interrupts received. Exiting immediately. Note that data loss may have occurred.

  overrides... fail
╷
│ Error: Test interrupted
│
│ The test operation could not be completed due to an interrupt signal.
│ Please read the remaining diagnostics carefully for any sign of failed
│ state cleanup or dangling resources.
╵
╷
│ Error: Create time sleep error
│
│   with time_sleep.wait_5_seconds,
│   on main.tf line 7, in resource "time_sleep" "wait_5_seconds":
│    7: resource "time_sleep" "wait_5_seconds" {
│
│ Original Error: context canceled
╵
╷
│ Error: execution halted
│
╵
╷
│ Error: execution halted
│
╵
main.tftest.hcl... tearing down
main.tftest.hcl... fail

Failure! 1 passed, 1 failed.
`

	if diff := cmp.Diff(expectedOut, actualOut); len(diff) > 0 {
		t.Errorf("expected:\n%s\nactual:\n%s\ndiff:\n%s", expectedOut, actualOut, diff)
	}

	// We want to make sure the cancel signal actually made it through.
	// Luckily we can access the test runs directly in the mock client.
	tr := mock.TestRuns.modules[module.ID][0]
	if tr.Status != tfe.TestRunCanceled {
		t.Errorf("expected test run to have been cancelled but was %s", tr.Status)
	}

	if mock.TestRuns.cancels != 2 {
		t.Errorf("incorrect number of cancels, expected 2 but was %d", mock.TestRuns.cancels)
	}
}

func TestTest_LongRunningTest(t *testing.T) {

	streams, done := terminal.StreamsForTesting(t)
	view := views.NewTest(arguments.ViewHuman, views.NewView(streams))

	colorize := mockColorize()
	colorize.Disable = true

	mock := NewMockClient()
	client := &tfe.Client{
		ConfigurationVersions: mock.ConfigurationVersions,
		Organizations:         mock.Organizations,
		RegistryModules:       mock.RegistryModules,
		TestRuns:              mock.TestRuns,
	}

	if _, err := client.Organizations.Create(context.Background(), tfe.OrganizationCreateOptions{
		Name: tfe.String("organisation"),
	}); err != nil {
		t.Fatalf("failed to create organisation: %v", err)
	}

	if _, err := client.RegistryModules.Create(context.Background(), "organisation", tfe.RegistryModuleCreateOptions{
		Name:         tfe.String("name"),
		Provider:     tfe.String("provider"),
		RegistryName: "app.terraform.io",
		Namespace:    "organisation",
	}); err != nil {
		t.Fatalf("failed to create registry module: %v", err)
	}

	runner := TestSuiteRunner{
		// Configuration data.
		ConfigDirectory:  "testdata/test-long-running",
		TestingDirectory: "tests",
		Config:           nil, // We don't need this for this test.
		Source:           "app.terraform.io/organisation/name/provider",

		// Cancellation controls, we won't be doing any cancellations in this
		// test.
		Stopped:      false,
		Cancelled:    false,
		StoppedCtx:   context.Background(),
		CancelledCtx: context.Background(),

		// Test Options, empty for this test.
		GlobalVariables: nil,
		Verbose:         false,
		Filters:         nil,

		// Outputs
		Renderer: &jsonformat.Renderer{
			Streams:             streams,
			Colorize:            colorize,
			RunningInAutomation: false,
		},
		View:    view,
		Streams: streams,

		// Networking
		Services:       nil, // Don't need this when the client is overridden.
		clientOverride: client,
	}

	_, diags := runner.Test()
	if len(diags) > 0 {
		t.Errorf("found diags and expected none: %s", diags.ErrWithWarnings())
	}

	output := done(t)
	actual := output.All()

	// The long running test logs actually contain additional progress updates,
	// but this test should ignore them and just show the usual output.

	expected := `main.tftest.hcl... in progress
  just_go... pass
main.tftest.hcl... tearing down
main.tftest.hcl... pass

Success! 1 passed, 0 failed.
`

	if diff := cmp.Diff(expected, actual); len(diff) > 0 {
		t.Errorf("expected:\n%s\nactual:\n%s\ndiff:\n%s", expected, actual, diff)
	}
}

func TestTest_LongRunningTestJSON(t *testing.T) {

	streams, done := terminal.StreamsForTesting(t)
	view := views.NewTest(arguments.ViewHuman, views.NewView(streams))

	colorize := mockColorize()
	colorize.Disable = true

	mock := NewMockClient()
	client := &tfe.Client{
		ConfigurationVersions: mock.ConfigurationVersions,
		Organizations:         mock.Organizations,
		RegistryModules:       mock.RegistryModules,
		TestRuns:              mock.TestRuns,
	}

	if _, err := client.Organizations.Create(context.Background(), tfe.OrganizationCreateOptions{
		Name: tfe.String("organisation"),
	}); err != nil {
		t.Fatalf("failed to create organisation: %v", err)
	}

	if _, err := client.RegistryModules.Create(context.Background(), "organisation", tfe.RegistryModuleCreateOptions{
		Name:         tfe.String("name"),
		Provider:     tfe.String("provider"),
		RegistryName: "app.terraform.io",
		Namespace:    "organisation",
	}); err != nil {
		t.Fatalf("failed to create registry module: %v", err)
	}

	runner := TestSuiteRunner{
		// Configuration data.
		ConfigDirectory:  "testdata/test-long-running",
		TestingDirectory: "tests",
		Config:           nil, // We don't need this for this test.
		Source:           "app.terraform.io/organisation/name/provider",

		// Cancellation controls, we won't be doing any cancellations in this
		// test.
		Stopped:      false,
		Cancelled:    false,
		StoppedCtx:   context.Background(),
		CancelledCtx: context.Background(),

		// Test Options, empty for this test.
		GlobalVariables: nil,
		Verbose:         false,
		Filters:         nil,

		// Outputs
		Renderer: nil, // This should force the logs to render as JSON.
		View:     view,
		Streams:  streams,

		// Networking
		Services:       nil, // Don't need this when the client is overridden.
		clientOverride: client,
	}

	_, diags := runner.Test()
	if len(diags) > 0 {
		t.Errorf("found diags and expected none: %s", diags.ErrWithWarnings())
	}

	output := done(t)
	actual := output.All()

	// This test should still include the progress updates as we're doing the
	// JSON output.

	expected := `{"@level":"info","@message":"Terraform 1.7.0-dev","@module":"terraform.ui","@timestamp":"2023-09-28T14:57:09.175210+02:00","terraform":"1.7.0-dev","type":"version","ui":"1.2"}
{"@level":"info","@message":"Found 1 file and 1 run block","@module":"terraform.ui","@timestamp":"2023-09-28T14:57:09.189212+02:00","test_abstract":{"main.tftest.hcl":["just_go"]},"type":"test_abstract"}
{"@level":"info","@message":"main.tftest.hcl... in progress","@module":"terraform.ui","@testfile":"main.tftest.hcl","@timestamp":"2023-09-28T14:57:09.189386+02:00","test_file":{"path":"main.tftest.hcl","progress":"starting"},"type":"test_file"}
{"@level":"info","@message":"  \"just_go\"... in progress","@module":"terraform.ui","@testfile":"main.tftest.hcl","@testrun":"just_go","@timestamp":"2023-09-28T14:57:09.189429+02:00","test_run":{"path":"main.tftest.hcl","run":"just_go","progress":"starting","elapsed":0},"type":"test_run"}
{"@level":"info","@message":"  \"just_go\"... in progress","@module":"terraform.ui","@testfile":"main.tftest.hcl","@testrun":"just_go","@timestamp":"2023-09-28T14:57:11.341278+02:00","test_run":{"path":"main.tftest.hcl","run":"just_go","progress":"running","elapsed":2152},"type":"test_run"}
{"@level":"info","@message":"  \"just_go\"... in progress","@module":"terraform.ui","@testfile":"main.tftest.hcl","@testrun":"just_go","@timestamp":"2023-09-28T14:57:13.343465+02:00","test_run":{"path":"main.tftest.hcl","run":"just_go","progress":"running","elapsed":4154},"type":"test_run"}
{"@level":"info","@message":"  \"just_go\"... pass","@module":"terraform.ui","@testfile":"main.tftest.hcl","@testrun":"just_go","@timestamp":"2023-09-28T14:57:14.381552+02:00","test_run":{"path":"main.tftest.hcl","run":"just_go","progress":"complete","status":"pass"},"type":"test_run"}
{"@level":"info","@message":"main.tftest.hcl... tearing down","@module":"terraform.ui","@testfile":"main.tftest.hcl","@timestamp":"2023-09-28T14:57:14.381655+02:00","test_file":{"path":"main.tftest.hcl","progress":"teardown"},"type":"test_file"}
{"@level":"info","@message":"  \"just_go\"... tearing down","@module":"terraform.ui","@testfile":"main.tftest.hcl","@testrun":"just_go","@timestamp":"2023-09-28T14:57:14.381712+02:00","test_run":{"path":"main.tftest.hcl","run":"just_go","progress":"teardown","elapsed":0},"type":"test_run"}
{"@level":"info","@message":"  \"just_go\"... tearing down","@module":"terraform.ui","@testfile":"main.tftest.hcl","@testrun":"just_go","@timestamp":"2023-09-28T14:57:16.477705+02:00","test_run":{"path":"main.tftest.hcl","run":"just_go","progress":"teardown","elapsed":2096},"type":"test_run"}
{"@level":"info","@message":"main.tftest.hcl... pass","@module":"terraform.ui","@testfile":"main.tftest.hcl","@timestamp":"2023-09-28T14:57:17.517309+02:00","test_file":{"path":"main.tftest.hcl","progress":"complete","status":"pass"},"type":"test_file"}
{"@level":"info","@message":"Success! 1 passed, 0 failed.","@module":"terraform.ui","@timestamp":"2023-09-28T14:57:17.517494+02:00","test_summary":{"status":"pass","passed":1,"failed":0,"errored":0,"skipped":0},"type":"test_summary"}
`

	if diff := cmp.Diff(expected, actual); len(diff) > 0 {
		t.Errorf("expected:\n%s\nactual:\n%s\ndiff:\n%s", expected, actual, diff)
	}
}
