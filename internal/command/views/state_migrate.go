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
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Message text used in human output.
const (
	// Notify the user that any preparation steps are over and the migration is starting.
	StateMigrationStartMessage = "[reset][bold]Migrating state from %s to %s...[reset]"

	// Notify the user that everything has finished successfully; migration and lockfile+backend state file updates.
	StateMigrationFinalizedMessage = "[reset][bold]Finished migrating state from %s to %s.[reset]"

	// Notify the user that an error has occurred, but there have been changes to where state is stored.
	// Hopefully the errors accompanying this message are actionable by users, but if not we expect a bug report.
	StateMigrationPostStepsInterruptedMessage = `[reset][bold]Finished migrating state from %s to %s, but an error occurred before Terraform was finished.[reset]

Your state has been copied to the new destination, but Terraform was unable to perform final operations to enable future commands to use your migrated state. Either Terraform was unable to record the new provider used for the destination state store to your dependency lock file, or the backend state file was unable to be updated. Please check the errors message(s) above for more information.

The successful migration means you will have two copies of your state, both in the source and destination locations.

If you can address the errors you can retry this command safely. Otherwise, please report the issue to the Terraform team with the error messages and your configuration.
`

	// Notify the user that the migration failed. This may be due to a misconfiguration, e.g. insufficient permissions to interact with a service.
	// We expect these errors to either be actionable by users, or to originate from a state store provider (but reports shouldn't come to us unless due to a backend).
	StateMigrationFailureMessage = `[reset][bold]Failed to migrate state from %s to %s.[reset]

Something went wrong while migrating the state. Please check the errors message(s) above for more information.

The "terraform state migrate" command does not modify the source state, so you can retry this command safely after addressing errors. When the command does succeed you will have two copies of your state, both in the source and destination locations.

Make sure you're supplying all the necessary attribute values for both the source and destination state stores. Remember, some values may need to be supplied via environment variables for either of the source or destination locations. If you continue to experience issues please report the issue to either the Terraform team when using a backend, or to the relevant provider development team when using a pluggable state store.
`
)

type stateMigrationFailureMode string

// In the human view these values are only used to control which human-readable message is printed to stdout.
// In the JSON view these values are used as the value of a field indicating when the error occurred.
// Therefore these strings are user-facing in JSON output and should not be changed!
const (
	DuringMigration          stateMigrationFailureMode = "error_during_migration"
	DuringLockfile           stateMigrationFailureMode = "error_updating_provider_lockfile"
	DuringWorkDirStateUpdate stateMigrationFailureMode = "error_updating_workdir_state"
)

type StateMigrate interface {
	Log(message string, params ...any)
	Diagnostics(diags tfdiags.Diagnostics)

	LogStateMigrationStart(source, destination string)
	LogStateMigrationComplete()
	LogStateMigrationErrored(failMode stateMigrationFailureMode, source, destination string)
	LogStateMigrationFinalized(source, destination string)

	LogMigrationSourceInitializationStart()
	LogMigrationSourceInitializationComplete()
	LogMigrationDestinationInitializationStart()
	LogMigrationDestinationInitializationComplete()

	ProviderInstallationLogger
	ProviderLockingLogger

	StateStoreProviderTrustLogger

	Spacer // The `state migrate` command logs empty lines to space-out different sections of human-readable output
}

func NewStateMigrate(viewType arguments.ViewType, view *View) StateMigrate {
	switch viewType {
	case arguments.ViewHuman:
		return &StateMigrateHuman{view: view}
	default:
		panic(fmt.Sprintf("unsupported view type: %s", viewType))
	}
}

var (
	_ StateMigrate                  = (*StateMigrateHuman)(nil)
	_ ProviderInstallationLogger    = (*StateMigrateHuman)(nil)
	_ StateStoreProviderTrustLogger = (*StateMigrateHuman)(nil)
	_ Spacer                        = (*StateMigrateHuman)(nil)
)

type StateMigrateHuman struct {
	view *View
}

func (s *StateMigrateHuman) Diagnostics(diags tfdiags.Diagnostics) {
	s.view.Diagnostics(diags)
}

func (s *StateMigrateHuman) LogStateMigrationStart(source string, destination string) {
	msg := fmt.Sprintf(StateMigrationStartMessage, source, destination)
	s.log(msg)
}

func (s *StateMigrateHuman) LogStateMigrationComplete() {
	// no-op in human view
}

