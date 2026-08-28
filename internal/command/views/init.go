// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"fmt"
	"strings"

	tfaddr "github.com/hashicorp/terraform-registry-address"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/views/json"
	"github.com/hashicorp/terraform/internal/getproviders"
	"github.com/hashicorp/terraform/internal/policy"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// The Init view is used for the init command.
type Init interface {
	Diagnostics(diags tfdiags.Diagnostics)
	PolicyResult(addr string, resp policy.EvaluationResponse)
	PolicyDiagnostics(diags policy.Diagnostics)
	Output(messageCode InitMessageCode, params ...any)

	// LogConfigurationCopyingStart describes the start of copying a module to create the root module in an empty directory.
	LogConfigurationCopyingStart(moduleSource string)

	// LogInitializingStateStoreProviderStart indicates progress during installation of a state store provider.
	// This is signposted as distinct from general provider download, which may happen later in init.
	LogInstallStateStoreProviderStart(providerAddr addrs.Provider, cons getproviders.VersionConstraints, storeType string)

	// LogInitializingStateStoreStart indicates progress initializing a state store.
	LogInitializingStateStoreStart(storeType string)

	// LogInitializingBackendStart indicates progress initializing a backend.
	LogInitializingBackendStart()

	ModuleInstallationLogger
	ProviderInstallationLogger
	ProviderLockingLogger

	StateStoreProviderTrustLogger

	prepareMessage(messageCode InitMessageCode, params ...any) string

	Spacer // The `init` command logs empty lines to space-out different sections of human-readable output
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

var (
	_ Init                       = (*InitHuman)(nil)
	_ ProviderInstallationLogger = (*InitHuman)(nil)
)

func (v *InitHuman) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}

func (v *InitHuman) Spacer() {
	v.view.Spacer()
}

func (v *InitHuman) PolicyDiagnostics(diags policy.Diagnostics) {
	v.view.PolicyDiagnostics(diags)
}

func (v *InitHuman) PolicyResult(addr string, resp policy.EvaluationResponse) {
	v.view.PolicyResult(addr, resp)
}

func (v *InitHuman) Output(messageCode InitMessageCode, params ...any) {
	v.print(v.prepareMessage(messageCode, params...))
}

func (v *InitHuman) LogConfigurationCopyingStart(moduleSource string) {
	template := "[reset][bold]Copying configuration[reset] from %q..."
	v.print(fmt.Sprintf(template, moduleSource))
}

func (v *InitHuman) LogInitializingStateStoreStart(storeType string) {
	template := "\n[reset][bold]Initializing the state store %q..."
	v.print(fmt.Sprintf(template, storeType))
}

func (v *InitHuman) LogInitializingBackendStart() {
	msg := "\n[reset][bold]Initializing the backend..."
	v.print(msg)
}

func (v *InitHuman) LogInstallProvidersStart() {
	msg := "\n[reset][bold]Initializing provider plugins..."
	v.print(msg)
}

func (v *InitHuman) LogInstallStateStoreProviderStart(pAddr tfaddr.Provider, cons getproviders.VersionConstraints, storeType string) {
	consSuffix := ""
	if len(cons) > 0 {
		consSuffix = fmt.Sprintf(" (%s)", getproviders.VersionConstraintsString(cons))
	}
	params := []any{pAddr.ForDisplay(), consSuffix, storeType}
	msg := fmt.Sprintf(logInstallStateStoreProviderStartMessageHuman, params...)
	v.print(msg)
}

// Implements StateStoreProviderTrustLogger interface.
func (v *InitHuman) LogInteractiveApproval() {
	v.print(logInteractiveApprovalMessageHuman)
}

// Implements StateStoreProviderTrustLogger interface.
func (v *InitHuman) LogInteractiveRejection() {
	v.print(logInteractiveRejectionMessageHuman)
}

// Implements StateStoreProviderTrustLogger interface.
func (v *InitHuman) LogAutomaticApproval() {
	v.print(logInteractiveAutomaticApprovalMessageHuman)
}

func (v *InitHuman) LogFindingMatchingVersion(providerAddr addrs.Provider, versionConstraints getproviders.VersionConstraints) {
	params := []any{providerAddr.ForDisplay(), getproviders.VersionConstraintsString(versionConstraints)}
	v.print(v.prepareMessage(FindingMatchingVersionMessage, params...))
}

