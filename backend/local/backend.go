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
	"strings"
	"sync"

	"github.com/hashicorp/terraform/tfdiags"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/command/clistate"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/cli"
	"github.com/mitchellh/colorstring"
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
	// CLI and Colorize control the CLI output. If CLI is nil then no CLI
	// output will be done. If CLIColor is nil then no coloring will be done.
	CLI      cli.Ui
	CLIColor *colorstring.Colorize

	// ShowDiagnostics prints diagnostic messages to the UI.
	ShowDiagnostics func(vals ...interface{})

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

	// We only want to create a single instance of a local state, so store them
	// here as they're loaded.
	states map[string]state.State

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

	// RunningInAutomation indicates that commands are being run by an
	// automated system rather than directly at a command prompt.
	//
	// This is a hint not to produce messages that expect that a user can
	// run a follow-up command, perhaps because Terraform is running in
	// some sort of workflow automation tool that abstracts away the
	// exact commands that are being run.
	RunningInAutomation bool

	opLock sync.Mutex
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
			// environment_dir was previously a deprecated alias for
			// workspace_dir, but now removed.
		},
	}
}

func (b *Local) ValidateConfig(obj cty.Value) tfdiags.Diagnostics {
	if b.Backend != nil {
		return b.Backend.ValidateConfig(obj)
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

	return diags
}

func (b *Local) Configure(obj cty.Value) tfdiags.Diagnostics {
	if b.Backend != nil {
		return b.Backend.Configure(obj)
	}

	var diags tfdiags.Diagnostics

	type Config struct {
		Path         string `hcl:"path,optional"`
		WorkspaceDir string `hcl:"workspace_dir,optional"`
	}

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

func (b *Local) State(name string) (state.State, error) {
	statePath, stateOutPath, backupPath := b.StatePaths(name)

	// If we have a backend handling state, delegate to that.
	if b.Backend != nil {
		return b.Backend.State(name)
	}

	if s, ok := b.states[name]; ok {
		return s, nil
	}

	if err := b.createState(name); err != nil {
		return nil, err
	}

	// Otherwise, we need to load the state.
	var s state.State = &state.LocalState{
		Path:    statePath,
		PathOut: stateOutPath,
	}

	// If we are backing up the state, wrap it
	if backupPath != "" {
		s = &state.BackupState{
			Real: s,
			Path: backupPath,
		}
	}

	if b.states == nil {
		b.states = map[string]state.State{}
	}
	b.states[name] = s
	return s, nil
}

// DeleteState removes a named state.
// The "default" state cannot be removed.
func (b *Local) DeleteState(name string) error {
	// If we have a backend handling state, defer to that.
	if b.Backend != nil {
		return b.Backend.DeleteState(name)
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

func (b *Local) States() ([]string, error) {
	// If we have a backend handling state, defer to that.
	if b.Backend != nil {
		return b.Backend.States()
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

// Operation implements backend.Enhanced
//
// This will initialize an in-memory terraform.Context to perform the
// operation within this process.
//
// The given operation parameter will be merged with the ContextOpts on
// the structure with the following rules. If a rule isn't specified and the
// name conflicts, assume that the field is overwritten if set.
func (b *Local) Operation(ctx context.Context, op *backend.Operation) (*backend.RunningOperation, error) {
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

	if op.LockState {
		op.StateLocker = clistate.NewLocker(stopCtx, op.StateLockTimeout, b.CLI, b.Colorize())
	} else {
		op.StateLocker = clistate.NewNoopLocker()
	}

	// Do it
	go func() {
		defer done()
		defer stop()
		defer cancel()

		// the state was locked during context creation, unlock the state when
		// the operation completes
		defer func() {
			err := op.StateLocker.Unlock(nil)
			if err != nil {
				b.ShowDiagnostics(err)
			}
			runningOp.Result = backend.OperationFailure
		}()

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
	opState state.State) (canceled bool) {
	// Wait for the operation to finish or for us to be interrupted so
	// we can handle it properly.
	select {
	case <-stopCtx.Done():
		if b.CLI != nil {
			b.CLI.Output("stopping operation...")
		}

		// try to force a PersistState just in case the process is terminated
		// before we can complete.
		if err := opState.PersistState(); err != nil {
			// We can't error out from here, but warn the user if there was an error.
			// If this isn't transient, we will catch it again below, and
			// attempt to save the state another way.
			if b.CLI != nil {
				b.CLI.Error(fmt.Sprintf(earlyStateWriteErrorFmt, err))
			}
		}

		// Stop execution
		go tfCtx.Stop()

		select {
		case <-cancelCtx.Done():
			log.Println("[WARN] running operation canceled")
			// if the operation was canceled, we need to return immediately
			canceled = true
		case <-doneCh:
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

// ReportResult is a helper for the common chore of setting the status of
// a running operation and showing any diagnostics produced during that
// operation.
//
// If the given diagnostics contains errors then the operation's result
// will be set to backend.OperationFailure. It will be set to
// backend.OperationSuccess otherwise. It will then use b.ShowDiagnostics
// to show the given diagnostics before returning.
//
// Callers should feel free to do each of these operations separately in
// more complex cases where e.g. diagnostics are interleaved with other
// output, but terminating immediately after reporting error diagnostics is
// common and can be expressed concisely via this method.
func (b *Local) ReportResult(op *backend.RunningOperation, diags tfdiags.Diagnostics) {
	if diags.HasErrors() {
		op.Result = backend.OperationFailure
	} else {
		op.Result = backend.OperationSuccess
	}
	b.ShowDiagnostics(diags)
}

// Colorize returns the Colorize structure that can be used for colorizing
// output. This is gauranteed to always return a non-nil value and so is useful
// as a helper to wrap any potentially colored strings.
func (b *Local) Colorize() *colorstring.Colorize {
	if b.CLIColor != nil {
		return b.CLIColor
	}

	return &colorstring.Colorize{
		Colors:  colorstring.DefaultColors,
		Disable: true,
	}
}

func (b *Local) schemaConfigure(ctx context.Context) error {
	d := schema.FromContextBackendConfig(ctx)

	// Set the path if it is set
	pathRaw, ok := d.GetOk("path")
	if ok {
		path := pathRaw.(string)
		if path == "" {
			return fmt.Errorf("configured path is empty")
		}

		b.StatePath = path
		b.StateOutPath = path
	}

	if raw, ok := d.GetOk("workspace_dir"); ok {
		path := raw.(string)
		if path != "" {
			b.StateWorkspaceDir = path
		}
	}

	// Legacy name, which ConflictsWith workspace_dir
	if raw, ok := d.GetOk("environment_dir"); ok {
		path := raw.(string)
		if path != "" {
			b.StateWorkspaceDir = path
		}
	}

	return nil
}

// StatePaths returns the StatePath, StateOutPath, and StateBackupPath as
// configured from the CLI.
func (b *Local) StatePaths(name string) (string, string, string) {
	statePath := b.StatePath
	stateOutPath := b.StateOutPath
	backupPath := b.StateBackupPath

	if name == "" {
		name = backend.DefaultStateName
	}

	if name == backend.DefaultStateName {
		if statePath == "" {
			statePath = DefaultStateFilename
		}
	} else {
		statePath = filepath.Join(b.stateWorkspaceDir(), name, DefaultStateFilename)
	}

	if stateOutPath == "" {
		stateOutPath = statePath
	}

	switch backupPath {
	case "-":
		backupPath = ""
	case "":
		backupPath = stateOutPath + DefaultBackupExtension
	}

	return statePath, stateOutPath, backupPath
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

func (b *Local) pluginInitRequired(providerErr *terraform.ResourceProviderError) {
	b.CLI.Output(b.Colorize().Color(fmt.Sprintf(
		strings.TrimSpace(errPluginInit)+"\n",
		providerErr)))
}

// this relies on multierror to format the plugin errors below the copy
const errPluginInit = `
[reset][bold][yellow]Plugin reinitialization required. Please run "terraform init".[reset]
[yellow]Reason: Could not satisfy plugin requirements.

Plugins are external binaries that Terraform uses to access and manipulate
resources. The configuration provided requires plugins which can't be located,
don't satisfy the version constraints, or are otherwise incompatible.

[reset][red]%s

[reset][yellow]Terraform automatically discovers provider requirements from your
configuration, including providers used in child modules. To see the
requirements and constraints from each module, run "terraform providers".
`
