package local

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/cli"
	"github.com/mitchellh/colorstring"
)

const (
	DefaultEnvDir          = "terraform.tfstate.d"
	DefaultEnvFile         = "environment"
	DefaultStateFilename   = "terraform.tfstate"
	DefaultDataDir         = ".terraform"
	DefaultBackupExtension = ".backup"
)

var ErrEnvNotSupported = errors.New("environments not supported")

// Local is an implementation of EnhancedBackend that performs all operations
// locally. This is the "default" backend and implements normal Terraform
// behavior as it is well known.
type Local struct {
	// CLI and Colorize control the CLI output. If CLI is nil then no CLI
	// output will be done. If CLIColor is nil then no coloring will be done.
	CLI      cli.Ui
	CLIColor *colorstring.Colorize

	// The State* paths are set from the CLI options, and may be left blank to
	// use the defaults. If the actual paths for the local backend state are
	// needed, use the StatePaths method.
	//
	// StatePath is the local path where state is read from.
	//
	// StateOutPath is the local path where the state will be written.
	// If this is empty, it will default to StatePath.
	//
	// StateBackupPath is the local path where a backup file will be written.
	// Set this to "-" to disable state backup.
	StatePath       string
	StateOutPath    string
	StateBackupPath string

	// we only want to create a single instance of the local state
	state state.State
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

	schema *schema.Backend
	opLock sync.Mutex
	once   sync.Once

	// workingDir is where the State* paths should be relative to.
	// This is currently only used for tests.
	workingDir string
}

func (b *Local) Input(
	ui terraform.UIInput, c *terraform.ResourceConfig) (*terraform.ResourceConfig, error) {
	b.once.Do(b.init)

	f := b.schema.Input
	if b.Backend != nil {
		f = b.Backend.Input
	}

	return f(ui, c)
}

func (b *Local) Validate(c *terraform.ResourceConfig) ([]string, []error) {
	b.once.Do(b.init)

	f := b.schema.Validate
	if b.Backend != nil {
		f = b.Backend.Validate
	}

	return f(c)
}

func (b *Local) Configure(c *terraform.ResourceConfig) error {
	b.once.Do(b.init)

	f := b.schema.Configure
	if b.Backend != nil {
		f = b.Backend.Configure
	}

	return f(c)
}

func (b *Local) States() ([]string, string, error) {
	// If we have a backend handling state, defer to that.
	if b.Backend != nil {
		if b, ok := b.Backend.(backend.MultiState); ok {
			return b.States()
		} else {
			return nil, "", ErrEnvNotSupported
		}
	}

	// the listing always start with "default"
	envs := []string{backend.DefaultStateName}

	current, err := b.currentStateName()
	if err != nil {
		return nil, "", err
	}

	entries, err := ioutil.ReadDir(filepath.Join(b.workingDir, DefaultEnvDir))
	// no error if there's no envs configured
	if os.IsNotExist(err) {
		return envs, backend.DefaultStateName, nil
	}
	if err != nil {
		return nil, "", err
	}

	currentExists := false
	var listed []string
	for _, entry := range entries {
		if entry.IsDir() {
			name := filepath.Base(entry.Name())
			if name == current {
				currentExists = true
			}
			listed = append(listed, name)
		}
	}

	// current was out of sync for some reason, so return defualt
	if !currentExists {
		current = backend.DefaultStateName
	}

	sort.Strings(listed)
	envs = append(envs, listed...)

	return envs, current, nil
}

// DeleteState removes a named state.
// The "default" state cannot be removed.
func (b *Local) DeleteState(name string) error {
	// If we have a backend handling state, defer to that.
	if b.Backend != nil {
		if b, ok := b.Backend.(backend.MultiState); ok {
			return b.DeleteState(name)
		} else {
			return ErrEnvNotSupported
		}
	}

	if name == "" {
		return errors.New("empty state name")
	}

	if name == backend.DefaultStateName {
		return errors.New("cannot delete default state")
	}

	_, current, err := b.States()
	if err != nil {
		return err
	}

	// if we're deleting the current state, we change back to the default
	if name == current {
		if err := b.ChangeState(backend.DefaultStateName); err != nil {
			return err
		}
	}

	return os.RemoveAll(filepath.Join(b.workingDir, DefaultEnvDir, name))
}