func (s *StateMigrateHuman) LogStateMigrationErrored(failMode stateMigrationFailureMode, source, destination string) {
	// The JSON object describes slightly different failures that led to an error.
	// So different messages are be logged depending which happened.
	var msg string
	switch failMode {
	case DuringMigration:
		// migration itself failed
		msg = fmt.Sprintf(StateMigrationFailureMessage, source, destination)
	case DuringLockfile, DuringWorkDirStateUpdate:
		// migration succeeded by updates in the working directory failed
		msg = fmt.Sprintf(StateMigrationPostStepsInterruptedMessage, source, destination)
	default:
		panic(fmt.Sprintf("(*StateMigrateHuman)LogStateMigrationErrored: called incorrectly with unknown failure mode : %q", failMode))
	}

	s.log(msg)
}

func (s *StateMigrateHuman) LogMigrationSourceInitializationStart() {
	// no-op in human view
}

func (s *StateMigrateHuman) LogMigrationSourceInitializationComplete() {
	// no-op in human view
}

func (s *StateMigrateHuman) LogMigrationDestinationInitializationStart() {
	// no-op in human view
}

func (s *StateMigrateHuman) LogMigrationDestinationInitializationComplete() {
	// no-op in human view
}

func (s *StateMigrateHuman) LogStateMigrationFinalized(source string, destination string) {
	msg := fmt.Sprintf(StateMigrationFinalizedMessage, source, destination)
	s.log(msg)
}

func (s *StateMigrateHuman) Log(message string, params ...any) {
	s.log(fmt.Sprintf(message, params...))
}

func (s *StateMigrateHuman) log(preparedMessage string) {
	msg := s.view.colorize.Color(strings.TrimSpace(preparedMessage))
	s.view.streams.Println(msg)
}

// Implements Spacer
func (s *StateMigrateHuman) Spacer() {
	s.view.Spacer()
}

// Implements ProviderInstallationLogger interface.
func (s *StateMigrateHuman) Output(code InitMessageCode, params ...any) {
	msg, ok := MessageRegistry[code]
	if !ok {
		panic("missing message for InstallingProviderMessage init message code")
	}
	s.Log(msg.HumanValue, params...)
}

// Implements ProviderInstallationLogger interface.
func (s *StateMigrateHuman) LogInstallStateStoreProviderStart(pAddr tfaddr.Provider, cons getproviders.VersionConstraints, storeType string) {
	consSuffix := ""
	if len(cons) > 0 {
		consSuffix = fmt.Sprintf(" (%s)", getproviders.VersionConstraintsString(cons))
	}
	params := []any{pAddr.ForDisplay(), consSuffix, storeType}
	msg := fmt.Sprintf(logInstallingStateStoreProviderStartMessageHuman, params...)
	s.log(msg)
}

// Implements StateStoreProviderTrustLogger interface.
func (s *StateMigrateHuman) LogInteractiveApproval() {
	s.log(logInteractiveApprovalMessageHuman)
}

// Implements StateStoreProviderTrustLogger interface.
func (s *StateMigrateHuman) LogInteractiveRejection() {
	s.log(logInteractiveRejectionMessageHuman)
}

// Implements StateStoreProviderTrustLogger interface.
func (s *StateMigrateHuman) LogAutomaticApproval() {
	s.log(logInteractiveAutomaticApprovalMessageHuman)
}

// Implements ProviderInstallationLogger interface.
func (s *StateMigrateHuman) LogFindingMatchingVersion(providerAddr addrs.Provider, versionConstraints getproviders.VersionConstraints) {
	params := []any{providerAddr.ForDisplay(), getproviders.VersionConstraintsString(versionConstraints)}
	msg := s.prepareMessage(FindingMatchingVersionMessage, params...)
	s.log(msg)
}

// Implements ProviderInstallationLogger interface.
func (s *StateMigrateHuman) LogFindingLatestVersion(providerAddr addrs.Provider) {
	params := []any{providerAddr.ForDisplay()}
	msg := s.prepareMessage(FindingLatestVersionMessage, params...)
	s.log(msg)
}

// Implements ProviderInstallationLogger interface.
func (s *StateMigrateHuman) LogProviderVersionAlreadyInstalled(providerAddr addrs.Provider, version getproviders.Version) {
	params := []any{providerAddr.ForDisplay(), version}
	msg := s.prepareMessage(ProviderAlreadyInstalledMessage, params...)
	s.log(msg)
}

// Implements ProviderInstallationLogger interface.
func (s *StateMigrateHuman) LogUsingProviderVersionFromCacheDir(providerAddr addrs.Provider, version getproviders.Version) {
	params := []any{providerAddr.ForDisplay(), version}
	msg := s.prepareMessage(UsingProviderFromCacheDirInfo, params...)
	s.log(msg)
}

// Implements ProviderInstallationLogger interface.
func (s *StateMigrateHuman) LogBuiltInProviderAvailable(providerAddr addrs.Provider) {
	params := []any{providerAddr.ForDisplay()}
	msg := s.prepareMessage(BuiltInProviderAvailableMessage, params...)
	s.log(msg)
}

