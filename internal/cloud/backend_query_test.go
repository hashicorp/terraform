// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cloud

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/cli"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/backend/backendrun"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/clistate"
	"github.com/hashicorp/terraform/internal/command/jsonformat"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/depsfile"
	"github.com/hashicorp/terraform/internal/initwd"
	"github.com/hashicorp/terraform/internal/states/statemgr"
	"github.com/hashicorp/terraform/internal/terminal"
)

func testOperationQuery(t *testing.T, configDir string) (*backendrun.Operation, func(), func(*testing.T) *terminal.TestOutput) {
	t.Helper()

	return testOperationQueryWithTimeout(t, configDir, 0)
}

func testOperationQueryWithTimeout(t *testing.T, configDir string, timeout time.Duration) (*backendrun.Operation, func(), func(*testing.T) *terminal.TestOutput) {
	t.Helper()

	_, configLoader, configCleanup := initwd.MustLoadConfigForTests(t, configDir, "tests")

	streams, done := terminal.StreamsForTesting(t)
	view := views.NewView(streams)
	stateLockerView := views.NewStateLocker(arguments.ViewHuman, view)
	operationView := views.NewQueryOperation(arguments.ViewHuman, false, view)

	// Many of our tests use an overridden "null" provider that's just in-memory
	// inside the test process, not a separate plugin on disk.
	depLocks := depsfile.NewLocks()
	depLocks.SetProviderOverridden(addrs.MustParseProviderSourceString("registry.terraform.io/hashicorp/null"))

	return &backendrun.Operation{
		ConfigDir:       configDir,
		ConfigLoader:    configLoader,
		StateLocker:     clistate.NewLocker(timeout, stateLockerView),
		Type:            backendrun.OperationTypePlan,
		View:            operationView,
		DependencyLocks: depLocks,
		Query:           true,
	}, configCleanup, done
}

func TestCloud_queryBasic(t *testing.T) {
	b, bCleanup := testBackendWithName(t)
	defer bCleanup()

	op, configCleanup, done := testOperationQuery(t, "./testdata/query")
	defer configCleanup()
	defer done(t)

	op.Workspace = testBackendSingleWorkspaceName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Result != backendrun.OperationSuccess {
		t.Fatalf("operation failed: %s", b.CLI.(*cli.MockUi).ErrorWriter.String())
	}

	output := b.CLI.(*cli.MockUi).OutputWriter.String()
	if !strings.Contains(output, "Running query in HCP Terraform") {
		t.Fatalf("expected HCP Terraform header in output: %s", output)
	}
	if !strings.Contains(output, "list.concept_pet.pets   id=") {
		t.Fatalf("expected query results in output: %s", output)
	}

	stateMgr, _ := b.StateMgr(testBackendSingleWorkspaceName)
	// An error suggests that the state was not unlocked after the operation finished
	if _, err := stateMgr.Lock(statemgr.NewLockInfo()); err != nil {
		t.Fatalf("unexpected error locking state after successful plan: %s", err.Error())
	}
}

func TestCloud_queryJSONBasic(t *testing.T) {
	b, bCleanup := testBackendWithName(t)
	defer bCleanup()

	stream, close := terminal.StreamsForTesting(t)

	b.renderer = &jsonformat.Renderer{
		Streams:  stream,
		Colorize: mockColorize(),
	}

	op, configCleanup, done := testOperationQuery(t, "./testdata/query-json-basic")
	defer configCleanup()
	defer done(t)

	op.Workspace = testBackendSingleWorkspaceName

	mockSROWorkspace(t, b, op.Workspace)

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Result != backendrun.OperationSuccess {
		t.Fatalf("operation failed: %s", b.CLI.(*cli.MockUi).ErrorWriter.String())
	}

	outp := close(t)
	gotOut := outp.Stdout()

	expectedOut := `list.concept_pet.pets   id=large-roughy,legs=2      This is a large-roughy
list.concept_pet.pets   id=able-werewolf,legs=5     This is a able-werewolf
list.concept_pet.pets   id=complete-gannet,legs=6   This is a complete-gannet
list.concept_pet.pets   id=charming-beagle,legs=3   This is a charming-beagle
list.concept_pet.pets   id=legal-lamprey,legs=2     This is a legal-lamprey

`
	if diff := cmp.Diff(expectedOut, gotOut); diff != "" {
		t.Fatalf("expected query results output to be %s, got %s: diff: %s", expectedOut, gotOut, diff)
	}

	stateMgr, _ := b.StateMgr(testBackendSingleWorkspaceName)
	// An error suggests that the state was not unlocked after the operation finished
	if _, err := stateMgr.Lock(statemgr.NewLockInfo()); err != nil {
		t.Fatalf("unexpected error locking state after successful plan: %s", err.Error())
	}
}

func TestCloud_queryJSONWithDiags(t *testing.T) {
	b, bCleanup := testBackendWithName(t)
	defer bCleanup()

	stream, close := terminal.StreamsForTesting(t)

	b.renderer = &jsonformat.Renderer{
		Streams:  stream,
		Colorize: mockColorize(),
	}

	op, configCleanup, done := testOperationQuery(t, "./testdata/query-json-diag")
	defer configCleanup()
	defer done(t)

	op.Workspace = testBackendSingleWorkspaceName

	mockSROWorkspace(t, b, op.Workspace)

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Result != backendrun.OperationSuccess {
		t.Fatalf("operation failed: %s", b.CLI.(*cli.MockUi).ErrorWriter.String())
	}

	testOut := close(t)
	output := testOut.Stdout()

	// Warning diagnostic message
	testString := "Warning: Something went wrong"
	if !strings.Contains(output, testString) {
		t.Fatalf("Expected %q to contain %q but it did not", output, testString)
	}
}
