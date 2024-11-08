// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package command

// This file contains all the Backend-related function calls on Meta,
// exported and private.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"

	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/backend/backendrun"
	backendInit "github.com/hashicorp/terraform/internal/backend/init"
	backendLocal "github.com/hashicorp/terraform/internal/backend/local"
	"github.com/hashicorp/terraform/internal/cloud"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/clistate"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/command/workdir"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/states/statemgr"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
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

	// ViewType will set console output format for the
	// initialization operation (JSON or human-readable).
	ViewType arguments.ViewType
}

// BackendWithRemoteTerraformVersion is a shared interface between the 'remote' and 'cloud' backends
// for simplified type checking when calling functions common to those particular backends.
type BackendWithRemoteTerraformVersion interface {
	IgnoreVersionConflict()
	VerifyWorkspaceTerraformVersion(workspace string) tfdiags.Diagnostics
	IsLocalOperations() bool
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
func (m *Meta) Backend(opts *BackendOpts) (backendrun.OperationsBackend, tfdiags.Diagnostics) {
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

		log.Printf("[TRACE] Meta.Backend: instantiated backend of type %T", b)
	}

	// Set up the CLI opts we pass into backends that support it.
	cliOpts, err := m.backendCLIOpts()
	if err != nil {
		if errs := providerPluginErrors(nil); errors.As(err, &errs) {
			// This is a special type returned by m.providerFactories, which
			// indicates one or more inconsistencies between the dependency
			// lock file and the provider plugins actually available in the
			// local cache directory.
			//
			// If initialization is allowed, we ignore this error, as it may
			// be resolved by the later step where providers are fetched.
			if !opts.Init {
				var buf bytes.Buffer
				for addr, err := range errs {
					fmt.Fprintf(&buf, "\n  - %s: %s", addr, err)
				}
				suggestion := "To download the plugins required for this configuration, run:\n  terraform init"
				if m.RunningInAutomation {
					// Don't mention "terraform init" specifically if we're running in an automation wrapper
					suggestion = "You must install the required plugins before running Terraform operations."
				}
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Required plugins are not installed",
					fmt.Sprintf(
						"The installed provider plugins are not consistent with the packages selected in the dependency lock file:%s\n\nTerraform uses external plugins to integrate with a variety of different infrastructure services. %s",
						buf.String(), suggestion,
					),
				))
				return nil, diags
			}
		} else {
			// All other errors just get generic handling.
			diags = diags.Append(err)
			return nil, diags
		}
	}
	cliOpts.Validation = true

	// FIXME: Temporarily exposing ViewType and View to the backend.
	// This is a workaround until the backend is refactored to support
	// native View handling.
	cliOpts.ViewType = opts.ViewType
	cliOpts.View = m.View

	// If the backend supports CLI initialization, do it.
	if cli, ok := b.(backendrun.CLI); ok {
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
	if enhanced, ok := b.(backendrun.OperationsBackend); ok {
		log.Printf("[TRACE] Meta.Backend: backend %T supports operations", b)
		return enhanced, nil
	}

	// We either have a non-enhanced backend or no backend configured at
	// all. In either case, we use local as our enhanced backend and the
	// non-enhanced (if any) as the state backend.

	if !opts.ForceLocal {
		log.Printf("[TRACE] Meta.Backend: backend %T does not support operations, so wrapping it in a local backend", b)
	}

	// Build the local backend
	local := backendLocal.NewWithBackend(b)
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
		m.backendState = &workdir.BackendState{
			Type:      "local",
			ConfigRaw: json.RawMessage("{}"),
		}
	}

	return local, nil
}

// selectWorkspace gets a list of existing workspaces and then checks
// if the currently selected workspace is valid. If not, it will ask
// the user to select a workspace from the list.
func (m *Meta) selectWorkspace(b backend.Backend) error {
	workspaces, err := b.Workspaces()
	if err == backend.ErrWorkspacesNotSupported {
		return nil
	}
	if err != nil {
		return fmt.Errorf("Failed to get existing workspaces: %s", err)
	}
	if len(workspaces) == 0 {
		if c, ok := b.(*cloud.Cloud); ok && m.input {
			// len is always 1 if using Name; 0 means we're using Tags and there
			// aren't any matching workspaces. Which might be normal and fine, so
			// let's just ask:
			name, err := m.UIInput().Input(context.Background(), &terraform.InputOpts{
				Id:          "create-workspace",
				Query:       "\n[reset][bold][yellow]No workspaces found.[reset]",
				Description: fmt.Sprintf(inputCloudInitCreateWorkspace, c.WorkspaceMapping.DescribeTags()),
			})
			if err != nil {
				return fmt.Errorf("Couldn't create initial workspace: %w", err)
			}
			name = strings.TrimSpace(name)
			if name == "" {
				return fmt.Errorf("Couldn't create initial workspace: no name provided")
			}
			log.Printf("[TRACE] Meta.selectWorkspace: selecting the new HCP Terraform workspace requested by the user (%s)", name)
			return m.SetWorkspace(name)
		} else {
			return errors.New(strings.TrimSpace(errBackendNoExistingWorkspaces))
		}
	}

	// Get the currently selected workspace.
	workspace, err := m.Workspace()
	if err != nil {
		return err
	}

	// Check if any of the existing workspaces matches the selected
	// workspace and create a numbered list of existing workspaces.
	var list strings.Builder
	for i, w := range workspaces {
		if w == workspace {
			log.Printf("[TRACE] Meta.selectWorkspace: the currently selected workspace is present in the configured backend (%s)", workspace)
			return nil
		}
		fmt.Fprintf(&list, "%d. %s\n", i+1, w)
	}

	// If the backend only has a single workspace, select that as the current workspace
	if len(workspaces) == 1 {
		log.Printf("[TRACE] Meta.selectWorkspace: automatically selecting the single workspace provided by the backend (%s)", workspaces[0])
		return m.SetWorkspace(workspaces[0])
	}

	if !m.input {
		return fmt.Errorf("Currently selected workspace %q does not exist", workspace)
	}

	// Otherwise, ask the user to select a workspace from the list of existing workspaces.
	v, err := m.UIInput().Input(context.Background(), &terraform.InputOpts{
		Id: "select-workspace",
		Query: fmt.Sprintf(
			"\n[reset][bold][yellow]The currently selected workspace (%s) does not exist.[reset]",
			workspace),
		Description: fmt.Sprintf(
			strings.TrimSpace(inputBackendSelectWorkspace), list.String()),
	})
	if err != nil {
		return fmt.Errorf("Failed to select workspace: %s", err)
	}

	idx, err := strconv.Atoi(v)
	if err != nil || (idx < 1 || idx > len(workspaces)) {
		return fmt.Errorf("Failed to select workspace: input not a valid number")
	}

	workspace = workspaces[idx-1]
	log.Printf("[TRACE] Meta.selectWorkspace: setting the current workspace according to user selection (%s)", workspace)
	return m.SetWorkspace(workspace)
}

