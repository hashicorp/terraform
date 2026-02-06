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
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/backend/backendrun"
	backendInit "github.com/hashicorp/terraform/internal/backend/init"
	"github.com/hashicorp/terraform/internal/backend/local"
	backendLocal "github.com/hashicorp/terraform/internal/backend/local"
	backendPluggable "github.com/hashicorp/terraform/internal/backend/pluggable"
	"github.com/hashicorp/terraform/internal/cloud"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/clistate"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/command/workdir"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/depsfile"
	"github.com/hashicorp/terraform/internal/didyoumean"
	"github.com/hashicorp/terraform/internal/getproviders/providerreqs"
	"github.com/hashicorp/terraform/internal/getproviders/reattach"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/states/statemgr"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
	tfversion "github.com/hashicorp/terraform/version"
)

// BackendOpts are the options used to initialize a backendrun.OperationsBackend.
type BackendOpts struct {
	// BackendConfig is a representation of the backend configuration block given in
	// the root module, or nil if no such block is present.
	BackendConfig *configs.Backend

	// StateStoreConfig is a representation of the state_store configuration block given in
	// the root module, or nil if no such block is present.
	StateStoreConfig *configs.StateStore

	ProviderRequirements *configs.RequiredProviders

	// Locks allows state-migration logic to detect when the provider used for pluggable state storage
	// during the last init (i.e. what's in the backend state file) is mismatched with the provider
	// version in use currently.
	Locks *depsfile.Locks

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

	// CreateDefaultWorkspace signifies whether the operations backend should create
	// the default workspace or not
	CreateDefaultWorkspace bool
}

// BackendWithRemoteTerraformVersion is a shared interface between the 'remote' and 'cloud' backends
// for simplified type checking when calling functions common to those particular backends.
type BackendWithRemoteTerraformVersion interface {
	IgnoreVersionConflict()
	VerifyWorkspaceTerraformVersion(workspace string) tfdiags.Diagnostics
	IsLocalOperations() bool
}

// Backend initializes and returns the operations backend for this CLI session.
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

	// If the result of loading a backend is an operations backend,
	// then return that as-is. This works even if b == nil (it will be !ok).
	if enhanced, ok := b.(backendrun.OperationsBackend); ok {
		log.Printf("[TRACE] Meta.Backend: backend %T supports operations", b)
		return enhanced, nil
	}

	// We either have a non-operations backend configured for state storage
	// or none configured at all. In either case, we use local as our operations backend
	// and the state-storage backend (if any) to manage state.

	if !opts.ForceLocal {
		log.Printf("[TRACE] Meta.Backend: backend %T does not support operations, so wrapping it in a local backend", b)
	}

	// Build the local operations backend
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
	stateStoreInUse := opts.StateStoreConfig != nil
	if !stateStoreInUse && m.backendConfigState == nil {
		// NOTE: This synthetic object is intentionally _not_ retained in the
		// on-disk record of the backend configuration, which was already dealt
		// with inside backendFromConfig, because we still need that codepath
		// to be able to recognize the lack of a config as distinct from
		// explicitly setting local until we do some more refactoring here.
		m.backendConfigState = &workdir.BackendConfigState{
			Type:      "local",
			ConfigRaw: json.RawMessage("{}"),
		}
	}

	return local, diags
}

// selectWorkspace gets a list of existing workspaces and then checks
// if the currently selected workspace is valid. If not, it will ask
// the user to select a workspace from the list.
func (m *Meta) selectWorkspace(b backend.Backend) error {
	workspaces, diags := b.Workspaces()
	if diags.HasErrors() && diags.Err().Error() == backend.ErrWorkspacesNotSupported.Error() {
		return nil
	}
	if diags.HasErrors() {
		return fmt.Errorf("Failed to get existing workspaces: %s", diags.Err())
	}
	if diags.HasWarnings() {
		log.Printf("[WARN] selectWorkspace: warning(s) returned when getting workspaces: %s", diags.ErrWithWarnings())
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
			return &errBackendNoExistingWorkspaces{}
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

// BackendForLocalPlan is similar to Backend, but uses settings that were
// stored in a plan when preparing the returned operations backend.
// The plan's data may describe `backend` or `state_store` configuration.
//
// The current workspace name is also stored as part of the plan, and so this
// method will check that it matches the currently-selected workspace name
// and produce error diagnostics if not.
func (m *Meta) BackendForLocalPlan(plan *plans.Plan) (backendrun.OperationsBackend, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	// Check the workspace name in the plan matches the current workspace
	currentWorkspace, err := m.Workspace()
	if err != nil {
		diags = diags.Append(fmt.Errorf("error determining current workspace when initializing a backend from the plan file: %w", err))
		return nil, diags
	}
	var plannedWorkspace string
	var isCloud bool
	switch {
	case plan.StateStore != nil:
		plannedWorkspace = plan.StateStore.Workspace
		isCloud = false
	case plan.Backend != nil:
		plannedWorkspace = plan.Backend.Workspace
		isCloud = plan.Backend.Type == "cloud"
	default:
		panic(fmt.Sprintf("Workspace data missing from plan file. Current workspace is %q. This is a bug in Terraform and should be reported.", currentWorkspace))
	}
	if currentWorkspace != plannedWorkspace {
		return nil, diags.Append(&errWrongWorkspaceForPlan{
			currentWorkspace: currentWorkspace,
			plannedWorkspace: plannedWorkspace,
			isCloud:          isCloud,
		})
	}

	var b backend.Backend
	switch {
	case plan.StateStore != nil:
		settings := plan.StateStore

		// BackendForLocalPlan is used in the context of an apply command using a plan file,
		// so we can read locks directly from the lock file and trust it contains what we need.
		locks, lockDiags := m.lockedDependencies()
		diags = diags.Append(lockDiags)
		if lockDiags.HasErrors() {
			return nil, diags
		}

		factories, err := m.ProviderFactoriesFromLocks(locks)
		if err != nil {
			// This may happen if the provider isn't present in the provider cache.
			// This should be caught earlier by logic that diffs the config against the backend state file.
			return nil, diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Provider unavailable",
				Detail: fmt.Sprintf("Terraform experienced an error when trying to use provider %s (%q) to initialize the %q state store: %s",
					settings.Provider.Source.Type,
					settings.Provider.Source,
					settings.Type,
					err),
			})
		}

		factory, exists := factories[*settings.Provider.Source]
		if !exists {
			return nil, diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Provider unavailable",
				Detail: fmt.Sprintf("The provider %s (%q) is required to initialize the %q state store, but the matching provider factory is missing. This is a bug in Terraform and should be reported.",
					settings.Provider.Source.Type,
					settings.Provider.Source,
					settings.Type,
				),
			})
		}

		provider, err := factory()
		if err != nil {
			diags = diags.Append(fmt.Errorf("error when obtaining provider instance during state store initialization: %w", err))
			return nil, diags
		}
		log.Printf("[TRACE] Meta.BackendForLocalPlan: launched instance of provider %s (%q)",
			settings.Provider.Source.Type,
			settings.Provider.Source,
		)

		// We purposefully don't have a deferred call to the provider's Close method here because the calling code needs a
		// running provider instance inside the returned backend.Backend instance.
		// Stopping the provider process is the responsibility of the calling code.

		resp := provider.GetProviderSchema()

		if len(resp.StateStores) == 0 {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Provider does not support pluggable state storage",
				Detail: fmt.Sprintf("There are no state stores implemented by provider %s (%q)",
					settings.Provider.Source.Type,
					settings.Provider.Source),
			})
			return nil, diags
		}

		stateStoreSchema, exists := resp.StateStores[settings.Type]
		if !exists {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "State store not implemented by the provider",
				Detail: fmt.Sprintf("State store %q is not implemented by provider %s (%q)",
					settings.Type,
					settings.Provider.Source.Type,
					settings.Provider.Source,
				),
			})
			return nil, diags
		}

		// Get the provider config from the backend state file.
		providerConfigVal, err := settings.Provider.Config.Decode(resp.Provider.Body.ImpliedType())
		if err != nil {
			diags = diags.Append(
				&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Error reading provider configuration state",
					Detail: fmt.Sprintf("Terraform experienced an error reading provider configuration for provider %s (%q) while configuring state store %s: %s",
						settings.Provider.Source.Type,
						settings.Provider.Source,
						settings.Type,
						err,
					),
				},
			)
			return nil, diags
		}

		// Get the state store config from the backend state file.
		stateStoreConfigVal, err := settings.Config.Decode(stateStoreSchema.Body.ImpliedType())
		if err != nil {
			diags = diags.Append(
				&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Error reading state store configuration state",
					Detail: fmt.Sprintf("Terraform experienced an error reading state store configuration for state store %s in provider %s (%q): %s",
						settings.Type,
						settings.Provider.Source.Type,
						settings.Provider.Source,
						err,
					),
				},
			)
			return nil, diags
		}

		// Validate and configure the provider
		//
		// NOTE: there are no marks we need to remove at this point.
		// We haven't added marks since the provider config from the backend state was used
		// because the state-storage provider's config isn't going to be presented to the user via terminal output or diags.
		validateResp := provider.ValidateProviderConfig(providers.ValidateProviderConfigRequest{
			Config: providerConfigVal,
		})
		diags = diags.Append(validateResp.Diagnostics)
		if diags.HasErrors() {
			return nil, diags
		}

		configureResp := provider.ConfigureProvider(providers.ConfigureProviderRequest{
			TerraformVersion: tfversion.SemVer.String(),
			Config:           providerConfigVal,
		})
		diags = diags.Append(configureResp.Diagnostics)
		if diags.HasErrors() {
			return nil, diags
		}

		// Now that the provider is configured we can begin using the state store through
		// the backend.Backend interface.
		p, err := backendPluggable.NewPluggable(provider, settings.Type)
		if err != nil {
			diags = diags.Append(err)
			return nil, diags
		}

		// Validate and configure the state store
		//
		// Note: we do not use the value returned from PrepareConfig for state stores,
		// however that old approach is still used with backends for compatibility reasons.
		_, validateDiags := p.PrepareConfig(stateStoreConfigVal)
		diags = diags.Append(validateDiags)
		if validateDiags.HasErrors() {
			return nil, diags
		}

		configureDiags := p.Configure(stateStoreConfigVal)
		diags = diags.Append(configureDiags)
		if configureDiags.HasErrors() {
			return nil, diags
		}
		log.Printf("[TRACE] Meta.BackendForLocalPlan: finished configuring state store %s in provider %s (%q)",
			settings.Type,
			settings.Provider.Source.Type,
			settings.Provider.Source,
		)

		// The fully configured Pluggable is used as the instance of backend.Backend
		b = p

	default:
		settings := plan.Backend

		f := backendInit.Backend(settings.Type)
		if f == nil {
			diags = diags.Append(errBackendSavedUnknown{settings.Type})
			return nil, diags
		}
		b = f()
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

