// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// The Init view is used for the init command.
type Init interface {
	Diagnostics(diags tfdiags.Diagnostics)
	Output(messageCode InitMessageCode, params ...any)
	LogInitMessage(messageCode InitMessageCode, params ...any)
	Log(message string, params ...any)
	PrepareMessage(messageCode InitMessageCode, params ...any) string
}

// NewInit returns Init implementation for the given ViewType.
func NewInit(vt arguments.ViewType, view *View) Init {
	switch vt {
	case arguments.ViewJSON:
		return &InitJSON{
			view: NewJSONView(view),
		}
	case arguments.ViewHuman:
		return &InitHuman{
			view: view,
		}
	default:
		panic(fmt.Sprintf("unknown view type %v", vt))
	}
}

// The InitHuman implementation renders human-readable text logs, suitable for
// a scrolling terminal.
type InitHuman struct {
	view *View
}

var _ Init = (*InitHuman)(nil)

func (v *InitHuman) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}

func (v *InitHuman) Output(messageCode InitMessageCode, params ...any) {
	v.view.streams.Println(v.PrepareMessage(messageCode, params...))
}

func (v *InitHuman) LogInitMessage(messageCode InitMessageCode, params ...any) {
	v.view.streams.Println(v.PrepareMessage(messageCode, params...))
}

// this implements log method for use by interfaces that need to log generic string messages, e.g used for logging in hook_module_install.go
func (v *InitHuman) Log(message string, params ...any) {
	v.view.streams.Println(strings.TrimSpace(fmt.Sprintf(message, params...)))
}

func (v *InitHuman) PrepareMessage(messageCode InitMessageCode, params ...any) string {
	message, ok := MessageRegistry[messageCode]
	if !ok {
		// display the message code as fallback if not found in the message registry
		return string(messageCode)
	}

	if message.HumanValue == "" {
		// no need to apply colorization if the message is empty
		return message.HumanValue
	}

	return v.view.colorize.Color(strings.TrimSpace(fmt.Sprintf(message.HumanValue, params...)))
}

// The InitJSON implementation renders streaming JSON logs, suitable for
// integrating with other software.
type InitJSON struct {
	view *JSONView
}

var _ Init = (*InitJSON)(nil)

func (v *InitJSON) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}

func (v *InitJSON) Output(messageCode InitMessageCode, params ...any) {
	// don't add empty messages to json output
	preppedMessage := v.PrepareMessage(messageCode, params...)
	if preppedMessage == "" {
		return
	}

	// Logged data includes by default:
	// @level as "info"
	// @module as "terraform.ui" (See NewJSONView)
	// @timestamp formatted in the default way
	//
	// In the method below we:
	// * Set @message as the first argument value
	// * Annotate with extra data:
	//     "type":"init_output"
	//     "message_code":"<value>"
	v.view.log.Info(
		preppedMessage,
		"type", "init_output",
		"message_code", string(messageCode),
	)
}

func (v *InitJSON) LogInitMessage(messageCode InitMessageCode, params ...any) {
	preppedMessage := v.PrepareMessage(messageCode, params...)
	if preppedMessage == "" {
		return
	}

	v.view.Log(preppedMessage)
}

// this implements log method for use by services that need to log generic string messages, e.g usage logging in hook_module_install.go
func (v *InitJSON) Log(message string, params ...any) {
	v.view.Log(strings.TrimSpace(fmt.Sprintf(message, params...)))
}

func (v *InitJSON) PrepareMessage(messageCode InitMessageCode, params ...any) string {
	message, ok := MessageRegistry[messageCode]
	if !ok {
		// display the message code as fallback if not found in the message registry
		return string(messageCode)
	}

	return strings.TrimSpace(fmt.Sprintf(message.JSONValue, params...))
}

// InitMessage represents a message string in both json and human decorated text format.
type InitMessage struct {
	HumanValue string
	JSONValue  string
}

