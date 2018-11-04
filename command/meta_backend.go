package command

// This file contains all the Backend-related function calls on Meta,
// exported and private.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hcldec"
	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"

	"github.com/hashicorp/terraform/backend"
	backendinit "github.com/hashicorp/terraform/backend/init"
	backendlocal "github.com/hashicorp/terraform/backend/local"
	"github.com/hashicorp/terraform/command/clistate"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/terraform"
	"github.com/hashicorp/terraform/tfdiags"
)

// BackendOpts are the options used to initialize a backend.Backend.
type BackendOpts struct {
	// Config is a representation of the backend configuration block given in
	// the root module, or nil if no such block is present.
	Config *configs.Backend

	// ConfigOverride is an hcl.Body that, if non-nil, will be used with
	// configs.MergeBodies to override the type-specific backend configuration
	// arguments in Config.
	ConfigOverride hcl.Body

	// Init should be set to true if initialization is allowed. If this is
	// false, then any configuration that requires configuration will show
	// an error asking the user to reinitialize.
	Init bool

	// ForceLocal will force a purely local backend, including state.
	// You probably don't want to set this.
	ForceLocal bool
}

// Backend initializes and returns the backend for this CLI session.
//
// The backend is used to perform the actual Terraform operations. This
// abstraction enables easily sliding in new Terraform behavior such as
// remote state storage, remote operations, etc. while allowing the CLI
// to remain mostly identical.
//
// This will initialize a new backend for each call, which can carry some
// overhead with it. Please reuse the returned value for optimal behavior.
//
// Only one backend should be used per Meta. This function is stateful
// and is unsafe to create multiple backends used at once. This function
// can be called multiple times with each backend being "live" (usable)
// one at a time.
//
// A side-effect of this method is the population of m.backendState, recording
// the final resolved backend configuration after dealing with overrides from
// the "terraform init" command line, etc.
func (m *Meta) Backend(opts *BackendOpts) (backend.Enhanced, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	// If no opts are set, then initialize
	if opts == nil {
		opts = &BackendOpts{}
	}

	// Initialize a backend from the config unless we're forcing a purely
	// local operation.
	var b backend.Backend
	if !opts.ForceLocal {
		var backendDiags tfdiags.Diagnostics
		b, backendDiags = m.backendFromConfig(opts)
		diags = diags.Append(backendDiags)

		if diags.HasErrors() {
			return nil, diags
		}

		log.Printf("[INFO] command: backend initialized: %T", b)
	}

	// Setup the CLI opts we pass into backends that support it
	cliOpts := m.backendCLIOpts()
	cliOpts.Validation = true

	// If the backend supports CLI initialization, do it.
	if cli, ok := b.(backend.CLI); ok {
		if err := cli.CLIInit(cliOpts); err != nil {
			diags = diags.Append(fmt.Errorf(
				"Error initializing backend %T: %s\n\n"+
					"This is a bug; please report it to the backend developer",
				b, err,
			))
			return nil, diags
		}
	}

	// If the result of loading the backend is an enhanced backend,
	// then return that as-is. This works even if b == nil (it will be !ok).
	if enhanced, ok := b.(backend.Enhanced); ok {
		return enhanced, nil
	}

	// We either have a non-enhanced backend or no backend configured at
	// all. In either case, we use local as our enhanced backend and the
	// non-enhanced (if any) as the state backend.

	if !opts.ForceLocal {
		log.Printf("[INFO] command: backend %T is not enhanced, wrapping in local", b)
	}

	// Build the local backend
	local := &backendlocal.Local{Backend: b}
	if err := local.CLIInit(cliOpts); err != nil {
		// Local backend isn't allowed to fail. It would be a bug.
		panic(err)
	}

	// If we got here from backendFromConfig returning nil then m.backendState
	// won't be set, since that codepath considers that to be no backend at all,
	// but our caller considers that to be the local backend with no config
	// and so we'll synthesize a backend state so other code doesn't need to
	// care about this special case.
	//
	// FIXME: We should refactor this so that we more directly and explicitly
	// treat the local backend as the default, including in the UI shown to
	// the user, since the local backend should only be used when learning or
	// in exceptional cases and so it's better to help the user learn that
	// by introducing it as a concept.
	if m.backendState == nil {
		// NOTE: This synthetic object is intentionally _not_ retained in the
		// on-disk record of the backend configuration, which was already dealt
		// with inside backendFromConfig, because we still need that codepath
		// to be able to recognize the lack of a config as distinct from
		// explicitly setting local until we do some more refactoring here.
		m.backendState = &terraform.BackendState{
			Type:      "local",
			ConfigRaw: json.RawMessage("{}"),
		}
	}

	return local, nil
}

