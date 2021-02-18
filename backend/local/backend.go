package local

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/command/views"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/states/statemgr"
	"github.com/hashicorp/terraform/terraform"
	"github.com/hashicorp/terraform/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

const (
	DefaultWorkspaceDir    = "terraform.tfstate.d"
	DefaultWorkspaceFile   = "environment"
	DefaultStateFilename   = "terraform.tfstate"
	DefaultBackupExtension = ".backup"
)

// Local is an implementation of EnhancedBackend that performs all operations
// locally. This is the "default" backend and implements normal Terraform
// behavior as it is well known.
type Local struct {
	// The State* paths are set from the backend config, and may be left blank
	// to use the defaults. If the actual paths for the local backend state are
	// needed, use the StatePaths method.
	//
	// StatePath is the local path where state is read from.
	//
	// StateOutPath is the local path where the state will be written.
	// If this is empty, it will default to StatePath.
	//
	// StateBackupPath is the local path where a backup file will be written.
	// Set this to "-" to disable state backup.
	//
	// StateWorkspaceDir is the path to the folder containing data for
	// non-default workspaces. This defaults to DefaultWorkspaceDir if not set.
	StatePath         string
	StateOutPath      string
	StateBackupPath   string
	StateWorkspaceDir string

	// The OverrideState* paths are set based on per-operation CLI arguments
	// and will override what'd be built from the State* fields if non-empty.
	// While the interpretation of the State* fields depends on the active
	// workspace, the OverrideState* fields are always used literally.
	OverrideStatePath       string
	OverrideStateOutPath    string
	OverrideStateBackupPath string

	// We only want to create a single instance of a local state, so store them
	// here as they're loaded.
	states map[string]statemgr.Full

	// Terraform context. Many of these will be overridden or merged by
	// Operation. See Operation for more details.
	ContextOpts *terraform.ContextOpts

	// OpInput will ask for necessary input prior to performing any operations.
	//
	// OpValidation will perform validation prior to running an operation. The
	// variable naming doesn't match the style of others since we have a func
	// Validate.
	OpInput      bool
	OpValidation bool

	// Backend, if non-nil, will use this backend for non-enhanced behavior.
	// This allows local behavior with remote state storage. It is a way to
	// "upgrade" a non-enhanced backend to an enhanced backend with typical
	// behavior.
	//
	// If this is nil, local performs normal state loading and storage.
	Backend backend.Backend

	// opLock locks operations
	opLock sync.Mutex
}

var _ backend.Backend = (*Local)(nil)

// New returns a new initialized local backend.
func New() *Local {
	return NewWithBackend(nil)
}

// NewWithBackend returns a new local backend initialized with a
// dedicated backend for non-enhanced behavior.
func NewWithBackend(backend backend.Backend) *Local {
	return &Local{
		Backend: backend,
	}
}

func (b *Local) ConfigSchema() *configschema.Block {
	if b.Backend != nil {
		return b.Backend.ConfigSchema()
	}
	return &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"path": {
				Type:     cty.String,
				Optional: true,
			},
			"workspace_dir": {
				Type:     cty.String,
				Optional: true,
			},
		},
	}
}

func (b *Local) PrepareConfig(obj cty.Value) (cty.Value, tfdiags.Diagnostics) {
	if b.Backend != nil {
		return b.Backend.PrepareConfig(obj)
	}

	var diags tfdiags.Diagnostics

	if val := obj.GetAttr("path"); !val.IsNull() {
		p := val.AsString()
		if p == "" {
			diags = diags.Append(tfdiags.AttributeValue(
				tfdiags.Error,
				"Invalid local state file path",
				`The "path" attribute value must not be empty.`,
				cty.Path{cty.GetAttrStep{Name: "path"}},
			))
		}
	}

	if val := obj.GetAttr("workspace_dir"); !val.IsNull() {
		p := val.AsString()
		if p == "" {
			diags = diags.Append(tfdiags.AttributeValue(
				tfdiags.Error,
				"Invalid local workspace directory path",
				`The "workspace_dir" attribute value must not be empty.`,
				cty.Path{cty.GetAttrStep{Name: "workspace_dir"}},
			))
		}
	}

	return obj, diags
}

func (b *Local) Configure(obj cty.Value) tfdiags.Diagnostics {
	if b.Backend != nil {
		return b.Backend.Configure(obj)
	}

	var diags tfdiags.Diagnostics

	if val := obj.GetAttr("path"); !val.IsNull() {
		p := val.AsString()
		b.StatePath = p
		b.StateOutPath = p
	} else {
		b.StatePath = DefaultStateFilename
		b.StateOutPath = DefaultStateFilename
	}

	if val := obj.GetAttr("workspace_dir"); !val.IsNull() {
		p := val.AsString()
		b.StateWorkspaceDir = p
	} else {
		b.StateWorkspaceDir = DefaultWorkspaceDir
	}

	return diags
}