var MessageRegistry map[InitMessageCode]InitMessage = map[InitMessageCode]InitMessage{
	"copying_configuration_message": {
		HumanValue: "[reset][bold]Copying configuration[reset] from %q...",
		JSONValue:  "Copying configuration from %q...",
	},
	"output_init_empty_message": {
		HumanValue: outputInitEmpty,
		JSONValue:  outputInitEmptyJSON,
	},
	"output_init_success_message": {
		HumanValue: outputInitSuccess,
		JSONValue:  outputInitSuccessJSON,
	},
	"output_init_success_cloud_message": {
		HumanValue: outputInitSuccessCloud,
		JSONValue:  outputInitSuccessCloudJSON,
	},
	"output_init_success_cli_message": {
		HumanValue: outputInitSuccessCLI,
		JSONValue:  outputInitSuccessCLI_JSON,
	},
	"output_init_success_cli_cloud_message": {
		HumanValue: outputInitSuccessCLICloud,
		JSONValue:  outputInitSuccessCLICloudJSON,
	},
	"upgrading_modules_message": {
		HumanValue: "[reset][bold]Upgrading modules...",
		JSONValue:  "Upgrading modules...",
	},
	"initializing_modules_message": {
		HumanValue: "[reset][bold]Initializing modules...",
		JSONValue:  "Initializing modules...",
	},
	"initializing_terraform_cloud_message": {
		HumanValue: "\n[reset][bold]Initializing HCP Terraform...",
		JSONValue:  "Initializing HCP Terraform...",
	},
	"initializing_backend_message": {
		HumanValue: "\n[reset][bold]Initializing the backend...",
		JSONValue:  "Initializing the backend...",
	},
	"initializing_provider_plugin_message": {
		HumanValue: "\n[reset][bold]Initializing provider plugins...",
		JSONValue:  "Initializing provider plugins...",
	},
	"initializing_provider_plugin_from_config_message": {
		HumanValue: "\n[reset][bold]Initializing provider plugins found in the configuration...",
		JSONValue:  "Initializing provider plugins found in the configuration...",
	},
	"initializing_provider_plugin_from_state_message": {
		HumanValue: "\n[reset][bold]Initializing provider plugins found in the state...",
		JSONValue:  "Initializing provider plugins found in the state...",
	},
	"initializing_state_store_message": {
		HumanValue: "\n[reset][bold]Initializing the state store...",
		JSONValue:  "Initializing the state store...",
	},
	"default_workspace_created_message": {
		HumanValue: defaultWorkspaceCreatedInfo,
		JSONValue:  defaultWorkspaceCreatedInfo,
	},
	"dependencies_lock_changes_info": {
		HumanValue: dependenciesLockChangesInfo,
		JSONValue:  dependenciesLockChangesInfo,
	},
	"lock_info": {
		HumanValue: previousLockInfoHuman,
		JSONValue:  previousLockInfoJSON,
	},
	"provider_already_installed_message": {
		HumanValue: "- Using previously-installed %s v%s",
		JSONValue:  "%s v%s: Using previously-installed provider version",
	},
	"built_in_provider_available_message": {
		HumanValue: "- %s is built in to Terraform",
		JSONValue:  "%s is built in to Terraform",
	},
	"reusing_previous_version_info": {
		HumanValue: "- Reusing previous version of %s from the dependency lock file",
		JSONValue:  "%s: Reusing previous version from the dependency lock file",
	},
	"reusing_version_during_state_provider_init": {
		HumanValue: "- Reusing previous version of %s",
		JSONValue:  "%s: Reusing previous version of %s",
	},
	"finding_matching_version_message": {
		HumanValue: "- Finding %s versions matching %q...",
		JSONValue:  "Finding matching versions for provider: %s, version_constraint: %q",
	},
	"finding_latest_version_message": {
		HumanValue: "- Finding latest version of %s...",
		JSONValue:  "%s: Finding latest version...",
	},
	"using_provider_from_cache_dir_info": {
		HumanValue: "- Using %s v%s from the shared cache directory",
		JSONValue:  "%s v%s: Using from the shared cache directory",
	},
	"installing_provider_message": {
		HumanValue: "- Installing %s v%s...",
		JSONValue:  "Installing provider version: %s v%s...",
	},
	"key_id": {
		HumanValue: ", key ID [reset][bold]%s[reset]",
		JSONValue:  "key_id: %s",
	},
	"installed_provider_version_info": {
		HumanValue: "- Installed %s v%s (%s%s)",
		JSONValue:  "Installed provider version: %s v%s (%s%s)",
	},
	"partner_and_community_providers_message": {
		HumanValue: partnerAndCommunityProvidersInfo,
		JSONValue:  partnerAndCommunityProvidersInfo,
	},
	"init_config_error": {
		HumanValue: errInitConfigError,
		JSONValue:  errInitConfigErrorJSON,
	},
	"state_store_unset": {
		HumanValue: "[reset][green]\n\nSuccessfully unset the state store %q. Terraform will now operate locally.",
		JSONValue:  "Successfully unset the state store %q. Terraform will now operate locally.",
	},
	"state_store_migrate_backend": {
		HumanValue: "Migrating from %q state store to %q backend.",
		JSONValue:  "Migrating from %q state store to %q backend.",
	},
	"backend_configured_success": {
		HumanValue: backendConfiguredSuccessHuman,
		JSONValue:  backendConfiguredSuccessJSON,
	},
	"backend_configured_unset": {
		HumanValue: backendConfiguredUnsetHuman,
		JSONValue:  backendConfiguredUnsetJSON,
	},
	"backend_migrate_to_cloud": {
		HumanValue: "Migrating from backend %q to HCP Terraform.",
		JSONValue:  "Migrating from backend %q to HCP Terraform.",
	},
	"backend_migrate_from_cloud": {
		HumanValue: "Migrating from HCP Terraform to backend %q.",
		JSONValue:  "Migrating from HCP Terraform to backend %q.",
	},
	"backend_cloud_change_in_place": {
		HumanValue: "HCP Terraform configuration has changed.",
		JSONValue:  "HCP Terraform configuration has changed.",
	},
	"backend_migrate_type_change": {
		HumanValue: backendMigrateTypeChangeHuman,
		JSONValue:  backendMigrateTypeChangeJSON,
	},
	"backend_reconfigure": {
		HumanValue: backendReconfigureHuman,
		JSONValue:  backendReconfigureJSON,
	},
	"backend_migrate_local": {
		HumanValue: backendMigrateLocalHuman,
		JSONValue:  backendMigrateLocalJSON,
	},
	"backend_cloud_migrate_local": {
		HumanValue: "Migrating from HCP Terraform or Terraform Enterprise to local state.",
		JSONValue:  "Migrating from HCP Terraform or Terraform Enterprise to local state.",
	},
	"backend_cloud_migrate_state_store": {
		HumanValue: "Migrating from HCP Terraform Terraform Enterprise to state store %q.",
		JSONValue:  "Migrating from HCP Terraform Terraform Enterprise to state store %q.",
	},
	"backend_migrate_state_store": {
		HumanValue: "Migrating from backend %q to state store %q.",
		JSONValue:  "Migrating from backend %q to state store %q.",
	},
	"state_store_migrate_local": {
		HumanValue: stateMigrateLocalHuman,
		JSONValue:  stateMigrateLocalJSON,
	},
	"empty_message": {
		HumanValue: "",
		JSONValue:  "",
	},
}

