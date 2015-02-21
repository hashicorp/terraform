package state

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/hashicorp/terraform/terraform"
)

func TestLocalState(t *testing.T) {
	f, err := ioutil.TempFile("", "tf")
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	defer os.Remove(f.Name())

	err = terraform.WriteState(TestStateInitial, f)
	f.Close()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	TestState(t, &LocalState{
		Path: f.Name(),
	})
}

func TestLocalState_impl(t *testing.T) {
	var _ StateReader = new(LocalState)
	var _ StateWriter = new(LocalState)
	var _ StatePersister = new(LocalState)
	var _ StateRefresher = new(LocalState)
}