// BackendForLocalPlan is similar to Backend, but uses backend settings that were
// stored in a plan.
//
// The current workspace name is also stored as part of the plan, and so this
// method will check that it matches the currently-selected workspace name
// and produce error diagnostics if not.
func (m *Meta) BackendForLocalPlan(settings plans.Backend) (backendrun.OperationsBackend, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	f := backendInit.Backend(settings.Type)
	if f == nil {
		diags = diags.Append(fmt.Errorf(strings.TrimSpace(errBackendSavedUnknown), settings.Type))
		return nil, diags
	}
	b := f()
	log.Printf("[TRACE] Meta.BackendForLocalPlan: instantiated backend of type %T", b)

	schema := b.ConfigSchema()
	configVal, err := settings.Config.Decode(schema.ImpliedType())
	if err != nil {
		diags = diags.Append(fmt.Errorf("saved backend configuration is invalid: %w", err))
		return nil, diags
	}

	newVal, validateDiags := b.PrepareConfig(configVal)
	diags = diags.Append(validateDiags)
	if validateDiags.HasErrors() {
		return nil, diags
	}

	configureDiags := b.Configure(newVal)
	diags = diags.Append(configureDiags)
	if configureDiags.HasErrors() {
		return nil, diags
	}

	// If the backend supports CLI initialization, do it.
	if cli, ok := b.(backendrun.CLI); ok {
		cliOpts, err := m.backendCLIOpts()
		if err != nil {
			diags = diags.Append(err)
			return nil, diags
		}
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
	if enhanced, ok := b.(backendrun.OperationsBackend); ok {
		log.Printf("[TRACE] Meta.BackendForPlan: backend %T supports operations", b)
		if err := m.setupEnhancedBackendAliases(enhanced); err != nil {
			diags = diags.Append(err)
			return nil, diags
		}
		return enhanced, nil
	}

	// Otherwise, we'll wrap our state-only remote backend in the local backend
	// to cause any operations to be run locally.
	log.Printf("[TRACE] Meta.BackendForLocalPlan: backend %T does not support operations, so wrapping it in a local backend", b)
	cliOpts, err := m.backendCLIOpts()
	if err != nil {
		diags = diags.Append(err)
		return nil, diags
	}
	cliOpts.Validation = false // don't validate here in case config contains file(...) calls where the file doesn't exist
	local := backendLocal.NewWithBackend(b)
	if err := local.CLIInit(cliOpts); err != nil {
		// Local backend should never fail, so this is always a bug.
		panic(err)
	}

	return local, diags
}

// backendCLIOpts returns a backendrun.CLIOpts object that should be passed to
// a backend that supports local CLI operations.
func (m *Meta) backendCLIOpts() (*backendrun.CLIOpts, error) {
	contextOpts, err := m.contextOpts()
	if contextOpts == nil && err != nil {
		return nil, err
	}
	return &backendrun.CLIOpts{
		CLI:                 m.Ui,
		CLIColor:            m.Colorize(),
		Streams:             m.Streams,
		StatePath:           m.statePath,
		StateOutPath:        m.stateOutPath,
		StateBackupPath:     m.backupPath,
		ContextOpts:         contextOpts,
		Input:               m.Input(),
		RunningInAutomation: m.RunningInAutomation,
	}, err
}

// Operation initializes a new backend.Operation struct.
//
// This prepares the operation. After calling this, the caller is expected
// to modify fields of the operation such as Sequence to specify what will
// be called.
func (m *Meta) Operation(b backend.Backend, vt arguments.ViewType) *backendrun.Operation {
	schema := b.ConfigSchema()
	workspace, err := m.Workspace()
	if err != nil {
		// An invalid workspace error would have been raised when creating the
		// backend, and the caller should have already exited. Seeing the error
		// here first is a bug, so panic.
		panic(fmt.Sprintf("invalid workspace: %s", err))
	}
	planOutBackend, err := m.backendState.ForPlan(schema, workspace)
	if err != nil {
		// Always indicates an implementation error in practice, because
		// errors here indicate invalid encoding of the backend configuration
		// in memory, and we should always have validated that by the time
		// we get here.
		panic(fmt.Sprintf("failed to encode backend configuration for plan: %s", err))
	}

	stateLocker := clistate.NewNoopLocker()
	if m.stateLock {
		view := views.NewStateLocker(vt, m.View)
		stateLocker = clistate.NewLocker(m.stateLockTimeout, view)
	}

	depLocks, diags := m.lockedDependencies()
	if diags.HasErrors() {
		// We can't actually report errors from here, but m.lockedDependencies
		// should always have been called earlier to prepare the "ContextOpts"
		// for the backend anyway, so we should never actually get here in
		// a real situation. If we do get here then the backend will inevitably
		// fail downstream somwhere if it tries to use the empty depLocks.
		log.Printf("[WARN] Failed to load dependency locks while preparing backend operation (ignored): %s", diags.Err().Error())
	}

	return &backendrun.Operation{
		PlanOutBackend:  planOutBackend,
		Targets:         m.targets,
		UIIn:            m.UIInput(),
		UIOut:           m.Ui,
		Workspace:       workspace,
		StateLocker:     stateLocker,
		DependencyLocks: depLocks,
	}
}

// backendConfig returns the local configuration for the backend
func (m *Meta) backendConfig(opts *BackendOpts) (*configs.Backend, int, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	if opts.Config == nil {
		// check if the config was missing, or just not required
		conf, moreDiags := m.loadBackendConfig(".")
		diags = diags.Append(moreDiags)
		if moreDiags.HasErrors() {
			return nil, 0, diags
		}

		if conf == nil {
			log.Println("[TRACE] Meta.Backend: no config given or present on disk, so returning nil config")
			return nil, 0, nil
		}

		log.Printf("[TRACE] Meta.Backend: BackendOpts.Config not set, so using settings loaded from %s", conf.DeclRange)
		opts.Config = conf
	}

	c := opts.Config

	if c == nil {
		log.Println("[TRACE] Meta.Backend: no explicit backend config, so returning nil config")
		return nil, 0, nil
	}

	bf := backendInit.Backend(c.Type)
	if bf == nil {
		detail := fmt.Sprintf("There is no backend type named %q.", c.Type)
		if msg, removed := backendInit.RemovedBackends[c.Type]; removed {
			detail = msg
		}

		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid backend type",
			Detail:   detail,
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
		log.Println("[TRACE] Meta.Backend: merging -backend-config=... CLI overrides into backend configuration")
		configBody = configs.MergeBodies(configBody, opts.ConfigOverride)
	}

	log.Printf("[TRACE] Meta.Backend: built configuration for %q backend with hash value %d", c.Type, configHash)

	// We'll shallow-copy configs.Backend here so that we can replace the
	// body without affecting others that hold this reference.
	configCopy := *c
	configCopy.Config = configBody
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
	// directory is kept in a *state-like* file, using a subset of the oldstate
	// snapshot version 3. It is not actually a Terraform state, and so only
	// the "backend" portion of it is actually used.
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
	sMgr := &clistate.LocalState{Path: statePath}
	if err := sMgr.RefreshState(); err != nil {
		diags = diags.Append(fmt.Errorf("Failed to load state: %s", err))
		return nil, diags
	}

	// Load the state, it must be non-nil for the tests below but can be empty
	s := sMgr.State()
	if s == nil {
		log.Printf("[TRACE] Meta.Backend: backend has not previously been initialized in this working directory")
		s = workdir.NewBackendStateFile()
	} else if s.Backend != nil {
		log.Printf("[TRACE] Meta.Backend: working directory was previously initialized for %q backend", s.Backend.Type)
	} else {
		log.Printf("[TRACE] Meta.Backend: working directory was previously initialized but has no backend (is using legacy remote state?)")
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

	// This switch statement covers all the different combinations of
	// configuring new backends, updating previously-configured backends, etc.
	switch {
	// No configuration set at all. Pure local state.
	case c == nil && s.Backend.Empty():
		log.Printf("[TRACE] Meta.Backend: using default local state only (no backend configuration, and no existing initialized backend)")
		return nil, nil

	// We're unsetting a backend (moving from backend => local)
	case c == nil && !s.Backend.Empty():
		log.Printf("[TRACE] Meta.Backend: previously-initialized %q backend is no longer present in config", s.Backend.Type)

		initReason := fmt.Sprintf("Unsetting the previously set backend %q", s.Backend.Type)
		if !opts.Init {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Backend initialization required, please run \"terraform init\"",
				fmt.Sprintf(strings.TrimSpace(errBackendInit), initReason),
			))
			return nil, diags
		}

		if s.Backend.Type != "cloud" && !m.migrateState {
			diags = diags.Append(migrateOrReconfigDiag)
			return nil, diags
		}

		return m.backend_c_r_S(c, cHash, sMgr, true, opts)

	// Configuring a backend for the first time or -reconfigure flag was used
	case c != nil && s.Backend.Empty():
		log.Printf("[TRACE] Meta.Backend: moving from default local state only to %q backend", c.Type)
		if !opts.Init {
			if c.Type == "cloud" {
				initReason := "Initial configuration of HCP Terraform or Terraform Enterprise"
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"HCP Terraform or Terraform Enterprise initialization required: please run \"terraform init\"",
					fmt.Sprintf(strings.TrimSpace(errBackendInitCloud), initReason),
				))
			} else {
				initReason := fmt.Sprintf("Initial configuration of the requested backend %q", c.Type)
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Backend initialization required, please run \"terraform init\"",
					fmt.Sprintf(strings.TrimSpace(errBackendInit), initReason),
				))
			}
			return nil, diags
		}
		return m.backend_C_r_s(c, cHash, sMgr, opts)
	// Potentially changing a backend configuration
	case c != nil && !s.Backend.Empty():
		// We are not going to migrate if...
		//
		// We're not initializing
		// AND the backend cache hash values match, indicating that the stored config is valid and completely unchanged.
		// AND we're not providing any overrides. An override can mean a change overriding an unchanged backend block (indicated by the hash value).
		if (uint64(cHash) == s.Backend.Hash) && (!opts.Init || opts.ConfigOverride == nil) {
			log.Printf("[TRACE] Meta.Backend: using already-initialized, unchanged %q backend configuration", c.Type)
			savedBackend, diags := m.savedBackend(sMgr)
			// Verify that selected workspace exist. Otherwise prompt user to create one
			if opts.Init && savedBackend != nil {
				if err := m.selectWorkspace(savedBackend); err != nil {
					diags = diags.Append(err)
					return nil, diags
				}
			}
			return savedBackend, diags
		}

		// If our configuration (the result of both the literal configuration and given
		// -backend-config options) is the same, then we're just initializing a previously
		// configured backend. The literal configuration may differ, however, so while we
		// don't need to migrate, we update the backend cache hash value.
		if !m.backendConfigNeedsMigration(c, s.Backend) {
			log.Printf("[TRACE] Meta.Backend: using already-initialized %q backend configuration", c.Type)
			savedBackend, moreDiags := m.savedBackend(sMgr)
			diags = diags.Append(moreDiags)
			if moreDiags.HasErrors() {
				return nil, diags
			}

			// It's possible for a backend to be unchanged, and the config itself to
			// have changed by moving a parameter from the config to `-backend-config`
			// In this case, we update the Hash.
			moreDiags = m.updateSavedBackendHash(cHash, sMgr)
			if moreDiags.HasErrors() {
				return nil, diags
			}
			// Verify that selected workspace exist. Otherwise prompt user to create one
			if opts.Init && savedBackend != nil {
				if err := m.selectWorkspace(savedBackend); err != nil {
					diags = diags.Append(err)
					return nil, diags
				}
			}

			return savedBackend, diags
		}
		log.Printf("[TRACE] Meta.Backend: backend configuration has changed (from type %q to type %q)", s.Backend.Type, c.Type)

		cloudMode := cloud.DetectConfigChangeType(s.Backend, c, false)

		if !opts.Init {
			//user ran another cmd that is not init but they are required to initialize because of a potential relevant change to their backend configuration
			initDiag := m.determineInitReason(s.Backend.Type, c.Type, cloudMode)
			diags = diags.Append(initDiag)
			return nil, diags
		}

		if !cloudMode.InvolvesCloud() && !m.migrateState {
			diags = diags.Append(migrateOrReconfigDiag)
			return nil, diags
		}

		log.Printf("[WARN] backend config has changed since last init")
		return m.backend_C_r_S_changed(c, cHash, sMgr, true, opts)

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

