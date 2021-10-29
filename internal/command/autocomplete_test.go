package command

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/states/statefile"
	"io/ioutil"
	"os"
	"reflect"
	"testing"

	"github.com/mitchellh/cli"
	"github.com/posener/complete"
)

func TestMetaCompletePredictWorkspaceName(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	os.MkdirAll(td, 0755)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	// make sure a vars file doesn't interfere
	err := ioutil.WriteFile(DefaultVarsFilename, nil, 0644)
	if err != nil {
		t.Fatal(err)
	}

	ui := new(cli.MockUi)
	meta := &Meta{Ui: ui}

	predictor := meta.completePredictWorkspaceName()

	got := predictor.Predict(complete.Args{
		Last: "",
	})
	want := []string{"default"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, want)
	}
}

func TestMetaCompletePredictResourceName(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	os.MkdirAll(td, 0755)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	// Create a new state file
	state := states.NewState()
	rootModule := state.RootModule()
	if rootModule == nil {
		t.Errorf("root module is nil; want valid object")
	}
	rootModule.SetResourceInstanceCurrent(
		addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "test_thing",
			Name: "baz",
		}.Instance(addrs.IntKey(0)),
		&states.ResourceInstanceObjectSrc{
			Status:        states.ObjectReady,
			SchemaVersion: 1,
			AttrsJSON:     []byte(`{"woozles":"confuzles"}`),
		},
		addrs.AbsProviderConfig{
			Provider: addrs.NewDefaultProvider("test"),
			Module:   addrs.RootModule,
		},
	)
	stateFile := statefile.New(state, "", 0)
	f, err := os.Create(DefaultStateFilename)
	if err != nil {
		t.Error(err)
	}
	if err := statefile.Write(stateFile, f); err != nil {
		t.Error(err)
	}

	// Test the complete predictor
	ui := new(cli.MockUi)
	meta := &Meta{Ui: ui}
	predictor := meta.completePredictResourceName()
	got := predictor.Predict(complete.Args{
		Last: "",
	})
	want := []string{"test_thing.baz[0]"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, want)
	}
}