// BackendForPlan is similar to Backend, but uses backend settings that were
// stored in a plan.
//
// The current workspace name is also stored as part of the plan, and so this
// method will check that it matches the currently-selected workspace name
// and produce error diagnostics if not.
func (m *Meta) BackendForPlan(settings plans.Backend) (backend.Enhanced, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	f := backendinit.Backend(settings.Type)
	if f == nil {
		diags = diags.Append(fmt.Errorf(strings.TrimSpace(errBackendSavedUnknown), settings.Type))
		return nil, diags
	}
	b := f()

	schema := b.ConfigSchema()
	configVal, err := settings.Config.Decode(schema.ImpliedType())
	if err != nil {
		diags = diags.Append(errwrap.Wrapf("saved backend configuration is invalid: {{err}}", err))
		return nil, diags
	}

	validateDiags := b.ValidateConfig(configVal)
	diags = diags.Append(validateDiags)
	if validateDiags.HasErrors() {
		return nil, diags
	}

	configureDiags := b.Configure(configVal)
	diags = diags.Append(configureDiags)

	// If the backend supports CLI initialization, do it.
	if cli, ok := b.(backend.CLI); ok {
		cliOpts := m.backendCLIOpts()
		if err := cli.CLIInit(cliOpts); err != nil {
			diags = diags.Append(fmt.Errorf(
				"Error initializing backend %T: %s\n\n"+
					"This is a bug; please report it to the backend developer",
				b, err,
			))
			return nil, diags
		}
	}

	// If the result of loading the backend is an enhanced backend,
	// then return that as-is. This works even if b == nil (it will be !ok).
	if enhanced, ok := b.(backend.Enhanced); ok {
		return enhanced, nil
	}

	// Otherwise, we'll wrap our state-only remote backend in the local backend
	// to cause any operations to be run locally.
	cliOpts := m.backendCLIOpts()
	cliOpts.Validation = false // don't validate here in case config contains file(...) calls where the file doesn't exist
	local := &backendlocal.Local{Backend: b}
	if err := local.CLIInit(cliOpts); err != nil {
		// Local backend should never fail, so this is always a bug.
		panic(err)
	}

	return local, diags
}

// backendCLIOpts returns a backend.CLIOpts object that should be passed to
// a backend that supports local CLI operations.
func (m *Meta) backendCLIOpts() *backend.CLIOpts {
	return &backend.CLIOpts{
		CLI:                 m.Ui,
		CLIColor:            m.Colorize(),
		ShowDiagnostics:     m.showDiagnostics,
		StatePath:           m.statePath,
		StateOutPath:        m.stateOutPath,
		StateBackupPath:     m.backupPath,
		ContextOpts:         m.contextOpts(),
		Input:               m.Input(),
		RunningInAutomation: m.RunningInAutomation,
	}
}

// IsLocalBackend returns true if the backend is a local backend. We use this
// for some checks that require a remote backend.
func (m *Meta) IsLocalBackend(b backend.Backend) bool {
	// Is it a local backend?
	bLocal, ok := b.(*backendlocal.Local)

	// If it is, does it not have an alternate state backend?
	if ok {
		ok = bLocal.Backend == nil
	}

	return ok
}

// Operation initializes a new backend.Operation struct.
//
// This prepares the operation. After calling this, the caller is expected
// to modify fields of the operation such as Sequence to specify what will
// be called.
func (m *Meta) Operation(b backend.Backend) *backend.Operation {
	schema := b.ConfigSchema()
	workspace := m.Workspace()
	planOutBackend, err := m.backendState.ForPlan(schema, workspace)
	if err != nil {
		// Always indicates an implementation error in practice, because
		// errors here indicate invalid encoding of the backend configuration
		// in memory, and we should always have validated that by the time
		// we get here.
		panic(fmt.Sprintf("failed to encode backend configuration for plan: %s", err))
	}

	return &backend.Operation{
		PlanOutBackend:   planOutBackend,
		Targets:          m.targets,
		UIIn:             m.UIInput(),
		UIOut:            m.Ui,
		Workspace:        workspace,
		LockState:        m.stateLock,
		StateLockTimeout: m.stateLockTimeout,
	}
}

// backendConfig returns the local configuration for the backend
func (m *Meta) backendConfig(opts *BackendOpts) (*configs.Backend, int, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	if opts.Config == nil {
		// check if the config was missing, or just not required
		conf, err := m.loadBackendConfig(".")
		if err != nil {
			return nil, 0, err
		}

		if conf == nil {
			log.Println("[INFO] command: no config, returning nil")
			return nil, 0, nil
		}

		log.Println("[WARN] BackendOpts.Config not set, but config found")
		opts.Config = conf
	}

	c := opts.Config

	if c == nil {
		log.Println("[INFO] command: no explicit backend config")
		return nil, 0, nil
	}

	bf := backendinit.Backend(c.Type)
	if bf == nil {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid backend type",
			Detail:   fmt.Sprintf("There is no backend type named %q.", c.Type),
			Subject:  &c.TypeRange,
		})
		return nil, 0, diags
	}
	b := bf()

	configSchema := b.ConfigSchema()
	configBody := c.Config
	configHash := c.Hash(configSchema)

	// If we have an override configuration body then we must apply it now.
	if opts.ConfigOverride != nil {
		configBody = configs.MergeBodies(configBody, opts.ConfigOverride)
	}

	// We'll shallow-copy configs.Backend here so that we can replace the
	// body without affecting others that hold this reference.
	configCopy := *c
	c.Config = configBody
	return &configCopy, configHash, diags
}

