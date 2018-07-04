package command

// This file contains all the Backend-related function calls on Meta,
// exported and private.

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"strings"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/hcl"
	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/command/clistate"
	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/mapstructure"

	backendInit "github.com/hashicorp/terraform/backend/init"
	backendLocal "github.com/hashicorp/terraform/backend/local"
)

// BackendOpts are the options used to initialize a backend.Backend.
type BackendOpts struct {
	// Module is the root module from which we will extract the terraform and
	// backend configuration.
	Config *config.Config

	// ConfigFile is a path to a file that contains configuration that
	// is merged directly into the backend configuration when loaded
	// from a file.
	ConfigFile string

	// ConfigExtra is extra configuration to merge into the backend
	// configuration after the extra file above.
	ConfigExtra map[string]interface{}

	// Plan is a plan that is being used. If this is set, the backend
	// configuration and output configuration will come from this plan.
	Plan *terraform.Plan

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
func (m *Meta) Backend(opts *BackendOpts) (backend.Enhanced, error) {
	// If no opts are set, then initialize
	if opts == nil {
		opts = &BackendOpts{}
	}

	// Initialize a backend from the config unless we're forcing a purely
	// local operation.
	var b backend.Backend
	if !opts.ForceLocal {
		var err error

		// If we have a plan then, we get the the backend from there. Otherwise,
		// the backend comes from the configuration.
		if opts.Plan != nil {
			b, err = m.backendFromPlan(opts)
		} else {
			b, err = m.backendFromConfig(opts)
		}
		if err != nil {
			return nil, err
		}

		log.Printf("[INFO] command: backend initialized: %T", b)
	}

	// Setup the CLI opts we pass into backends that support it.
	cliOpts := &backend.CLIOpts{
		CLI:                 m.Ui,
		CLIColor:            m.Colorize(),
		StatePath:           m.statePath,
		StateOutPath:        m.stateOutPath,
		StateBackupPath:     m.backupPath,
		ContextOpts:         m.contextOpts(),
		Input:               m.Input(),
		RunningInAutomation: m.RunningInAutomation,
	}

	// Don't validate if we have a plan. Validation is normally harmless here,
	// but validation requires interpolation, and `file()` function calls may
	// not have the original files in the current execution context.
	cliOpts.Validation = opts.Plan == nil

	// If the backend supports CLI initialization, do it.
	if cli, ok := b.(backend.CLI); ok {
		if err := cli.CLIInit(cliOpts); err != nil {
			return nil, fmt.Errorf(
				"Error initializing backend %T: %s\n\n"+
					"This is a bug, please report it to the backend developer",
				b, err)
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
	local := &backendLocal.Local{Backend: b}
	if err := local.CLIInit(cliOpts); err != nil {
		// Local backend isn't allowed to fail. It would be a bug.
		panic(err)
	}

	return local, nil
}

// IsLocalBackend returns true if the backend is a local backend. We use this
// for some checks that require a remote backend.
func (m *Meta) IsLocalBackend(b backend.Backend) bool {
	// Is it a local backend?
	bLocal, ok := b.(*backendLocal.Local)

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
func (m *Meta) Operation() *backend.Operation {
	return &backend.Operation{
		PlanOutBackend:   m.backendState,
		Targets:          m.targets,
		UIIn:             m.UIInput(),
		UIOut:            m.Ui,
		Workspace:        m.Workspace(),
		LockState:        m.stateLock,
		StateLockTimeout: m.stateLockTimeout,
	}
}

// backendConfig returns the local configuration for the backend
func (m *Meta) backendConfig(opts *BackendOpts) (*config.Backend, error) {
	if opts.Config == nil {
		// check if the config was missing, or just not required
		conf, err := m.Config(".")
		if err != nil {
			return nil, err
		}

		if conf == nil {
			log.Println("[INFO] command: no config, returning nil")
			return nil, nil
		}

		log.Println("[WARN] BackendOpts.Config not set, but config found")
		opts.Config = conf
	}

	c := opts.Config

	// If there is no Terraform configuration block, no backend config
	if c.Terraform == nil {
		log.Println("[INFO] command: empty terraform config, returning nil")
		return nil, nil
	}

	// Get the configuration for the backend itself.
	backend := c.Terraform.Backend
	if backend == nil {
		log.Println("[INFO] command: empty backend config, returning nil")
		return nil, nil
	}

	// If we have a config file set, load that and merge.
	if opts.ConfigFile != "" {
		log.Printf(
			"[DEBUG] command: loading extra backend config from: %s",
			opts.ConfigFile)
		rc, err := m.backendConfigFile(opts.ConfigFile)
		if err != nil {
			return nil, fmt.Errorf(
				"Error loading extra configuration file for backend: %s", err)
		}

		// Merge in the configuration
		backend.RawConfig = backend.RawConfig.Merge(rc)
	}

	// If we have extra config values, merge that
	if len(opts.ConfigExtra) > 0 {
		log.Printf(
			"[DEBUG] command: adding extra backend config from CLI")
		rc, err := config.NewRawConfig(opts.ConfigExtra)
		if err != nil {
			return nil, fmt.Errorf(
				"Error adding extra backend configuration from CLI: %s", err)
		}

		// Merge in the configuration
		backend.RawConfig = backend.RawConfig.Merge(rc)
	}

	// Validate the backend early. We have to do this before the normal
	// config validation pass since backend loading happens earlier.
	if errs := backend.Validate(); len(errs) > 0 {
		return nil, multierror.Append(nil, errs...)
	}

	// Return the configuration which may or may not be set
	return backend, nil
}

// backendConfigFile loads the extra configuration to merge with the
// backend configuration from an extra file if specified by
// BackendOpts.ConfigFile.
func (m *Meta) backendConfigFile(path string) (*config.RawConfig, error) {
	// Read the file
	d, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Parse it
	hclRoot, err := hcl.Parse(string(d))
	if err != nil {
		return nil, err
	}

	// Decode it
	var c map[string]interface{}
	if err := hcl.DecodeObject(&c, hclRoot); err != nil {
		return nil, err
	}

	return config.NewRawConfig(c)
}

// backendFromConfig returns the initialized (not configured) backend
// directly from the config/state..
//
// This function handles any edge cases around backend config loading. For
// example: legacy remote state, new config changes, backend type changes,
// etc.
//
// This function may query the user for input unless input is disabled, in
// which case this function will error.
func (m *Meta) backendFromConfig(opts *BackendOpts) (backend.Backend, error) {
	// Get the local backend configuration.
	c, err := m.backendConfig(opts)
	if err != nil {
		return nil, fmt.Errorf("Error loading backend config: %s", err)
	}

	// cHash defaults to zero unless c is set
	var cHash uint64
	if c != nil {
		// We need to rehash to get the value since we may have merged the
		// config with an extra ConfigFile. We don't do this when merging
		// because we do want the ORIGINAL value on c so that we store
		// that to not detect drift. This is covered in tests.
		cHash = c.Rehash()
	}

	// Get the path to where we store a local cache of backend configuration
	// if we're using a remote backend. This may not yet exist which means
	// we haven't used a non-local backend before. That is okay.
	statePath := filepath.Join(m.DataDir(), DefaultStateFilename)
	sMgr := &state.LocalState{Path: statePath}
	if err := sMgr.RefreshState(); err != nil {
		return nil, fmt.Errorf("Error loading state: %s", err)
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

	// This giant switch statement covers all eight possible combinations
	// of state settings between: configuring new backends, saved (previously-
	// configured) backends, and legacy remote state.
	switch {
	// No configuration set at all. Pure local state.
	case c == nil && s.Remote.Empty() && s.Backend.Empty():
		return nil, nil

	// We're unsetting a backend (moving from backend => local)
	case c == nil && s.Remote.Empty() && !s.Backend.Empty():
		if !opts.Init {
			initReason := fmt.Sprintf(
				"Unsetting the previously set backend %q",
				s.Backend.Type)
			m.backendInitRequired(initReason)
			return nil, errBackendInitRequired
		}

		return m.backend_c_r_S(c, sMgr, true)

	// We have a legacy remote state configuration but no new backend config
	case c == nil && !s.Remote.Empty() && s.Backend.Empty():
		return m.backend_c_R_s(c, sMgr)

	// We have a legacy remote state configuration simultaneously with a
	// saved backend configuration while at the same time disabling backend
	// configuration.
	//
	// This is a naturally impossible case: Terraform will never put you
	// in this state, though it is theoretically possible through manual edits
	case c == nil && !s.Remote.Empty() && !s.Backend.Empty():
		if !opts.Init {
			initReason := fmt.Sprintf(
				"Unsetting the previously set backend %q",
				s.Backend.Type)
			m.backendInitRequired(initReason)
			return nil, errBackendInitRequired
		}

		return m.backend_c_R_S(c, sMgr)

	// Configuring a backend for the first time.
	case c != nil && s.Remote.Empty() && s.Backend.Empty():
		if !opts.Init {
			initReason := fmt.Sprintf(
				"Initial configuration of the requested backend %q",
				c.Type)
			m.backendInitRequired(initReason)
			return nil, errBackendInitRequired
		}

		return m.backend_C_r_s(c, sMgr)

	// Potentially changing a backend configuration
	case c != nil && s.Remote.Empty() && !s.Backend.Empty():
		// If our configuration is the same, then we're just initializing
		// a previously configured remote backend.
		if !s.Backend.Empty() {
			hash := s.Backend.Hash
			// on init we need an updated hash containing any extra options
			// that were added after merging.
			if opts.Init {
				hash = s.Backend.Rehash()
			}
			if hash == cHash {
				return m.backend_C_r_S_unchanged(c, sMgr)
			}
		}

		if !opts.Init {
			initReason := fmt.Sprintf(
				"Backend configuration changed for %q",
				c.Type)
			m.backendInitRequired(initReason)
			return nil, errBackendInitRequired
		}

		log.Printf(
			"[WARN] command: backend config change! saved: %d, new: %d",
			s.Backend.Hash, cHash)
		return m.backend_C_r_S_changed(c, sMgr, true)

	// Configuring a backend for the first time while having legacy
	// remote state. This is very possible if a Terraform user configures
	// a backend prior to ever running Terraform on an old state.
	case c != nil && !s.Remote.Empty() && s.Backend.Empty():
		if !opts.Init {
			initReason := fmt.Sprintf(
				"Initial configuration for backend %q",
				c.Type)
			m.backendInitRequired(initReason)
			return nil, errBackendInitRequired
		}

		return m.backend_C_R_s(c, sMgr)

	// Configuring a backend with both a legacy remote state set
	// and a pre-existing backend saved.
	case c != nil && !s.Remote.Empty() && !s.Backend.Empty():
		// If the hashes are the same, we have a legacy remote state with
		// an unchanged stored backend state.
		hash := s.Backend.Hash
		if opts.Init {
			hash = s.Backend.Rehash()
		}
		if hash == cHash {
			if !opts.Init {
				initReason := fmt.Sprintf(
					"Legacy remote state found with configured backend %q",
					c.Type)
				m.backendInitRequired(initReason)
				return nil, errBackendInitRequired
			}

			return m.backend_C_R_S_unchanged(c, sMgr, true)
		}

		if !opts.Init {
			initReason := fmt.Sprintf(
				"Reconfiguring the backend %q",
				c.Type)
			m.backendInitRequired(initReason)
			return nil, errBackendInitRequired
		}

		// We have change in all three
		return m.backend_C_R_S_changed(c, sMgr)
	default:
		// This should be impossible since all state possibilties are
		// tested above, but we need a default case anyways and we should
		// protect against the scenario where a case is somehow removed.
		return nil, fmt.Errorf(
			"Unhandled backend configuration state. This is a bug. Please\n"+
				"report this error with the following information.\n\n"+
				"Config Nil: %v\n"+
				"Saved Backend Empty: %v\n"+
				"Legacy Remote Empty: %v\n",
			c == nil, s.Backend.Empty(), s.Remote.Empty())
	}
}

// backendFromPlan loads the backend from a given plan file.
func (m *Meta) backendFromPlan(opts *BackendOpts) (backend.Backend, error) {
	// Precondition check
	if opts.Plan == nil {
		panic("plan should not be nil")
	}

	// We currently don't allow "-state" to be specified.
	if m.statePath != "" {
		return nil, fmt.Errorf(
			"State path cannot be specified with a plan file. The plan itself contains\n" +
				"the state to use. If you wish to change that, please create a new plan\n" +
				"and specify the state path when creating the plan.")
	}

	planBackend := opts.Plan.Backend
	planState := opts.Plan.State
	if planState == nil {
		// The state can be nil, we just have to make it empty for the logic
		// in this function.
		planState = terraform.NewState()
	}

	// Validation only for non-local plans
	local := planState.Remote.Empty() && planBackend.Empty()
	if !local {
		// We currently don't allow "-state-out" to be specified.
		if m.stateOutPath != "" {
			return nil, fmt.Errorf(strings.TrimSpace(errBackendPlanStateFlag))
		}
	}

	/*
		// Determine the path where we'd be writing state
		path := DefaultStateFilename
		if !planState.Remote.Empty() || !planBackend.Empty() {
			path = filepath.Join(m.DataDir(), DefaultStateFilename)
		}

		// If the path exists, then we need to verify we're writing the same
		// state lineage. If the path doesn't exist that's okay.
		_, err := os.Stat(path)
		if err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("Error checking state destination: %s", err)
		}
		if err == nil {
			// The file exists, we need to read it and compare
			if err := m.backendFromPlan_compareStates(state, path); err != nil {
				return nil, err
			}
		}
	*/

	// If we have a stateOutPath, we must also specify it as the
	// input path so we can check it properly. We restore it after this
	// function exits.
	original := m.statePath
	m.statePath = m.stateOutPath
	defer func() { m.statePath = original }()

	var b backend.Backend
	var err error
	switch {
	// No remote state at all, all local
	case planState.Remote.Empty() && planBackend.Empty():
		log.Printf("[INFO] command: initializing local backend from plan (not set)")

		// Get the local backend
		b, err = m.Backend(&BackendOpts{ForceLocal: true})

	// New backend configuration set
	case planState.Remote.Empty() && !planBackend.Empty():
		log.Printf(
			"[INFO] command: initializing backend from plan: %s",
			planBackend.Type)

		b, err = m.backendInitFromSaved(planBackend)

	// Legacy remote state set
	case !planState.Remote.Empty() && planBackend.Empty():
		log.Printf(
			"[INFO] command: initializing legacy remote backend from plan: %s",
			planState.Remote.Type)

		// Write our current state to an inmemory state just so that we
		// have it in the format of state.State
		inmem := &state.InmemState{}
		inmem.WriteState(planState)

		// Get the backend through the normal means of legacy state
		b, err = m.backend_c_R_s(nil, inmem)

	// Both set, this can't happen in a plan.
	case !planState.Remote.Empty() && !planBackend.Empty():
		return nil, fmt.Errorf(strings.TrimSpace(errBackendPlanBoth))
	}

	// If we had an error, return that
	if err != nil {
		return nil, err
	}

	env := m.Workspace()

	// Get the state so we can determine the effect of using this plan
	realMgr, err := b.State(env)
	if err != nil {
		return nil, fmt.Errorf("Error reading state: %s", err)
	}

	if m.stateLock {
		stateLocker := clistate.NewLocker(context.Background(), m.stateLockTimeout, m.Ui, m.Colorize())
		if err := stateLocker.Lock(realMgr, "backend from plan"); err != nil {
			return nil, fmt.Errorf("Error locking state: %s", err)
		}
		defer stateLocker.Unlock(nil)
	}

	if err := realMgr.RefreshState(); err != nil {
		return nil, fmt.Errorf("Error reading state: %s", err)
	}
	real := realMgr.State()
	if real != nil {
		// If they're not the same lineage, don't allow this
		if !real.SameLineage(planState) {
			return nil, fmt.Errorf(strings.TrimSpace(errBackendPlanLineageDiff))
		}

		// Compare ages
		comp, err := real.CompareAges(planState)
		if err != nil {
			return nil, fmt.Errorf("Error comparing state ages for safety: %s", err)
		}
		switch comp {
		case terraform.StateAgeEqual:
			// State ages are equal, this is perfect

		case terraform.StateAgeReceiverOlder:
			// Real state is somehow older, this is okay.

		case terraform.StateAgeReceiverNewer:
			// If we have an older serial it is a problem but if we have a
			// differing serial but are still identical, just let it through.
			if real.Equal(planState) {
				log.Printf(
					"[WARN] command: state in plan has older serial, but Equal is true")
				break
			}

			// The real state is newer, this is not allowed.
			return nil, fmt.Errorf(
				strings.TrimSpace(errBackendPlanOlder),
				planState.Serial, real.Serial)
		}
	}

	// Write the state
	newState := opts.Plan.State.DeepCopy()
	if newState != nil {
		newState.Remote = nil
		newState.Backend = nil
	}

	// realMgr locked above
	if err := realMgr.WriteState(newState); err != nil {
		return nil, fmt.Errorf("Error writing state: %s", err)
	}
	if err := realMgr.PersistState(); err != nil {
		return nil, fmt.Errorf("Error writing state: %s", err)
	}

	return b, nil
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
func (m *Meta) backend_c_r_S(
	c *config.Backend, sMgr state.State, output bool) (backend.Backend, error) {
	s := sMgr.State()

	// Get the backend type for output
	backendType := s.Backend.Type

	m.Ui.Output(fmt.Sprintf(strings.TrimSpace(outputBackendMigrateLocal), s.Backend.Type))

	// Grab a purely local backend to get the local state if it exists
	localB, err := m.Backend(&BackendOpts{ForceLocal: true})
	if err != nil {
		return nil, fmt.Errorf(strings.TrimSpace(errBackendLocalRead), err)
	}

	// Initialize the configured backend
	b, err := m.backend_C_r_S_unchanged(c, sMgr)
	if err != nil {
		return nil, fmt.Errorf(
			strings.TrimSpace(errBackendSavedUnsetConfig), s.Backend.Type, err)
	}

	// Perform the migration
	err = m.backendMigrateState(&backendMigrateOpts{
		OneType: s.Backend.Type,
		TwoType: "local",
		One:     b,
		Two:     localB,
	})
	if err != nil {
		return nil, err
	}

	// Remove the stored metadata
	s.Backend = nil
	if err := sMgr.WriteState(s); err != nil {
		return nil, fmt.Errorf(strings.TrimSpace(errBackendClearSaved), err)
	}
	if err := sMgr.PersistState(); err != nil {
		return nil, fmt.Errorf(strings.TrimSpace(errBackendClearSaved), err)
	}

	if output {
		m.Ui.Output(m.Colorize().Color(fmt.Sprintf(
			"[reset][green]\n\n"+
				strings.TrimSpace(successBackendUnset), backendType)))
	}

	// Return no backend
	return nil, nil
}

// Legacy remote state
func (m *Meta) backend_c_R_s(
	c *config.Backend, sMgr state.State) (backend.Backend, error) {
	s := sMgr.State()

	// Warn the user
	m.Ui.Warn(strings.TrimSpace(warnBackendLegacy) + "\n")

	// We need to convert the config to map[string]interface{} since that
	// is what the backends expect.
	var configMap map[string]interface{}
	if err := mapstructure.Decode(s.Remote.Config, &configMap); err != nil {
		return nil, fmt.Errorf("Error configuring remote state: %s", err)
	}

	// Create the config
	rawC, err := config.NewRawConfig(configMap)
	if err != nil {
		return nil, fmt.Errorf("Error configuring remote state: %s", err)
	}
	config := terraform.NewResourceConfig(rawC)

	// Get the backend
	f := backendInit.Backend(s.Remote.Type)
	if f == nil {
		return nil, fmt.Errorf(strings.TrimSpace(errBackendLegacyUnknown), s.Remote.Type)
	}
	b := f()

	// Configure
	if err := b.Configure(config); err != nil {
		return nil, fmt.Errorf(errBackendLegacyConfig, err)
	}

	return b, nil
}

// Unsetting backend, saved backend, legacy remote state
func (m *Meta) backend_c_R_S(
	c *config.Backend, sMgr state.State) (backend.Backend, error) {
	// Notify the user
	m.Ui.Output(m.Colorize().Color(fmt.Sprintf(
		"[reset]%s\n\n",
		strings.TrimSpace(outputBackendUnsetWithLegacy))))

	// Get the backend type for later
	backendType := sMgr.State().Backend.Type

	// First, perform the configured => local tranasition
	if _, err := m.backend_c_r_S(c, sMgr, false); err != nil {
		return nil, err
	}

	// Grab a purely local backend
	localB, err := m.Backend(&BackendOpts{ForceLocal: true})
	if err != nil {
		return nil, fmt.Errorf(errBackendLocalRead, err)
	}

	// Grab the state
	s := sMgr.State()

	m.Ui.Output(strings.TrimSpace(outputBackendMigrateLegacy))
	// Initialize the legacy backend
	oldB, err := m.backendInitFromLegacy(s.Remote)
	if err != nil {
		return nil, err
	}

	// Perform the migration
	err = m.backendMigrateState(&backendMigrateOpts{
		OneType: s.Remote.Type,
		TwoType: "local",
		One:     oldB,
		Two:     localB,
	})
	if err != nil {
		return nil, err
	}

	// Unset the remote state
	s = sMgr.State()
	if s == nil {
		s = terraform.NewState()
	}
	s.Remote = nil
	if err := sMgr.WriteState(s); err != nil {
		return nil, fmt.Errorf(strings.TrimSpace(errBackendClearLegacy), err)
	}
	if err := sMgr.PersistState(); err != nil {
		return nil, fmt.Errorf(strings.TrimSpace(errBackendClearLegacy), err)
	}

	m.Ui.Output(m.Colorize().Color(fmt.Sprintf(
		"[reset][green]\n\n"+
			strings.TrimSpace(successBackendUnset), backendType)))

	return nil, nil
}

// Configuring a backend for the first time with legacy remote state.
func (m *Meta) backend_C_R_s(
	c *config.Backend, sMgr state.State) (backend.Backend, error) {
	// Notify the user
	m.Ui.Output(m.Colorize().Color(fmt.Sprintf(
		"[reset]%s\n\n",
		strings.TrimSpace(outputBackendConfigureWithLegacy))))

	// First, configure the new backend
	b, err := m.backendInitFromConfig(c)
	if err != nil {
		return nil, err
	}

	// Next, save the new configuration. This will not overwrite our
	// legacy remote state. We'll handle that after.
	s := sMgr.State()
	if s == nil {
		s = terraform.NewState()
	}
	s.Backend = &terraform.BackendState{
		Type:   c.Type,
		Config: c.RawConfig.Raw,
		Hash:   c.Hash,
	}
	if err := sMgr.WriteState(s); err != nil {
		return nil, fmt.Errorf(errBackendWriteSaved, err)
	}
	if err := sMgr.PersistState(); err != nil {
		return nil, fmt.Errorf(errBackendWriteSaved, err)
	}

	// I don't know how this is possible but if we don't have remote
	// state config anymore somehow, just return the backend. This
	// shouldn't be possible, though.
	if s.Remote.Empty() {
		return b, nil
	}

	m.Ui.Output(strings.TrimSpace(outputBackendMigrateLegacy))
	// Initialize the legacy backend
	oldB, err := m.backendInitFromLegacy(s.Remote)
	if err != nil {
		return nil, err
	}

	// Perform the migration
	err = m.backendMigrateState(&backendMigrateOpts{
		OneType: s.Remote.Type,
		TwoType: c.Type,
		One:     oldB,
		Two:     b,
	})
	if err != nil {
		return nil, err
	}

	// Unset the remote state
	s = sMgr.State()
	if s == nil {
		s = terraform.NewState()
	}
	s.Remote = nil
	if err := sMgr.WriteState(s); err != nil {
		return nil, fmt.Errorf(strings.TrimSpace(errBackendClearLegacy), err)
	}
	if err := sMgr.PersistState(); err != nil {
		return nil, fmt.Errorf(strings.TrimSpace(errBackendClearLegacy), err)
	}

	m.Ui.Output(m.Colorize().Color(fmt.Sprintf(
		"[reset][green]\n"+strings.TrimSpace(successBackendSet), s.Backend.Type)))

	return b, nil
}

// Configuring a backend for the first time.
func (m *Meta) backend_C_r_s(
	c *config.Backend, sMgr state.State) (backend.Backend, error) {
	// Get the backend
	b, err := m.backendInitFromConfig(c)
	if err != nil {
		return nil, err
	}

	// Grab a purely local backend to get the local state if it exists
	localB, err := m.Backend(&BackendOpts{ForceLocal: true})
	if err != nil {
		return nil, fmt.Errorf(errBackendLocalRead, err)
	}

	env := m.Workspace()

	localState, err := localB.State(env)
	if err != nil {
		return nil, fmt.Errorf(errBackendLocalRead, err)
	}
	if err := localState.RefreshState(); err != nil {
		return nil, fmt.Errorf(errBackendLocalRead, err)
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
			return nil, err
		}

		// we usually remove the local state after migration to prevent
		// confusion, but adding a default local backend block to the config
		// can get us here too. Don't delete our state if the old and new paths
		// are the same.
		erase := true
		if newLocalB, ok := b.(*backendLocal.Local); ok {
			if localB, ok := localB.(*backendLocal.Local); ok {
				if newLocalB.StatePath == localB.StatePath {
					erase = false
				}
			}
		}

		if erase {
			// We always delete the local state, unless that was our new state too.
			if err := localState.WriteState(nil); err != nil {
				return nil, fmt.Errorf(errBackendMigrateLocalDelete, err)
			}
			if err := localState.PersistState(); err != nil {
				return nil, fmt.Errorf(errBackendMigrateLocalDelete, err)
			}
		}
	}

	if m.stateLock {
		stateLocker := clistate.NewLocker(context.Background(), m.stateLockTimeout, m.Ui, m.Colorize())
		if err := stateLocker.Lock(sMgr, "backend from plan"); err != nil {
			return nil, fmt.Errorf("Error locking state: %s", err)
		}
		defer stateLocker.Unlock(nil)
	}

	// Store the metadata in our saved state location
	s := sMgr.State()
	if s == nil {
		s = terraform.NewState()
	}
	s.Backend = &terraform.BackendState{
		Type:   c.Type,
		Config: c.RawConfig.Raw,
		Hash:   c.Hash,
	}

	if err := sMgr.WriteState(s); err != nil {
		return nil, fmt.Errorf(errBackendWriteSaved, err)
	}
	if err := sMgr.PersistState(); err != nil {
		return nil, fmt.Errorf(errBackendWriteSaved, err)
	}

	m.Ui.Output(m.Colorize().Color(fmt.Sprintf(
		"[reset][green]\n"+strings.TrimSpace(successBackendSet), s.Backend.Type)))

	// Return the backend
	return b, nil
}

// Changing a previously saved backend.
func (m *Meta) backend_C_r_S_changed(
	c *config.Backend, sMgr state.State, output bool) (backend.Backend, error) {
	if output {
		// Notify the user
		m.Ui.Output(m.Colorize().Color(fmt.Sprintf(
			"[reset]%s\n\n",
			strings.TrimSpace(outputBackendReconfigure))))
	}

	// Get the old state
	s := sMgr.State()

	// Get the backend
	b, err := m.backendInitFromConfig(c)
	if err != nil {
		return nil, fmt.Errorf(
			"Error initializing new backend: %s", err)
	}

	// no need to confuse the user if the backend types are the same
	if s.Backend.Type != c.Type {
		m.Ui.Output(strings.TrimSpace(fmt.Sprintf(outputBackendMigrateChange, s.Backend.Type, c.Type)))
	}

	// Grab the existing backend
	oldB, err := m.backend_C_r_S_unchanged(c, sMgr)
	if err != nil {
		return nil, fmt.Errorf(
			"Error loading previously configured backend: %s", err)
	}

	// Perform the migration
	err = m.backendMigrateState(&backendMigrateOpts{
		OneType: s.Backend.Type,
		TwoType: c.Type,
		One:     oldB,
		Two:     b,
	})
	if err != nil {
		return nil, err
	}

	if m.stateLock {
		stateLocker := clistate.NewLocker(context.Background(), m.stateLockTimeout, m.Ui, m.Colorize())
		if err := stateLocker.Lock(sMgr, "backend from plan"); err != nil {
			return nil, fmt.Errorf("Error locking state: %s", err)
		}
		defer stateLocker.Unlock(nil)
	}

	// Update the backend state
	s = sMgr.State()
	if s == nil {
		s = terraform.NewState()
	}
	s.Backend = &terraform.BackendState{
		Type:   c.Type,
		Config: c.RawConfig.Raw,
		Hash:   c.Hash,
	}

	if err := sMgr.WriteState(s); err != nil {
		return nil, fmt.Errorf(errBackendWriteSaved, err)
	}
	if err := sMgr.PersistState(); err != nil {
		return nil, fmt.Errorf(errBackendWriteSaved, err)
	}

	if output {
		m.Ui.Output(m.Colorize().Color(fmt.Sprintf(
			"[reset][green]\n"+strings.TrimSpace(successBackendSet), s.Backend.Type)))
	}

	return b, nil
}

// Initiailizing an unchanged saved backend
func (m *Meta) backend_C_r_S_unchanged(
	c *config.Backend, sMgr state.State) (backend.Backend, error) {
	s := sMgr.State()

	// it's possible for a backend to be unchanged, and the config itself to
	// have changed by moving a parameter from the config to `-backend-config`
	// In this case we only need to update the Hash.
	if c != nil && s.Backend.Hash != c.Hash {
		s.Backend.Hash = c.Hash
		if err := sMgr.WriteState(s); err != nil {
			return nil, fmt.Errorf(errBackendWriteSaved, err)
		}
	}

	// Create the config. We do this from the backend state since this
	// has the complete configuration data whereas the config itself
	// may require input.
	rawC, err := config.NewRawConfig(s.Backend.Config)
	if err != nil {
		return nil, fmt.Errorf("Error configuring backend: %s", err)
	}
	config := terraform.NewResourceConfig(rawC)

	// Get the backend
	f := backendInit.Backend(s.Backend.Type)
	if f == nil {
		return nil, fmt.Errorf(strings.TrimSpace(errBackendSavedUnknown), s.Backend.Type)
	}
	b := f()

	// Configure
	if err := b.Configure(config); err != nil {
		return nil, fmt.Errorf(errBackendSavedConfig, s.Backend.Type, err)
	}

	return b, nil
}

// Initiailizing a changed saved backend with legacy remote state.
func (m *Meta) backend_C_R_S_changed(
	c *config.Backend, sMgr state.State) (backend.Backend, error) {
	// Notify the user
	m.Ui.Output(m.Colorize().Color(fmt.Sprintf(
		"[reset]%s\n\n",
		strings.TrimSpace(outputBackendSavedWithLegacyChanged))))

	// Reconfigure the backend first
	if _, err := m.backend_C_r_S_changed(c, sMgr, false); err != nil {
		return nil, err
	}

	// Handle the case where we have all set but unchanged
	b, err := m.backend_C_R_S_unchanged(c, sMgr, false)
	if err != nil {
		return nil, err
	}

	// Output success message
	m.Ui.Output(m.Colorize().Color(fmt.Sprintf(
		"[reset][green]\n\n"+
			strings.TrimSpace(successBackendReconfigureWithLegacy), c.Type)))

	return b, nil
}

// Initiailizing an unchanged saved backend with legacy remote state.
func (m *Meta) backend_C_R_S_unchanged(
	c *config.Backend, sMgr state.State, output bool) (backend.Backend, error) {
	if output {
		// Notify the user
		m.Ui.Output(m.Colorize().Color(fmt.Sprintf(
			"[reset]%s\n\n",
			strings.TrimSpace(outputBackendSavedWithLegacy))))
	}

	// Load the backend from the state
	s := sMgr.State()
	b, err := m.backendInitFromSaved(s.Backend)
	if err != nil {
		return nil, err
	}

	m.Ui.Output(strings.TrimSpace(outputBackendMigrateLegacy))

	// Initialize the legacy backend
	oldB, err := m.backendInitFromLegacy(s.Remote)
	if err != nil {
		return nil, err
	}

	// Perform the migration
	err = m.backendMigrateState(&backendMigrateOpts{
		OneType: s.Remote.Type,
		TwoType: s.Backend.Type,
		One:     oldB,
		Two:     b,
	})
	if err != nil {
		return nil, err
	}

	if m.stateLock {
		stateLocker := clistate.NewLocker(context.Background(), m.stateLockTimeout, m.Ui, m.Colorize())
		if err := stateLocker.Lock(sMgr, "backend from plan"); err != nil {
			return nil, fmt.Errorf("Error locking state: %s", err)
		}
		defer stateLocker.Unlock(nil)
	}

	// Unset the remote state
	s = sMgr.State()
	if s == nil {
		s = terraform.NewState()
	}
	s.Remote = nil

	if err := sMgr.WriteState(s); err != nil {
		return nil, fmt.Errorf(strings.TrimSpace(errBackendClearLegacy), err)
	}
	if err := sMgr.PersistState(); err != nil {
		return nil, fmt.Errorf(strings.TrimSpace(errBackendClearLegacy), err)
	}

	if output {
		m.Ui.Output(m.Colorize().Color(fmt.Sprintf(
			"[reset][green]\n\n"+
				strings.TrimSpace(successBackendLegacyUnset), s.Backend.Type)))
	}

	return b, nil
}

//-------------------------------------------------------------------
// Reusable helper functions for backend management
//-------------------------------------------------------------------

func (m *Meta) backendInitFromConfig(c *config.Backend) (backend.Backend, error) {
	// Create the config.
	config := terraform.NewResourceConfig(c.RawConfig)

	// Get the backend
	f := backendInit.Backend(c.Type)
	if f == nil {
		return nil, fmt.Errorf(strings.TrimSpace(errBackendNewUnknown), c.Type)
	}
	b := f()

	// TODO: test
	// Ask for input if we have input enabled
	if m.Input() {
		var err error
		config, err = b.Input(m.UIInput(), config)
		if err != nil {
			return nil, fmt.Errorf(
				"Error asking for input to configure the backend %q: %s",
				c.Type, err)
		}
	}

	// Validate
	warns, errs := b.Validate(config)
	for _, warning := range warns {
		// We just write warnings directly to the UI. This isn't great
		// since we're a bit deep here to be pushing stuff out into the
		// UI, but sufficient to let us print out deprecation warnings
		// and the like.
		m.Ui.Warn(warning)
	}
	if len(errs) > 0 {
		return nil, fmt.Errorf(
			"Error configuring the backend %q: %s",
			c.Type, multierror.Append(nil, errs...))
	}

	// Configure
	if err := b.Configure(config); err != nil {
		return nil, fmt.Errorf(errBackendNewConfig, c.Type, err)
	}

	return b, nil
}

func (m *Meta) backendInitFromLegacy(s *terraform.RemoteState) (backend.Backend, error) {
	// We need to convert the config to map[string]interface{} since that
	// is what the backends expect.
	var configMap map[string]interface{}
	if err := mapstructure.Decode(s.Config, &configMap); err != nil {
		return nil, fmt.Errorf("Error configuring remote state: %s", err)
	}

	// Create the config
	rawC, err := config.NewRawConfig(configMap)
	if err != nil {
		return nil, fmt.Errorf("Error configuring remote state: %s", err)
	}
	config := terraform.NewResourceConfig(rawC)

	// Get the backend
	f := backendInit.Backend(s.Type)
	if f == nil {
		return nil, fmt.Errorf(strings.TrimSpace(errBackendLegacyUnknown), s.Type)
	}
	b := f()

	// Configure
	if err := b.Configure(config); err != nil {
		return nil, fmt.Errorf(errBackendLegacyConfig, err)
	}

	return b, nil
}

func (m *Meta) backendInitFromSaved(s *terraform.BackendState) (backend.Backend, error) {
	// Create the config. We do this from the backend state since this
	// has the complete configuration data whereas the config itself
	// may require input.
	rawC, err := config.NewRawConfig(s.Config)
	if err != nil {
		return nil, fmt.Errorf("Error configuring backend: %s", err)
	}
	config := terraform.NewResourceConfig(rawC)

	// Get the backend
	f := backendInit.Backend(s.Type)
	if f == nil {
		return nil, fmt.Errorf(strings.TrimSpace(errBackendSavedUnknown), s.Type)
	}
	b := f()

	// Configure
	if err := b.Configure(config); err != nil {
		return nil, fmt.Errorf(errBackendSavedConfig, s.Type, err)
	}

	return b, nil
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

const warnBackendLegacy = `
Deprecation warning: This environment is configured to use legacy remote state.
Remote state changed significantly in Terraform 0.9. Please update your remote
state configuration to use the new 'backend' settings. For now, Terraform
will continue to use your existing settings. Legacy remote state support
will be removed in Terraform 0.11.

You can find a guide for upgrading here:

https://www.terraform.io/docs/backends/legacy-0-8.html
`
