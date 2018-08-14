package planfile

import (
	"io/ioutil"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
	version "github.com/hashicorp/go-version"

	"github.com/hashicorp/terraform/configs/configload"
	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/states/statefile"
)

func TestRoundtrip(t *testing.T) {
	fixtureDir := filepath.Join("testdata", "test-config")
	loader, err := configload.NewLoader(&configload.Config{
		ModulesDir: filepath.Join(fixtureDir, ".terraform", "modules"),
	})
	if err != nil {
		t.Fatal(err)
	}

	_, snapIn, diags := loader.LoadConfigWithSnapshot(fixtureDir)
	if diags.HasErrors() {
		t.Fatal(diags.Error())
	}

	// Just a minimal state file so we can test that it comes out again at all.
	// We don't need to test the entire thing because the state file
	// serialization is already tested in its own package.
	stateFileIn := &statefile.File{
		TerraformVersion: version.Must(version.NewVersion("1.0.0")),
		Serial:           1,
		Lineage:          "abc123",
		State:            states.NewState(),
	}

	// Minimal plan too, since the serialization of the tfplan portion of the
	// file is tested more fully in tfplan_test.go .
	planIn := &plans.Plan{
		Changes: &plans.Changes{
			Resources:   []*plans.ResourceInstanceChangeSrc{},
			RootOutputs: map[string]*plans.OutputChangeSrc{},
		},
		ProviderSHA256s: map[string][]byte{},
		VariableValues: map[string]plans.DynamicValue{
			"foo": plans.DynamicValue([]byte("foo placeholder")),
		},
		Backend: plans.Backend{
			Type:      "local",
			Config:    plans.DynamicValue([]byte("config placeholder")),
			Workspace: "default",
		},
	}

	workDir, err := ioutil.TempDir("", "tf-planfile")
	if err != nil {
		t.Fatal(err)
	}
	planFn := filepath.Join(workDir, "tfplan")

	err = Create(planFn, snapIn, stateFileIn, planIn)
	if err != nil {
		t.Fatalf("failed to create plan file: %s", err)
	}

	pr, err := Open(planFn)
	if err != nil {
		t.Fatalf("failed to open plan file for reading: %s", err)
	}

	t.Run("ReadPlan", func(t *testing.T) {
		planOut, err := pr.ReadPlan()
		if err != nil {
			t.Fatalf("failed to read plan: %s", err)
		}
		if !reflect.DeepEqual(planIn, planOut) {
			t.Errorf("plan did not survive round-trip\nresult: %sinput: %s", spew.Sdump(planOut), spew.Sdump(planIn))
		}
	})

	t.Run("ReadStateFile", func(t *testing.T) {
		stateFileOut, err := pr.ReadStateFile()
		if err != nil {
			t.Fatalf("failed to read state: %s", err)
		}
		if !reflect.DeepEqual(stateFileIn, stateFileOut) {
			t.Errorf("state file did not survive round-trip\nresult: %sinput: %s", spew.Sdump(stateFileOut), spew.Sdump(stateFileIn))
		}
	})

	t.Run("ReadConfigSnapshot", func(t *testing.T) {
		snapOut, err := pr.ReadConfigSnapshot()
		if err != nil {
			t.Fatalf("failed to read config snapshot: %s", err)
		}
		if !reflect.DeepEqual(snapIn, snapOut) {
			t.Errorf("config snapshot did not survive round-trip\nresult: %sinput: %s", spew.Sdump(snapOut), spew.Sdump(snapIn))
		}
	})

	t.Run("ReadConfig", func(t *testing.T) {
		// Reading from snapshots is tested in the configload package, so
		// here we'll just test that we can successfully do it, to see if the
		// glue code in _this_ package is correct.
		_, diags := pr.ReadConfig()
		if diags.HasErrors() {
			t.Errorf("when reading config: %s", diags.Err())
		}
	})
}
