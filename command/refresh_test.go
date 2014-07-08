package command

import (
	"io/ioutil"
	"os"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/cli"
)

func TestRefresh(t *testing.T) {
	state := &terraform.State{
		Resources: map[string]*terraform.ResourceState{
			"test_instance.foo": &terraform.ResourceState{
				ID:   "bar",
				Type: "test_instance",
			},
		},
	}
	statePath := testStateFile(t, state)

	p := testProvider()
	ui := new(cli.MockUi)
	c := &RefreshCommand{
		ContextOpts: testCtxConfig(p),
		Ui:          ui,
	}

	p.RefreshFn = nil
	p.RefreshReturn = &terraform.ResourceState{ID: "yes"}

	args := []string{
		statePath,
		testFixturePath("refresh"),
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	if !p.RefreshCalled {
		t.Fatal("refresh should be called")
	}

	f, err := os.Open(statePath)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	newState, err := terraform.ReadState(f)
	f.Close()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := newState.Resources["test_instance.foo"]
	expected := p.RefreshReturn
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestRefresh_outPath(t *testing.T) {
	state := &terraform.State{
		Resources: map[string]*terraform.ResourceState{
			"test_instance.foo": &terraform.ResourceState{
				ID:   "bar",
				Type: "test_instance",
			},
		},
	}
	statePath := testStateFile(t, state)

	// Output path
	outf, err := ioutil.TempFile("", "tf")
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	outPath := outf.Name()
	outf.Close()
	os.Remove(outPath)

	p := testProvider()
	ui := new(cli.MockUi)
	c := &RefreshCommand{
		ContextOpts: testCtxConfig(p),
		Ui:          ui,
	}

	p.RefreshFn = nil
	p.RefreshReturn = &terraform.ResourceState{ID: "yes"}

	args := []string{
		"-out", outPath,
		statePath,
		testFixturePath("refresh"),
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	f, err := os.Open(statePath)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	newState, err := terraform.ReadState(f)
	f.Close()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !reflect.DeepEqual(newState, state) {
		t.Fatalf("bad: %#v", newState)
	}

	f, err = os.Open(outPath)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	newState, err = terraform.ReadState(f)
	f.Close()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := newState.Resources["test_instance.foo"]
	expected := p.RefreshReturn
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad: %#v", actual)
	}
}