type InitMessageCode string

const (
	// Following message codes are used and documented EXTERNALLY
	// Keep docs/internals/machine-readable-ui.mdx up to date with
	// this list when making changes here.
	CopyingConfigurationMessage       InitMessageCode = "copying_configuration_message"
	EmptyMessage                      InitMessageCode = "empty_message"
	OutputInitEmptyMessage            InitMessageCode = "output_init_empty_message"
	OutputInitSuccessMessage          InitMessageCode = "output_init_success_message"
	OutputInitSuccessCloudMessage     InitMessageCode = "output_init_success_cloud_message"
	OutputInitSuccessCLIMessage       InitMessageCode = "output_init_success_cli_message"
	OutputInitSuccessCLICloudMessage  InitMessageCode = "output_init_success_cli_cloud_message"
	UpgradingModulesMessage           InitMessageCode = "upgrading_modules_message"
	InitializingTerraformCloudMessage InitMessageCode = "initializing_terraform_cloud_message"
	InitializingModulesMessage        InitMessageCode = "initializing_modules_message"
	InitializingBackendMessage        InitMessageCode = "initializing_backend_message"
	InitializingStateStoreMessage     InitMessageCode = "initializing_state_store_message"
	DefaultWorkspaceCreatedMessage    InitMessageCode = "default_workspace_created_message"
	InitializingProviderPluginMessage InitMessageCode = "initializing_provider_plugin_message"
	LockInfo                          InitMessageCode = "lock_info"
	DependenciesLockChangesInfo       InitMessageCode = "dependencies_lock_changes_info"

	//// Message codes below are ONLY used INTERNALLY (for now)

	// InitializingProviderPluginFromConfigMessage indicates the beginning of installing of providers described in configuration
	InitializingProviderPluginFromConfigMessage InitMessageCode = "initializing_provider_plugin_from_config_message"
	// InitializingProviderPluginFromStateMessage indicates the beginning of installing of providers described in state
	InitializingProviderPluginFromStateMessage InitMessageCode = "initializing_provider_plugin_from_state_message"
	// DependenciesLockPendingChangesInfo indicates when a provider installation step will reuse a provider from a previous installation step in the current operation
	ReusingVersionIdentifiedFromConfig InitMessageCode = "reusing_version_during_state_provider_init"

	// InitConfigError indicates problems encountered during initialisation
	InitConfigError InitMessageCode = "init_config_error"
	// BackendConfiguredSuccessMessage indicates successful backend configuration
	BackendConfiguredSuccessMessage InitMessageCode = "backend_configured_success"
	// BackendConfiguredUnsetMessage indicates successful backend unsetting
	BackendConfiguredUnsetMessage InitMessageCode = "backend_configured_unset"
	// BackendMigrateToCloudMessage indicates migration to HCP Terraform
	BackendMigrateToCloudMessage InitMessageCode = "backend_migrate_to_cloud"
	// BackendMigrateFromCloudMessage indicates migration from HCP Terraform
	BackendMigrateFromCloudMessage InitMessageCode = "backend_migrate_from_cloud"
	// BackendCloudChangeInPlaceMessage indicates HCP Terraform configuration change
	BackendCloudChangeInPlaceMessage InitMessageCode = "backend_cloud_change_in_place"
	// BackendMigrateTypeChangeMessage indicates backend type change
	BackendMigrateTypeChangeMessage InitMessageCode = "backend_migrate_type_change"
	// BackendReconfigureMessage indicates backend reconfiguration
	BackendReconfigureMessage InitMessageCode = "backend_reconfigure"
	// BackendMigrateLocalMessage indicates migration to local backend
	BackendMigrateLocalMessage InitMessageCode = "backend_migrate_local"
	// BackendCloudMigrateLocalMessage indicates migration from cloud to local
	BackendCloudMigrateLocalMessage InitMessageCode = "backend_cloud_migrate_local"
	// BackendCloudMigrateStateStoreMessage indicates migration from cloud to a state store
	BackendCloudMigrateStateStoreMessage InitMessageCode = "backend_cloud_migrate_state_store"
	// BackendMigrateStateStoreMessage indicates migration from a backend to a state store
	BackendMigrateStateStoreMessage InitMessageCode = "backend_migrate_state_store"
	// StateMigrateLocalMessage indicates migration from state store to local
	StateMigrateLocalMessage InitMessageCode = "state_store_migrate_local"
	// FindingMatchingVersionMessage indicates that Terraform is looking for a provider version that matches the constraint during installation
	FindingMatchingVersionMessage InitMessageCode = "finding_matching_version_message"
	// InstalledProviderVersionInfo describes a successfully installed provider along with its version
	InstalledProviderVersionInfo InitMessageCode = "installed_provider_version_info"
	// ReusingPreviousVersionInfo indicates a provider which is locked to a specific version during installation
	ReusingPreviousVersionInfo InitMessageCode = "reusing_previous_version_info"
	// BuiltInProviderAvailableMessage indicates a built-in provider in use during installation
	BuiltInProviderAvailableMessage InitMessageCode = "built_in_provider_available_message"
	// ProviderAlreadyInstalledMessage indicates a provider that is already installed during installation
	ProviderAlreadyInstalledMessage InitMessageCode = "provider_already_installed_message"
	// KeyID indicates the key ID used to sign of a successfully installed provider
	KeyID InitMessageCode = "key_id"
	// InstallingProviderMessage indicates that a provider is being installed (from a remote location)
	InstallingProviderMessage InitMessageCode = "installing_provider_message"
	// FindingLatestVersionMessage indicates that Terraform is looking for the latest version of a provider during installation (no constraint was supplied)
	FindingLatestVersionMessage InitMessageCode = "finding_latest_version_message"
	// UsingProviderFromCacheDirInfo indicates that a provider is being linked from a system-wide cache
	UsingProviderFromCacheDirInfo InitMessageCode = "using_provider_from_cache_dir_info"
	// PartnerAndCommunityProvidersMessage is a message concerning partner and community providers and how these are signed
	PartnerAndCommunityProvidersMessage InitMessageCode = "partner_and_community_providers_message"
)