// backendFromConfig returns the initialized (not configured) backend
// directly from the config/state..
//
// This function handles various edge cases around backend config loading. For
// example: new config changes, backend type changes, etc.
//
// As of the 0.12 release it can no longer migrate from legacy remote state
// to backends, and will instead instruct users to use 0.11 or earlier as
// a stepping-stone to do that migration.
//
// This function may query the user for input unless input is disabled, in
// which case this function will error.
func (m *Meta) backendFromConfig(opts *BackendOpts) (backend.Backend, tfdiags.Diagnostics) {
	// Get the local backend configuration.
	c, cHash, diags := m.backendConfig(opts)
	if diags.HasErrors() {
		return nil, diags
	}

	// ------------------------------------------------------------------------
	// For historical reasons, current backend configuration for a working
	// directory is kept in a *state-like* file, using the legacy state
	// structures in the Terraform package. It is not actually a Terraform
	// state, and so only the "backend" portion of it is actually used.
	//
	// The remainder of this code often confusingly refers to this as a "state",
	// so it's unfortunately important to remember that this is not actually
	// what we _usually_ think of as "state", and is instead a local working
	// directory "backend configuration state" that is never persisted anywhere.
	//
	// Since the "real" state has since moved on to be represented by
	// states.State, we can recognize the special meaning of state that applies
	// to this function and its callees by their continued use of the
	// otherwise-obsolete terraform.State.
	// ------------------------------------------------------------------------

	// Get the path to where we store a local cache of backend configuration
	// if we're using a remote backend. This may not yet exist which means
	// we haven't used a non-local backend before. That is okay.
	statePath := filepath.Join(m.DataDir(), DefaultStateFilename)
	sMgr := &state.LocalState{Path: statePath}
	if err := sMgr.RefreshState(); err != nil {
		diags = diags.Append(fmt.Errorf("Failed to load state: %s", err))
		return nil, diags
	}

	// Load the state, it must be non-nil for the tests below but can be empty
	s := sMgr.State()
	if s == nil {
		log.Printf("[DEBUG] command: no data state file found for backend config")
		s = terraform.NewState()
	}

	// if we want to force reconfiguration of the backend, we set the backend
	// state to nil on this copy. This will direct us through the correct
	// configuration path in the switch statement below.
	if m.reconfigure {
		s.Backend = nil
	}

	// Upon return, we want to set the state we're using in-memory so that
	// we can access it for commands.
	m.backendState = nil
	defer func() {
		if s := sMgr.State(); s != nil && !s.Backend.Empty() {
			m.backendState = s.Backend
		}
	}()

	if !s.Remote.Empty() {
		// Legacy remote state is no longer supported. User must first
		// migrate with Terraform 0.11 or earlier.
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Legacy remote state not supported",
			"This working directory is configured for legacy remote state, which is no longer supported from Terraform v0.12 onwards. To migrate this environment, first run \"terraform init\" under a Terraform 0.11 release, and then upgrade Terraform again.",
		))
		return nil, diags
	}

	// This switch statement covers all the different combinations of
	// configuring new backends, updating previously-configured backends, etc.
	switch {
	// No configuration set at all. Pure local state.
	case c == nil && s.Backend.Empty():
		return nil, nil

	// We're unsetting a backend (moving from backend => local)
	case c == nil && !s.Backend.Empty():
		if !opts.Init {
			initReason := fmt.Sprintf(
				"Unsetting the previously set backend %q",
				s.Backend.Type)
			m.backendInitRequired(initReason)
			diags = diags.Append(errBackendInitRequired)
			return nil, diags
		}

		return m.backend_c_r_S(c, cHash, sMgr, true)

	// Configuring a backend for the first time.
	case c != nil && s.Backend.Empty():
		if !opts.Init {
			initReason := fmt.Sprintf(
				"Initial configuration of the requested backend %q",
				c.Type)
			m.backendInitRequired(initReason)
			diags = diags.Append(errBackendInitRequired)
			return nil, diags
		}

		return m.backend_C_r_s(c, cHash, sMgr)

	// Potentially changing a backend configuration
	case c != nil && !s.Backend.Empty():
		// If our configuration is the same, then we're just initializing
		// a previously configured remote backend.
		if !s.Backend.Empty() {
			storedHash := s.Backend.Hash
			if storedHash == cHash {
				return m.backend_C_r_S_unchanged(c, cHash, sMgr)
			}
		}

		if !opts.Init {
			initReason := fmt.Sprintf(
				"Backend configuration changed for %q",
				c.Type)
			m.backendInitRequired(initReason)
			diags = diags.Append(errBackendInitRequired)
			return nil, diags
		}

		log.Printf(
			"[WARN] command: backend config change! saved: %d, new: %d",
			s.Backend.Hash, cHash)
		return m.backend_C_r_S_changed(c, cHash, sMgr, true)

	default:
		diags = diags.Append(fmt.Errorf(
			"Unhandled backend configuration state. This is a bug. Please\n"+
				"report this error with the following information.\n\n"+
				"Config Nil: %v\n"+
				"Saved Backend Empty: %v\n",
			c == nil, s.Backend.Empty(),
		))
		return nil, diags
	}
}

