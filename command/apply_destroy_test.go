package command

import (
	"os"
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/mitchellh/cli"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/providers"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/states/statefile"
	"github.com/hashicorp/terraform/terraform"
)

func TestApply_destroy(t *testing.T) {
	originalState := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "foo",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"bar"}`),
				Status:    states.ObjectReady,
			},
			addrs.ProviderConfig{Type: "test"}.Absolute(addrs.RootModuleInstance),
		)
	})
	statePath := testStateFile(t, originalState)

	p := testProvider()
	p.GetSchemaReturn = &terraform.ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"test_instance": {
				Attributes: map[string]*configschema.Attribute{
					"id":  {Type: cty.String, Computed: true},
					"ami": {Type: cty.String, Optional: true},
				},
			},
		},
	}

	ui := new(cli.MockUi)
	c := &ApplyCommand{
		Destroy: true,
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	// Run the apply command pointing to our existing state
	args := []string{
		"-auto-approve",
		"-state", statePath,
		testFixturePath("apply"),
	}
	if code := c.Run(args); code != 0 {
		t.Log(ui.OutputWriter.String())
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	// Verify a new state exists
	if _, err := os.Stat(statePath); err != nil {
		t.Fatalf("err: %s", err)
	}

	f, err := os.Open(statePath)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	defer f.Close()

	stateFile, err := statefile.Read(f)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if stateFile.State == nil {
		t.Fatal("state should not be nil")
	}

	actualStr := strings.TrimSpace(stateFile.State.String())
	expectedStr := strings.TrimSpace(testApplyDestroyStr)
	if actualStr != expectedStr {
		t.Fatalf("bad:\n\n%s\n\n%s", actualStr, expectedStr)
	}

	// Should have a backup file
	f, err = os.Open(statePath + DefaultBackupExtension)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	backupStateFile, err := statefile.Read(f)
	f.Close()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actualStr = strings.TrimSpace(backupStateFile.State.String())
	expectedStr = strings.TrimSpace(originalState.String())
	if actualStr != expectedStr {
		t.Fatalf("bad:\n\n%s\n\n%s", actualStr, expectedStr)
	}
}

func TestApply_destroyLockedState(t *testing.T) {
	originalState := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "foo",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"bar"}`),
				Status:    states.ObjectReady,
			},
			addrs.ProviderConfig{Type: "test"}.Absolute(addrs.RootModuleInstance),
		)
	})
	statePath := testStateFile(t, originalState)

	unlock, err := testLockState(testDataDir, statePath)
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()

	p := testProvider()
	ui := new(cli.MockUi)
	c := &ApplyCommand{
		Destroy: true,
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	// Run the apply command pointing to our existing state
	args := []string{
		"-auto-approve",
		"-state", statePath,
		testFixturePath("apply"),
	}

	if code := c.Run(args); code == 0 {
		t.Fatal("expected error")
	}

	output := ui.ErrorWriter.String()
	if !strings.Contains(output, "lock") {
		t.Fatal("command output does not look like a lock error:", output)
	}
}

func TestApply_destroyPlan(t *testing.T) {
	planPath := testPlanFileNoop(t)

	p := testProvider()
	ui := new(cli.MockUi)
	c := &ApplyCommand{
		Destroy: true,
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	// Run the apply command pointing to our existing state
	args := []string{
		planPath,
	}
	if code := c.Run(args); code != 1 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}
}

func TestApply_destroyTargeted(t *testing.T) {
	originalState := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "foo",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"i-ab123"}`),
				Status:    states.ObjectReady,
			},
			addrs.ProviderConfig{Type: "test"}.Absolute(addrs.RootModuleInstance),
		)
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_load_balancer",
				Name: "foo",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"i-abc123"}`),
				Status:    states.ObjectReady,
			},
			addrs.ProviderConfig{Type: "test"}.Absolute(addrs.RootModuleInstance),
		)
	})
	statePath := testStateFile(t, originalState)

	p := testProvider()
	p.GetSchemaReturn = &terraform.ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"test_instance": {
				Attributes: map[string]*configschema.Attribute{
					"id": {Type: cty.String, Computed: true},
				},
			},
			"test_load_balancer": {
				Attributes: map[string]*configschema.Attribute{
					"id":        {Type: cty.String, Computed: true},
					"instances": {Type: cty.List(cty.String), Optional: true},
				},
			},
		},
	}
	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
		return providers.PlanResourceChangeResponse{
			PlannedState: req.ProposedNewState,
		}
	}

	ui := new(cli.MockUi)
	c := &ApplyCommand{
		Destroy: true,
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	// Run the apply command pointing to our existing state
	args := []string{
		"-auto-approve",
		"-target", "test_instance.foo",
		"-state", statePath,
		testFixturePath("apply-destroy-targeted"),
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	// Verify a new state exists
	if _, err := os.Stat(statePath); err != nil {
		t.Fatalf("err: %s", err)
	}

	f, err := os.Open(statePath)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	defer f.Close()

	stateFile, err := statefile.Read(f)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if stateFile == nil || stateFile.State == nil {
		t.Fatal("state should not be nil")
	}

	spew.Config.DisableMethods = true
	if !stateFile.State.Empty() {
		t.Fatalf("unexpected final state\ngot: %s\nwant: empty state", spew.Sdump(stateFile.State))
	}

	// Should have a backup file
	f, err = os.Open(statePath + DefaultBackupExtension)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	backupStateFile, err := statefile.Read(f)
	f.Close()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actualStr := strings.TrimSpace(backupStateFile.State.String())
	expectedStr := strings.TrimSpace(originalState.String())
	if actualStr != expectedStr {
		t.Fatalf("bad:\n\nactual:\n%s\n\nexpected:\nb%s", actualStr, expectedStr)
	}
}

const testApplyDestroyStr = `
<no state>
`
