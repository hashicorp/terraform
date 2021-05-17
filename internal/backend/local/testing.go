package local

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/states/statemgr"
	"github.com/hashicorp/terraform/internal/terraform"
)

// TestLocal returns a configured Local struct with temporary paths and
// in-memory ContextOpts.
//
// No operations will be called on the returned value, so you can still set
// public fields without any locks.
func TestLocal(t *testing.T) (*Local, func()) {
	t.Helper()
	tempDir := testTempDir(t)

	local := New()
	local.StatePath = filepath.Join(tempDir, "state.tfstate")
	local.StateOutPath = filepath.Join(tempDir, "state.tfstate")
	local.StateBackupPath = filepath.Join(tempDir, "state.tfstate.bak")
	local.StateWorkspaceDir = filepath.Join(tempDir, "state.tfstate.d")
	local.ContextOpts = &terraform.ContextOpts{}

	cleanup := func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Fatal("error cleanup up test:", err)
		}
	}

	return local, cleanup
}

// TestLocalProvider modifies the ContextOpts of the *Local parameter to
// have a provider with the given name.
func TestLocalProvider(t *testing.T, b *Local, name string, schema *terraform.ProviderSchema) *terraform.MockProvider {
	// Build a mock resource provider for in-memory operations
	p := new(terraform.MockProvider)

	if schema == nil {
		schema = &terraform.ProviderSchema{} // default schema is empty
	}
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		Provider:      providers.Schema{Block: schema.Provider},
		ProviderMeta:  providers.Schema{Block: schema.ProviderMeta},
		ResourceTypes: map[string]providers.Schema{},
		DataSources:   map[string]providers.Schema{},
	}
	for name, res := range schema.ResourceTypes {
		p.GetProviderSchemaResponse.ResourceTypes[name] = providers.Schema{
			Block:   res,
			Version: int64(schema.ResourceTypeSchemaVersions[name]),
		}
	}
	for name, dat := range schema.DataSources {
		p.GetProviderSchemaResponse.DataSources[name] = providers.Schema{Block: dat}
	}

	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
		rSchema, _ := schema.SchemaForResourceType(addrs.ManagedResourceMode, req.TypeName)
		if rSchema == nil {
			rSchema = &configschema.Block{} // default schema is empty
		}
		plannedVals := map[string]cty.Value{}
		for name, attrS := range rSchema.Attributes {
			val := req.ProposedNewState.GetAttr(name)
			if attrS.Computed && val.IsNull() {
				val = cty.UnknownVal(attrS.Type)
			}
			plannedVals[name] = val
		}
		for name := range rSchema.BlockTypes {
			// For simplicity's sake we just copy the block attributes over
			// verbatim, since this package's mock providers are all relatively
			// simple -- we're testing the backend, not esoteric provider features.
			plannedVals[name] = req.ProposedNewState.GetAttr(name)
		}

		return providers.PlanResourceChangeResponse{
			PlannedState:   cty.ObjectVal(plannedVals),
			PlannedPrivate: req.PriorPrivate,
		}
	}
	p.ReadResourceFn = func(req providers.ReadResourceRequest) providers.ReadResourceResponse {
		return providers.ReadResourceResponse{NewState: req.PriorState}
	}
	p.ReadDataSourceFn = func(req providers.ReadDataSourceRequest) providers.ReadDataSourceResponse {
		return providers.ReadDataSourceResponse{State: req.Config}
	}

	// Initialize the opts
	if b.ContextOpts == nil {
		b.ContextOpts = &terraform.ContextOpts{}
	}

	// Set up our provider
	b.ContextOpts.Providers = map[addrs.Provider]providers.Factory{
		addrs.NewDefaultProvider(name): providers.FactoryFixed(p),
	}

	return p

}

// TestLocalSingleState is a backend implementation that wraps Local
// and modifies it to only support single states (returns
// ErrWorkspacesNotSupported for multi-state operations).
//
// This isn't an actual use case, this is exported just to provide a
// easy way to test that behavior.
type TestLocalSingleState struct {
	*Local
}