//-------------------------------------------------------------------
// Backend Config Scenarios
//
// The functions below cover handling all the various scenarios that
// can exist when loading a backend. They are named in the format of
// "backend_C_R_S" where C, R, S may be upper or lowercase. Lowercase
// means it is false, uppercase means it is true. The full set of eight
// possible cases is handled.
//
// The fields are:
//
//   * C - Backend configuration is set and changed in TF files
//   * R - Legacy remote state is set
//   * S - Backend configuration is set in the state
//
//-------------------------------------------------------------------

// Unconfiguring a backend (moving from backend => local).
func (m *Meta) backend_c_r_S(c *configs.Backend, cHash int, sMgr *state.LocalState, output bool) (backend.Backend, tfdiags.Diagnostics) {
	s := sMgr.State()

	// Get the backend type for output
	backendType := s.Backend.Type

	m.Ui.Output(fmt.Sprintf(strings.TrimSpace(outputBackendMigrateLocal), s.Backend.Type))

	// Grab a purely local backend to get the local state if it exists
	localB, diags := m.Backend(&BackendOpts{ForceLocal: true})
	if diags.HasErrors() {
		return nil, diags
	}

	// Initialize the configured backend
	b, moreDiags := m.backend_C_r_S_unchanged(c, cHash, sMgr)
	diags = diags.Append(moreDiags)
	if moreDiags.HasErrors() {
		return nil, diags
	}

	// Perform the migration
	err := m.backendMigrateState(&backendMigrateOpts{
		OneType: s.Backend.Type,
		TwoType: "local",
		One:     b,
		Two:     localB,
	})
	if err != nil {
		diags = diags.Append(err)
		return nil, diags
	}

	// Remove the stored metadata
	s.Backend = nil
	if err := sMgr.WriteState(s); err != nil {
		diags = diags.Append(fmt.Errorf(strings.TrimSpace(errBackendClearSaved), err))
		return nil, diags
	}
	if err := sMgr.PersistState(); err != nil {
		diags = diags.Append(fmt.Errorf(strings.TrimSpace(errBackendClearSaved), err))
		return nil, diags
	}

	if output {
		m.Ui.Output(m.Colorize().Color(fmt.Sprintf(
			"[reset][green]\n\n"+
				strings.TrimSpace(successBackendUnset), backendType)))
	}

	// Return no backend
	return nil, diags
}

// Legacy remote state
func (m *Meta) backend_c_R_s(c *configs.Backend, sMgr *state.LocalState) (backend.Backend, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	m.Ui.Error(strings.TrimSpace(errBackendLegacy) + "\n")

	diags = diags.Append(fmt.Errorf("Cannot initialize legacy remote state"))
	return nil, diags
}

// Unsetting backend, saved backend, legacy remote state
func (m *Meta) backend_c_R_S(c *configs.Backend, cHash int, sMgr *state.LocalState) (backend.Backend, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	m.Ui.Error(strings.TrimSpace(errBackendLegacy) + "\n")

	diags = diags.Append(fmt.Errorf("Cannot initialize legacy remote state"))
	return nil, diags
}

// Configuring a backend for the first time with legacy remote state.
func (m *Meta) backend_C_R_s(c *configs.Backend, sMgr *state.LocalState) (backend.Backend, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	m.Ui.Error(strings.TrimSpace(errBackendLegacy) + "\n")

	diags = diags.Append(fmt.Errorf("Cannot initialize legacy remote state"))
	return nil, diags
}

