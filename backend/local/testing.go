package local

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform/terraform"
)

// TestLocal returns a configured Local struct with temporary paths and
// in-memory ContextOpts.
//
// No operations will be called on the returned value, so you can still set
// public fields without any locks.
func TestLocal(t *testing.T) *Local {
	tempDir := testTempDir(t)
	return &Local{
		StatePath:       filepath.Join(tempDir, "state.tfstate"),
		StateOutPath:    filepath.Join(tempDir, "state.tfstate"),
		StateBackupPath: filepath.Join(tempDir, "state.tfstate.bak"),
		ContextOpts:     &terraform.ContextOpts{},
	}
}

// TestLocalProvider modifies the ContextOpts of the *Local parameter to
// have a provider with the given name.
func TestLocalProvider(t *testing.T, b *Local, name string) *terraform.MockResourceProvider {
	// Build a mock resource provider for in-memory operations
	p := new(terraform.MockResourceProvider)
	p.DiffReturn = &terraform.InstanceDiff{}
	p.RefreshFn = func(
		info *terraform.InstanceInfo,
		s *terraform.InstanceState) (*terraform.InstanceState, error) {
		return s, nil
	}
	p.ResourcesReturn = []terraform.ResourceType{
		terraform.ResourceType{
			Name: "test_instance",
		},
	}

	// Initialize the opts
	if b.ContextOpts == nil {
		b.ContextOpts = &terraform.ContextOpts{}
	}
	if b.ContextOpts.Providers == nil {
		b.ContextOpts.Providers = make(map[string]terraform.ResourceProviderFactory)
	}

	// Setup our provider
	b.ContextOpts.Providers[name] = func() (terraform.ResourceProvider, error) {
		return p, nil
	}

	return p
}

func testTempDir(t *testing.T) string {
	d, err := ioutil.TempDir("", "tf")
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	return d
}