func (v *InitHuman) LogFindingLatestVersion(providerAddr addrs.Provider) {
	params := []any{providerAddr.ForDisplay()}
	v.print(v.prepareMessage(FindingLatestVersionMessage, params...))
}

func (v *InitHuman) LogProviderVersionAlreadyInstalled(providerAddr addrs.Provider, version getproviders.Version) {
	params := []any{providerAddr.ForDisplay(), version}
	v.print(v.prepareMessage(ProviderAlreadyInstalledMessage, params...))
}

func (v *InitHuman) LogUsingProviderVersionFromCacheDir(providerAddr addrs.Provider, version getproviders.Version) {
	params := []any{providerAddr.ForDisplay(), version}
	v.print(v.prepareMessage(UsingProviderFromCacheDirInfo, params...))
}

func (v *InitHuman) LogBuiltInProviderAvailable(providerAddr addrs.Provider) {
	params := []any{providerAddr.ForDisplay()}
	v.print(v.prepareMessage(BuiltInProviderAvailableMessage, params...))
}

func (v *InitHuman) LogInstallProviderVersionStart(providerAddr addrs.Provider, version getproviders.Version) {
	params := []any{providerAddr.ForDisplay(), version}
	v.print(v.prepareMessage(InstallingProviderMessage, params...))
}

func (v *InitHuman) LogReusingPreviousProviderVersion(providerAddr addrs.Provider, version getproviders.Version) {
	params := []any{version, providerAddr.ForDisplay()}
	v.print(v.prepareMessage(ReusingPreviousVersionInfo, params...))
}

func (v *InitHuman) LogInstallProviderVersionComplete(providerAddr addrs.Provider, version getproviders.Version, auth *getproviders.PackageAuthenticationResult) {
	params := []any{providerAddr.ForDisplay(), version, auth, ""} // add empty key id to the end
	v.print(v.prepareMessage(InstalledProviderVersionInfo, params...))
}

func (v *InitHuman) LogInstallProviderVersionCompleteWithKeyID(providerAddr addrs.Provider, version getproviders.Version, auth *getproviders.PackageAuthenticationResult, keyID string) {
	keyDetails := fmt.Sprintf(", key ID [reset][bold]%s[reset]", keyID) // key id needs to be formatted for human output
	params := []any{providerAddr.ForDisplay(), version, auth, keyDetails}
	v.print(v.prepareMessage(InstalledProviderVersionInfo, params...))
}

func (v *InitHuman) LogPartnerAndCommunityProviders() {
	v.print(v.prepareMessage(PartnerAndCommunityProvidersMessage))
}

// Implements ProviderLockingLogger
func (v *InitHuman) LogProviderLockfileCreated() {
	v.print(createdLockInfoHuman)
}

// Implements ProviderLockingLogger
func (v *InitHuman) LogProviderLockfileUpdated() {
	msg := strings.TrimSpace(dependenciesLockChangesInfo)
	v.print(msg)
}

// Implements ModuleInstallationLogger
//
// See logging in hook_module_install.go
func (v *InitHuman) LogModuleDownload(message string) {
	v.print(strings.TrimSpace(message))
}

// Implements ModuleInstallationLogger
//
// See logging in hook_module_install.go
func (v *InitHuman) LogModuleInstallation(message string) {
	v.print(message)
}

// Implements ModuleInstallationLogger
func (v *InitHuman) LogModuleUpgrade() {
	msg := "[reset][bold]Upgrading modules..."
	v.print(msg)
}

// Implements ModuleInstallationLogger
func (v *InitHuman) LogModuleInitialization() {
	msg := "[reset][bold]Initializing modules..."
	v.print(msg)
}

// print formats (trims whitespace & applies colour) and
// prints the formatted message to the stdout stream.
func (v *InitHuman) print(message string) {
	message = v.view.colorize.Color(strings.TrimSpace(message))
	v.view.streams.Println(message)
}

// prepareMessage retrieves a message template matching the InitMessageCode and
// returns a formatted string made using the template and param argument(s).
//
// As this is implemented on InitHuman the human message template is used.
func (v *InitHuman) prepareMessage(messageCode InitMessageCode, params ...any) string {
	message, ok := MessageRegistry[messageCode]
	if !ok {
		// display the message code as fallback if not found in the message registry
		return string(messageCode)
	}

	if message.HumanValue == "" {
		panic("unexpected empty message for init message code: " + string(messageCode))
	}

	return fmt.Sprintf(message.HumanValue, params...)
}