// Configuring a backend for the first time.
func (m *Meta) backend_C_r_s(c *configs.Backend, cHash int, sMgr *state.LocalState) (backend.Backend, tfdiags.Diagnostics) {
	// Get the backend
	b, configVal, diags := m.backendInitFromConfig(c)
	if diags.HasErrors() {
		return nil, diags
	}

	// Grab a purely local backend to get the local state if it exists
	localB, localBDiags := m.Backend(&BackendOpts{ForceLocal: true})
	if localBDiags.HasErrors() {
		diags = diags.Append(localBDiags)
		return nil, diags
	}

	workspace := m.Workspace()

	localState, err := localB.StateMgr(workspace)
	if err != nil {
		diags = diags.Append(fmt.Errorf(errBackendLocalRead, err))
		return nil, diags
	}
	if err := localState.RefreshState(); err != nil {
		diags = diags.Append(fmt.Errorf(errBackendLocalRead, err))
		return nil, diags
	}

	// If the local state is not empty, we need to potentially do a
	// state migration to the new backend (with user permission), unless the
	// destination is also "local"
	if localS := localState.State(); !localS.Empty() {
		// Perform the migration
		err = m.backendMigrateState(&backendMigrateOpts{
			OneType: "local",
			TwoType: c.Type,
			One:     localB,
			Two:     b,
		})
		if err != nil {
			diags = diags.Append(err)
			return nil, diags
		}

		// we usually remove the local state after migration to prevent
		// confusion, but adding a default local backend block to the config
		// can get us here too. Don't delete our state if the old and new paths
		// are the same.
		erase := true
		if newLocalB, ok := b.(*backendlocal.Local); ok {
			if localB, ok := localB.(*backendlocal.Local); ok {
				if newLocalB.StatePath == localB.StatePath {
					erase = false
				}
			}
		}

		if erase {
			// We always delete the local state, unless that was our new state too.
			if err := localState.WriteState(nil); err != nil {
				diags = diags.Append(fmt.Errorf(errBackendMigrateLocalDelete, err))
				return nil, diags
			}
			if err := localState.PersistState(); err != nil {
				diags = diags.Append(fmt.Errorf(errBackendMigrateLocalDelete, err))
				return nil, diags
			}
		}
	}

	if m.stateLock {
		stateLocker := clistate.NewLocker(context.Background(), m.stateLockTimeout, m.Ui, m.Colorize())
		if err := stateLocker.Lock(sMgr, "backend from plan"); err != nil {
			diags = diags.Append(fmt.Errorf("Error locking state: %s", err))
			return nil, diags
		}
		defer stateLocker.Unlock(nil)
	}

	configJSON, err := ctyjson.Marshal(configVal, b.ConfigSchema().ImpliedType())
	if err != nil {
		diags = diags.Append(fmt.Errorf("Can't serialize backend configuration as JSON: %s", err))
		return nil, diags
	}

	// Store the metadata in our saved state location
	s := sMgr.State()
	if s == nil {
		s = terraform.NewState()
	}
	s.Backend = &terraform.BackendState{
		Type:      c.Type,
		ConfigRaw: json.RawMessage(configJSON),
		Hash:      cHash,
	}

	if err := sMgr.WriteState(s); err != nil {
		diags = diags.Append(fmt.Errorf(errBackendWriteSaved, err))
		return nil, diags
	}
	if err := sMgr.PersistState(); err != nil {
		diags = diags.Append(fmt.Errorf(errBackendWriteSaved, err))
		return nil, diags
	}

	m.Ui.Output(m.Colorize().Color(fmt.Sprintf(
		"[reset][green]\n"+strings.TrimSpace(successBackendSet), s.Backend.Type)))

	// Return the backend
	return b, diags
}

// Changing a previously saved backend.
func (m *Meta) backend_C_r_S_changed(c *configs.Backend, cHash int, sMgr *state.LocalState, output bool) (backend.Backend, tfdiags.Diagnostics) {
	if output {
		// Notify the user
		m.Ui.Output(m.Colorize().Color(fmt.Sprintf(
			"[reset]%s\n\n",
			strings.TrimSpace(outputBackendReconfigure))))
	}

	// Get the old state
	s := sMgr.State()

	// Get the backend
	b, configVal, diags := m.backendInitFromConfig(c)
	if diags.HasErrors() {
		return nil, diags
	}

	// no need to confuse the user if the backend types are the same
	if s.Backend.Type != c.Type {
		m.Ui.Output(strings.TrimSpace(fmt.Sprintf(outputBackendMigrateChange, s.Backend.Type, c.Type)))
	}

	// Grab the existing backend
	oldB, oldBDiags := m.backend_C_r_S_unchanged(c, cHash, sMgr)
	diags = diags.Append(oldBDiags)
	if oldBDiags.HasErrors() {
		return nil, diags
	}

	// Perform the migration
	err := m.backendMigrateState(&backendMigrateOpts{
		OneType: s.Backend.Type,
		TwoType: c.Type,
		One:     oldB,
		Two:     b,
	})
	if err != nil {
		diags = diags.Append(err)
		return nil, diags
	}

	if m.stateLock {
		stateLocker := clistate.NewLocker(context.Background(), m.stateLockTimeout, m.Ui, m.Colorize())
		if err := stateLocker.Lock(sMgr, "backend from plan"); err != nil {
			diags = diags.Append(fmt.Errorf("Error locking state: %s", err))
			return nil, diags
		}
		defer stateLocker.Unlock(nil)
	}

	configJSON, err := ctyjson.Marshal(configVal, b.ConfigSchema().ImpliedType())
	if err != nil {
		diags = diags.Append(fmt.Errorf("Can't serialize backend configuration as JSON: %s", err))
		return nil, diags
	}

	// Update the backend state
	s = sMgr.State()
	if s == nil {
		s = terraform.NewState()
	}
	s.Backend = &terraform.BackendState{
		Type:      c.Type,
		ConfigRaw: json.RawMessage(configJSON),
		Hash:      cHash,
	}

	if err := sMgr.WriteState(s); err != nil {
		diags = diags.Append(fmt.Errorf(errBackendWriteSaved, err))
		return nil, diags
	}
	if err := sMgr.PersistState(); err != nil {
		diags = diags.Append(fmt.Errorf(errBackendWriteSaved, err))
		return nil, diags
	}

	if output {
		m.Ui.Output(m.Colorize().Color(fmt.Sprintf(
			"[reset][green]\n"+strings.TrimSpace(successBackendSet), s.Backend.Type)))
	}

	return b, diags
}