// Operation initializes a new backendrun.Operation struct.
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

	var planOutBackend *plans.Backend
	var planOutStateStore *plans.StateStore
	switch {
	case m.backendConfigState != nil && m.stateStoreConfigState != nil:
		// Both set
		panic("failed to encode backend configuration for plan: both backend and state_store data present but they are mutually exclusive")
	case m.stateStoreConfigState != nil:
		// To access the provider schema, we need to access the underlying backends
		var providerSchema *configschema.Block
		lb := b.(*local.Local)
		p := lb.Backend.(*backendPluggable.Pluggable)
		providerSchema = p.ProviderSchema()

		planOutStateStore, err = m.stateStoreConfigState.PlanData(schema, providerSchema, workspace)
		if err != nil {
			// Always indicates an implementation error in practice, because
			// errors here indicate invalid encoding of the state_store configuration
			// in memory, and we should always have validated that by the time
			// we get here.
			panic(fmt.Sprintf("failed to encode state_store configuration for plan: %s", err))
		}
	default:
		// Either backendConfigState is set, or it's nil; PlanData method can handle either.
		planOutBackend, err = m.backendConfigState.PlanData(schema, nil, workspace)
		if err != nil {
			// Always indicates an implementation error in practice, because
			// errors here indicate invalid encoding of the backend configuration
			// in memory, and we should always have validated that by the time
			// we get here.
			panic(fmt.Sprintf("failed to encode backend configuration for plan: %s", err))
		}
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

	op := &backendrun.Operation{
		// These two fields are mutually exclusive; one is being assigned a nil value below.
		PlanOutBackend:    planOutBackend,
		PlanOutStateStore: planOutStateStore,

		Targets:         m.targets,
		UIIn:            m.UIInput(),
		UIOut:           m.Ui,
		Workspace:       workspace,
		StateLocker:     stateLocker,
		DependencyLocks: depLocks,
	}

	if op.PlanOutBackend != nil && op.PlanOutStateStore != nil {
		panic("failed to prepare operation: both backend and state_store configurations are present")
	}

	return op
}

