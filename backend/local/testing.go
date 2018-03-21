package local

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/terraform"
	"github.com/hashicorp/terraform/tfdiags"
)

// TestLocal returns a configured Local struct with temporary paths and
// in-memory ContextOpts.
//
// No operations will be called on the returned value, so you can still set
// public fields without any locks.
func TestLocal(t *testing.T) (*Local, func()) {
	t.Helper()

	tempDir := testTempDir(t)
	var local *Local
	local = &Local{
		StatePath:         filepath.Join(tempDir, "state.tfstate"),
		StateOutPath:      filepath.Join(tempDir, "state.tfstate"),
		StateBackupPath:   filepath.Join(tempDir, "state.tfstate.bak"),
		StateWorkspaceDir: filepath.Join(tempDir, "state.tfstate.d"),
		ContextOpts:       &terraform.ContextOpts{},
		ShowDiagnostics: func(vals ...interface{}) {
			var diags tfdiags.Diagnostics
			diags = diags.Append(vals...)
			for _, diag := range diags {
				t.Log(diag.Description().Summary)
				if local.CLI != nil {
					local.CLI.Error(diag.Description().Summary)
				}
			}
		},
	}
	cleanup := func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Fatal("error clecanup up test:", err)
		}
	}

	return local, cleanup
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

	// Setup our provider
	b.ContextOpts.ProviderResolver = terraform.ResourceProviderResolverFixed(
		map[string]terraform.ResourceProviderFactory{
			name: terraform.ResourceProviderFactoryFixed(p),
		},
	)

	return p
}

// TestNewLocalSingle is a factory for creating a TestLocalSingleState.
// This function matches the signature required for backend/init.
func TestNewLocalSingle() backend.Backend {
	return &TestLocalSingleState{Local: &Local{}}
}

// TestLocalSingleState is a backend implementation that wraps Local
// and modifies it to only support single states (returns
// ErrNamedStatesNotSupported for multi-state operations).
//
// This isn't an actual use case, this is exported just to provide a
// easy way to test that behavior.
type TestLocalSingleState struct {
	*Local
}

func (b *TestLocalSingleState) State(name string) (state.State, error) {
	if name != backend.DefaultStateName {
		return nil, backend.ErrNamedStatesNotSupported
	}

	return b.Local.State(name)
}

func (b *TestLocalSingleState) States() ([]string, error) {
	return nil, backend.ErrNamedStatesNotSupported
}

func (b *TestLocalSingleState) DeleteState(string) error {
	return backend.ErrNamedStatesNotSupported
}

// TestNewLocalNoDefault is a factory for creating a TestLocalNoDefaultState.
// This function matches the signature required for backend/init.
func TestNewLocalNoDefault() backend.Backend {
	return &TestLocalNoDefaultState{Local: &Local{}}
}

// TestLocalNoDefaultState is a backend implementation that wraps
// Local and modifies it to support named states, but not the
// default state. It returns ErrDefaultStateNotSupported when the
// DefaultStateName is used.
type TestLocalNoDefaultState struct {
	*Local
}

func (b *TestLocalNoDefaultState) State(name string) (state.State, error) {
	if name == backend.DefaultStateName {
		return nil, backend.ErrDefaultStateNotSupported
	}
	return b.Local.State(name)
}

func (b *TestLocalNoDefaultState) States() ([]string, error) {
	states, err := b.Local.States()
	if err != nil {
		return nil, err
	}

	filtered := states[:0]
	for _, name := range states {
		if name != backend.DefaultStateName {
			filtered = append(filtered, name)
		}
	}

	return filtered, nil
}

func (b *TestLocalNoDefaultState) DeleteState(name string) error {
	if name == backend.DefaultStateName {
		return backend.ErrDefaultStateNotSupported
	}
	return b.Local.DeleteState(name)
}

func testTempDir(t *testing.T) string {
	d, err := ioutil.TempDir("", "tf")
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	return d
}