func (m *Meta) determineInitReason(previousBackendType string, currentBackendType string, cloudMode cloud.ConfigChangeMode) tfdiags.Diagnostics {
	initReason := ""
	switch cloudMode {
	case cloud.ConfigMigrationIn:
		initReason = fmt.Sprintf("Changed from backend %q to HCP Terraform", previousBackendType)
	case cloud.ConfigMigrationOut:
		initReason = fmt.Sprintf("Changed from HCP Terraform to backend %q", previousBackendType)
	case cloud.ConfigChangeInPlace:
		initReason = "HCP Terraform configuration block has changed"
	default:
		switch {
		case previousBackendType != currentBackendType:
			initReason = fmt.Sprintf("Backend type changed from %q to %q", previousBackendType, currentBackendType)
		default:
			initReason = "Backend configuration block has changed"
		}
	}

	var diags tfdiags.Diagnostics
	switch cloudMode {
	case cloud.ConfigChangeInPlace:
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"HCP Terraform or Terraform Enterprise initialization required: please run \"terraform init\"",
			fmt.Sprintf(strings.TrimSpace(errBackendInitCloud), initReason),
		))
	case cloud.ConfigMigrationIn:
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"HCP Terraform or Terraform Enterprise initialization required: please run \"terraform init\"",
			fmt.Sprintf(strings.TrimSpace(errBackendInitCloud), initReason),
		))
	default:
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Backend initialization required: please run \"terraform init\"",
			fmt.Sprintf(strings.TrimSpace(errBackendInit), initReason),
		))
	}

	return diags
}

