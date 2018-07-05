package command

import (
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/cli"
	"github.com/zclconf/go-cty/cty"
)

func TestApply_destroy(t *testing.T) {
	originalState := &terraform.State{
		Modules: []*terraform.ModuleState{
			&terraform.ModuleState{
				Path: []string{"root"},
				Resources: map[string]*terraform.ResourceState{
					"test_instance.foo": &terraform.ResourceState{
						Type: "test_instance",
						Primary: &terraform.InstanceState{
							ID: "bar",
						},
					},
				},
			},
		},
	}

	statePath := testStateFile(t, originalState)

	p := testProvider()
	p.GetSchemaReturn = applyFixtureSchema()

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

	state, err := terraform.ReadState(f)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if state == nil {
		t.Fatal("state should not be nil")
	}

	actualStr := strings.TrimSpace(state.String())
	expectedStr := strings.TrimSpace(testApplyDestroyStr)
	if actualStr != expectedStr {
		t.Fatalf("bad:\n\n%s\n\n%s", actualStr, expectedStr)
	}

	// Should have a backup file
	f, err = os.Open(statePath + DefaultBackupExtension)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	backupState, err := terraform.ReadState(f)
	f.Close()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actualStr = strings.TrimSpace(backupState.String())
	expectedStr = strings.TrimSpace(originalState.String())
	if actualStr != expectedStr {
		t.Fatalf("bad:\n\n%s\n\n%s", actualStr, expectedStr)
	}
}

func TestApply_destroyLockedState(t *testing.T) {
	originalState := &terraform.State{
		Modules: []*terraform.ModuleState{
			&terraform.ModuleState{
				Path: []string{"root"},
				Resources: map[string]*terraform.ResourceState{
					"test_instance.foo": &terraform.ResourceState{
						Type: "test_instance",
						Primary: &terraform.InstanceState{
							ID: "bar",
						},
					},
				},
			},
		},
	}

	statePath := testStateFile(t, originalState)

	unlock, err := testLockState("./testdata", statePath)
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
	planPath := testPlanFile(t, &terraform.Plan{
		Config: testModule(t, "apply"),
	})

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
	originalState := &terraform.State{
		Modules: []*terraform.ModuleState{
			&terraform.ModuleState{
				Path: []string{"root"},
				Resources: map[string]*terraform.ResourceState{
					"test_instance.foo": &terraform.ResourceState{
						Type: "test_instance",
						Primary: &terraform.InstanceState{
							ID: "i-ab123",
						},
					},
					"test_load_balancer.foo": &terraform.ResourceState{
						Type: "test_load_balancer",
						Primary: &terraform.InstanceState{
							ID: "lb-abc123",
						},
					},
				},
			},
		},
	}

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
					"instances": {Type: cty.List(cty.String), Optional: true},
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

	state, err := terraform.ReadState(f)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if state == nil {
		t.Fatal("state should not be nil")
	}

	actualStr := strings.TrimSpace(state.String())
	expectedStr := strings.TrimSpace(testApplyDestroyStr)
	if actualStr != expectedStr {
		t.Fatalf("bad:\n\n%s\n\nexpected:\n\n%s", actualStr, expectedStr)
	}

	// Should have a backup file
	f, err = os.Open(statePath + DefaultBackupExtension)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	backupState, err := terraform.ReadState(f)
	f.Close()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actualStr = strings.TrimSpace(backupState.String())
	expectedStr = strings.TrimSpace(originalState.String())
	if actualStr != expectedStr {
		t.Fatalf("bad:\n\nactual:\n%s\n\nexpected:\nb%s", actualStr, expectedStr)
	}
}

const testApplyDestroyStr = `
<no state>
`
