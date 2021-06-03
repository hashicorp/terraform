package local

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/clistate"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/configs/configload"
	"github.com/hashicorp/terraform/internal/initwd"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/plans/planfile"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/states/statefile"
	"github.com/hashicorp/terraform/internal/terminal"
	"github.com/zclconf/go-cty/cty"
)

func TestLocalContext(t *testing.T) {
	configDir := "./testdata/empty"
	b, cleanup := TestLocal(t)
	defer cleanup()

	_, configLoader, configCleanup := initwd.MustLoadConfigForTests(t, configDir)
	defer configCleanup()

	streams, _ := terminal.StreamsForTesting(t)
	view := views.NewView(streams)
	stateLocker := clistate.NewLocker(0, views.NewStateLocker(arguments.ViewHuman, view))

	op := &backend.Operation{
		ConfigDir:    configDir,
		ConfigLoader: configLoader,
		Workspace:    backend.DefaultStateName,
		StateLocker:  stateLocker,
	}

	_, _, diags := b.Context(op)
	if diags.HasErrors() {
		t.Fatalf("unexpected error: %s", diags.Err().Error())
	}

	// Context() retains a lock on success
	assertBackendStateLocked(t, b)
}

func TestLocalContext_error(t *testing.T) {
	configDir := "./testdata/apply"
	b, cleanup := TestLocal(t)
	defer cleanup()

	_, configLoader, configCleanup := initwd.MustLoadConfigForTests(t, configDir)
	defer configCleanup()

	streams, _ := terminal.StreamsForTesting(t)
	view := views.NewView(streams)
	stateLocker := clistate.NewLocker(0, views.NewStateLocker(arguments.ViewHuman, view))

	op := &backend.Operation{
		ConfigDir:    configDir,
		ConfigLoader: configLoader,
		Workspace:    backend.DefaultStateName,
		StateLocker:  stateLocker,
	}

	_, _, diags := b.Context(op)
	if !diags.HasErrors() {
		t.Fatal("unexpected success")
	}

	// Context() unlocks the state on failure
	assertBackendStateUnlocked(t, b)
}

func TestLocalContext_stalePlan(t *testing.T) {
	configDir := "./testdata/apply"
	b, cleanup := TestLocal(t)
	defer cleanup()

	_, configLoader, configCleanup := initwd.MustLoadConfigForTests(t, configDir)
	defer configCleanup()

	// Write an empty state file with serial 3
	sf, err := os.Create(b.StatePath)
	if err != nil {
		t.Fatalf("unexpected error creating state file %s: %s", b.StatePath, err)
	}
	if err := statefile.Write(statefile.New(states.NewState(), "boop", 3), sf); err != nil {
		t.Fatalf("unexpected error writing state file: %s", err)
	}

	// Refresh the state
	sm, err := b.StateMgr("")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if err := sm.RefreshState(); err != nil {
		t.Fatalf("unexpected error refreshing state: %s", err)
	}

	// Create a minimal plan which also has state file serial 2, so is stale
	backendConfig := cty.ObjectVal(map[string]cty.Value{
		"path":          cty.NullVal(cty.String),
		"workspace_dir": cty.NullVal(cty.String),
	})
	backendConfigRaw, err := plans.NewDynamicValue(backendConfig, backendConfig.Type())
	if err != nil {
		t.Fatal(err)
	}
	plan := &plans.Plan{
		UIMode:  plans.NormalMode,
		Changes: plans.NewChanges(),
		Backend: plans.Backend{
			Type:   "local",
			Config: backendConfigRaw,
		},
		PrevRunState: states.NewState(),
		PriorState:   states.NewState(),
	}
	prevStateFile := statefile.New(plan.PrevRunState, "boop", 1)
	stateFile := statefile.New(plan.PriorState, "boop", 2)

	// Roundtrip through serialization as expected by the operation
	outDir := testTempDir(t)
	defer os.RemoveAll(outDir)
	planPath := filepath.Join(outDir, "plan.tfplan")
	if err := planfile.Create(planPath, configload.NewEmptySnapshot(), prevStateFile, stateFile, plan); err != nil {
		t.Fatalf("unexpected error writing planfile: %s", err)
	}
	planFile, err := planfile.Open(planPath)
	if err != nil {
		t.Fatalf("unexpected error reading planfile: %s", err)
	}

	streams, _ := terminal.StreamsForTesting(t)
	view := views.NewView(streams)
	stateLocker := clistate.NewLocker(0, views.NewStateLocker(arguments.ViewHuman, view))

	op := &backend.Operation{
		ConfigDir:    configDir,
		ConfigLoader: configLoader,
		PlanFile:     planFile,
		Workspace:    backend.DefaultStateName,
		StateLocker:  stateLocker,
	}

	_, _, diags := b.Context(op)
	if !diags.HasErrors() {
		t.Fatal("unexpected success")
	}

	// Context() unlocks the state on failure
	assertBackendStateUnlocked(t, b)
}