// The InitJSON implementation renders streaming JSON logs, suitable for
// integrating with other software.
type InitJSON struct {
	view *JSONView
}

var (
	_ Init                       = (*InitJSON)(nil)
	_ ProviderInstallationLogger = (*InitJSON)(nil)
)

func (v *InitJSON) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}

func (v *InitJSON) Spacer() {
	v.view.Spacer()
}

func (v *InitJSON) PolicyDiagnostics(diags policy.Diagnostics) {
	v.view.PolicyDiagnostics(diags)
}

func (v *InitJSON) PolicyResult(addr string, resp policy.EvaluationResponse) {
	v.view.PolicyResult(addr, resp)
}

func (v *InitJSON) Output(messageCode InitMessageCode, params ...any) {
	preppedMessage := v.prepareMessage(messageCode, params...)
	v.initOutputLog(preppedMessage, messageCode)
}

func (v *InitJSON) initOutputLog(preppedMessage string, messageCode any) {
	switch messageCode.(type) {
	case InitMessageCode, json.MessageType:
		// ok, tolerated custom string types
	default:
		panic(fmt.Sprintf("initOutputLog: unexpected argument type %T", messageCode))
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
		"message_code", messageCode,
	)
}

func (v *InitJSON) LogConfigurationCopyingStart(moduleSource string) {
	template := "Copying configuration from %q..."
	v.initOutputLog(fmt.Sprintf(template, moduleSource), json.CopyingConfigurationMessage)
}

func (v *InitJSON) LogInitializingStateStoreStart(storeType string) {
	template := "Initializing the state store %q..."
	v.initOutputLog(fmt.Sprintf(template, storeType), json.InitializingStateStoreMessage)
}

func (v *InitJSON) LogInitializingBackendStart() {
	msg := "Initializing the backend..."
	v.initOutputLog(msg, json.InitializingBackendMessage)
}

// logInitMessage is an internalised version of an old method `LogInitMessage`.
// New methods have since been added that replace the old `LogInitMessage` method,
// but to ensure that the same JSON output is produced we keep `logInitMessage` to
// be reused by the newer methods.
//
// Logs produced via this method are not annotated with any extra data.
// By default they contain:
// * @level as "info"
// * @module as "terraform.ui" (See NewJSONView)
// * @timestamp formatted in the default way
// * @message set as the string constructed from this method's arguments
func (v *InitJSON) logInitMessage(messageCode InitMessageCode, params ...any) {
	preppedMessage := v.prepareMessage(messageCode, params...)
	if preppedMessage == "" {
		return
	}

	v.view.Log(preppedMessage)
}

func (v *InitJSON) LogInstallProvidersStart() {
	msg := "Initializing provider plugins..."
	v.initOutputLog(msg, json.InitializingProviderPluginMessage)
}

func (v *InitJSON) LogInstallStateStoreProviderStart(pAddr tfaddr.Provider, cons getproviders.VersionConstraints, storeType string) {
	consSuffix := ""
	if len(cons) > 0 {
		consSuffix = fmt.Sprintf(" (%s)", getproviders.VersionConstraintsString(cons))
	}
	params := []any{pAddr.ForDisplay(), consSuffix, storeType}
	msg := fmt.Sprintf(logInstallStateStoreProviderStartMessageJSON, params...)

	v.view.log.Info(
		msg,
		"type", json.LogStateStoreProviderInstallationStart,
	)
}

// Implements StateStoreProviderTrustLogger interface.
func (v *InitJSON) LogInteractiveApproval() {
	v.view.log.Info(
		logInteractiveApprovalMessageJSON,
		"type", json.LogProviderInteractiveApproval,
	)
}

// Implements StateStoreProviderTrustLogger interface.
func (v *InitJSON) LogInteractiveRejection() {
	v.view.log.Info(
		logInteractiveRejectionMessageJSON,
		"type", json.LogProviderInteractiveRejection,
	)
}

// Implements StateStoreProviderTrustLogger interface.
func (v *InitJSON) LogAutomaticApproval() {
	v.view.log.Info(
		logInteractiveAutomaticApprovalMessageJSON,
		"type", json.LogProviderAutomaticApproval,
	)
}

func (v *InitJSON) LogFindingMatchingVersion(providerAddr addrs.Provider, versionConstraints getproviders.VersionConstraints) {
	params := []any{providerAddr.ForDisplay(), getproviders.VersionConstraintsString(versionConstraints)}

	// This was previously logged via LogInitMessage, so we need to match implementation of that method
	// to ensure the same JSON log is produced.
	v.logInitMessage(FindingMatchingVersionMessage, params...)
}