const outputInitEmpty = `
[reset][bold]Terraform initialized in an empty directory![reset]

The directory has no Terraform configuration files. You may begin working
with Terraform immediately by creating Terraform configuration files.
`

const outputInitEmptyJSON = `
Terraform initialized in an empty directory!

The directory has no Terraform configuration files. You may begin working
with Terraform immediately by creating Terraform configuration files.
`

const outputInitSuccess = `
[reset][bold][green]Terraform has been successfully initialized![reset][green]
`

const outputInitSuccessJSON = `
Terraform has been successfully initialized!
`

const outputInitSuccessCloud = `
[reset][bold][green]HCP Terraform has been successfully initialized![reset][green]
`

const outputInitSuccessCloudJSON = `
HCP Terraform has been successfully initialized!
`

const outputInitSuccessCLI = `[reset][green]
You may now begin working with Terraform. Try running "terraform plan" to see
any changes that are required for your infrastructure. All Terraform commands
should now work.

If you ever set or change modules or backend configuration for Terraform,
rerun this command to reinitialize your working directory. If you forget, other
commands will detect it and remind you to do so if necessary.
`

const outputInitSuccessCLI_JSON = `
You may now begin working with Terraform. Try running "terraform plan" to see
any changes that are required for your infrastructure. All Terraform commands
should now work.

If you ever set or change modules or backend configuration for Terraform,
rerun this command to reinitialize your working directory. If you forget, other
commands will detect it and remind you to do so if necessary.
`