// Initiailizing an unchanged saved backend
func (m *Meta) backend_C_r_S_unchanged(c *configs.Backend, cHash int, sMgr *state.LocalState) (backend.Backend, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	s := sMgr.State()

	// it's possible for a backend to be unchanged, and the config itself to
	// have changed by moving a parameter from the config to `-backend-config`
	// In this case we only need to update the Hash.
	if c != nil && s.Backend.Hash != cHash {
		s.Backend.Hash = cHash
		if err := sMgr.WriteState(s); err != nil {
			diags = diags.Append(err)
			return nil, diags
		}
	}

	// Get the backend
	f := backendinit.Backend(s.Backend.Type)
	if f == nil {
		diags = diags.Append(fmt.Errorf(strings.TrimSpace(errBackendSavedUnknown), s.Backend.Type))
		return nil, diags
	}
	b := f()

	// The configuration saved in the working directory state file is used
	// in this case, since it will contain any additional values that
	// were provided via -backend-config arguments on terraform init.
	schema := b.ConfigSchema()
	configVal, err := s.Backend.Config(schema)
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to decode current backend config",
			fmt.Sprintf("The backend configuration created by the most recent run of \"terraform init\" could not be decoded: %s. The configuration may have been initialized by an earlier version that used an incompatible configuration structure. Run \"terraform init -reconfigure\" to force re-initialization of the backend.", err),
		))
		return nil, diags
	}

	// Validate the config and then configure the backend
	validDiags := b.ValidateConfig(configVal)
	diags = diags.Append(validDiags)
	if validDiags.HasErrors() {
		return nil, diags
	}
	configDiags := b.Configure(configVal)
	diags = diags.Append(configDiags)
	if configDiags.HasErrors() {
		return nil, diags
	}

	return b, diags
}

// Initiailizing a changed saved backend with legacy remote state.
func (m *Meta) backend_C_R_S_changed(c *configs.Backend, sMgr *state.LocalState) (backend.Backend, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	m.Ui.Error(strings.TrimSpace(errBackendLegacy) + "\n")

	diags = diags.Append(fmt.Errorf("Cannot initialize legacy remote state"))
	return nil, diags
}

// Initiailizing an unchanged saved backend with legacy remote state.
func (m *Meta) backend_C_R_S_unchanged(c *configs.Backend, sMgr *state.LocalState, output bool) (backend.Backend, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	m.Ui.Error(strings.TrimSpace(errBackendLegacy) + "\n")

	diags = diags.Append(fmt.Errorf("Cannot initialize legacy remote state"))
	return nil, diags
}

//-------------------------------------------------------------------
// Reusable helper functions for backend management
//-------------------------------------------------------------------

func (m *Meta) backendInitFromConfig(c *configs.Backend) (backend.Backend, cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	// Get the backend
	f := backendinit.Backend(c.Type)
	if f == nil {
		diags = diags.Append(fmt.Errorf(strings.TrimSpace(errBackendNewUnknown), c.Type))
		return nil, cty.NilVal, diags
	}
	b := f()

	schema := b.ConfigSchema()
	decSpec := schema.NoneRequired().DecoderSpec()
	configVal, hclDiags := hcldec.Decode(c.Config, decSpec, nil)
	diags = diags.Append(hclDiags)
	if hclDiags.HasErrors() {
		return nil, cty.NilVal, diags
	}

	// TODO: test
	if m.Input() {
		var err error
		configVal, err = m.inputForSchema(configVal, schema)
		if err != nil {
			diags = diags.Append(fmt.Errorf("Error asking for input to configure backend %q: %s", c.Type, err))
		}
	}

	validateDiags := b.ValidateConfig(configVal)
	diags = diags.Append(validateDiags.InConfigBody(c.Config))
	if validateDiags.HasErrors() {
		return nil, cty.NilVal, diags
	}

	configureDiags := b.Configure(configVal)
	diags = diags.Append(configureDiags.InConfigBody(c.Config))

	return b, configVal, diags
}

