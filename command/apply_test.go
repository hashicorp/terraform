package command

import (
	"io/ioutil"
	"os"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/cli"
)

func TestApply(t *testing.T) {
	tf, err := ioutil.TempFile("", "tf")
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	statePath := tf.Name()
	tf.Close()
	os.Remove(tf.Name())

	p := testProvider()
	ui := new(cli.MockUi)
	c := &ApplyCommand{
		TFConfig: testTFConfig(p),
		Ui:       ui,
	}

	args := []string{
		"-init",
		"-state", statePath,
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

func TestApply_noState(t *testing.T) {
	p := testProvider()
	ui := new(cli.MockUi)
	c := &ApplyCommand{
		TFConfig: testTFConfig(p),
		Ui:       ui,
	}

	args := []string{
		"-state=",
		testFixturePath("apply"),
	}
	if code := c.Run(args); code != 1 {
		t.Fatalf("bad: \n%s", ui.OutputWriter.String())
	}
}

func TestApply_state(t *testing.T) {
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
	c := &ApplyCommand{
		TFConfig: testTFConfig(p),
		Ui:       ui,
	}

	// Run the apply command pointing to our existing state
	args := []string{
		"-state", statePath,
		testFixturePath("apply"),
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	// Verify that the provider was called with the existing state
	expectedState := originalState.Resources["test_instance.foo"]
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
		"-state=idontexist.tfstate",
		testFixturePath("apply"),
	}
	// TODO
	return
	if code := c.Run(args); code != 1 {
		t.Fatalf("bad: \n%s", ui.OutputWriter.String())
	}
}