// Change to the named state, creating it if it doesn't exist.
func (b *Local) ChangeState(name string) error {
	// If we have a backend handling state, defer to that.
	if b.Backend != nil {
		if b, ok := b.Backend.(backend.MultiState); ok {
			return b.ChangeState(name)
		} else {
			return ErrEnvNotSupported
		}
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("state name cannot be empty")
	}

	envs, current, err := b.States()
	if err != nil {
		return err
	}

	if name == current {
		return nil
	}

	exists := false
	for _, env := range envs {
		if env == name {
			exists = true
			break
		}
	}

	if !exists {
		if err := b.createState(name); err != nil {
			return err
		}
	}

	err = os.MkdirAll(filepath.Join(b.workingDir, DefaultDataDir), 0755)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(
		filepath.Join(b.workingDir, DefaultDataDir, DefaultEnvFile),
		[]byte(name),
		0644,
	)
	if err != nil {
		return err
	}

	// remove the current state so it's reloaded on the next call to State
	b.state = nil

	return nil
}

func (b *Local) State() (state.State, error) {
	// If we have a backend handling state, defer to that.
	if b.Backend != nil {
		return b.Backend.State()
	}

	if b.state != nil {
		return b.state, nil
	}

	statePath, stateOutPath, backupPath, err := b.StatePaths()
	if err != nil {
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

	b.state = s
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
	// Determine the function to call for our operation
	var f func(context.Context, *backend.Operation, *backend.RunningOperation)
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
	runningCtx, runningCtxCancel := context.WithCancel(context.Background())
	runningOp := &backend.RunningOperation{Context: runningCtx}

	// Do it
	go func() {
		defer b.opLock.Unlock()
		defer runningCtxCancel()
		f(ctx, op, runningOp)
	}()

	// Return
	return runningOp, nil
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

func (b *Local) init() {
	b.schema = &schema.Backend{
		Schema: map[string]*schema.Schema{
			"path": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
		},

		ConfigureFunc: b.schemaConfigure,
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

	return nil
}

// StatePaths returns the StatePath, StateOutPath, and StateBackupPath as
// configured by the current environment. If backups are disabled,
// StateBackupPath will be an empty string.
func (b *Local) StatePaths() (string, string, string, error) {
	statePath := b.StatePath
	stateOutPath := b.StateOutPath
	backupPath := b.StateBackupPath

	if statePath == "" {
		path, err := b.statePath()
		if err != nil {
			return "", "", "", err
		}
		statePath = path
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

	return statePath, stateOutPath, backupPath, nil
}

func (b *Local) statePath() (string, error) {
	_, current, err := b.States()
	if err != nil {
		return "", err
	}
	path := DefaultStateFilename

	if current != backend.DefaultStateName && current != "" {
		path = filepath.Join(b.workingDir, DefaultEnvDir, current, DefaultStateFilename)
	}
	return path, nil
}

func (b *Local) createState(name string) error {
	stateNames, _, err := b.States()
	if err != nil {
		return err
	}

	for _, n := range stateNames {
		if name == n {
			// state exists, nothing to do
			return nil
		}
	}

	err = os.MkdirAll(filepath.Join(b.workingDir, DefaultEnvDir, name), 0755)
	if err != nil {
		return err
	}

	return nil
}

// currentStateName returns the name of the current named state as set in the
// configuration files.
// If there are no configured environments, currentStateName returns "default"
func (b *Local) currentStateName() (string, error) {
	contents, err := ioutil.ReadFile(filepath.Join(b.workingDir, DefaultDataDir, DefaultEnvFile))
	if os.IsNotExist(err) {
		return backend.DefaultStateName, nil
	}
	if err != nil {
		return "", err
	}

	if fromFile := strings.TrimSpace(string(contents)); fromFile != "" {
		return fromFile, nil
	}

	return backend.DefaultStateName, nil
}
