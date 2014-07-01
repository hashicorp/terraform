package command

import (
	"io/ioutil"
	"os"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/cli"
)

func TestPlan_destroy(t *testing.T) {
	originalState := &terraform.State{
		Resources: map[string]*terraform.ResourceState{
			"test_instance.foo": &terraform.ResourceState{
				ID:   "bar",
				Type: "test_instance",
			},
		},
	}

	outPath := testTempFile(t)
	statePath := testStateFile(t, originalState)

	p := testProvider()
	ui := new(cli.MockUi)
	c := &PlanCommand{
		TFConfig: testTFConfig(p),
		Ui:       ui,
	}

	args := []string{
		"-destroy",
		"-out", outPath,
		"-state", statePath,
		testFixturePath("plan"),
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	plan := testReadPlan(t, outPath)
	for _, r := range plan.Diff.Resources {
		if !r.Destroy {
			t.Fatalf("bad: %#v", r)
		}
	}
}
func TestPlan_noState(t *testing.T) {
	p := testProvider()
	ui := new(cli.MockUi)
	c := &PlanCommand{
		TFConfig: testTFConfig(p),
		Ui:       ui,
	}

	args := []string{
		testFixturePath("plan"),
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	// Verify that refresh was called
	if !p.RefreshCalled {
		t.Fatal("refresh should be called")
	}

	// Verify that the provider was called with the existing state
	expectedState := &terraform.ResourceState{
		Type: "test_instance",
	}
	if !reflect.DeepEqual(p.DiffState, expectedState) {
		t.Fatalf("bad: %#v", p.DiffState)
	}
}

func TestPlan_outPath(t *testing.T) {
	tf, err := ioutil.TempFile("", "tf")
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	outPath := tf.Name()
	os.Remove(tf.Name())

	p := testProvider()
	ui := new(cli.MockUi)
	c := &PlanCommand{
		TFConfig: testTFConfig(p),
		Ui:       ui,
	}

	p.DiffReturn = &terraform.ResourceDiff{
		Destroy: true,
	}

	args := []string{
		"-out", outPath,
		testFixturePath("plan"),
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	f, err := os.Open(outPath)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	defer f.Close()

	if _, err := terraform.ReadPlan(f); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestPlan_refresh(t *testing.T) {
	p := testProvider()
	ui := new(cli.MockUi)
	c := &PlanCommand{
		TFConfig: testTFConfig(p),
		Ui:       ui,
	}

	args := []string{
		"-refresh=false",
		testFixturePath("plan"),
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	if p.RefreshCalled {
		t.Fatal("refresh should not be called")
	}
}

func TestPlan_state(t *testing.T) {
	// Write out some prior state
	tf, err := ioutil.TempFile("", "tf")
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	statePath := tf.Name()
	defer os.Remove(tf.Name())

	originalState := &terraform.State{
		Resources: map[string]*terraform.ResourceState{
			"test_instance.foo": &terraform.ResourceState{
				ID:   "bar",
				Type: "test_instance",
			},
		},
	}

	err = terraform.WriteState(originalState, tf)
	tf.Close()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	p := testProvider()
	ui := new(cli.MockUi)
	c := &PlanCommand{
		TFConfig: testTFConfig(p),
		Ui:       ui,
	}

	args := []string{
		"-state", statePath,
		testFixturePath("plan"),
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	// Verify that the provider was called with the existing state
	expectedState := originalState.Resources["test_instance.foo"]
	if !reflect.DeepEqual(p.DiffState, expectedState) {
		t.Fatalf("bad: %#v", p.DiffState)
	}
}