const outputInitSuccessCLICloud = `[reset][green]
You may now begin working with HCP Terraform. Try running "terraform plan" to
see any changes that are required for your infrastructure.

If you ever set or change modules or Terraform Settings, run "terraform init"
again to reinitialize your working directory.
`

const outputInitSuccessCLICloudJSON = `
You may now begin working with HCP Terraform. Try running "terraform plan" to
see any changes that are required for your infrastructure.

If you ever set or change modules or Terraform Settings, run "terraform init"
again to reinitialize your working directory.
`

const previousLockInfoHuman = `
Terraform has created a lock file [bold].terraform.lock.hcl[reset] to record the provider
selections it made above. Include this file in your version control repository
so that Terraform can guarantee to make the same selections by default when
you run "terraform init" in the future.`

const previousLockInfoJSON = `
Terraform has created a lock file .terraform.lock.hcl to record the provider
selections it made above. Include this file in your version control repository
so that Terraform can guarantee to make the same selections by default when
you run "terraform init" in the future.`

const defaultWorkspaceCreatedInfo = `
Terraform created an empty state file for the default workspace in your state store
because it didn't exist. If this was not intended, read the init command's documentation for
more guidance:
https://developer.hashicorp.com/terraform/cli/commands/init
`

const dependenciesLockChangesInfo = `
Terraform has made some changes to the provider dependency selections recorded
in the .terraform.lock.hcl file. Review those changes and commit them to your
version control system if they represent changes you intended to make.`

const partnerAndCommunityProvidersInfo = "\nPartner and community providers are signed by their developers.\n" +
	"If you'd like to know more about provider signing, you can read about it here:\n" +
	"https://developer.hashicorp.com/terraform/cli/plugins/signing"

const errInitConfigError = `
[reset]Terraform encountered problems during initialisation, including problems
with the configuration, described below.

The Terraform configuration must be valid before initialization so that
Terraform can determine which modules and providers need to be installed.
`

const errInitConfigErrorJSON = `
Terraform encountered problems during initialisation, including problems
with the configuration, described below.

The Terraform configuration must be valid before initialization so that
Terraform can determine which modules and providers need to be installed.
`

const backendConfiguredSuccessHuman = `[reset][green]
Successfully configured the backend %q! Terraform will automatically
use this backend unless the backend configuration changes.`

const backendConfiguredSuccessJSON = `Successfully configured the backend %q! Terraform will automatically
use this backend unless the backend configuration changes.`

const backendConfiguredUnsetHuman = `[reset][green]

Successfully unset the backend %q. Terraform will now operate locally.`

const backendConfiguredUnsetJSON = `Successfully unset the backend %q. Terraform will now operate locally.`

const backendMigrateTypeChangeHuman = `[reset]Terraform detected that the backend type changed from %q to %q.
`

const backendMigrateTypeChangeJSON = `Terraform detected that the backend type changed from %q to %q.`

const backendReconfigureHuman = `[reset][bold]Backend configuration changed![reset]

Terraform has detected that the configuration specified for the backend
has changed. Terraform will now check for existing state in the backends.
`

const backendReconfigureJSON = `Backend configuration changed!

Terraform has detected that the configuration specified for the backend
has changed. Terraform will now check for existing state in the backends.`

const backendMigrateLocalHuman = `Terraform has detected you're unconfiguring your previously set %q backend.`

const backendMigrateLocalJSON = `Terraform has detected you're unconfiguring your previously set %q backend.`

const stateMigrateLocalHuman = `Terraform has detected you're unconfiguring your previously set %q state store.`

const stateMigrateLocalJSON = `Terraform has detected you're unconfiguring your previously set %q state store.`
