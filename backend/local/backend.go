package local

import (
	"context"
	"fmt"
	"sync"

	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/cli"
	"github.com/mitchellh/colorstring"
)

// Local is an implementation of EnhancedBackend that performs all operations
// locally. This is the "default" backend and implements normal Terraform
// behavior as it is well known.
type Local struct {
	// CLI and Colorize control the CLI output. If CLI is nil then no CLI
	// output will be done. If CLIColor is nil then no coloring will be done.
	CLI      cli.Ui
	CLIColor *colorstring.Colorize

	// StatePath is the local path where state is read from.
	//
	// StateOutPath is the local path where the state will be written.
	// If this is empty, it will default to StatePath.
	//
	// StateBackupPath is the local path where a backup file will be written.
	// If this is empty, no backup will be taken.
	StatePath       string
	StateOutPath    string
	StateBackupPath string

	// we only want to create a single instance of the local state
	state state.State

	// ContextOpts are the base context options to set when initializing a
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

func (b *Local) State() (state.State, error) {
	// If we have a backend handling state, defer to that.
	if b.Backend != nil {
		return b.Backend.State()
	}

	if b.state != nil {
		return b.state, nil
	}

	// Otherwise, we need to load the state.
	var s state.State = &state.LocalState{
		Path:    b.StatePath,
		PathOut: b.StateOutPath,
	}

	// Load the state as a sanity check
	if err := s.RefreshState(); err != nil {
		return nil, errwrap.Wrapf("Error reading local state: {{err}}", err)
	}

	// If we are backing up the state, wrap it
	if path := b.StateBackupPath; path != "" {
		s = &state.BackupState{
			Real: s,
			Path: path,
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
	}

	return nil
}