func (b *Local) Workspaces() ([]string, error) {
	// If we have a backend handling state, defer to that.
	if b.Backend != nil {
		return b.Backend.Workspaces()
	}

	// the listing always start with "default"
	envs := []string{backend.DefaultStateName}

	entries, err := ioutil.ReadDir(b.stateWorkspaceDir())
	// no error if there's no envs configured
	if os.IsNotExist(err) {
		return envs, nil
	}
	if err != nil {
		return nil, err
	}

	var listed []string
	for _, entry := range entries {
		if entry.IsDir() {
			listed = append(listed, filepath.Base(entry.Name()))
		}
	}

	sort.Strings(listed)
	envs = append(envs, listed...)

	return envs, nil
}

// DeleteWorkspace removes a workspace.
//
// The "default" workspace cannot be removed.
func (b *Local) DeleteWorkspace(name string) error {
	// If we have a backend handling state, defer to that.
	if b.Backend != nil {
		return b.Backend.DeleteWorkspace(name)
	}

	if name == "" {
		return errors.New("empty state name")
	}

	if name == backend.DefaultStateName {
		return errors.New("cannot delete default state")
	}

	delete(b.states, name)
	return os.RemoveAll(filepath.Join(b.stateWorkspaceDir(), name))
}

func (b *Local) StateMgr(name string) (statemgr.Full, error) {
	// If we have a backend handling state, delegate to that.
	if b.Backend != nil {
		return b.Backend.StateMgr(name)
	}

	if s, ok := b.states[name]; ok {
		return s, nil
	}

	if err := b.createState(name); err != nil {
		return nil, err
	}

	statePath, stateOutPath, backupPath := b.StatePaths(name)
	log.Printf("[TRACE] backend/local: state manager for workspace %q will:\n - read initial snapshot from %s\n - write new snapshots to %s\n - create any backup at %s", name, statePath, stateOutPath, backupPath)

	s := statemgr.NewFilesystemBetweenPaths(statePath, stateOutPath)
	if backupPath != "" {
		s.SetBackupPath(backupPath)
	}

	if b.states == nil {
		b.states = map[string]statemgr.Full{}
	}
	b.states[name] = s
	return s, nil
}

// Operation implements backend.Enhanced
//
// This will initialize an in-memory terraform.Context to perform the
// operation within this process.
//
// The given operation parameter will be merged with the ContextOpts on
// the structure with the following rules. If a rule isn't specified and the
// name conflicts, assume that the field is overwritten if set.
func (b *Local) Operation(ctx context.Context, op *backend.Operation) (*backend.RunningOperation, error) {
	if op.View == nil {
		panic("Operation called with nil View")
	}

	// Determine the function to call for our operation
	var f func(context.Context, context.Context, *backend.Operation, *backend.RunningOperation)
	switch op.Type {
	case backend.OperationTypeRefresh:
		f = b.opRefresh
	case backend.OperationTypePlan:
		f = b.opPlan
	case backend.OperationTypeApply:
		f = b.opApply
	default:
		return nil, fmt.Errorf(
			"Unsupported operation type: %s\n\n"+
				"This is a bug in Terraform and should be reported. The local backend\n"+
				"is built-in to Terraform and should always support all operations.",
			op.Type)
	}

	// Lock
	b.opLock.Lock()

	// Build our running operation
	// the runninCtx is only used to block until the operation returns.
	runningCtx, done := context.WithCancel(context.Background())
	runningOp := &backend.RunningOperation{
		Context: runningCtx,
	}

	// stopCtx wraps the context passed in, and is used to signal a graceful Stop.
	stopCtx, stop := context.WithCancel(ctx)
	runningOp.Stop = stop

	// cancelCtx is used to cancel the operation immediately, usually
	// indicating that the process is exiting.
	cancelCtx, cancel := context.WithCancel(context.Background())
	runningOp.Cancel = cancel

	op.StateLocker = op.StateLocker.WithContext(stopCtx)

	// Do it
	go func() {
		defer done()
		defer stop()
		defer cancel()

		defer b.opLock.Unlock()
		f(stopCtx, cancelCtx, op, runningOp)
	}()

	// Return
	return runningOp, nil
}

