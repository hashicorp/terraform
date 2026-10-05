// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"fmt"
	"strings"

	version "github.com/hashicorp/go-version"
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
	JSONOutputVersionLogger

	Diagnostics(diags tfdiags.Diagnostics)
	PolicyResult(addr string, resp policy.EvaluationResponse)
	PolicyDiagnostics(diags policy.Diagnostics)

	// LogConfigurationCopyingStart describes the start of copying a module to create the root module in an empty directory.
	LogConfigurationCopyingStart(moduleSource string)

	// LogInstallStateStoreProviderStart indicates progress during installation of a state store provider.
	// This is signposted as distinct from general provider download, which may happen later in init.
	LogInstallStateStoreProviderStart(providerAddr addrs.Provider, cons getproviders.VersionConstraints, storeType string)

	// LogInitializingStateStoreStart indicates progress during initialization of a state store.
	LogInitializingStateStoreStart(storeType string)

	// LogInitializingBackendStart indicates progress initializing a backend.
	LogInitializingBackendStart()

	// LogMigrateBackendUnsetStart indicates that the state is going to be migrated after unsetting/successfully removing from the configuration.
	LogMigrateBackendUnsetStart(backendType string)

	// LogMigrateBackendUnsetEnd indicates that the state is going to be migrated after the backend has been unset/removed from the configuration.
	LogMigrateBackendUnsetEnd(backendType string)

	// LogMigrateBackendReconfigured indicates that the backend has been reconfigured successfully.
	LogMigrateBackendReconfigured()

	// LogMigrateFromBackendToBackend indicates that the state is going to be migrated from one backend type to another.
	LogMigrateFromBackendToBackend(oldType, newType string)

	// LogMigrateFromBackendToCloud indicates that the state is going to be migrated from the backend to the cloud.
	LogMigrateFromBackendToCloud(backendType string)

	// LogMigrateFromCloudToBackend indicates that the state is going to be migrated from the cloud to the backend.
	LogMigrateFromCloudToBackend(backendType string)

	// LogMigrateFromCloudToLocal indicates that the state is going to be migrated from the cloud to a local backend.
	LogMigrateFromCloudToLocal()

	// LogMigrateCloudConfigurationChanged indicates that the backend is being migrated following a change in cloud configuration.
	LogMigrateCloudConfigurationChanged()

	// LogInitializingHCPTerraformStart indicates progress initializing the `cloud` backend.
	LogInitializingHCPTerraformStart()

	// LogModuleUpgrade describes the start of upgrading a module during init.
	LogModuleUpgrade()

	// LogModuleInitialization describes the start of initializing a module during init.
	LogModuleInitialization()

	// LogInitSuccess reports a successful init command completing
	LogInitSuccess()

	// LogInitSuccessCloud is just like LogInitSuccess but uses HCP Terraform-specific language
	LogInitSuccessCloud()

	// LogInitSuccessEmpty reports a successful init command completing, but notes that the config was empty
	LogInitSuccessEmpty()

	// LogCallToActionCLI is used when Terraform is not running in automation and prompt users about using the primary workflow
	LogCallToActionCLI()

	// LogCallToActionCLICloud is just like LogCallToActionCLI but uses HCP Terraform-specific language
	LogCallToActionCLICloud()

	// LogBackendConfiguredSuccess reports that the backend was successfully configured.
	LogBackendConfiguredSuccess(backendType string)

	ModuleInstallationLogger
	ProviderInstallationLogger
	ProviderLockingLogger

	StateStoreProviderTrustLogger

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

func (v *InitHuman) Version() {}