// backendConfig returns the local configuration for the backend
func (m *Meta) backendConfig(opts *BackendOpts) (*configs.Backend, int, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	if opts.BackendConfig == nil {
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
		opts.BackendConfig = conf
	}

	c := opts.BackendConfig

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

// stateStoreConfig returns the local 'state_store' configuration
// This method:
// > Ensures that that state store type exists in the linked provider.
// > Returns config that is the combination of config and any config overrides originally supplied via the CLI.
// > Returns a hash of the config in the configuration files, i.e. excluding overrides
func (m *Meta) stateStoreConfig(opts *BackendOpts) (*configs.StateStore, int, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	c := opts.StateStoreConfig
	if c == nil {
		// We choose to not to re-parse the config to look for data if it's missing,
		// which currently happens in the similar `backendConfig` method.
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Missing state store configuration",
			Detail:   "Terraform attempted to configure a state store when no parsed 'state_store' configuration was present. This is a bug in Terraform and should be reported.",
		})
		return nil, 0, diags
	}

	if errs := c.VerifyDependencySelections(opts.Locks, opts.ProviderRequirements); len(errs) > 0 {
		var buf strings.Builder
		for _, err := range errs {
			fmt.Fprintf(&buf, "\n  - %s", err.Error())
		}
		var suggestion string
		switch {
		case opts.Locks == nil:
			// If we get here then it suggests that there's a caller that we
			// didn't yet update to populate DependencyLocks, which is a bug.
			panic("This run has no dependency lock information provided at all, which is a bug in Terraform; please report it!")
		case opts.Locks.Empty():
			suggestion = "To make the initial dependency selections that will initialize the dependency lock file, run:\n  terraform init"
		default:
			suggestion = "To update the locked dependency selections to match a changed configuration, run:\n  terraform init -upgrade"
		}
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Inconsistent dependency lock file",
			fmt.Sprintf(
				"The following dependency selections recorded in the lock file are inconsistent with the current configuration:%s\n\n%s",
				buf.String(), suggestion,
			),
		))
		return nil, 0, diags
	}

	// Get the provider version from locks, as this impacts the hash
	// NOTE: this assumes that we will never allow users to override config definint which provider is used for state storage
	stateStoreProviderVersion, vDiags := getStateStorageProviderVersion(opts.StateStoreConfig, opts.Locks)
	diags = diags.Append(vDiags)
	if vDiags.HasErrors() {
		return nil, 0, diags
	}

	pFactory, pDiags := m.StateStoreProviderFactoryFromConfig(opts.StateStoreConfig, opts.Locks)
	diags = diags.Append(pDiags)
	if pDiags.HasErrors() {
		return nil, 0, diags
	}

	provider, err := pFactory()
	if err != nil {
		diags = diags.Append(fmt.Errorf("error when obtaining provider instance during state store initialization: %w", err))
		return nil, 0, diags
	}
	defer provider.Close() // Stop the child process once we're done with it here.

	resp := provider.GetProviderSchema()

	if len(resp.StateStores) == 0 {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Provider does not support pluggable state storage",
			Detail: fmt.Sprintf("There are no state stores implemented by provider %s (%q)",
				c.Provider.Name,
				c.ProviderAddr),
			Subject: &c.DeclRange,
		})
		return nil, 0, diags
	}

	stateStoreSchema, exists := resp.StateStores[c.Type]
	if !exists {
		suggestions := slices.Sorted(maps.Keys(resp.StateStores))
		suggestion := didyoumean.NameSuggestion(c.Type, suggestions)
		if suggestion != "" {
			suggestion = fmt.Sprintf(" Did you mean %q?", suggestion)
		}
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "State store not implemented by the provider",
			Detail: fmt.Sprintf("State store %q is not implemented by provider %s (%q)%s",
				c.Type, c.Provider.Name,
				c.ProviderAddr, suggestion),
			Subject: &c.DeclRange,
		})
		return nil, 0, diags
	}

	// We know that the provider contains a state store with the correct type name.
	// Validation of the config against the schema happens later.
	// For now, we:
	// > Get a hash of the present config
	// > Apply any overrides

	configBody := c.Config
	hash, diags := c.Hash(stateStoreSchema.Body, resp.Provider.Body, stateStoreProviderVersion)

	// If we have an override configuration body then we must apply it now.
	if opts.ConfigOverride != nil {
		log.Println("[TRACE] Meta.Backend: merging -backend-config=... CLI overrides into state_store configuration")
		configBody = configs.MergeBodies(configBody, opts.ConfigOverride)
	}

	log.Printf("[TRACE] Meta.Backend: built configuration for %q state_store with hash value %d", c.Type, hash)

	// We'll shallow-copy configs.StateStore here so that we can replace the
	// body without affecting others that hold this reference.
	configCopy := *c
	configCopy.Config = configBody
	return &configCopy, hash, diags
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
	var diags tfdiags.Diagnostics

	// Get the local 'backend' or 'state_store' configuration.
	var backendConfig *configs.Backend
	var stateStoreConfig *configs.StateStore
	var cHash int
	if opts.StateStoreConfig != nil {
		// state store has been parsed from config and is included in opts
		var ssDiags tfdiags.Diagnostics
		stateStoreConfig, cHash, ssDiags = m.stateStoreConfig(opts)
		diags = diags.Append(ssDiags)
		if ssDiags.HasErrors() {
			return nil, diags
		}
	} else {
		// backend config may or may not have been parsed and included in opts,
		// or may not exist in config at all (default/implied local backend)
		var beDiags tfdiags.Diagnostics
		backendConfig, cHash, beDiags = m.backendConfig(opts)
		diags = diags.Append(beDiags)
		if beDiags.HasErrors() {
			return nil, diags
		}
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
		diags = diags.Append(fmt.Errorf("Failed to load the backend state file: %s", err))
		return nil, diags
	}

	// Load the state, it must be non-nil for the tests below but can be empty
	s := sMgr.State()
	if s == nil {
		log.Printf("[TRACE] Meta.Backend: backend has not previously been initialized in this working directory")
		s = workdir.NewBackendStateFile()
	} else if s.Backend != nil {
		log.Printf("[TRACE] Meta.Backend: working directory was previously initialized for %q backend", s.Backend.Type)
	} else if s.StateStore != nil {
		log.Printf("[TRACE] Meta.Backend: working directory was previously initialized for %q state_store using provider %q, version %s",
			s.StateStore.Type,
			s.StateStore.Provider.Source,
			s.StateStore.Provider.Version)
	} else {
		log.Printf("[TRACE] Meta.Backend: working directory was previously initialized but has no backend (is using legacy remote state?)")
	}

	// if we want to force reconfiguration of the backend or state store, we set the backend
	// and state_store state to nil on this copy. This will direct us through the correct
	if m.reconfigure {
		s.Backend = nil
		s.StateStore = nil
	}

	// Upon return, we want to set the state we're using in-memory so that
	// we can access it for commands.
	//
	// Currently the only command using these values is the `plan` command,
	// which records the data in the plan file.
	m.backendConfigState = nil
	m.stateStoreConfigState = nil
	defer func() {
		s := sMgr.State()
		switch {
		case s == nil:
			// Do nothing
			/* If there is no backend state file then either:
			1. The working directory isn't initialized yet.
				The user is either in the process of running an init command, in which case the values set via this deferred function will not be used,
				or they are performing a non-init command that will be interrupted by an error before these values are used in downstream
			2. There isn't any backend or state_store configuration and an implied local backend is in use.
				This is valid and will mean m.backendConfigState is nil until the calling code adds a synthetic object in:
				https://github.com/hashicorp/terraform/blob/3eea12a1d810a17e9c8e43cf7774817641ca9bc1/internal/command/meta_backend.go#L213-L234
			*/
		case !s.Backend.Empty():
			m.backendConfigState = s.Backend
		case !s.StateStore.Empty():
			m.stateStoreConfigState = s.StateStore
		}
	}()

	// This switch statement covers all the different combinations of
	// configuring new backends, updating previously-configured backends, etc.
	switch {
	// No configuration set at all. Pure local state.
	case backendConfig == nil && s.Backend.Empty() &&
		stateStoreConfig == nil && s.StateStore.Empty():
		log.Printf("[TRACE] Meta.Backend: using default local state only (no backend configuration, and no existing initialized backend)")
		return nil, nil

	// We're unsetting a backend (moving from backend => local)
	case backendConfig == nil && !s.Backend.Empty() &&
		stateStoreConfig == nil && s.StateStore.Empty():
		log.Printf("[TRACE] Meta.Backend: previously-initialized %q backend is no longer present in config", s.Backend.Type)

		initReason := fmt.Sprintf("Unsetting the previously set backend %q", s.Backend.Type)
		if !opts.Init {
			diags = diags.Append(errBackendInitDiag(initReason))
			return nil, diags
		}

		if s.Backend.Type != "cloud" && !m.migrateState {
			diags = diags.Append(migrateOrReconfigDiag)
			return nil, diags
		}

		return m.backend_c_r_S(backendConfig, cHash, sMgr, true, opts)

	// We're unsetting a state_store (moving from state_store => local)
	case stateStoreConfig == nil && !s.StateStore.Empty() &&
		backendConfig == nil && s.Backend.Empty():
		log.Printf("[TRACE] Meta.Backend: previously-initialized state_store %q in provider %s (%q) is no longer present in config",
			s.StateStore.Type,
			s.StateStore.Provider.Source.Type,
			s.StateStore.Provider.Source,
		)

		initReason := fmt.Sprintf("Unsetting the previously set state store %q", s.StateStore.Type)
		if !opts.Init {
			diags = diags.Append(errStateStoreInitDiag(initReason))
			return nil, diags
		}

		if !m.migrateState {
			diags = diags.Append(migrateOrReconfigStateStoreDiag)
			return nil, diags
		}

		// Grab a purely local backend to be the destination for migrated state
		localB, moreDiags := m.Backend(&BackendOpts{ForceLocal: true, Init: true})
		diags = diags.Append(moreDiags)
		if moreDiags.HasErrors() {
			return nil, diags
		}

		v := views.NewInit(opts.ViewType, m.View)
		v.Output(views.InitMessageCode("state_store_unset"), s.StateStore.Type)

		return m.stateStore_to_backend(sMgr, "local", localB, nil, opts.ViewType)

	// Configuring a backend for the first time or -reconfigure flag was used
	case backendConfig != nil && s.Backend.Empty() &&
		stateStoreConfig == nil && s.StateStore.Empty():
		log.Printf("[TRACE] Meta.Backend: moving from default local state only to %q backend", backendConfig.Type)
		if !opts.Init {
			if backendConfig.Type == "cloud" {
				initReason := "Initial configuration of HCP Terraform or Terraform Enterprise"
				diags = diags.Append(errBackendInitCloudDiag(initReason))
			} else {
				initReason := fmt.Sprintf("Initial configuration of the requested backend %q", backendConfig.Type)
				diags = diags.Append(errBackendInitDiag(initReason))
			}
			return nil, diags
		}
		return m.backend_C_r_s(backendConfig, cHash, sMgr, opts)

	// Configuring a state store for the first time or -reconfigure flag was used
	case stateStoreConfig != nil && s.StateStore.Empty() &&
		backendConfig == nil && s.Backend.Empty():
		log.Printf("[TRACE] Meta.Backend: moving from default local state only to state_store %q in provider %s (%q)",
			stateStoreConfig.Type,
			stateStoreConfig.Provider.Name,
			stateStoreConfig.ProviderAddr,
		)

		if !opts.Init {
			initReason := fmt.Sprintf("Initial configuration of the requested state_store %q in provider %s (%q)",
				stateStoreConfig.Type,
				stateStoreConfig.Provider.Name,
				stateStoreConfig.ProviderAddr,
			)
			diags = diags.Append(errStateStoreInitDiag(initReason))
			return nil, diags
		}

		return m.stateStore_C_s(stateStoreConfig, cHash, sMgr, opts)

	// Migration from state store to backend
	case backendConfig != nil && s.Backend.Empty() &&
		stateStoreConfig == nil && !s.StateStore.Empty():
		log.Printf("[TRACE] Meta.Backend: config has changed from state_store %q in provider %s (%q) to backend %q",
			s.StateStore.Type,
			s.StateStore.Provider.Source.Type,
			s.StateStore.Provider.Source,
			backendConfig.Type,
		)

		if !opts.Init {
			initReason := fmt.Sprintf("Migrating from state store %q to backend %q",
				s.StateStore.Type, backendConfig.Type)
			diags = diags.Append(errBackendInitDiag(initReason))
			return nil, diags
		}

		b, configVal, moreDiags := m.backendInitFromConfig(backendConfig)
		diags = diags.Append(moreDiags)
		if moreDiags.HasErrors() {
			return nil, diags
		}

		v := views.NewInit(opts.ViewType, m.View)
		v.Output(views.InitMessageCode("state_store_migrate_backend"), s.StateStore.Type, backendConfig.Type)

		newBackendCfgState := &workdir.BackendConfigState{
			Type: backendConfig.Type,
		}
		newBackendCfgState.SetConfig(configVal, b.ConfigSchema())
		newBackendCfgState.Hash = uint64(cHash)

		return m.stateStore_to_backend(sMgr, backendConfig.Type, b, newBackendCfgState, opts.ViewType)

	// Migration from backend to state store
	case backendConfig == nil && !s.Backend.Empty() &&
		stateStoreConfig != nil && s.StateStore.Empty():
		log.Printf("[TRACE] Meta.Backend: config has changed from backend %q to state_store %q in provider %s (%q)",
			s.Backend.Type,
			stateStoreConfig.Type,
			stateStoreConfig.Provider.Name,
			stateStoreConfig.ProviderAddr,
		)

		if !opts.Init {
			initReason := fmt.Sprintf("Migrating from backend %q to state store %q in provider %s (%q)",
				s.Backend.Type, stateStoreConfig.Type,
				stateStoreConfig.Provider.Name, stateStoreConfig.ProviderAddr)
			diags = diags.Append(errBackendInitDiag(initReason))
			return nil, diags
		}

		return m.backend_to_stateStore(s.Backend, sMgr, stateStoreConfig, cHash, opts)

	// Potentially changing a backend configuration
	case backendConfig != nil && !s.Backend.Empty() &&
		stateStoreConfig == nil && s.StateStore.Empty():
		// We are not going to migrate if...
		//
		// We're not initializing
		// AND the backend cache hash values match, indicating that the stored config is valid and completely unchanged.
		// AND we're not providing any overrides. An override can mean a change overriding an unchanged backend block (indicated by the hash value).
		if (uint64(cHash) == s.Backend.Hash) && (!opts.Init || opts.ConfigOverride == nil) {
			log.Printf("[TRACE] Meta.Backend: using already-initialized, unchanged %q backend configuration", backendConfig.Type)
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
		if !m.backendConfigNeedsMigration(backendConfig, s.Backend) {
			log.Printf("[TRACE] Meta.Backend: using already-initialized %q backend configuration", backendConfig.Type)
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
		log.Printf("[TRACE] Meta.Backend: backend configuration has changed (from type %q to type %q)", s.Backend.Type, backendConfig.Type)

		cloudMode := cloud.DetectConfigChangeType(s.Backend, backendConfig, false)

		if !opts.Init {
			// user ran another cmd that is not init but they are required to initialize because of a potential relevant change to their backend configuration
			initDiag := m.determineInitReason(s.Backend.Type, backendConfig.Type, cloudMode)
			diags = diags.Append(initDiag)
			return nil, diags
		}

		if !cloudMode.InvolvesCloud() && !m.migrateState {
			diags = diags.Append(migrateOrReconfigDiag)
			return nil, diags
		}

		log.Printf("[WARN] backend config has changed since last init")
		return m.backend_C_r_S_changed(backendConfig, cHash, sMgr, true, opts)

	// Potentially changing a state store configuration
	case backendConfig == nil && s.Backend.Empty() &&
		stateStoreConfig != nil && !s.StateStore.Empty():
		// When implemented, this will need to handle multiple scenarios like:
		// > Changing to using a different provider for PSS.
		// > Changing to using a different version of the same provider for PSS.
		// >>>> Navigating state upgrades that do not force an explicit migration &&
		//      identifying when migration is required.
		// > Changing to using a different store in the same version of the provider.
		// > Changing how the provider is configured.
		// > Changing how the store is configured.
		// > Allowing values to be moved between partial overrides and config

		// We're not initializing
		// AND the config's and backend state file's hash values match, indicating that the stored config is valid and completely unchanged.
		// AND we're not providing any overrides. An override can mean a change overriding an unchanged backend block (indicated by the hash value).
		if (uint64(cHash) == s.StateStore.Hash) && (!opts.Init || opts.ConfigOverride == nil) {
			log.Printf("[TRACE] Meta.Backend: using already-initialized, unchanged %q state_store configuration", stateStoreConfig.Type)
			savedStateStore, sssDiags := m.savedStateStore(sMgr)
			diags = diags.Append(sssDiags)
			// Verify that selected workspace exist. Otherwise prompt user to create one
			if opts.Init && savedStateStore != nil {
				if err := m.selectWorkspace(savedStateStore); err != nil {
					diags = diags.Append(err)
					return nil, diags
				}
			}
			return savedStateStore, diags
		}

		// Above caters only for unchanged config
		// but this switch case will also handle changes,
		// which isn't implemented yet.
		return nil, diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Not implemented yet",
			Detail:   "Changing a state store configuration is not implemented yet",
		})

	default:
		diags = diags.Append(fmt.Errorf(
			"Unhandled backend configuration state. This is a bug. Please\n"+
				"report this error with the following information.\n\n"+
				"Backend Config Nil: %v\n"+
				"Saved Backend Empty: %v\n"+
				"StateStore Config Nil: %v\n"+
				"Saved StateStore Empty: %v\n",
			backendConfig == nil,
			s.Backend.Empty(),
			stateStoreConfig == nil,
			s.StateStore.Empty(),
		))
		return nil, diags
	}
}

