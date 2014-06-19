package command

import (
	"io/ioutil"
	"os"
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
