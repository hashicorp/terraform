package command

import (
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/cli"
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
	ui := new(cli.MockUi)
	c := &ApplyCommand{
		Destroy: true,
		Meta: Meta{
			ContextOpts: testCtxConfig(p),
			Ui:          ui,
		},
	}

	// Run the apply command pointing to our existing state
	args := []string{
		"-force",
		"-state", statePath,
		testFixturePath("apply"),
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

func TestApply_destroyPlan(t *testing.T) {
	planPath := testPlanFile(t, &terraform.Plan{
		Module: testModule(t, "apply"),
	})

	p := testProvider()
	ui := new(cli.MockUi)
	c := &ApplyCommand{
		Destroy: true,
		Meta: Meta{
			ContextOpts: testCtxConfig(p),
			Ui:          ui,
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
	ui := new(cli.MockUi)
	c := &ApplyCommand{
		Destroy: true,
		Meta: Meta{
			ContextOpts: testCtxConfig(p),
			Ui:          ui,
		},
	}

	// Run the apply command pointing to our existing state
	args := []string{
		"-force",
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
		t.Fatalf("bad:\n\nactual:\n%s\n\nexpected:\nb%s", actualStr, expectedStr)
	}
}

const testApplyDestroyStr = `
<no state>
`