// opWait waits for the operation to complete, and a stop signal or a
// cancelation signal.
func (b *Local) opWait(
	doneCh <-chan struct{},
	stopCtx context.Context,
	cancelCtx context.Context,
	tfCtx *terraform.Context,
	opStateMgr statemgr.Persister,
	view views.Operation) (canceled bool) {
	// Wait for the operation to finish or for us to be interrupted so
	// we can handle it properly.
	select {
	case <-stopCtx.Done():
		view.Stopping()

		// try to force a PersistState just in case the process is terminated
		// before we can complete.
		if err := opStateMgr.PersistState(); err != nil {
			// We can't error out from here, but warn the user if there was an error.
			// If this isn't transient, we will catch it again below, and
			// attempt to save the state another way.
			var diags tfdiags.Diagnostics
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Error saving current state",
				fmt.Sprintf(earlyStateWriteErrorFmt, err),
			))
			view.Diagnostics(diags)
		}

		// Stop execution
		log.Println("[TRACE] backend/local: waiting for the running operation to stop")
		go tfCtx.Stop()

		select {
		case <-cancelCtx.Done():
			log.Println("[WARN] running operation was forcefully canceled")
			// if the operation was canceled, we need to return immediately
			canceled = true
		case <-doneCh:
			log.Println("[TRACE] backend/local: graceful stop has completed")
		}
	case <-cancelCtx.Done():
		// this should not be called without first attempting to stop the
		// operation
		log.Println("[ERROR] running operation canceled without Stop")
		canceled = true
	case <-doneCh:
	}
	return
}

// StatePaths returns the StatePath, StateOutPath, and StateBackupPath as
// configured from the CLI.
func (b *Local) StatePaths(name string) (stateIn, stateOut, backupOut string) {
	statePath := b.OverrideStatePath
	stateOutPath := b.OverrideStateOutPath
	backupPath := b.OverrideStateBackupPath

	isDefault := name == backend.DefaultStateName || name == ""

	baseDir := ""
	if !isDefault {
		baseDir = filepath.Join(b.stateWorkspaceDir(), name)
	}

	if statePath == "" {
		if isDefault {
			statePath = b.StatePath // s.StatePath applies only to the default workspace, since StateWorkspaceDir is used otherwise
		}
		if statePath == "" {
			statePath = filepath.Join(baseDir, DefaultStateFilename)
		}
	}
	if stateOutPath == "" {
		stateOutPath = statePath
	}
	if backupPath == "" {
		backupPath = b.StateBackupPath
	}
	switch backupPath {
	case "-":
		backupPath = ""
	case "":
		backupPath = stateOutPath + DefaultBackupExtension
	}

	return statePath, stateOutPath, backupPath
}

// PathsConflictWith returns true if any state path used by a workspace in
// the receiver is the same as any state path used by the other given
// local backend instance.
//
// This should be used when "migrating" from one local backend configuration to
// another in order to avoid deleting the "old" state snapshots if they are
// in the same files as the "new" state snapshots.
func (b *Local) PathsConflictWith(other *Local) bool {
	otherPaths := map[string]struct{}{}
	otherWorkspaces, err := other.Workspaces()
	if err != nil {
		// If we can't enumerate the workspaces then we'll conservatively
		// assume that paths _do_ overlap, since we can't be certain.
		return true
	}
	for _, name := range otherWorkspaces {
		p, _, _ := other.StatePaths(name)
		otherPaths[p] = struct{}{}
	}

	ourWorkspaces, err := other.Workspaces()
	if err != nil {
		// If we can't enumerate the workspaces then we'll conservatively
		// assume that paths _do_ overlap, since we can't be certain.
		return true
	}

	for _, name := range ourWorkspaces {
		p, _, _ := b.StatePaths(name)
		if _, exists := otherPaths[p]; exists {
			return true
		}
	}
	return false
}

// this only ensures that the named directory exists
func (b *Local) createState(name string) error {
	if name == backend.DefaultStateName {
		return nil
	}

	stateDir := filepath.Join(b.stateWorkspaceDir(), name)
	s, err := os.Stat(stateDir)
	if err == nil && s.IsDir() {
		// no need to check for os.IsNotExist, since that is covered by os.MkdirAll
		// which will catch the other possible errors as well.
		return nil
	}

	err = os.MkdirAll(stateDir, 0755)
	if err != nil {
		return err
	}

	return nil
}

// stateWorkspaceDir returns the directory where state environments are stored.
func (b *Local) stateWorkspaceDir() string {
	if b.StateWorkspaceDir != "" {
		return b.StateWorkspaceDir
	}

	return DefaultWorkspaceDir
}

const earlyStateWriteErrorFmt = `Error: %s

Terraform encountered an error attempting to save the state before cancelling the current operation. Once the operation is complete another attempt will be made to save the final state.`