// backendFromState returns the initialized (not configured) backend directly
// from the backend state. This should be used only when a user runs
// `terraform init -backend=false`. This function returns a local backend if
// there is no backend state or no backend configured.
func (m *Meta) backendFromState(_ context.Context) (backend.Backend, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	// Get the path to where we store a local cache of backend configuration
	// if we're using a remote backend. This may not yet exist which means
	// we haven't used a non-local backend before. That is okay.
	statePath := filepath.Join(m.DataDir(), DefaultStateFilename)
	sMgr := &clistate.LocalState{Path: statePath}
	if err := sMgr.RefreshState(); err != nil {
		diags = diags.Append(fmt.Errorf("Failed to load state: %s", err))
		return nil, diags
	}
	s := sMgr.State()
	if s == nil {
		// no state, so return a local backend
		log.Printf("[TRACE] Meta.Backend: backend has not previously been initialized in this working directory")
		return backendLocal.New(), diags
	}
	if s.Backend == nil {
		// s.Backend is nil, so return a local backend
		log.Printf("[TRACE] Meta.Backend: working directory was previously initialized but has no backend (is using legacy remote state?)")
		return backendLocal.New(), diags
	}
	log.Printf("[TRACE] Meta.Backend: working directory was previously initialized for %q backend", s.Backend.Type)

	//backend init function
	if s.Backend.Type == "" {
		return backendLocal.New(), diags
	}
	f := backendInit.Backend(s.Backend.Type)
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
	newVal, validDiags := b.PrepareConfig(configVal)
	diags = diags.Append(validDiags)
	if validDiags.HasErrors() {
		return nil, diags
	}

	configDiags := b.Configure(newVal)
	diags = diags.Append(configDiags)
	if configDiags.HasErrors() {
		return nil, diags
	}

	// If the result of loading the backend is an enhanced backend,
	// then set up enhanced backend service aliases.
	if enhanced, ok := b.(backendrun.OperationsBackend); ok {
		log.Printf("[TRACE] Meta.BackendForPlan: backend %T supports operations", b)

		if err := m.setupEnhancedBackendAliases(enhanced); err != nil {
			diags = diags.Append(err)
			return nil, diags
		}
	}

	return b, diags
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
	c *configs.Backend, cHash int, sMgr *clistate.LocalState, output bool, opts *BackendOpts) (backend.Backend, tfdiags.Diagnostics) {

	var diags tfdiags.Diagnostics

	vt := arguments.ViewJSON
	// Set default viewtype if none was set as the StateLocker needs to know exactly
	// what viewType we want to have.
	if opts == nil || opts.ViewType != vt {
		vt = arguments.ViewHuman
	}

	s := sMgr.State()

	cloudMode := cloud.DetectConfigChangeType(s.Backend, c, false)
	diags = diags.Append(m.assertSupportedCloudInitOptions(cloudMode))
	if diags.HasErrors() {
		return nil, diags
	}

	// Get the backend type for output
	backendType := s.Backend.Type

	if cloudMode == cloud.ConfigMigrationOut {
		m.Ui.Output("Migrating from HCP Terraform or Terraform Enterprise to local state.")
	} else {
		m.Ui.Output(fmt.Sprintf(strings.TrimSpace(outputBackendMigrateLocal), s.Backend.Type))
	}

	// Grab a purely local backend to get the local state if it exists
	localB, moreDiags := m.Backend(&BackendOpts{ForceLocal: true, Init: true})
	diags = diags.Append(moreDiags)
	if moreDiags.HasErrors() {
		return nil, diags
	}

	// Initialize the configured backend
	b, moreDiags := m.savedBackend(sMgr)
	diags = diags.Append(moreDiags)
	if moreDiags.HasErrors() {
		return nil, diags
	}

	// Perform the migration
	err := m.backendMigrateState(&backendMigrateOpts{
		SourceType:      s.Backend.Type,
		DestinationType: "local",
		Source:          b,
		Destination:     localB,
		ViewType:        vt,
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

// Configuring a backend for the first time.
func (m *Meta) backend_C_r_s(c *configs.Backend, cHash int, sMgr *clistate.LocalState, opts *BackendOpts) (backend.Backend, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	vt := arguments.ViewJSON
	// Set default viewtype if none was set as the StateLocker needs to know exactly
	// what viewType we want to have.
	if opts == nil || opts.ViewType != vt {
		vt = arguments.ViewHuman
	}

	// Grab a purely local backend to get the local state if it exists
	localB, localBDiags := m.Backend(&BackendOpts{ForceLocal: true, Init: true})
	if localBDiags.HasErrors() {
		diags = diags.Append(localBDiags)
		return nil, diags
	}

	workspaces, err := localB.Workspaces()
	if err != nil {
		diags = diags.Append(fmt.Errorf(errBackendLocalRead, err))
		return nil, diags
	}

	var localStates []statemgr.Full
	for _, workspace := range workspaces {
		localState, err := localB.StateMgr(workspace)
		if err != nil {
			diags = diags.Append(fmt.Errorf(errBackendLocalRead, err))
			return nil, diags
		}
		if err := localState.RefreshState(); err != nil {
			diags = diags.Append(fmt.Errorf(errBackendLocalRead, err))
			return nil, diags
		}

		// We only care about non-empty states.
		if localS := localState.State(); !localS.Empty() {
			log.Printf("[TRACE] Meta.Backend: will need to migrate workspace states because of existing %q workspace", workspace)
			localStates = append(localStates, localState)
		} else {
			log.Printf("[TRACE] Meta.Backend: ignoring local %q workspace because its state is empty", workspace)
		}
	}

	cloudMode := cloud.DetectConfigChangeType(nil, c, len(localStates) > 0)
	diags = diags.Append(m.assertSupportedCloudInitOptions(cloudMode))
	if diags.HasErrors() {
		return nil, diags
	}

	// Get the backend
	b, configVal, moreDiags := m.backendInitFromConfig(c)
	diags = diags.Append(moreDiags)
	if diags.HasErrors() {
		return nil, diags
	}

	if len(localStates) > 0 {
		// Perform the migration
		err = m.backendMigrateState(&backendMigrateOpts{
			SourceType:      "local",
			DestinationType: c.Type,
			Source:          localB,
			Destination:     b,
			ViewType:        vt,
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
		if newLocalB, ok := b.(*backendLocal.Local); ok {
			if localB, ok := localB.(*backendLocal.Local); ok {
				if newLocalB.PathsConflictWith(localB) {
					erase = false
					log.Printf("[TRACE] Meta.Backend: both old and new backends share the same local state paths, so not erasing old state")
				}
			}
		}

		if erase {
			log.Printf("[TRACE] Meta.Backend: removing old state snapshots from old backend")
			for _, localState := range localStates {
				// We always delete the local state, unless that was our new state too.
				if err := localState.WriteState(nil); err != nil {
					diags = diags.Append(fmt.Errorf(errBackendMigrateLocalDelete, err))
					return nil, diags
				}
				if err := localState.PersistState(nil); err != nil {
					diags = diags.Append(fmt.Errorf(errBackendMigrateLocalDelete, err))
					return nil, diags
				}
			}
		}
	}

	if m.stateLock {
		view := views.NewStateLocker(vt, m.View)
		stateLocker := clistate.NewLocker(m.stateLockTimeout, view)
		if err := stateLocker.Lock(sMgr, "backend from plan"); err != nil {
			diags = diags.Append(fmt.Errorf("Error locking state: %s", err))
			return nil, diags
		}
		defer stateLocker.Unlock()
	}

	configJSON, err := ctyjson.Marshal(configVal, b.ConfigSchema().ImpliedType())
	if err != nil {
		diags = diags.Append(fmt.Errorf("Can't serialize backend configuration as JSON: %s", err))
		return nil, diags
	}

	// Store the metadata in our saved state location
	s := sMgr.State()
	if s == nil {
		s = workdir.NewBackendStateFile()
	}
	s.Backend = &workdir.BackendState{
		Type:      c.Type,
		ConfigRaw: json.RawMessage(configJSON),
		Hash:      uint64(cHash),
	}

	// Verify that selected workspace exists in the backend.
	if opts.Init && b != nil {
		err := m.selectWorkspace(b)
		if err != nil {
			diags = diags.Append(err)

			// FIXME: A compatibility oddity with the 'remote' backend.
			// As an awkward legacy UX, when the remote backend is configured and there
			// are no workspaces, the output to the user saying that there are none and
			// the user should create one with 'workspace new' takes the form of an
			// error message - even though it's happy path, expected behavior.
			//
			// Therefore, only return nil with errored diags for everything else, and
			// allow the remote backend to continue and write its configuration to state
			// even though no workspace is selected.
			if c.Type != "remote" {
				return nil, diags
			}
		}
	}

	if err := sMgr.WriteState(s); err != nil {
		diags = diags.Append(fmt.Errorf(errBackendWriteSaved, err))
		return nil, diags
	}
	if err := sMgr.PersistState(); err != nil {
		diags = diags.Append(fmt.Errorf(errBackendWriteSaved, err))
		return nil, diags
	}

	// By now the backend is successfully configured.  If using HCP Terraform, the success
	// message is handled as part of the final init message
	if _, ok := b.(*cloud.Cloud); !ok {
		m.Ui.Output(m.Colorize().Color(fmt.Sprintf(
			"[reset][green]\n"+strings.TrimSpace(successBackendSet), s.Backend.Type)))
	}

	return b, diags
}

// Changing a previously saved backend.
func (m *Meta) backend_C_r_S_changed(c *configs.Backend, cHash int, sMgr *clistate.LocalState, output bool, opts *BackendOpts) (backend.Backend, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	vt := arguments.ViewJSON
	// Set default viewtype if none was set as the StateLocker needs to know exactly
	// what viewType we want to have.
	if opts == nil || opts.ViewType != vt {
		vt = arguments.ViewHuman
	}

	// Get the old state
	s := sMgr.State()

	cloudMode := cloud.DetectConfigChangeType(s.Backend, c, false)
	diags = diags.Append(m.assertSupportedCloudInitOptions(cloudMode))
	if diags.HasErrors() {
		return nil, diags
	}

	if output {
		// Notify the user
		switch cloudMode {
		case cloud.ConfigChangeInPlace:
			m.Ui.Output("HCP Terraform configuration has changed.")
		case cloud.ConfigMigrationIn:
			m.Ui.Output(fmt.Sprintf("Migrating from backend %q to HCP Terraform.", s.Backend.Type))
		case cloud.ConfigMigrationOut:
			m.Ui.Output(fmt.Sprintf("Migrating from HCP Terraform to backend %q.", c.Type))
		default:
			if s.Backend.Type != c.Type {
				output := fmt.Sprintf(outputBackendMigrateChange, s.Backend.Type, c.Type)
				m.Ui.Output(m.Colorize().Color(fmt.Sprintf(
					"[reset]%s\n",
					strings.TrimSpace(output))))
			} else {
				m.Ui.Output(m.Colorize().Color(fmt.Sprintf(
					"[reset]%s\n",
					strings.TrimSpace(outputBackendReconfigure))))
			}
		}
	}

	// Get the backend
	b, configVal, moreDiags := m.backendInitFromConfig(c)
	diags = diags.Append(moreDiags)
	if moreDiags.HasErrors() {
		return nil, diags
	}

	// If this is a migration into, out of, or irrelevant to HCP Terraform
	// mode then we will do state migration here. Otherwise, we just update
	// the working directory initialization directly, because HCP Terraform
	// doesn't have configurable state storage anyway -- we're only changing
	// which workspaces are relevant to this configuration, not where their
	// state lives.
	if cloudMode != cloud.ConfigChangeInPlace {
		// Grab the existing backend
		oldB, oldBDiags := m.savedBackend(sMgr)
		diags = diags.Append(oldBDiags)
		if oldBDiags.HasErrors() {
			return nil, diags
		}

		// Perform the migration
		err := m.backendMigrateState(&backendMigrateOpts{
			SourceType:      s.Backend.Type,
			DestinationType: c.Type,
			Source:          oldB,
			Destination:     b,
			ViewType:        vt,
		})
		if err != nil {
			diags = diags.Append(err)
			return nil, diags
		}

		if m.stateLock {
			view := views.NewStateLocker(vt, m.View)
			stateLocker := clistate.NewLocker(m.stateLockTimeout, view)
			if err := stateLocker.Lock(sMgr, "backend from plan"); err != nil {
				diags = diags.Append(fmt.Errorf("Error locking state: %s", err))
				return nil, diags
			}
			defer stateLocker.Unlock()
		}
	}

	configJSON, err := ctyjson.Marshal(configVal, b.ConfigSchema().ImpliedType())
	if err != nil {
		diags = diags.Append(fmt.Errorf("Can't serialize backend configuration as JSON: %s", err))
		return nil, diags
	}

	// Update the backend state
	s = sMgr.State()
	if s == nil {
		s = workdir.NewBackendStateFile()
	}
	s.Backend = &workdir.BackendState{
		Type:      c.Type,
		ConfigRaw: json.RawMessage(configJSON),
		Hash:      uint64(cHash),
	}

	// Verify that selected workspace exist. Otherwise prompt user to create one
	if opts.Init && b != nil {
		if err := m.selectWorkspace(b); err != nil {
			diags = diags.Append(err)
			return b, diags
		}
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
		// By now the backend is successfully configured.  If using HCP Terraform, the success
		// message is handled as part of the final init message
		if _, ok := b.(*cloud.Cloud); !ok {
			m.Ui.Output(m.Colorize().Color(fmt.Sprintf(
				"[reset][green]\n"+strings.TrimSpace(successBackendSet), s.Backend.Type)))
		}
	}

	return b, diags
}

// Initializing a saved backend from the cache file (legacy state file)
//
// TODO: This is extremely similar to Meta.backendFromState() but for legacy reasons this is the
// function used by the migration APIs within this file. The other handles 'init -backend=false',
// specifically.
func (m *Meta) savedBackend(sMgr *clistate.LocalState) (backend.Backend, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	s := sMgr.State()

	// Get the backend
	f := backendInit.Backend(s.Backend.Type)
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
	newVal, validDiags := b.PrepareConfig(configVal)
	diags = diags.Append(validDiags)
	if validDiags.HasErrors() {
		return nil, diags
	}

	configDiags := b.Configure(newVal)
	diags = diags.Append(configDiags)
	if configDiags.HasErrors() {
		return nil, diags
	}

	// If the result of loading the backend is an enhanced backend,
	// then set up enhanced backend service aliases.
	if enhanced, ok := b.(backendrun.OperationsBackend); ok {
		log.Printf("[TRACE] Meta.BackendForPlan: backend %T supports operations", b)

		if err := m.setupEnhancedBackendAliases(enhanced); err != nil {
			diags = diags.Append(err)
			return nil, diags
		}
	}

	return b, diags
}

func (m *Meta) updateSavedBackendHash(cHash int, sMgr *clistate.LocalState) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	s := sMgr.State()

	if s.Backend.Hash != uint64(cHash) {
		s.Backend.Hash = uint64(cHash)
		if err := sMgr.WriteState(s); err != nil {
			diags = diags.Append(err)
		}
	}

	return diags
}

//-------------------------------------------------------------------
// Reusable helper functions for backend management
//-------------------------------------------------------------------

// backendConfigNeedsMigration returns true if migration might be required to
// move from the configured backend to the given cached backend config.
//
// This must be called with the synthetic *configs.Backend that results from
// merging in any command-line options for correct behavior.
//
// If either the given configuration or cached configuration are invalid then
// this function will conservatively assume that migration is required,
// expecting that the migration code will subsequently deal with the same
// errors.
func (m *Meta) backendConfigNeedsMigration(c *configs.Backend, s *workdir.BackendState) bool {
	if s == nil || s.Empty() {
		log.Print("[TRACE] backendConfigNeedsMigration: no cached config, so migration is required")
		return true
	}
	if c.Type != s.Type {
		log.Printf("[TRACE] backendConfigNeedsMigration: type changed from %q to %q, so migration is required", s.Type, c.Type)
		return true
	}

	// We need the backend's schema to do our comparison here.
	f := backendInit.Backend(c.Type)
	if f == nil {
		log.Printf("[TRACE] backendConfigNeedsMigration: no backend of type %q, which migration codepath must handle", c.Type)
		return true // let the migration codepath deal with the missing backend
	}
	b := f()

	schema := b.ConfigSchema()
	decSpec := schema.NoneRequired().DecoderSpec()
	givenVal, diags := hcldec.Decode(c.Config, decSpec, nil)
	if diags.HasErrors() {
		log.Printf("[TRACE] backendConfigNeedsMigration: failed to decode given config; migration codepath must handle problem: %s", diags.Error())
		return true // let the migration codepath deal with these errors
	}

	cachedVal, err := s.Config(schema)
	if err != nil {
		log.Printf("[TRACE] backendConfigNeedsMigration: failed to decode cached config; migration codepath must handle problem: %s", err)
		return true // let the migration codepath deal with the error
	}

	// If we get all the way down here then it's the exact equality of the
	// two decoded values that decides our outcome. It's safe to use RawEquals
	// here (rather than Equals) because we know that unknown values can
	// never appear in backend configurations.
	if cachedVal.RawEquals(givenVal) {
		log.Print("[TRACE] backendConfigNeedsMigration: given configuration matches cached configuration, so no migration is required")
		return false
	}
	log.Print("[TRACE] backendConfigNeedsMigration: configuration values have changed, so migration is required")
	return true
}

func (m *Meta) backendInitFromConfig(c *configs.Backend) (backend.Backend, cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	// Get the backend
	f := backendInit.Backend(c.Type)
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

	if !configVal.IsWhollyKnown() {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Unknown values within backend definition",
			"The `terraform` configuration block should contain only concrete and static values. Another diagnostic should contain more information about which part of the configuration is problematic."))
		return nil, cty.NilVal, diags
	}

	// TODO: test
	if m.Input() {
		var err error
		configVal, err = m.inputForSchema(configVal, schema)
		if err != nil {
			diags = diags.Append(fmt.Errorf("Error asking for input to configure backend %q: %s", c.Type, err))
		}

		// We get an unknown here if the if the user aborted input, but we can't
		// turn that into a config value, so set it to null and let the provider
		// handle it in PrepareConfig.
		if !configVal.IsKnown() {
			configVal = cty.NullVal(configVal.Type())
		}
	}

	newVal, validateDiags := b.PrepareConfig(configVal)
	diags = diags.Append(validateDiags.InConfigBody(c.Config, ""))
	if validateDiags.HasErrors() {
		return nil, cty.NilVal, diags
	}

	configureDiags := b.Configure(newVal)
	diags = diags.Append(configureDiags.InConfigBody(c.Config, ""))

	// If the result of loading the backend is an enhanced backend,
	// then set up enhanced backend service aliases.
	if enhanced, ok := b.(backendrun.OperationsBackend); ok {
		log.Printf("[TRACE] Meta.BackendForPlan: backend %T supports operations", b)
		if err := m.setupEnhancedBackendAliases(enhanced); err != nil {
			diags = diags.Append(err)
			return nil, cty.NilVal, diags
		}
	}

	return b, configVal, diags
}

// Helper method to get aliases from the enhanced backend and alias them
// in the Meta service discovery. It's unfortunate that the Meta backend
// is modifying the service discovery at this level, but the owner
// of the service discovery pointer does not have easy access to the backend.
func (m *Meta) setupEnhancedBackendAliases(b backendrun.OperationsBackend) error {
	// Set up the service discovery aliases specified by the enhanced backend.
	serviceAliases, err := b.ServiceDiscoveryAliases()
	if err != nil {
		return err
	}

	for _, alias := range serviceAliases {
		m.Services.Alias(alias.From, alias.To)
	}
	return nil
}

// Helper method to ignore remote/cloud backend version conflicts. Only call this
// for commands which cannot accidentally upgrade remote state files.
func (m *Meta) ignoreRemoteVersionConflict(b backend.Backend) {
	if back, ok := b.(BackendWithRemoteTerraformVersion); ok {
		back.IgnoreVersionConflict()
	}
}

// Helper method to check the local Terraform version against the configured
// version in the remote workspace, returning diagnostics if they conflict.
func (m *Meta) remoteVersionCheck(b backend.Backend, workspace string) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	if back, ok := b.(BackendWithRemoteTerraformVersion); ok {
		// Allow user override based on command-line flag
		if m.ignoreRemoteVersion {
			back.IgnoreVersionConflict()
		}
		// If the override is set, this check will return a warning instead of
		// an error
		versionDiags := back.VerifyWorkspaceTerraformVersion(workspace)
		diags = diags.Append(versionDiags)
		// If there are no errors resulting from this check, we do not need to
		// check again
		if !diags.HasErrors() {
			back.IgnoreVersionConflict()
		}
	}

	return diags
}

// assertSupportedCloudInitOptions returns diagnostics with errors if the
// init-related command line options (implied inside the Meta receiver)
// are incompatible with the given cloud configuration change mode.
func (m *Meta) assertSupportedCloudInitOptions(mode cloud.ConfigChangeMode) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	if mode.InvolvesCloud() {
		log.Printf("[TRACE] Meta.Backend: HCP Terraform or Terraform Enterprise mode initialization type: %s", mode)
		if m.reconfigure {
			if mode.IsCloudMigration() {
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Invalid command-line option",
					"The -reconfigure option is unsupported when migrating to HCP Terraform, because activating HCP Terraform involves some additional steps.",
				))
			} else {
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Invalid command-line option",
					"The -reconfigure option is for in-place reconfiguration of state backends only, and is not needed when changing HCP Terraform settings.\n\nWhen using HCP Terraform, initialization automatically activates any new Cloud configuration settings.",
				))
			}
		}
		if m.migrateState {
			name := "-migrate-state"
			if m.forceInitCopy {
				// -force copy implies -migrate-state in "terraform init",
				// so m.migrateState is forced to true in this case even if
				// the user didn't actually specify it. We'll use the other
				// name here to avoid being confusing, then.
				name = "-force-copy"
			}
			if mode.IsCloudMigration() {
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Invalid command-line option",
					fmt.Sprintf("The %s option is for migration between state backends only, and is not applicable when using HCP Terraform.\n\nHCP Terraform migrations have additional steps, configured by interactive prompts.", name),
				))
			} else {
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Invalid command-line option",
					fmt.Sprintf("The %s option is for migration between state backends only, and is not applicable when using HCP Terraform.\n\nState storage is handled automatically by HCP Terraform and so the state storage location is not configurable.", name),
				))
			}
		}
	}
	return diags
}