func (m *Meta) backendInitFromSaved(s *terraform.BackendState) (backend.Backend, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	// Get the backend
	f := backendinit.Backend(s.Type)
	if f == nil {
		diags = diags.Append(fmt.Errorf(strings.TrimSpace(errBackendSavedUnknown), s.Type))
		return nil, diags
	}
	b := f()

	schema := b.ConfigSchema()
	configVal, err := s.Config(schema)
	if err != nil {
		diags = diags.Append(errwrap.Wrapf("saved backend configuration is invalid: {{err}}", err))
		return nil, diags
	}

	validateDiags := b.ValidateConfig(configVal)
	diags = diags.Append(validateDiags)
	if validateDiags.HasErrors() {
		return nil, diags
	}

	configureDiags := b.Configure(configVal)
	diags = diags.Append(configureDiags)

	return b, diags
}

func (m *Meta) backendInitRequired(reason string) {
	m.Ui.Output(m.Colorize().Color(fmt.Sprintf(
		"[reset]"+strings.TrimSpace(errBackendInit)+"\n", reason)))
}

//-------------------------------------------------------------------
// Output constants and initialization code
//-------------------------------------------------------------------

// errBackendInitRequired is the final error message shown when reinit
// is required for some reason. The error message includes the reason.
var errBackendInitRequired = errors.New(
	"Initialization required. Please see the error message above.")

const errBackendLegacyConfig = `
One or more errors occurred while configuring the legacy remote state.
If fixing these errors requires changing your remote state configuration,
you must switch your configuration to the new remote backend configuration.
You can learn more about remote backends at the URL below:

https://www.terraform.io/docs/backends/index.html

The error(s) configuring the legacy remote state:

%s
`

const errBackendLegacyUnknown = `
The legacy remote state type %q could not be found.

Terraform 0.9.0 shipped with backwards compatibility for all built-in
legacy remote state types. This error may mean that you were using a
custom Terraform build that perhaps supported a different type of
remote state.

Please check with the creator of the remote state above and try again.
`

const errBackendLocalRead = `
Error reading local state: %s

Terraform is trying to read your local state to determine if there is
state to migrate to your newly configured backend. Terraform can't continue
without this check because that would risk losing state. Please resolve the
error above and try again.
`

const errBackendMigrateLocalDelete = `
Error deleting local state after migration: %s

Your local state is deleted after successfully migrating it to the newly
configured backend. As part of the deletion process, a backup is made at
the standard backup path unless explicitly asked not to. To cleanly operate
with a backend, we must delete the local state file. Please resolve the
issue above and retry the command.
`

const errBackendMigrateNew = `
Error migrating local state to backend: %s

Your local state remains intact and unmodified. Please resolve the error
above and try again.
`

const errBackendNewConfig = `
Error configuring the backend %q: %s

Please update the configuration in your Terraform files to fix this error
then run this command again.
`

const errBackendNewRead = `
Error reading newly configured backend state: %s

Terraform is trying to read the state from your newly configured backend
to determine the copy process for your existing state. Backends are expected
to not error even if there is no state yet written. Please resolve the
error above and try again.
`

const errBackendNewUnknown = `
The backend %q could not be found.

This is the backend specified in your Terraform configuration file.
This error could be a simple typo in your configuration, but it can also
be caused by using a Terraform version that doesn't support the specified
backend type. Please check your configuration and your Terraform version.

If you'd like to run Terraform and store state locally, you can fix this
error by removing the backend configuration from your configuration.
`

const errBackendRemoteRead = `
Error reading backend state: %s

Terraform is trying to read the state from your configured backend to
determine if there is any migration steps necessary. Terraform can't continue
without this check because that would risk losing state. Please resolve the
error above and try again.
`

const errBackendSavedConfig = `
Error configuring the backend %q: %s

Please update the configuration in your Terraform files to fix this error.
If you'd like to update the configuration interactively without storing
the values in your configuration, run "terraform init".
`

const errBackendSavedUnsetConfig = `
Error configuring the existing backend %q: %s

Terraform must configure the existing backend in order to copy the state
from the existing backend, as requested. Please resolve the error and try
again. If you choose to not copy the existing state, Terraform will not
configure the backend. If the configuration is invalid, please update your
Terraform configuration with proper configuration for this backend first
before unsetting the backend.
`

const errBackendSavedUnknown = `
The backend %q could not be found.

This is the backend that this Terraform environment is configured to use
both in your configuration and saved locally as your last-used backend.
If it isn't found, it could mean an alternate version of Terraform was
used with this configuration. Please use the proper version of Terraform that
contains support for this backend.

If you'd like to force remove this backend, you must update your configuration
to not use the backend and run "terraform init" (or any other command) again.
`

const errBackendClearLegacy = `
Error clearing the legacy remote state configuration: %s

Terraform completed configuring your backend. It is now safe to remove
the legacy remote state configuration, but an error occurred while trying
to do so. Please look at the error above, resolve it, and try again.
`

const errBackendClearSaved = `
Error clearing the backend configuration: %s

Terraform removes the saved backend configuration when you're removing a
configured backend. This must be done so future Terraform runs know to not
use the backend configuration. Please look at the error above, resolve it,
and try again.
`