// Implements ProviderInstallationLogger interface.
func (s *StateMigrateHuman) LogInstallingProviderVersion(providerAddr addrs.Provider, version getproviders.Version) {
	params := []any{providerAddr.ForDisplay(), version}
	msg := s.prepareMessage(InstallingProviderMessage, params...)
	s.log(msg)
}

// Implements ProviderInstallationLogger interface.
func (s *StateMigrateHuman) LogReusingPreviousProviderVersion(providerAddr addrs.Provider, version getproviders.Version) {
	params := []any{version, providerAddr.ForDisplay()}
	msg := s.prepareMessage(ReusingPreviousVersionInfo, params...)
	s.log(msg)
}

// Implements ProviderInstallationLogger interface.
func (s *StateMigrateHuman) LogProviderVersionSuccess(providerAddr addrs.Provider, version getproviders.Version, auth *getproviders.PackageAuthenticationResult) {
	params := []any{providerAddr.ForDisplay(), version, auth, ""} // add empty key id to the end
	msg := s.prepareMessage(InstalledProviderVersionInfo, params...)
	s.log(msg)
}

// Implements ProviderInstallationLogger interface.
func (s *StateMigrateHuman) LogProviderVersionSuccessWithKeyID(providerAddr addrs.Provider, version getproviders.Version, auth *getproviders.PackageAuthenticationResult, keyID string) {
	keyDetails := fmt.Sprintf(", key ID [reset][bold]%s[reset]", keyID) // key id needs to be formatted for human output
	params := []any{providerAddr.ForDisplay(), version, auth, keyDetails}

	msg := s.prepareMessage(InstalledProviderVersionInfo, params...)
	s.log(msg)
}

// Implements ProviderInstallationLogger interface.
func (s *StateMigrateHuman) LogPartnerAndCommunityProviders() {
	msg := s.prepareMessage(PartnerAndCommunityProvidersMessage)
	s.log(msg)
}

// Implements DependencyLockLogger interface.
func (s *StateMigrateHuman) LogProviderLockfileCreated() {
	s.log(previousLockInfoHuman)
}

// Implements DependencyLockLogger interface.
func (s *StateMigrateHuman) LogProviderLockfileUpdated() {
	s.log(dependenciesLockChangesInfo)
}

// Implements ProviderInstallationLogger interface.
func (s *StateMigrateHuman) prepareMessage(code InitMessageCode, params ...any) string {
	message, ok := MessageRegistry[code]
	if !ok {
		// display the message code as fallback if not found in the message registry
		return string(code)
	}

	if message.HumanValue == "" {
		// no need to apply colorization if the message is empty
		return message.HumanValue
	}

	return s.view.colorize.Color(strings.TrimSpace(fmt.Sprintf(message.HumanValue, params...)))
}

type StateMigrateJSON struct {
	view *JSONView
}

var (
	_ ProviderLockingLogger         = (*StateMigrateJSON)(nil)
	_ StateStoreProviderTrustLogger = (*StateMigrateJSON)(nil)
	_ Spacer                        = (*StateMigrateJSON)(nil)
)

// Implements Spacer
func (s *StateMigrateJSON) Spacer() {
	// no-op for JSON output, since we don't want to log empty messages in JSON
}

// Implements StateStoreProviderTrustLogger interface.
func (s *StateMigrateJSON) LogInteractiveApproval() {
	s.view.log.Info(
		logInteractiveApprovalMessageJSON,
		"type", json.ProviderInteractiveApproval,
	)
}

// Implements StateStoreProviderTrustLogger interface.
func (s *StateMigrateJSON) LogInteractiveRejection() {
	s.view.log.Info(
		logInteractiveRejectionMessageJSON,
		"type", json.ProviderInteractiveRejection,
	)
}

// Implements StateStoreProviderTrustLogger interface.
func (s *StateMigrateJSON) LogAutomaticApproval() {
	s.view.log.Info(
		logInteractiveAutomaticApprovalMessageJSON,
		"type", json.ProviderAutomaticApproval,
	)
}

// Implements ProviderLockingLogger interface.
func (s *StateMigrateJSON) LogProviderLockfileCreated() {
	msg := strings.TrimSpace(previousLockInfoJSON)
	s.view.log.Info(
		msg,
		"type", json.ProviderLockfileCreated,
	)
}

// Implements ProviderLockingLogger interface.
func (s *StateMigrateJSON) LogProviderLockfileUpdated() {
	msg := strings.TrimSpace(dependenciesLockChangesInfo)
	s.view.log.Info(
		msg,
		"type", json.ProviderLockfileUpdated,
	)
}
