package command

import (
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/cli"
)

func TestApply(t *testing.T) {
	statePath := testTempFile(t)

	p := testProvider()
	ui := new(cli.MockUi)
	c := &ApplyCommand{
		TFConfig: testTFConfig(p),
		Ui:       ui,
	}

	args := []string{
		"-init",
		statePath,
		testFixturePath("apply"),
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

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
}

func TestApply_plan(t *testing.T) {
	planPath := testPlanFile(t, new(terraform.Plan))
	statePath := testTempFile(t)

	p := testProvider()
	ui := new(cli.MockUi)
	c := &ApplyCommand{
		TFConfig: testTFConfig(p),
		Ui:       ui,
	}

	args := []string{
		statePath,
		planPath,
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

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
}

func TestApply_shutdown(t *testing.T) {
	stopped := false
	stopCh := make(chan struct{})
	stopReplyCh := make(chan struct{})

	statePath := testTempFile(t)

	p := testProvider()
	shutdownCh := make(chan struct{})
	ui := new(cli.MockUi)
	c := &ApplyCommand{
		ShutdownCh: shutdownCh,
		TFConfig:   testTFConfig(p),
		Ui:         ui,
	}

	p.DiffFn = func(
		*terraform.ResourceState,
		*terraform.ResourceConfig) (*terraform.ResourceDiff, error) {
		return &terraform.ResourceDiff{
			Attributes: map[string]*terraform.ResourceAttrDiff{
				"ami": &terraform.ResourceAttrDiff{
					New: "bar",
				},
			},
		}, nil
	}
	p.ApplyFn = func(
		*terraform.ResourceState,
		*terraform.ResourceDiff) (*terraform.ResourceState, error) {
		if !stopped {
			stopped = true
			close(stopCh)
			<-stopReplyCh
		}

		return &terraform.ResourceState{
			ID: "foo",
			Attributes: map[string]string{
				"ami": "2",
			},
		}, nil
	}

	go func() {
		<-stopCh
		shutdownCh <- struct{}{}

		// This is really dirty, but we have no other way to assure that
		// tf.Stop() has been called. This doesn't assure it either, but
		// it makes it much more certain.
		time.Sleep(50 * time.Millisecond)

		close(stopReplyCh)
	}()

	args := []string{
		"-init",
		statePath,
		testFixturePath("apply-shutdown"),
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

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

	if len(state.Resources) != 1 {
		t.Fatalf("bad: %d", len(state.Resources))
	}
}

func TestApply_state(t *testing.T) {
	originalState := &terraform.State{
		Resources: map[string]*terraform.ResourceState{
			"test_instance.foo": &terraform.ResourceState{
				ID:   "bar",
				Type: "test_instance",
			},
		},
	}

	statePath := testStateFile(t, originalState)

	p := testProvider()
	p.DiffReturn = &terraform.ResourceDiff{
		Attributes: map[string]*terraform.ResourceAttrDiff{
			"ami": &terraform.ResourceAttrDiff{
				New: "bar",
			},
		},
	}

	ui := new(cli.MockUi)
	c := &ApplyCommand{
		TFConfig: testTFConfig(p),
		Ui:       ui,
	}

	// Run the apply command pointing to our existing state
	args := []string{
		statePath,
		testFixturePath("apply"),
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	// Verify that the provider was called with the existing state
	expectedState := originalState.Resources["test_instance.foo"]
	if !reflect.DeepEqual(p.DiffState, expectedState) {
		t.Fatalf("bad: %#v", p.DiffState)
	}

	if !reflect.DeepEqual(p.ApplyState, expectedState) {
		t.Fatalf("bad: %#v", p.ApplyState)
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
}

func TestApply_stateNoExist(t *testing.T) {
	p := testProvider()
	ui := new(cli.MockUi)
	c := &ApplyCommand{
		TFConfig: testTFConfig(p),
		Ui:       ui,
	}

	args := []string{
		"idontexist.tfstate",
		testFixturePath("apply"),
	}
	if code := c.Run(args); code != 1 {
		t.Fatalf("bad: \n%s", ui.OutputWriter.String())
	}
}