//-------------------------------------------------------------------
// Output constants and initialization code
//-------------------------------------------------------------------

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

const errBackendNewUnknown = `
The backend %q could not be found.

This is the backend specified in your Terraform configuration file.
This error could be a simple typo in your configuration, but it can also
be caused by using a Terraform version that doesn't support the specified
backend type. Please check your configuration and your Terraform version.

If you'd like to run Terraform and store state locally, you can fix this
error by removing the backend configuration from your configuration.
`

const errBackendNoExistingWorkspaces = `
No existing workspaces.

Use the "terraform workspace" command to create and select a new workspace.
If the backend already contains existing workspaces, you may need to update
the backend configuration.
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

const errBackendClearSaved = `
Error clearing the backend configuration: %s

Terraform removes the saved backend configuration when you're removing a
configured backend. This must be done so future Terraform runs know to not
use the backend configuration. Please look at the error above, resolve it,
and try again.
`

const errBackendInit = `
Reason: %s

The "backend" is the interface that Terraform uses to store state,
perform operations, etc. If this message is showing up, it means that the
Terraform configuration you're using is using a custom configuration for
the Terraform backend.

Changes to backend configurations require reinitialization. This allows
Terraform to set up the new configuration, copy existing state, etc. Please run
"terraform init" with either the "-reconfigure" or "-migrate-state" flags to
use the current configuration.