// determineInitReason is used in non-Init commands to interrupt the command early and prompt users to instead run an init command.
// That prompt needs to include the reason why init needs to be run, and it is determined here.
//
// Note: the calling code is responsible for determining that a change has occurred before invoking this
// method. This makes the default cases (config has changed) valid.
func (m *Meta) determineInitReason(previousBackendType string, currentBackendType string, cloudMode cloud.ConfigChangeMode) tfdiags.Diagnostics {
	initReason := ""
	switch cloudMode {
	case cloud.ConfigMigrationIn:
		initReason = fmt.Sprintf("Changed from backend %q to HCP Terraform", previousBackendType)
	case cloud.ConfigMigrationOut:
		initReason = fmt.Sprintf("Changed from HCP Terraform to backend %q", currentBackendType)
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
		diags = diags.Append(errBackendInitCloudDiag(initReason))
	case cloud.ConfigMigrationIn:
		diags = diags.Append(errBackendInitCloudDiag(initReason))
	default:
		diags = diags.Append(errBackendInitDiag(initReason))
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

	// Depending on the contents of the backend state file,
	// prepare a backend.Backend in the appropriate way.
	var b backend.Backend
	switch {
	case !s.StateStore.Empty():
		// state_store
		log.Printf("[TRACE] Meta.Backend: working directory was previously initialized for %q state store", s.StateStore.Type)
		var ssDiags tfdiags.Diagnostics
		b, ssDiags = m.savedStateStore(sMgr) // Relies on the state manager's internal state being refreshed above.
		diags = diags.Append(ssDiags)
		if ssDiags.HasErrors() {
			return nil, diags
		}
	case !s.Backend.Empty():
		// backend or cloud
		if s.Backend.Type == "" {
			return backendLocal.New(), diags
		}
		f := backendInit.Backend(s.Backend.Type)
		if f == nil {
			diags = diags.Append(errBackendSavedUnknown{s.Backend.Type})
			return nil, diags
		}
		b = f()

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

		log.Printf("[TRACE] Meta.Backend: working directory was previously initialized for %q backend", s.Backend.Type)
	default:
		// s.StateStore and s.Backend are empty, so return a local backend
		log.Printf("[TRACE] Meta.Backend: working directory was previously initialized but has no backend (is using legacy remote state?)")
		b = backendLocal.New()
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
	c *configs.Backend, cHash int, sMgr *clistate.LocalState, output bool, opts *BackendOpts,
) (backend.Backend, tfdiags.Diagnostics) {
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

	view := views.NewInit(vt, m.View)
	if cloudMode == cloud.ConfigMigrationOut {
		view.Output(views.BackendCloudMigrateLocalMessage)
	} else {
		view.Output(views.BackendMigrateLocalMessage, s.Backend.Type)
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
		diags = diags.Append(errBackendClearSaved{err})
		return nil, diags
	}
	if err := sMgr.PersistState(); err != nil {
		diags = diags.Append(errBackendClearSaved{err})
		return nil, diags
	}

	if output {
		view.Output(views.BackendConfiguredUnsetMessage, backendType)
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
	diags = diags.Append(localBDiags)
	if localBDiags.HasErrors() {
		return nil, diags
	}

	workspaces, wDiags := localB.Workspaces()
	if wDiags.HasErrors() {
		diags = diags.Append(wDiags.Warnings())
		diags = diags.Append(&errBackendLocalRead{wDiags.Err()})
		return nil, diags
	}

	var localStates []statemgr.Full
	for _, workspace := range workspaces {
		localState, sDiags := localB.StateMgr(workspace)
		if sDiags.HasErrors() {
			diags = diags.Append(sDiags.Warnings())
			diags = diags.Append(&errBackendLocalRead{sDiags.Err()})
			return nil, diags
		}
		if err := localState.RefreshState(); err != nil {
			diags = diags.Append(&errBackendLocalRead{err})
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
		err := m.backendMigrateState(&backendMigrateOpts{
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
					diags = diags.Append(&errBackendMigrateLocalDelete{err})
					return nil, diags
				}
				if err := localState.PersistState(nil); err != nil {
					diags = diags.Append(&errBackendMigrateLocalDelete{err})
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

	// Store the metadata in our saved state location
	s := sMgr.State()
	if s == nil {
		s = workdir.NewBackendStateFile()
	}
	s.Backend = &workdir.BackendConfigState{
		Type: c.Type,
		Hash: uint64(cHash),
	}
	err := s.Backend.SetConfig(configVal, b.ConfigSchema())
	if err != nil {
		diags = diags.Append(fmt.Errorf("Can't serialize backend configuration as JSON: %s", err))
		return nil, diags
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
		diags = diags.Append(errBackendWriteSavedDiag(err))
		return nil, diags
	}
	if err := sMgr.PersistState(); err != nil {
		diags = diags.Append(errBackendWriteSavedDiag(err))
		return nil, diags
	}

	// By now the backend is successfully configured.  If using HCP Terraform, the success
	// message is handled as part of the final init message
	if _, ok := b.(*cloud.Cloud); !ok {
		view := views.NewInit(vt, m.View)
		view.Output(views.BackendConfiguredSuccessMessage, s.Backend.Type)
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
		view := views.NewInit(vt, m.View)
		switch cloudMode {
		case cloud.ConfigChangeInPlace:
			view.Output(views.BackendCloudChangeInPlaceMessage)
		case cloud.ConfigMigrationIn:
			view.Output(views.BackendMigrateToCloudMessage, s.Backend.Type)
		case cloud.ConfigMigrationOut:
			view.Output(views.BackendMigrateFromCloudMessage, c.Type)
		default:
			if s.Backend.Type != c.Type {
				view.Output(views.BackendMigrateTypeChangeMessage, s.Backend.Type, c.Type)
			} else {
				view.Output(views.BackendReconfigureMessage)
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

	// Update the backend state
	s = sMgr.State()
	if s == nil {
		s = workdir.NewBackendStateFile()
	}
	s.Backend = &workdir.BackendConfigState{
		Type: c.Type,
		Hash: uint64(cHash),
	}
	err := s.Backend.SetConfig(configVal, b.ConfigSchema())
	if err != nil {
		diags = diags.Append(fmt.Errorf("Can't serialize backend configuration as JSON: %s", err))
		return nil, diags
	}

	// Verify that selected workspace exist. Otherwise prompt user to create one
	if opts.Init && b != nil {
		if err := m.selectWorkspace(b); err != nil {
			diags = diags.Append(err)
			return b, diags
		}
	}

	if err := sMgr.WriteState(s); err != nil {
		diags = diags.Append(errBackendWriteSavedDiag(err))
		return nil, diags
	}
	if err := sMgr.PersistState(); err != nil {
		diags = diags.Append(errBackendWriteSavedDiag(err))
		return nil, diags
	}

	if output {
		// By now the backend is successfully configured.  If using HCP Terraform, the success
		// message is handled as part of the final init message
		if _, ok := b.(*cloud.Cloud); !ok {
			view := views.NewInit(vt, m.View)
			view.Output(views.BackendConfiguredSuccessMessage, s.Backend.Type)
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
		diags = diags.Append(errBackendSavedUnknown{s.Backend.Type})
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
			diags = diags.Append(errBackendWriteSavedDiag(err))
		}
		// No need to call PersistState as it's a no-op
	}

	return diags
}

// backend returns an operations backend that may use a backend, cloud, or state_store block for state storage.
// Based on the supplied config, it prepares arguments to pass into (Meta).Backend, which returns the operations backend.
//
// This method should be used in NON-init operations only; it's incapable of processing new init command CLI flags used
// for partial configuration, however it will use the backend state file to use partial configuration from a previous
// init command.
func (m *Meta) backend(configPath string, viewType arguments.ViewType) (backendrun.OperationsBackend, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	if configPath == "" {
		configPath = "."
	}

	// Only return error diagnostics at this point. Any warnings will be caught
	// again later and duplicated in the output.
	root, mDiags := m.loadSingleModule(configPath)
	if mDiags.HasErrors() {
		diags = diags.Append(mDiags)
		return nil, diags
	}

	locks, lDiags := m.lockedDependencies()
	diags = diags.Append(lDiags)
	if lDiags.HasErrors() {
		return nil, diags
	}

	var opts *BackendOpts
	switch {
	case root.Backend != nil:
		opts = &BackendOpts{
			BackendConfig: root.Backend,
			Locks:         locks,
			ViewType:      viewType,
		}
	case root.CloudConfig != nil:
		backendConfig := root.CloudConfig.ToBackendConfig()
		opts = &BackendOpts{
			BackendConfig: &backendConfig,
			Locks:         locks,
			ViewType:      viewType,
		}
	case root.StateStore != nil:
		opts = &BackendOpts{
			StateStoreConfig:     root.StateStore,
			ProviderRequirements: root.ProviderRequirements,
			Locks:                locks,
			ViewType:             viewType,
		}
	default:
		// there is no config; defaults to local state storage
		opts = &BackendOpts{
			Locks:    locks,
			ViewType: viewType,
		}
	}

	// This method should not be used for init commands,
	// so we always set this value as false.
	opts.Init = false

	// Load the backend
	be, beDiags := m.Backend(opts)
	diags = diags.Append(beDiags)
	if beDiags.HasErrors() {
		return nil, diags
	}

	return be, diags
}

func (m *Meta) backend_to_stateStore(bcs *workdir.BackendConfigState, sMgr *clistate.LocalState, c *configs.StateStore, cHash int, opts *BackendOpts) (backend.Backend, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	vt := arguments.ViewJSON
	// Set default viewtype if none was set as the StateLocker needs to know exactly
	// what viewType we want to have.
	if opts == nil || opts.ViewType != vt {
		vt = arguments.ViewHuman
	}

	s := sMgr.State()

	cloudMode := cloud.DetectConfigChangeType(bcs, nil, false)
	diags = diags.Append(m.assertSupportedCloudInitOptions(cloudMode))
	if diags.HasErrors() {
		return nil, diags
	}

	view := views.NewInit(vt, m.View)
	if cloudMode == cloud.ConfigMigrationOut {
		view.Output(views.BackendCloudMigrateStateStoreMessage, c.Type)
	} else {
		view.Output(views.BackendMigrateStateStoreMessage, bcs.Type, c.Type)
	}

	// Initialize the configured backend
	b, moreDiags := m.savedBackend(sMgr)
	diags = diags.Append(moreDiags)
	if moreDiags.HasErrors() {
		return nil, diags
	}

	// Get the state store as an instance of backend.Backend
	ssBackend, storeConfigVal, providerConfigVal, moreDiags := m.stateStoreInitFromConfig(c, opts.Locks)
	diags = diags.Append(moreDiags)
	if diags.HasErrors() {
		return nil, diags
	}

	// Perform the migration
	err := m.backendMigrateState(&backendMigrateOpts{
		SourceType:      bcs.Type,
		DestinationType: c.Type,
		Source:          b,
		Destination:     ssBackend,
		ViewType:        vt,
	})
	if err != nil {
		diags = diags.Append(err)
		return nil, diags
	}

	rDiags := m.removeLocalState(bcs.Type, b)
	if rDiags.HasErrors() {
		diags = diags.Append(rDiags)
		return nil, diags
	}

	if m.stateLock {
		view := views.NewStateLocker(vt, m.View)
		stateLocker := clistate.NewLocker(m.stateLockTimeout, view)
		if err := stateLocker.Lock(sMgr, "init is initializing state_store first time"); err != nil {
			diags = diags.Append(fmt.Errorf("Error locking state: %s", err))
			return nil, diags
		}
		defer stateLocker.Unlock()
	}

	// Store the state_store metadata in our saved state location

	var pVersion *version.Version // This will remain nil for builtin providers or unmanaged providers.
	if c.ProviderAddr.IsBuiltIn() {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagWarning,
			Summary:  "State storage is using a builtin provider",
			Detail:   "Terraform is using a builtin provider for initializing state storage. Terraform will be less able to detect when state migrations are required in future init commands.",
		})
	} else {
		isReattached, err := reattach.IsProviderReattached(c.ProviderAddr, os.Getenv("TF_REATTACH_PROVIDERS"))
		if err != nil {
			diags = diags.Append(fmt.Errorf("Unable to determine if state storage provider is reattached while initializing state store for the first time. This is a bug in Terraform and should be reported: %w", err))
			return nil, diags
		}
		if isReattached {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagWarning,
				Summary:  "State storage provider is not managed by Terraform",
				Detail:   "Terraform is using a provider supplied via TF_REATTACH_PROVIDERS for initializing state storage. Terraform will be less able to detect when state migrations are required in future init commands.",
			})
		} else {
			// The provider is not built in and is being managed by Terraform
			// This is the most common scenario, by far.
			var vDiags tfdiags.Diagnostics
			pVersion, vDiags = getStateStorageProviderVersion(c, opts.Locks)
			diags = diags.Append(vDiags)
			if vDiags.HasErrors() {
				return nil, diags
			}
		}
	}

	// Update the stored metadata
	s.Backend = nil
	s.StateStore = &workdir.StateStoreConfigState{
		Type: c.Type,
		Hash: uint64(cHash),
		Provider: &workdir.ProviderConfigState{
			Source:  &c.ProviderAddr,
			Version: pVersion,
		},
	}
	err = s.StateStore.SetConfig(storeConfigVal, ssBackend.ConfigSchema())
	if err != nil {
		diags = diags.Append(fmt.Errorf("Failed to set state store configuration: %w", err))
		return nil, diags
	}

	// We need to briefly convert away from backend.Backend interface to use the method
	// for accessing the provider schema. In this method we _always_ expect the concrete value
	// to be backendPluggable.Pluggable.
	plug := ssBackend.(*backendPluggable.Pluggable)
	err = s.StateStore.Provider.SetConfig(providerConfigVal, plug.ProviderSchema())
	if err != nil {
		diags = diags.Append(fmt.Errorf("Failed to set state store provider configuration: %w", err))
		return nil, diags
	}

	// Update backend state file
	if err := sMgr.WriteState(s); err != nil {
		diags = diags.Append(errBackendWriteSavedDiag(err))
		return nil, diags
	}
	if err := sMgr.PersistState(); err != nil {
		diags = diags.Append(errBackendWriteSavedDiag(err))
		return nil, diags
	}

	return b, diags
}

func (m *Meta) removeLocalState(backendType string, b backend.Backend) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	if backendType != "local" {
		return diags
	}

	workspaces, wDiags := b.Workspaces()
	if wDiags.HasErrors() {
		diags = diags.Append(&errBackendLocalRead{wDiags.Err()})
		return diags
	}

	var localStates []statemgr.Full
	for _, workspace := range workspaces {
		localState, sDiags := b.StateMgr(workspace)
		if sDiags.HasErrors() {
			diags = diags.Append(&errBackendLocalRead{sDiags.Err()})
			return diags
		}
		if err := localState.RefreshState(); err != nil {
			diags = diags.Append(&errBackendLocalRead{err})
			return diags
		}

		// We only care about non-empty states.
		if localS := localState.State(); !localS.Empty() {
			log.Printf("[TRACE] Meta.Backend: will need to migrate workspace states because of existing %q workspace", workspace)
			localStates = append(localStates, localState)
		} else {
			log.Printf("[TRACE] Meta.Backend: ignoring local %q workspace because its state is empty", workspace)
		}
	}

	if len(localStates) > 0 {
		log.Printf("[TRACE] Meta.removeLocalState: removing old state snapshots (%d) from old backend", len(localStates))
		for idx, localState := range localStates {
			// We always delete the local state, unless that was our new state too.
			if err := localState.WriteState(nil); err != nil {
				diags = diags.Append(&errBackendMigrateLocalDelete{err})
				return diags
			}
			if err := localState.PersistState(nil); err != nil {
				diags = diags.Append(&errBackendMigrateLocalDelete{err})
				return diags
			}
			log.Printf("[DEBUG] Meta.removeLocalState: deleted local state for workspace %q", workspaces[idx])
		}
	}
	return diags
}

//-------------------------------------------------------------------
// State Store Config Scenarios
// The functions below cover handling all the various scenarios that
// can exist when loading a state store. They are named in the format of
// "stateStore_C_S" where C and S may be upper or lowercase. Lowercase
// means it is false, uppercase means it is true.
//
// The fields are:
//
//   * C - State store configuration is set and changed in TF files
//   * S - State store configuration is set in the state
//
//-------------------------------------------------------------------

// Configuring a state_store for the first time.
func (m *Meta) stateStore_C_s(c *configs.StateStore, stateStoreHash int, backendSMgr *clistate.LocalState, opts *BackendOpts) (backend.Backend, tfdiags.Diagnostics) {
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

	workspaces, wDiags := localB.Workspaces()
	if wDiags.HasErrors() {
		diags = diags.Append(&errBackendLocalRead{wDiags.Err()})
		return nil, diags
	}

	var localStates []statemgr.Full
	for _, workspace := range workspaces {
		localState, sDiags := localB.StateMgr(workspace)
		if sDiags.HasErrors() {
			diags = diags.Append(&errBackendLocalRead{sDiags.Err()})
			return nil, diags
		}
		if err := localState.RefreshState(); err != nil {
			diags = diags.Append(&errBackendLocalRead{err})
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

	// Get the state store as an instance of backend.Backend
	b, storeConfigVal, providerConfigVal, moreDiags := m.stateStoreInitFromConfig(c, opts.Locks)
	diags = diags.Append(moreDiags)
	if diags.HasErrors() {
		return nil, diags
	}

	if len(localStates) > 0 {
		// Migrate any local states into the new state store
		err := m.backendMigrateState(&backendMigrateOpts{
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

		// We remove the local state after migration to prevent confusion
		// As we're migrating to a state store we don't have insight into whether it stores
		// files locally at all, and whether those local files conflict with the location of
		// the old local state.
		log.Printf("[TRACE] Meta.Backend: removing old state snapshots from old backend")
		for _, localState := range localStates {
			// We always delete the local state, unless that was our new state too.
			if err := localState.WriteState(nil); err != nil {
				diags = diags.Append(&errBackendMigrateLocalDelete{err})
				return nil, diags
			}
			if err := localState.PersistState(nil); err != nil {
				diags = diags.Append(&errBackendMigrateLocalDelete{err})
				return nil, diags
			}
		}
	}

	if m.stateLock {
		view := views.NewStateLocker(vt, m.View)
		stateLocker := clistate.NewLocker(m.stateLockTimeout, view)
		if err := stateLocker.Lock(backendSMgr, "init is initializing state_store first time"); err != nil {
			diags = diags.Append(fmt.Errorf("Error locking state: %s", err))
			return nil, diags
		}
		defer stateLocker.Unlock()
	}

	// Store the state_store metadata in our saved state location
	s := backendSMgr.State()
	if s == nil {
		s = workdir.NewBackendStateFile()
	}

	var pVersion *version.Version // This will remain nil for builtin providers or unmanaged providers.
	if c.ProviderAddr.IsBuiltIn() {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagWarning,
			Summary:  "State storage is using a builtin provider",
			Detail:   "Terraform is using a builtin provider for initializing state storage. Terraform will be less able to detect when state migrations are required in future init commands.",
		})
	} else {
		isReattached, err := reattach.IsProviderReattached(c.ProviderAddr, os.Getenv("TF_REATTACH_PROVIDERS"))
		if err != nil {
			diags = diags.Append(fmt.Errorf("Unable to determine if state storage provider is reattached while initializing state store for the first time. This is a bug in Terraform and should be reported: %w", err))
			return nil, diags
		}
		if isReattached {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagWarning,
				Summary:  "State storage provider is not managed by Terraform",
				Detail:   "Terraform is using a provider supplied via TF_REATTACH_PROVIDERS for initializing state storage. Terraform will be less able to detect when state migrations are required in future init commands.",
			})
		} else {
			// The provider is not built in and is being managed by Terraform
			// This is the most common scenario, by far.
			var vDiags tfdiags.Diagnostics
			pVersion, vDiags = getStateStorageProviderVersion(c, opts.Locks)
			diags = diags.Append(vDiags)
			if vDiags.HasErrors() {
				return nil, diags
			}
		}
	}

	s.StateStore = &workdir.StateStoreConfigState{
		Type: c.Type,
		Hash: uint64(stateStoreHash),
		Provider: &workdir.ProviderConfigState{
			Source:  &c.ProviderAddr,
			Version: pVersion,
		},
	}
	err := s.StateStore.SetConfig(storeConfigVal, b.ConfigSchema())
	if err != nil {
		diags = diags.Append(fmt.Errorf("Failed to set state store configuration: %w", err))
		return nil, diags
	}

	// We need to briefly convert away from backend.Backend interface to use the method
	// for accessing the provider schema. In this method we _always_ expect the concrete value
	// to be backendPluggable.Pluggable.
	plug := b.(*backendPluggable.Pluggable)
	err = s.StateStore.Provider.SetConfig(providerConfigVal, plug.ProviderSchema())
	if err != nil {
		diags = diags.Append(fmt.Errorf("Failed to set state store provider configuration: %w", err))
		return nil, diags
	}

	// Verify that selected workspace exists in the state store.
	if opts.Init && b != nil {
		err := m.selectWorkspace(b)
		if err != nil {
			if errors.Is(err, &errBackendNoExistingWorkspaces{}) {
				// If there are no workspaces, Terraform either needs to create the default workspace here
				// or instruct the user to run a `terraform workspace new` command.
				ws, err := m.Workspace()
				if err != nil {
					diags = diags.Append(fmt.Errorf("Failed to check current workspace: %w", err))
					return nil, diags
				}

				if ws == backend.DefaultStateName {
					// Users control if the default workspace is created through the -create-default-workspace flag (defaults to true)
					if opts.CreateDefaultWorkspace {
						diags = diags.Append(m.createDefaultWorkspace(c, b))
						if !diags.HasErrors() {
							// Report workspace creation to the view
							view := views.NewInit(vt, m.View)
							view.Output(views.DefaultWorkspaceCreatedMessage)
						}
					} else {
						diags = diags.Append(&hcl.Diagnostic{
							Severity: hcl.DiagWarning,
							Summary:  "The default workspace does not exist",
							Detail:   "Terraform has been configured to skip creation of the default workspace in the state store. To create it, either remove the `-create-default-workspace=false` flag and re-run the 'init' command, or create it using a 'workspace new' command",
						})
					}
				} else {
					// User needs to run a `terraform workspace new` command to create the missing custom workspace.
					diags = append(diags, tfdiags.Sourceless(
						tfdiags.Error,
						fmt.Sprintf("Workspace %q has not been created yet", ws),
						fmt.Sprintf("State store %q in provider %s (%q) reports that no workspaces currently exist. To create the custom workspace %q use the command `terraform workspace new %s`.",
							c.Type,
							c.Provider.Name,
							c.ProviderAddr,
							ws,
							ws,
						),
					))
					return nil, diags
				}
			} else {
				// For all other errors, report via diagnostics
				diags = diags.Append(fmt.Errorf("Failed to select a workspace: %w", err))
			}
		}
	}
	if diags.HasErrors() {
		return nil, diags
	}

	// Update backend state file
	if err := backendSMgr.WriteState(s); err != nil {
		diags = diags.Append(errBackendWriteSavedDiag(err))
		return nil, diags
	}
	if err := backendSMgr.PersistState(); err != nil {
		diags = diags.Append(errBackendWriteSavedDiag(err))
		return nil, diags
	}

	return b, diags
}

// Migrating a state store to backend (including local).
func (m *Meta) stateStore_to_backend(ssSMgr *clistate.LocalState, dstBackendType string, dstBackend backend.Backend, newBackendState *workdir.BackendConfigState, viewType arguments.ViewType) (backend.Backend, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	s := ssSMgr.State()
	stateStoreType := s.StateStore.Type

	view := views.NewInit(viewType, m.View)
	view.Output(views.StateMigrateLocalMessage, stateStoreType)

	// Initialize the configured state store
	ss, moreDiags := m.savedStateStore(ssSMgr)
	diags = diags.Append(moreDiags)
	if moreDiags.HasErrors() {
		return nil, diags
	}

	// Perform the migration
	err := m.backendMigrateState(&backendMigrateOpts{
		SourceType:      stateStoreType,
		DestinationType: dstBackendType,
		Source:          ss,
		Destination:     dstBackend,
		ViewType:        viewType,
	})
	if err != nil {
		diags = diags.Append(err)
		return nil, diags
	}

	// Remove the stored metadata
	s.StateStore = nil
	s.Backend = newBackendState
	if err := ssSMgr.WriteState(s); err != nil {
		diags = diags.Append(errStateStoreClearSaved{err})
		return nil, diags
	}
	if err := ssSMgr.PersistState(); err != nil {
		diags = diags.Append(errStateStoreClearSaved{err})
		return nil, diags
	}

	// Return backend
	return dstBackend, diags
}

// getStateStorageProviderVersion gets the current version of the state store provider that's in use. This is achieved
// by inspecting the current locks.
//
// This function assumes that calling code has checked whether the provider is fully managed by Terraform,
// or is built-in, before using this method and is prepared to receive a nil Version.
func getStateStorageProviderVersion(c *configs.StateStore, locks *depsfile.Locks) (*version.Version, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	var pVersion *version.Version

	isReattached, err := reattach.IsProviderReattached(c.ProviderAddr, os.Getenv("TF_REATTACH_PROVIDERS"))
	if err != nil {
		diags = diags.Append(fmt.Errorf("Unable to determine if state storage provider is reattached while determining the version in use. This is a bug in Terraform and should be reported: %w", err))
		return nil, diags
	}
	if c.ProviderAddr.IsBuiltIn() || isReattached {
		return nil, nil // nil Version returned
	}

	pLock := locks.Provider(c.ProviderAddr)
	if pLock == nil {
		// This should never happen as the user would've already hit
		// an error earlier prompting them to run init
		diags = diags.Append(fmt.Errorf("The provider %s (%q) is not present in the lockfile, despite being used for state store %q. This is a bug in Terraform and should be reported.",
			c.Provider.Name,
			c.ProviderAddr,
			c.Type))
		return nil, diags
	}
	pVersion, err = providerreqs.GoVersionFromVersion(pLock.Version())
	if err != nil {
		diags = diags.Append(fmt.Errorf("Failed obtain the in-use version of provider %s (%q) used with state store %q. This is a bug in Terraform and should be reported: %w",
			c.Provider.Name,
			c.ProviderAddr,
			c.Type,
			err))
		return nil, diags
	}

	return pVersion, diags
}

// createDefaultWorkspace receives a backend made using a pluggable state store, and details about that store's config,
// and persists an empty state file in the default workspace. By creating this artifact we ensure that the default
// workspace is created and usable by Terraform in later operations.
func (m *Meta) createDefaultWorkspace(c *configs.StateStore, b backend.Backend) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	defaultSMgr, sDiags := b.StateMgr(backend.DefaultStateName)
	diags = diags.Append(sDiags)
	if sDiags.HasErrors() {
		diags = diags.Append(fmt.Errorf("Failed to create a state manager for state store %q in provider %s (%q). This is a bug in Terraform and should be reported: %w",
			c.Type,
			c.Provider.Name,
			c.ProviderAddr,
			sDiags.Err()))
		return diags
	}
	emptyState := states.NewState()
	if err := defaultSMgr.WriteState(emptyState); err != nil {
		diags = diags.Append(errStateStoreWorkspaceCreateDiag(err, c.Type))
		return diags
	}
	if err := defaultSMgr.PersistState(nil); err != nil {
		diags = diags.Append(errStateStoreWorkspaceCreateDiag(err, c.Type))
		return diags
	}

	return diags
}

// Initializing a saved state store from the backend state file (aka 'cache file', aka 'legacy state file')
func (m *Meta) savedStateStore(sMgr *clistate.LocalState) (backend.Backend, tfdiags.Diagnostics) {
	// We're preparing a state_store version of backend.Backend.
	//
	// The provider and state store will be configured using the backend state file.

	var diags tfdiags.Diagnostics

	s := sMgr.State()

	factory, pDiags := m.StateStoreProviderFactoryFromConfigState(s.StateStore)
	diags = diags.Append(pDiags)
	if pDiags.HasErrors() {
		return nil, diags
	}

	provider, err := factory()
	if err != nil {
		diags = diags.Append(fmt.Errorf("error when obtaining provider instance during state store initialization: %w", err))
		return nil, diags
	}
	// We purposefully don't have a deferred call to the provider's Close method here because the calling code needs a
	// running provider instance inside the returned backend.Backend instance.
	// Stopping the provider process is the responsibility of the calling code.

	resp := provider.GetProviderSchema()

	if len(resp.StateStores) == 0 {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Provider does not support pluggable state storage",
			Detail: fmt.Sprintf("There are no state stores implemented by provider %s (%q)",
				s.StateStore.Provider.Source.Type,
				s.StateStore.Provider.Source),
		})
		return nil, diags
	}

	stateStoreSchema, exists := resp.StateStores[s.StateStore.Type]
	if !exists {
		suggestions := slices.Sorted(maps.Keys(resp.StateStores))
		suggestion := didyoumean.NameSuggestion(s.StateStore.Type, suggestions)
		if suggestion != "" {
			suggestion = fmt.Sprintf(" Did you mean %q?", suggestion)
		}
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "State store not implemented by the provider",
			Detail: fmt.Sprintf("State store %q is not implemented by provider %s (%q)%s",
				s.StateStore.Type,
				s.StateStore.Provider.Source.Type,
				s.StateStore.Provider.Source,
				suggestion),
		})
		return nil, diags
	}

	// Get the provider config from the backend state file.
	providerConfigVal, err := s.StateStore.Provider.Config(resp.Provider.Body)
	if err != nil {
		diags = diags.Append(
			&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Error reading provider configuration state",
				Detail: fmt.Sprintf("Terraform experienced an error reading provider configuration for provider %s (%q) while configuring state store %s",
					s.StateStore.Provider.Source.Type,
					s.StateStore.Provider.Source,
					s.StateStore.Type,
				),
			},
		)
		return nil, diags
	}

	// Get the state store config from the backend state file.
	stateStoreConfigVal, err := s.StateStore.Config(stateStoreSchema.Body)
	if err != nil {
		diags = diags.Append(
			&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Error reading state store configuration state",
				Detail: fmt.Sprintf("Terraform experienced an error reading state store configuration for state store %s in provider %s (%q)",
					s.StateStore.Type,
					s.StateStore.Provider.Source.Type,
					s.StateStore.Provider.Source,
				),
			},
		)
		return nil, diags
	}

	// Validate and configure the provider
	//
	// NOTE: there are no marks we need to remove at this point.
	// We haven't added marks since the provider config from the backend state was used
	// because the state-storage provider's config isn't going to be presented to the user via terminal output or diags.
	validateResp := provider.ValidateProviderConfig(providers.ValidateProviderConfigRequest{
		Config: providerConfigVal,
	})
	diags = diags.Append(validateResp.Diagnostics)
	if diags.HasErrors() {
		return nil, diags
	}

	configureResp := provider.ConfigureProvider(providers.ConfigureProviderRequest{
		TerraformVersion: tfversion.SemVer.String(),
		Config:           providerConfigVal,
	})
	diags = diags.Append(configureResp.Diagnostics)
	if diags.HasErrors() {
		return nil, diags
	}

	// Now that the provider is configured we can begin using the state store through
	// the backend.Backend interface.
	p, err := backendPluggable.NewPluggable(provider, s.StateStore.Type)
	if err != nil {
		diags = diags.Append(err)
	}

	// Validate and configure the state store
	//
	// Note: we do not use the value returned from PrepareConfig for state stores,
	// however that old approach is still used with backends for compatibility reasons.
	_, validateDiags := p.PrepareConfig(stateStoreConfigVal)
	diags = diags.Append(validateDiags)

	configureDiags := p.Configure(stateStoreConfigVal)
	diags = diags.Append(configureDiags)

	return p, diags
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
func (m *Meta) backendConfigNeedsMigration(c *configs.Backend, s *workdir.BackendConfigState) bool {
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

// backendInitFromConfig returns an initialized and configured backend, using the backend.Backend interface.
// During this process:
// > Users are prompted for input if required attributes are missing.
// > The backend config is validated
// > The backend is configured
// > Service discovery is handled for operations backends (only relevant to `cloud` and `remote`)
func (m *Meta) backendInitFromConfig(c *configs.Backend) (backend.Backend, cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	// Get the backend
	f := backendInit.Backend(c.Type)
	if f == nil {
		diags = diags.Append(errBackendNewUnknown{c.Type})
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

// stateStoreInitFromConfig returns an initialized and configured state store, using the backend.Backend interface.
// During this process:
// > The provider is configured, after validating provider config
// > The state store is configured, after validating state_store config
//
// NOTE: the backend version of this method, `backendInitFromConfig`, prompts users for input if any required fields
// are missing from the backend config. In `stateStoreInitFromConfig` we don't do this, and instead users will see an error.
func (m *Meta) stateStoreInitFromConfig(c *configs.StateStore, locks *depsfile.Locks) (backend.Backend, cty.Value, cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	factory, pDiags := m.StateStoreProviderFactoryFromConfig(c, locks)
	diags = diags.Append(pDiags)
	if pDiags.HasErrors() {
		return nil, cty.NilVal, cty.NilVal, diags
	}

	provider, err := factory()
	if err != nil {
		diags = diags.Append(fmt.Errorf("error when obtaining provider instance during state store initialization: %w", err))
		return nil, cty.NilVal, cty.NilVal, diags
	}
	// We purposefully don't have a deferred call to the provider's Close method here because the calling code needs a
	// running provider instance inside the returned backend.Backend instance.
	// Stopping the provider process is the responsibility of the calling code.

	resp := provider.GetProviderSchema()

	if len(resp.StateStores) == 0 {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Provider does not support pluggable state storage",
			Detail: fmt.Sprintf("There are no state stores implemented by provider %s (%q)",
				c.Provider.Name,
				c.ProviderAddr),
			Subject: &c.DeclRange,
		})
		return nil, cty.NilVal, cty.NilVal, diags
	}

	schema, exists := resp.StateStores[c.Type]
	if !exists {
		suggestions := slices.Sorted(maps.Keys(resp.StateStores))
		suggestion := didyoumean.NameSuggestion(c.Type, suggestions)
		if suggestion != "" {
			suggestion = fmt.Sprintf(" Did you mean %q?", suggestion)
		}
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "State store not implemented by the provider",
			Detail: fmt.Sprintf("State store %q is not implemented by provider %s (%q)%s",
				c.Type, c.Provider.Name,
				c.ProviderAddr, suggestion),
			Subject: &c.DeclRange,
		})
		return nil, cty.NilVal, cty.NilVal, diags
	}

	// Handle the nested provider block.
	pDecSpec := resp.Provider.Body.DecoderSpec()
	pConfig := c.Provider.Config
	providerConfigVal, pDecDiags := hcldec.Decode(pConfig, pDecSpec, nil)
	diags = diags.Append(pDecDiags)

	// Handle the schema for the state store itself, excluding the provider block.
	ssdecSpec := schema.Body.DecoderSpec()
	stateStoreConfigVal, ssDecDiags := hcldec.Decode(c.Config, ssdecSpec, nil)
	diags = diags.Append(ssDecDiags)
	if ssDecDiags.HasErrors() {
		return nil, cty.NilVal, cty.NilVal, diags
	}

	// Validate and configure the provider
	//
	// NOTE: there are no marks we need to remove at this point.
	// We haven't added marks since the provider config from the backend state was used
	// because the state-storage provider's config isn't going to be presented to the user via terminal output or diags.
	validateResp := provider.ValidateProviderConfig(providers.ValidateProviderConfigRequest{
		Config: providerConfigVal,
	})
	diags = diags.Append(validateResp.Diagnostics)
	if validateResp.Diagnostics.HasErrors() {
		return nil, cty.NilVal, cty.NilVal, diags
	}

	configureResp := provider.ConfigureProvider(providers.ConfigureProviderRequest{
		TerraformVersion: tfversion.String(),
		Config:           providerConfigVal,
	})
	diags = diags.Append(configureResp.Diagnostics)
	if configureResp.Diagnostics.HasErrors() {
		return nil, cty.NilVal, cty.NilVal, diags
	}

	// Now that the provider is configured we can begin using the state store through
	// the backend.Backend interface.
	p, err := backendPluggable.NewPluggable(provider, c.Type)
	if err != nil {
		diags = diags.Append(err)
	}

	// Validate and configure the state store
	//
	// Note: we do not use the value returned from PrepareConfig for state stores,
	// however that old approach is still used with backends for compatibility reasons.
	_, validateDiags := p.PrepareConfig(stateStoreConfigVal)
	diags = diags.Append(validateDiags)

	configureDiags := p.Configure(stateStoreConfigVal)
	diags = diags.Append(configureDiags)

	return p, stateStoreConfigVal, providerConfigVal, diags
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