var (
	_ Init                          = (*InitHuman)(nil)
	_ JSONOutputVersionLogger       = (*InitHuman)(nil)
	_ Spacer                        = (*InitHuman)(nil)
	_ ProviderInstallationLogger    = (*InitHuman)(nil)
	_ ProviderLockingLogger         = (*InitHuman)(nil)
	_ StateStoreProviderTrustLogger = (*InitHuman)(nil)
	_ ModuleInstallationLogger      = (*InitHuman)(nil)
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

func (v *InitHuman) LogConfigurationCopyingStart(moduleSource string) {
	v.print(fmt.Sprintf("[reset][bold]Copying configuration[reset] from %q...", moduleSource))
}

func (v *InitHuman) LogInitializingBackendStart() {
	v.print("\n[reset][bold]Initializing the backend...")
}

func (v *InitHuman) LogMigrateBackendUnsetStart(backendType string) {
	v.print(fmt.Sprintf(`Terraform has detected you're unconfiguring your previously set %q backend.`, backendType))
}

func (v *InitHuman) LogMigrateBackendUnsetEnd(backendType string) {
	v.print(fmt.Sprintf(`[reset][green]

Successfully unset the backend %q. Terraform will now operate locally.`, backendType))
}

func (v *InitHuman) LogMigrateBackendReconfigured() {
	v.print(`[reset][bold]Backend configuration changed![reset]

Terraform has detected that the configuration specified for the backend
has changed. Terraform will now check for existing state in the backends.
`)
}

func (v *InitHuman) LogMigrateFromBackendToBackend(oldType, newType string) {
	v.print(fmt.Sprintf(`[reset]Terraform detected that the backend type changed from %q to %q.`, oldType, newType))
}

func (v *InitHuman) LogMigrateFromBackendToCloud(backendType string) {
	v.print(fmt.Sprintf("Migrating from backend %q to HCP Terraform.", backendType))
}

func (v *InitHuman) LogMigrateFromCloudToBackend(backendType string) {
	v.print(fmt.Sprintf("Migrating from HCP Terraform to backend %q.", backendType))
}

func (v *InitHuman) LogMigrateFromCloudToLocal() {
	v.print("Migrating from HCP Terraform or Terraform Enterprise to local state.")
}

func (v *InitHuman) LogMigrateCloudConfigurationChanged() {
	v.print("HCP Terraform configuration has changed.")
}

func (v *InitHuman) LogInitializingHCPTerraformStart() {
	v.print("\n[reset][bold]Initializing HCP Terraform...")
}

func (v *InitHuman) LogInitSuccess() {
	v.print(strings.TrimSpace(outputInitSuccess))
}

func (v *InitHuman) LogInitSuccessCloud() {
	v.print(strings.TrimSpace(outputInitSuccessCloud))
}

func (v *InitHuman) LogInitSuccessEmpty() {
	v.print(strings.TrimSpace(outputInitEmpty))
}

func (v *InitHuman) LogCallToActionCLI() {
	v.print(strings.TrimSpace(outputInitSuccessCLI))
}

func (v *InitHuman) LogCallToActionCLICloud() {
	v.print(strings.TrimSpace(outputInitSuccessCLICloud))
}

func (v *InitHuman) LogBackendConfiguredSuccess(backendType string) {
	v.print(fmt.Sprintf(strings.TrimSpace(backendConfiguredSuccessHuman), backendType))
}

func (v *InitHuman) LogInstallProvidersStart() {
	v.print("\n[reset][bold]Initializing provider plugins...")
}

func (v *InitHuman) LogInstallStateStoreProviderStart(pAddr tfaddr.Provider, cons getproviders.VersionConstraints, storeType string) {
	consSuffix := ""
	if len(cons) > 0 {
		consSuffix = fmt.Sprintf(" (%s)", getproviders.VersionConstraintsString(cons))
	}
	params := []any{pAddr.ForDisplay(), consSuffix, storeType}
	v.print(fmt.Sprintf(logInstallStateStoreProviderStartMessageHuman, params...))
}

func (v *InitHuman) LogInitializingStateStoreStart(storeType string) {
	v.print(fmt.Sprintf("\n[reset][bold]Initializing the state store %q...", storeType))
}

func (v *InitHuman) LogInteractiveApproval() {
	v.print(logInteractiveApprovalMessageHuman)
}

func (v *InitHuman) LogInteractiveRejection() {
	v.print(logInteractiveRejectionMessageHuman)
}

func (v *InitHuman) LogAutomaticApproval() {
	v.print(logInteractiveAutomaticApprovalMessageHuman)
}

func (v *InitHuman) LogFindingMatchingVersion(providerAddr addrs.Provider, versionConstraints getproviders.VersionConstraints) {
	v.print(fmt.Sprintf("- Finding %s versions matching %q...", providerAddr.ForDisplay(), getproviders.VersionConstraintsString(versionConstraints)))
}

func (v *InitHuman) LogFindingLatestVersion(providerAddr addrs.Provider) {
	v.print(fmt.Sprintf("- Finding latest version of %s...", providerAddr.ForDisplay()))
}

func (v *InitHuman) LogProviderVersionAlreadyInstalled(providerAddr addrs.Provider, version getproviders.Version) {
	v.print(fmt.Sprintf("- Using previously-installed %s v%s", providerAddr.ForDisplay(), version))
}

func (v *InitHuman) LogUsingProviderVersionFromCacheDir(providerAddr addrs.Provider, version getproviders.Version) {
	v.print(fmt.Sprintf("- Using %s v%s from the shared cache directory", providerAddr.ForDisplay(), version))
}

func (v *InitHuman) LogBuiltInProviderAvailable(providerAddr addrs.Provider) {
	v.print(fmt.Sprintf("- %s is built in to Terraform", providerAddr.ForDisplay()))
}

func (v *InitHuman) LogInstallProviderVersionStart(providerAddr addrs.Provider, version getproviders.Version) {
	v.print(fmt.Sprintf("- Installing %s v%s...", providerAddr.ForDisplay(), version))
}

func (v *InitHuman) LogReusingPreviousProviderVersion(providerAddr addrs.Provider, version getproviders.Version) {
	v.print(fmt.Sprintf("- Reusing version %s of %s from the dependency lock file", version, providerAddr.ForDisplay()))
}

func (v *InitHuman) LogInstallProviderVersionComplete(providerAddr addrs.Provider, version getproviders.Version, auth *getproviders.PackageAuthenticationResult) {
	v.print(fmt.Sprintf("- Installed %s v%s (%s%s)", providerAddr.ForDisplay(), version, auth, "")) // add empty key id to the end
}

func (v *InitHuman) LogInstallProviderVersionCompleteWithKeyID(providerAddr addrs.Provider, version getproviders.Version, auth *getproviders.PackageAuthenticationResult, keyID string) {
	keyDetails := fmt.Sprintf(", key ID [reset][bold]%s[reset]", keyID) // key id needs to be formatted for human output
	msg := fmt.Sprintf("- Installed %s v%s (%s%s)", providerAddr.ForDisplay(), version, auth, keyDetails)
	v.print(msg)
}

func (v *InitHuman) LogPartnerAndCommunityProviders() {
	v.print(logPartnerAndCommunityProviders)
}

func (v *InitHuman) LogProviderLockfileCreated() {
	v.print(createdLockInfoHuman)
}

func (v *InitHuman) LogProviderLockfileUpdated() {
	v.print(dependenciesLockChangesInfo)
}

// Implements ModuleInstallationLogger
//
// See logging in hook_module_install.go
func (v *InitHuman) LogModuleDownload(packageAddr string, version *version.Version, modulePath string) {
	var message string
	if version == nil {
		message = fmt.Sprintf(moduleDownloadHuman, packageAddr, modulePath)
	} else {
		message = fmt.Sprintf(moduleDownloadWithVersionHuman, packageAddr, version, modulePath)
	}
	v.print(strings.TrimSpace(message))
}

// Implements ModuleInstallationLogger
//
// See logging in hook_module_install.go
func (v *InitHuman) LogModuleInstallation(modulePath string) {
	v.print(fmt.Sprintf(moduleInstallationHuman, modulePath))
}

// Implements ModuleInstallationLogger
//
// See logging in hook_module_install.go
func (v *InitHuman) LogModuleInstallationWithLocalPath(modulePath, localDir string) {
	v.print(fmt.Sprintf(moduleInstallationWithLocalPathHuman, modulePath, localDir))
}

func (v *InitHuman) LogModuleUpgrade() {
	v.print("[reset][bold]Upgrading modules...")
}

func (v *InitHuman) LogModuleInitialization() {
	v.print("[reset][bold]Initializing modules...")
}

// print formats (trims whitespace & applies colour) and
// prints the formatted message to the stdout stream.
func (v *InitHuman) print(message string) {
	message = v.view.colorize.Color(strings.TrimSpace(message))
	v.view.streams.Println(message)
}

// The InitJSON implementation renders streaming JSON logs, suitable for
// integrating with other software.
type InitJSON struct {
	view *JSONView
}

var (
	_ Init                          = (*InitJSON)(nil)
	_ JSONOutputVersionLogger       = (*InitJSON)(nil)
	_ Spacer                        = (*InitJSON)(nil)
	_ ProviderInstallationLogger    = (*InitJSON)(nil)
	_ ProviderLockingLogger         = (*InitJSON)(nil)
	_ StateStoreProviderTrustLogger = (*InitJSON)(nil)
	_ ModuleInstallationLogger      = (*InitJSON)(nil)
)

func (v *InitJSON) Version() {
	v.view.Version()
}

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

func (v *InitJSON) initOutputLog(preppedMessage string, messageCode json.MessageType) {
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

func (v *InitJSON) LogConfigurationCopyingStart(moduleSource string) {
	v.initOutputLog(fmt.Sprintf("Copying configuration from %q...", moduleSource), json.MessageCopyingConfigurationMessage)
}

func (v *InitJSON) LogInitializingBackendStart() {
	v.initOutputLog("Initializing the backend...", json.MessageInitializingBackendMessage)
}

func (v *InitJSON) LogMigrateBackendUnsetStart(backendType string) {
	// `-json` and `-migrate-state` are mutually exclusive.
	panic("InitJSON: LogMigrateBackendUnsetStart not implemented")
}

func (v *InitJSON) LogMigrateBackendUnsetEnd(backendType string) {
	// `-json` and `-migrate-state` are mutually exclusive.
	panic("InitJSON: LogMigrateBackendUnsetEnd not implemented")
}

func (v *InitJSON) LogMigrateBackendReconfigured() {
	// `-json` and `-migrate-state` are mutually exclusive.
	panic("InitJSON: LogMigrateBackendReconfigured not implemented")
}

func (v *InitJSON) LogMigrateFromBackendToBackend(oldType, newType string) {
	// `-json` and `-migrate-state` are mutually exclusive.
	panic("InitJSON: LogMigrateFromBackendToBackend not implemented")
}

func (v *InitJSON) LogMigrateFromBackendToCloud(backendType string) {
	// `-json` and `-migrate-state` are mutually exclusive.
	panic("InitJSON: LogMigrateFromBackendToCloud not implemented")
}

func (v *InitJSON) LogMigrateFromCloudToBackend(backendType string) {
	// `-json` and `-migrate-state` are mutually exclusive.
	panic("InitJSON: LogMigrateFromCloudToBackend not implemented")
}

func (v *InitJSON) LogMigrateFromCloudToLocal() {
	// `-json` and `-migrate-state` are mutually exclusive.
	panic("InitJSON: LogMigrateFromCloudToLocal not implemented")
}

func (v *InitJSON) LogMigrateCloudConfigurationChanged() {
	// `-json` and `-migrate-state` are mutually exclusive.
	panic("InitJSON: LogMigrateCloudConfigurationChanged not implemented")
}

func (v *InitJSON) LogInitializingHCPTerraformStart() {
	v.initOutputLog("Initializing HCP Terraform...", json.MessageInitializingTerraformCloudMessage)
}

func (v *InitJSON) LogInitSuccess() {
	v.initOutputLog(strings.TrimSpace(outputInitSuccessJSON), json.MessageOutputInitSuccessMessage)
}

func (v *InitJSON) LogInitSuccessCloud() {
	v.initOutputLog(strings.TrimSpace(outputInitSuccessCloudJSON), json.MessageOutputInitSuccessCloudMessage)
}

func (v *InitJSON) LogInitSuccessEmpty() {
	v.initOutputLog(strings.TrimSpace(outputInitEmptyJSON), json.MessageOutputInitEmptyMessage)
}

func (v *InitJSON) LogCallToActionCLI() {
	v.initOutputLog(strings.TrimSpace(outputInitSuccessCLI_JSON), json.MessageOutputInitSuccessCLIMessage)
}

func (v *InitJSON) LogCallToActionCLICloud() {
	v.initOutputLog(strings.TrimSpace(outputInitSuccessCLICloudJSON), json.MessageOutputInitSuccessCLICloudMessage)
}

func (v *InitJSON) LogBackendConfiguredSuccess(backendType string) {
	v.initOutputLog(fmt.Sprintf(strings.TrimSpace(backendConfiguredSuccessJSON), backendType), json.MessageBackendConfiguredSuccess)
}

func (v *InitJSON) LogInstallProvidersStart() {
	v.initOutputLog("Initializing provider plugins...", json.MessageInitializingProviderPluginMessage)
}

func (v *InitJSON) LogInstallStateStoreProviderStart(pAddr tfaddr.Provider, cons getproviders.VersionConstraints, storeType string) {
	consSuffix := ""
	if len(cons) > 0 {
		consSuffix = fmt.Sprintf(" (%s)", getproviders.VersionConstraintsString(cons))
	}
	params := []any{pAddr.ForDisplay(), consSuffix, storeType}

	v.view.log.Info(
		fmt.Sprintf(logInstallStateStoreProviderStartMessageJSON, params...),
		"type", json.MessageStateStoreProviderInstallationStart,
	)
}

func (v *InitJSON) LogInitializingStateStoreStart(storeType string) {
	v.view.log.Info(
		fmt.Sprintf("Initializing the state store %q...", storeType),
		"type", json.MessageStateStoreInitializationStart,
	)
}

func (v *InitJSON) LogInteractiveApproval() {
	v.view.log.Info(
		logInteractiveApprovalMessageJSON,
		"type", json.MessageProviderInteractiveApproval,
	)
}

func (v *InitJSON) LogInteractiveRejection() {
	v.view.log.Info(
		logInteractiveRejectionMessageJSON,
		"type", json.MessageProviderInteractiveRejection,
	)
}

func (v *InitJSON) LogAutomaticApproval() {
	v.view.log.Info(
		logInteractiveAutomaticApprovalMessageJSON,
		"type", json.MessageProviderAutomaticApproval,
	)
}

func (v *InitJSON) LogFindingMatchingVersion(providerAddr addrs.Provider, versionConstraints getproviders.VersionConstraints) {
	v.view.Log(fmt.Sprintf("Finding matching versions for provider: %s, version_constraint: %q", providerAddr.ForDisplay(), getproviders.VersionConstraintsString(versionConstraints)))
}

func (v *InitJSON) LogFindingLatestVersion(providerAddr addrs.Provider) {
	v.view.Log(fmt.Sprintf("%s: Finding latest version...", providerAddr.ForDisplay()))
}

func (v *InitJSON) LogProviderVersionAlreadyInstalled(providerAddr addrs.Provider, version getproviders.Version) {
	v.view.Log(fmt.Sprintf("%s v%s: Using previously-installed provider version", providerAddr.ForDisplay(), version))
}

func (v *InitJSON) LogUsingProviderVersionFromCacheDir(providerAddr addrs.Provider, version getproviders.Version) {
	v.view.Log(fmt.Sprintf("%s v%s: Using from the shared cache directory", providerAddr.ForDisplay(), version))
}

func (v *InitJSON) LogBuiltInProviderAvailable(providerAddr addrs.Provider) {
	v.view.Log(fmt.Sprintf("%s is built in to Terraform", providerAddr.ForDisplay()))
}

func (v *InitJSON) LogInstallProviderVersionStart(providerAddr addrs.Provider, version getproviders.Version) {
	v.view.Log(fmt.Sprintf("Installing provider version: %s v%s...", providerAddr.ForDisplay(), version))
}

func (v *InitJSON) LogReusingPreviousProviderVersion(providerAddr addrs.Provider, version getproviders.Version) {
	v.view.Log(fmt.Sprintf("%s: Reusing version %s from the dependency lock file", providerAddr.ForDisplay(), version))
}

func (v *InitJSON) LogInstallProviderVersionComplete(providerAddr addrs.Provider, version getproviders.Version, auth *getproviders.PackageAuthenticationResult) {
	v.view.Log(fmt.Sprintf("Installed provider version: %s v%s (%s%s)", providerAddr.ForDisplay(), version, auth, "")) // empty key id at the end
}

func (v *InitJSON) LogInstallProviderVersionCompleteWithKeyID(providerAddr addrs.Provider, version getproviders.Version, auth *getproviders.PackageAuthenticationResult, keyID string) {
	keyDetails := fmt.Sprintf("key_id: %s", keyID) // key id needs to be formatted for JSON output
	msg := fmt.Sprintf("Installed provider version: %s v%s (%s%s)", providerAddr.ForDisplay(), version, auth, keyDetails)
	v.view.Log(msg)
}

func (v *InitJSON) LogPartnerAndCommunityProviders() {
	v.view.Log(logPartnerAndCommunityProviders)
}

func (v *InitJSON) LogProviderLockfileCreated() {
	v.initOutputLog(strings.TrimSpace(createdLockInfoJSON), json.MessageLockInfo)
}

func (v *InitJSON) LogProviderLockfileUpdated() {
	v.initOutputLog(strings.TrimSpace(dependenciesLockChangesInfo), json.MessageDependenciesLockChangesInfo)
}

// Implements ModuleInstallationLogger
//
// See logging in hook_module_install.go
func (v *InitJSON) LogModuleDownload(packageAddr string, version *version.Version, modulePath string) {
	var message string
	if version == nil {
		message = fmt.Sprintf(moduleDownloadHuman, packageAddr, modulePath)
	} else {
		message = fmt.Sprintf(moduleDownloadWithVersionHuman, packageAddr, version, modulePath)
	}

	v.view.Log(message)
}

// Implements ModuleInstallationLogger
//
// See logging in hook_module_install.go
func (v *InitJSON) LogModuleInstallation(modulePath string) {
	v.view.Log(fmt.Sprintf(moduleInstallationHuman, modulePath))
}

// Implements ModuleInstallationLogger
//
// See logging in hook_module_install.go
func (v *InitJSON) LogModuleInstallationWithLocalPath(modulePath, localDir string) {
	v.view.Log(fmt.Sprintf(moduleInstallationWithLocalPathHuman, modulePath, localDir))
}

func (v *InitJSON) LogModuleUpgrade() {
	v.initOutputLog("Upgrading modules...", json.MessageUpgradingModulesMessage)
}

func (v *InitJSON) LogModuleInitialization() {
	v.initOutputLog("Initializing modules...", json.MessageInitializingModulesMessage)
}

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

const (
	// LogInstallStateStoreProviderStart method's message templates
	logInstallStateStoreProviderStartMessageHuman = "[reset][bold]Installing provider %s%s for state store %q..."
	logInstallStateStoreProviderStartMessageJSON  = "Installing provider %s%s for state store %q..."
)