If the change reason above is incorrect, please verify your configuration
hasn't changed and try again. At this point, no changes to your existing
configuration or state have been made.
`

const errBackendInitCloud = `
Reason: %s.

Changes to the HCP Terraform configuration block require reinitialization, to discover any changes to the available workspaces.

To re-initialize, run:
  terraform init

Terraform has not yet made changes to your existing configuration or state.
`

const errBackendWriteSaved = `
Error saving the backend configuration: %s

Terraform saves the complete backend configuration in a local file for
configuring the backend on future operations. This cannot be disabled. Errors
are usually due to simple file permission errors. Please look at the error
above, resolve it, and try again.
`

const outputBackendMigrateChange = `
Terraform detected that the backend type changed from %q to %q.
`

const outputBackendMigrateLocal = `
Terraform has detected you're unconfiguring your previously set %q backend.
`

const outputBackendReconfigure = `
[reset][bold]Backend configuration changed![reset]

Terraform has detected that the configuration specified for the backend
has changed. Terraform will now check for existing state in the backends.
`

const inputCloudInitCreateWorkspace = `
There are no workspaces with the configured tags (%s)
in your HCP Terraform organization. To finish initializing, Terraform needs at
least one workspace available.

Terraform can create a properly tagged workspace for you now. Please enter a
name to create a new HCP Terraform workspace.
`

const successBackendUnset = `
Successfully unset the backend %q. Terraform will now operate locally.
`

const successBackendSet = `
Successfully configured the backend %q! Terraform will automatically
use this backend unless the backend configuration changes.
`

var migrateOrReconfigDiag = tfdiags.Sourceless(
	tfdiags.Error,
	"Backend configuration changed",
	"A change in the backend configuration has been detected, which may require migrating existing state.\n\n"+
		"If you wish to attempt automatic migration of the state, use \"terraform init -migrate-state\".\n"+
		`If you wish to store the current configuration with no changes to the state, use "terraform init -reconfigure".`)
