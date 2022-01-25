package cloud

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/clistate"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/initwd"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/states/statemgr"
	"github.com/hashicorp/terraform/internal/terminal"
	"github.com/mitchellh/cli"
)

func testOperationRefresh(t *testing.T, configDir string) (*backend.Operation, func(), func(*testing.T) *terminal.TestOutput) {
	t.Helper()

	return testOperationRefreshWithTimeout(t, configDir, 0)
}

func testOperationRefreshWithTimeout(t *testing.T, configDir string, timeout time.Duration) (*backend.Operation, func(), func(*testing.T) *terminal.TestOutput) {
	t.Helper()

	_, configLoader, configCleanup := initwd.MustLoadConfigForTests(t, configDir)

	streams, done := terminal.StreamsForTesting(t)
	view := views.NewView(streams)
	stateLockerView := views.NewStateLocker(arguments.ViewHuman, view)
	operationView := views.NewOperation(arguments.ViewHuman, false, view)

	return &backend.Operation{
		ConfigDir:    configDir,
		ConfigLoader: configLoader,
		PlanRefresh:  true,
		StateLocker:  clistate.NewLocker(timeout, stateLockerView),
		Type:         backend.OperationTypeRefresh,
		View:         operationView,
	}, configCleanup, done
}

func TestCloud_refreshBasicActuallyRunsApplyRefresh(t *testing.T) {
	b, bCleanup := testBackendWithName(t)
	defer bCleanup()

	op, configCleanup, done := testOperationRefresh(t, "./testdata/refresh")
	defer configCleanup()
	defer done(t)

	op.UIOut = b.CLI
	b.CLIColor = b.cliColorize()
	op.PlanMode = plans.RefreshOnlyMode
	op.Workspace = testBackendSingleWorkspaceName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Result != backend.OperationSuccess {
		t.Fatalf("operation failed: %s", b.CLI.(*cli.MockUi).ErrorWriter.String())
	}

	output := b.CLI.(*cli.MockUi).OutputWriter.String()
	if !strings.Contains(output, "Proceeding with 'terraform apply -refresh-only -auto-approve'") {
		t.Fatalf("expected TFC header in output: %s", output)
	}

	stateMgr, _ := b.StateMgr(testBackendSingleWorkspaceName)
	// An error suggests that the state was not unlocked after apply
	if _, err := stateMgr.Lock(statemgr.NewLockInfo()); err != nil {
		t.Fatalf("unexpected error locking state after apply: %s", err.Error())
	}
}