// TestNewLocalSingle is a factory for creating a TestLocalSingleState.
// This function matches the signature required for backend/init.
func TestNewLocalSingle() backend.Backend {
	return &TestLocalSingleState{Local: New()}
}

func (b *TestLocalSingleState) Workspaces() ([]string, error) {
	return nil, backend.ErrWorkspacesNotSupported
}

func (b *TestLocalSingleState) DeleteWorkspace(string) error {
	return backend.ErrWorkspacesNotSupported
}

func (b *TestLocalSingleState) StateMgr(name string) (statemgr.Full, error) {
	if name != backend.DefaultStateName {
		return nil, backend.ErrWorkspacesNotSupported
	}

	return b.Local.StateMgr(name)
}

// TestLocalNoDefaultState is a backend implementation that wraps
// Local and modifies it to support named states, but not the
// default state. It returns ErrDefaultWorkspaceNotSupported when
// the DefaultStateName is used.
type TestLocalNoDefaultState struct {
	*Local
}

// TestNewLocalNoDefault is a factory for creating a TestLocalNoDefaultState.
// This function matches the signature required for backend/init.
func TestNewLocalNoDefault() backend.Backend {
	return &TestLocalNoDefaultState{Local: New()}
}

func (b *TestLocalNoDefaultState) Workspaces() ([]string, error) {
	workspaces, err := b.Local.Workspaces()
	if err != nil {
		return nil, err
	}

	filtered := workspaces[:0]
	for _, name := range workspaces {
		if name != backend.DefaultStateName {
			filtered = append(filtered, name)
		}
	}

	return filtered, nil
}

func (b *TestLocalNoDefaultState) DeleteWorkspace(name string) error {
	if name == backend.DefaultStateName {
		return backend.ErrDefaultWorkspaceNotSupported
	}
	return b.Local.DeleteWorkspace(name)
}

func (b *TestLocalNoDefaultState) StateMgr(name string) (statemgr.Full, error) {
	if name == backend.DefaultStateName {
		return nil, backend.ErrDefaultWorkspaceNotSupported
	}
	return b.Local.StateMgr(name)
}

func testTempDir(t *testing.T) string {
	d, err := ioutil.TempDir("", "tf")
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	return d
}

func testStateFile(t *testing.T, path string, s *states.State) {
	stateFile := statemgr.NewFilesystem(path)
	stateFile.WriteState(s)
}

func mustProviderConfig(s string) addrs.AbsProviderConfig {
	p, diags := addrs.ParseAbsProviderConfigStr(s)
	if diags.HasErrors() {
		panic(diags.Err())
	}
	return p
}

func mustResourceInstanceAddr(s string) addrs.AbsResourceInstance {
	addr, diags := addrs.ParseAbsResourceInstanceStr(s)
	if diags.HasErrors() {
		panic(diags.Err())
	}
	return addr
}

// assertBackendStateUnlocked attempts to lock the backend state. Failure
// indicates that the state was indeed locked and therefore this function will
// return true.
func assertBackendStateUnlocked(t *testing.T, b *Local) bool {
	t.Helper()
	stateMgr, _ := b.StateMgr(backend.DefaultStateName)
	if _, err := stateMgr.Lock(statemgr.NewLockInfo()); err != nil {
		t.Errorf("state is already locked: %s", err.Error())
		return false
	}
	return true
}

// assertBackendStateLocked attempts to lock the backend state. Failure
// indicates that the state was already locked and therefore this function will
// return false.
func assertBackendStateLocked(t *testing.T, b *Local) bool {
	t.Helper()
	stateMgr, _ := b.StateMgr(backend.DefaultStateName)
	if _, err := stateMgr.Lock(statemgr.NewLockInfo()); err != nil {
		return true
	}
	t.Error("unexpected success locking state")
	return true
}