func (m *Meta) StateStoreProviderFactoryFromConfig(config *configs.StateStore, locks *depsfile.Locks) (providers.Factory, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	if config == nil || locks == nil {
		panic(fmt.Sprintf("nil config or nil locks passed to GetStateStoreProviderFactory: config %#v, locks %#v", config, locks))
	}

	if config.ProviderAddr.IsZero() {
		// This should not happen; this data is populated when parsing config,
		// even for builtin providers
		return nil, diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Unknown provider used for state storage",
			Detail:   "Terraform could not find the provider used with the state_store. This is a bug in Terraform and should be reported.",
			Subject:  &config.TypeRange,
		})
	}

	factories, err := m.ProviderFactoriesFromLocks(locks)
	if err != nil {
		// This may happen if the provider isn't present in the provider cache.
		// This should be caught earlier by logic that diffs the config against the backend state file.
		return nil, diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Provider unavailable",
			Detail: fmt.Sprintf("Terraform experienced an error when trying to use provider %s (%q) to initialize the %q state store: %s",
				config.Provider.Name,
				config.ProviderAddr,
				config.Type,
				err),
			Subject: &config.TypeRange,
		})
	}

	factory, exists := factories[config.ProviderAddr]
	if !exists {
		return nil, diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Provider unavailable",
			Detail: fmt.Sprintf("The provider %s (%q) is required to initialize the %q state store, but the matching provider factory is missing. This is a bug in Terraform and should be reported.",
				config.Provider.Name,
				config.ProviderAddr,
				config.Type,
			),
			Subject: &config.TypeRange,
		})
	}

	return factory, diags
}