func (v *InitJSON) LogFindingLatestVersion(providerAddr addrs.Provider) {
	params := []any{providerAddr.ForDisplay()}

	// This was previously logged via LogInitMessage, so we need to match implementation of that method
	// to ensure the same JSON log is produced.
	v.logInitMessage(FindingLatestVersionMessage, params...)
}

func (v *InitJSON) LogProviderVersionAlreadyInstalled(providerAddr addrs.Provider, version getproviders.Version) {
	params := []any{providerAddr.ForDisplay(), version}

	// This was previously logged via LogInitMessage, so we need to match implementation of that method
	// to ensure the same JSON log is produced.
	v.logInitMessage(ProviderAlreadyInstalledMessage, params...)
}

func (v *InitJSON) LogUsingProviderVersionFromCacheDir(providerAddr addrs.Provider, version getproviders.Version) {
	params := []any{providerAddr.ForDisplay(), version}

	// This was previously logged via LogInitMessage, so we need to match implementation of that method
	// to ensure the same JSON log is produced.
	v.logInitMessage(UsingProviderFromCacheDirInfo, params...)
}

func (v *InitJSON) LogBuiltInProviderAvailable(providerAddr addrs.Provider) {
	params := []any{providerAddr.ForDisplay()}

	// This was previously logged via LogInitMessage, so we need to match implementation of that method
	// to ensure the same JSON log is produced.
	v.logInitMessage(BuiltInProviderAvailableMessage, params...)
}

func (v *InitJSON) LogInstallProviderVersionStart(providerAddr addrs.Provider, version getproviders.Version) {
	params := []any{providerAddr.ForDisplay(), version}

	// This was previously logged via LogInitMessage, so we need to match implementation of that method
	// to ensure the same JSON log is produced.
	v.logInitMessage(InstallingProviderMessage, params...)
}

func (v *InitJSON) LogReusingPreviousProviderVersion(providerAddr addrs.Provider, version getproviders.Version) {
	params := []any{providerAddr.ForDisplay(), version}

	// This was previously logged via LogInitMessage, so we need to match implementation of that method
	// to ensure the same JSON log is produced.
	v.logInitMessage(ReusingPreviousVersionInfo, params...)
}

func (v *InitJSON) LogInstallProviderVersionComplete(providerAddr addrs.Provider, version getproviders.Version, auth *getproviders.PackageAuthenticationResult) {
	params := []any{providerAddr.ForDisplay(), version, auth, ""} // add empty key id to the end

	// This was previously logged via LogInitMessage, so we need to match implementation of that method
	// to ensure the same JSON log is produced.
	v.logInitMessage(InstalledProviderVersionInfo, params...)
}

func (v *InitJSON) LogInstallProviderVersionCompleteWithKeyID(providerAddr addrs.Provider, version getproviders.Version, auth *getproviders.PackageAuthenticationResult, keyID string) {
	keyDetails := fmt.Sprintf("key_id: %s", keyID) // key id needs to be formatted for JSON output
	params := []any{providerAddr.ForDisplay(), version, auth, keyDetails}

	// This was previously logged via LogInitMessage, so we need to match implementation of that method
	// to ensure the same JSON log is produced.
	v.logInitMessage(InstalledProviderVersionInfo, params...)
}

func (v *InitJSON) LogPartnerAndCommunityProviders() {
	// This was previously logged via LogInitMessage, so we need to match implementation of that method
	// to ensure the same JSON log is produced.
	v.logInitMessage(PartnerAndCommunityProvidersMessage)
}

// Implements ProviderLockingLogger
func (v *InitJSON) LogProviderLockfileCreated() {
	msg := strings.TrimSpace(createdLockInfoJSON)
	v.initOutputLog(msg, json.LockInfo)
}

// Implements ProviderLockingLogger
func (v *InitJSON) LogProviderLockfileUpdated() {
	msg := strings.TrimSpace(dependenciesLockChangesInfo)
	v.initOutputLog(msg, json.DependenciesLockChangesInfo)
}

// Implements ModuleInstallationLogger
//
// See logging in hook_module_install.go
func (v *InitJSON) LogModuleDownload(message string) {
	v.view.Log(message)
}

// Implements ModuleInstallationLogger
//
// See logging in hook_module_install.go
func (v *InitJSON) LogModuleInstallation(message string) {
	v.view.Log(message)
}