const errBackendInit = `
[reset][bold][yellow]Backend reinitialization required. Please run "terraform init".[reset]
[yellow]Reason: %s

The "backend" is the interface that Terraform uses to store state,
perform operations, etc. If this message is showing up, it means that the
Terraform configuration you're using is using a custom configuration for
the Terraform backend.

Changes to backend configurations require reinitialization. This allows
Terraform to setup the new configuration, copy existing state, etc. This is
only done during "terraform init". Please run that command now then try again.

If the change reason above is incorrect, please verify your configuration
hasn't changed and try again. At this point, no changes to your existing
configuration or state have been made.
`

const errBackendWriteSaved = `
Error saving the backend configuration: %s

Terraform saves the complete backend configuration in a local file for
configuring the backend on future operations. This cannot be disabled. Errors
are usually due to simple file permission errors. Please look at the error
above, resolve it, and try again.
`

const errBackendPlanBoth = `
The plan file contained both a legacy remote state and backend configuration.
This is not allowed. Please recreate the plan file with the latest version of
Terraform.
`

const errBackendPlanLineageDiff = `
The plan file contains a state with a differing lineage than the current
state. By continuing, your current state would be overwritten by the state
in the plan. Please either update the plan with the latest state or delete
your current state and try again.

"Lineage" is a unique identifier generated only once on the creation of
a new, empty state. If these values differ, it means they were created new
at different times. Therefore, Terraform must assume that they're completely
different states.

The most common cause of seeing this error is using a plan that was
created against a different state. Perhaps the plan is very old and the
state has since been recreated, or perhaps the plan was against a completely
different infrastructure.
`

const errBackendPlanStateFlag = `
The -state and -state-out flags cannot be set with a plan that has a remote
state. The plan itself contains the configuration for the remote backend to
store state. The state will be written there for consistency.

If you wish to change this behavior, please create a plan from local state.
You may use the state flags with plans from local state to affect where
the final state is written.
`

const errBackendPlanOlder = `
This plan was created against an older state than is current. Please create
a new plan file against the latest state and try again.

Terraform doesn't allow you to run plans that were created from older
states since it doesn't properly represent the latest changes Terraform
may have made, and can result in unsafe behavior.

Plan Serial:    %[1]d
Current Serial: %[2]d
`

const outputBackendMigrateChange = `
Terraform detected that the backend type changed from %q to %q.
`

const outputBackendMigrateLegacy = `
Terraform detected legacy remote state.
`

const outputBackendMigrateLocal = `
Terraform has detected you're unconfiguring your previously set %q backend.
`

const outputBackendConfigureWithLegacy = `
[reset][bold]New backend configuration detected with legacy remote state![reset]

Terraform has detected that you're attempting to configure a new backend.
At the same time, legacy remote state configuration was found. Terraform will
first configure the new backend, and then ask if you'd like to migrate
your remote state to the new backend.
`

const outputBackendReconfigure = `
[reset][bold]Backend configuration changed![reset]

Terraform has detected that the configuration specified for the backend
has changed. Terraform will now check for existing state in the backends.
`

const outputBackendSavedWithLegacy = `
[reset][bold]Legacy remote state was detected![reset]

Terraform has detected you still have legacy remote state enabled while
also having a backend configured. Terraform will now ask if you want to
migrate your legacy remote state data to the configured backend.
`

const outputBackendSavedWithLegacyChanged = `
[reset][bold]Legacy remote state was detected while also changing your current backend!reset]

Terraform has detected that you have legacy remote state, a configured
current backend, and you're attempting to reconfigure your backend. To handle
all of these changes, Terraform will first reconfigure your backend. After
this, Terraform will handle optionally copying your legacy remote state
into the newly configured backend.
`

const outputBackendUnsetWithLegacy = `
[reset][bold]Detected a request to unset the backend with legacy remote state present![reset]

Terraform has detected that you're attempting to unset a previously configured
backend (by not having the "backend" configuration set in your Terraform files).
At the same time, legacy remote state was detected. To handle this complex
scenario, Terraform will first unset your configured backend, and then
ask you how to handle the legacy remote state. This will be multi-step
process.
`

const successBackendLegacyUnset = `
Terraform has successfully migrated from legacy remote state to your
configured backend (%q).
`

const successBackendReconfigureWithLegacy = `
Terraform has successfully reconfigured your backend and migrate
from legacy remote state to the new backend.
`

const successBackendUnset = `
Successfully unset the backend %q. Terraform will now operate locally.
`

const successBackendSet = `
Successfully configured the backend %q! Terraform will automatically
use this backend unless the backend configuration changes.
`

const errBackendLegacy = `
This working directory is configured to use the legacy remote state features
from Terraform 0.8 or earlier. Remote state changed significantly in Terraform
0.9 and the automatic upgrade mechanism has now been removed.

To upgrade, please first use Terraform v0.11 to complete the upgrade steps:
    https://www.terraform.io/docs/backends/legacy-0-8.html
`