func (m *Meta) StateStoreProviderFactoryFromConfigState(cfgState *workdir.StateStoreConfigState) (providers.Factory, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	if cfgState == nil {
		panic("nil config passed to StateStoreProviderFactoryFromConfigState")
	}

	if cfgState.Provider == nil || cfgState.Provider.Source.IsZero() {
		// This should not happen; this data is populated when storing config state
		return nil, diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Unknown provider used for state storage",
			Detail:   "Terraform could not find the provider used with the state_store. This is a bug in Terraform and should be reported.",
		})
	}

	factories, err := m.ProviderFactories()
	if err != nil {
		// This may happen if the provider isn't present in the provider cache.
		// This should be caught earlier by logic that diffs the config against the backend state file.
		return nil, diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Provider unavailable",
			Detail: fmt.Sprintf("Terraform experienced an error when trying to use provider %s (%q) to initialize the %q state store: %s",
				cfgState.Type,
				cfgState.Provider.Source,
				cfgState.Type,
				err),
		})
	}

	factory, exists := factories[*cfgState.Provider.Source]
	if !exists {
		return nil, diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Provider unavailable",
			Detail: fmt.Sprintf("The provider %s (%q) is required to initialize the %q state store, but the matching provider factory is missing. This is a bug in Terraform and should be reported.",
				cfgState.Type,
				cfgState.Provider.Source,
				cfgState.Type,
			),
		})
	}

	return factory, diags
}

//-------------------------------------------------------------------
// Output constants and initialization code
//-------------------------------------------------------------------

const inputCloudInitCreateWorkspace = `
There are no workspaces with the configured tags (%s)
in your HCP Terraform organization. To finish initializing, Terraform needs at
least one workspace available.

Terraform can create a properly tagged workspace for you now. Please enter a
name to create a new HCP Terraform workspace.
`