// Implements ModuleInstallationLogger
func (v *InitJSON) LogModuleUpgrade() {
	msg := "Upgrading modules..."
	v.initOutputLog(msg, json.UpgradingModulesMessage)
}

// Implements ModuleInstallationLogger
func (v *InitJSON) LogModuleInitialization() {
	msg := "Initializing modules..."
	v.initOutputLog(msg, json.InitializingModulesMessage)
}

// prepareMessage retrieves a message template matching the InitMessageCode and
// returns a formatted string made using the template and param argument(s).
//
// As this is implemented on InitJSON the JSON message template is used.
func (v *InitJSON) prepareMessage(messageCode InitMessageCode, params ...any) string {
	message, ok := MessageRegistry[messageCode]
	if !ok {
		// display the message code as fallback if not found in the message registry
		return string(messageCode)
	}

	if message.JSONValue == "" {
		panic("unexpected empty message for init message code: " + string(messageCode))
	}

	return strings.TrimSpace(fmt.Sprintf(message.JSONValue, params...))
}

// InitMessage represents a message string in both json and human decorated text format.
type InitMessage struct {
	HumanValue string
	JSONValue  string
}

var MessageRegistry map[InitMessageCode]InitMessage = map[InitMessageCode]InitMessage{
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
	"initializing_terraform_cloud_message": {
		HumanValue: "\n[reset][bold]Initializing HCP Terraform...",
		JSONValue:  "Initializing HCP Terraform...",
	},
	"dependencies_lock_changes_info": {
		HumanValue: dependenciesLockChangesInfo,
		JSONValue:  dependenciesLockChangesInfo,
	},
	"provider_already_installed_message": {
		HumanValue: logProviderVersionAlreadyInstalledHuman,
		JSONValue:  logProviderVersionAlreadyInstalledJSON,
	},
	"built_in_provider_available_message": {
		HumanValue: logBuiltInProviderAvailableHuman,
		JSONValue:  logBuiltInProviderAvailableJSON,
	},
	"reusing_previous_version_info": {
		HumanValue: logReusingPreviousProviderVersionHuman,
		JSONValue:  logReusingPreviousProviderVersionJSON,
	},
	"finding_matching_version_message": {
		HumanValue: logFindingMatchingVersionHuman,
		JSONValue:  logFindingMatchingVersionJSON,
	},
	"finding_latest_version_message": {
		HumanValue: logFindingLatestVersionHuman,
		JSONValue:  logFindingLatestVersionJSON,
	},
	"using_provider_from_cache_dir_info": {
		HumanValue: logUsingProviderVersionFromCacheDirHuman,
		JSONValue:  logUsingProviderVersionFromCacheDirJSON,
	},
	"installing_provider_message": {
		HumanValue: logInstallProviderVersionStartHuman,
		JSONValue:  logInstallProviderVersionStartJSON,
	},
	"installed_provider_version_info": {
		HumanValue: logInstallProviderVersionCompleteHuman,
		JSONValue:  logInstallProviderVersionCompleteJSON,
	},
	"partner_and_community_providers_message": {
		HumanValue: logPartnerAndCommunityProviders,
		JSONValue:  logPartnerAndCommunityProviders,
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
}

type InitMessageCode string

const (
	// Following message codes are used and documented EXTERNALLY
	// Keep docs/internals/machine-readable-ui.mdx up to date with
	// this list when making changes here.
	OutputInitEmptyMessage            InitMessageCode = "output_init_empty_message"
	OutputInitSuccessMessage          InitMessageCode = "output_init_success_message"
	OutputInitSuccessCloudMessage     InitMessageCode = "output_init_success_cloud_message"
	OutputInitSuccessCLIMessage       InitMessageCode = "output_init_success_cli_message"
	OutputInitSuccessCLICloudMessage  InitMessageCode = "output_init_success_cli_cloud_message"
	InitializingTerraformCloudMessage InitMessageCode = "initializing_terraform_cloud_message"
	DependenciesLockChangesInfo       InitMessageCode = "dependencies_lock_changes_info"

	//// Message codes below are ONLY used INTERNALLY (for now)

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

const (
	// LogInstallStateStoreProviderStart method's message templates
	logInstallStateStoreProviderStartMessageHuman = "[reset][bold]Installing provider %s%s for state store %q..."
	logInstallStateStoreProviderStartMessageJSON  = "Installing provider %s%s for state store %q..."
)
