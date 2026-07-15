// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/getproviders"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Message text used in human output.
const (
	// Notify the user that any preparation steps are over and the migration is starting.
	StateMigrationStartMessage = "[reset][bold]Migrating state from %s to %s...[reset]"

	// Notify the user that everything has completed successfully.
	StateMigrationCompletedMessage = "[reset][bold]Finished migrating state from %s to %s.[reset]"

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

type StateMigrate interface {
	Log(message string, params ...any)
	Diagnostics(diags tfdiags.Diagnostics)

	ProviderInstaller
	Spacer // The `state migrate` command logs empty lines to space-out different sections of human-readable output
}

func NewStateMigrate(viewType arguments.ViewType, view *View) StateMigrate {
	switch viewType {
	case arguments.ViewHuman:
		return &StateMigrateHuman{
			view: view,
		}
	default:
		return &StateMigrateJSON{
			view: NewJSONView(view),
		}
	}
}

var (
	_ StateMigrate      = (*StateMigrateHuman)(nil)
	_ ProviderInstaller = (*StateMigrateHuman)(nil)
	_ Spacer            = (*StateMigrateHuman)(nil)
)

type StateMigrateHuman struct {
	view *View
}

func (s *StateMigrateHuman) Diagnostics(diags tfdiags.Diagnostics) {
	s.view.Diagnostics(diags)
}

// Plain logging of messages, without using message codes
func (s *StateMigrateHuman) Log(message string, params ...any) {
	s.log(fmt.Sprintf(message, params...))
}

// log is reused to ensure human output is always trimmed and colourised before printing to the output stream.
func (s *StateMigrateHuman) log(preparedMessage string) {
	msg := s.view.colorize.Color(strings.TrimSpace(preparedMessage))
	s.view.streams.Println(msg)
}

// Implements Spacer
func (s *StateMigrateHuman) Spacer() {
	s.view.Spacer()
}

// Implements ProviderInstaller interface.
func (s *StateMigrateHuman) Output(code InitMessageCode, params ...any) {
	msg, ok := MessageRegistry[code]
	if !ok {
		panic("missing message for InstallingProviderMessage init message code")
	}
	s.Log(msg.HumanValue, params...)
}

// Implements ProviderInstaller interface.
func (s *StateMigrateHuman) LogInitializingStateStoreProviderPlugin(storeType string) {
	params := []any{storeType}
	msg := s.prepareMessage(InitializingStateStoreProviderPluginMessage, params...)
	s.log(msg)
}

// Implements ProviderInstaller interface.
func (s *StateMigrateHuman) LogFindingMatchingVersion(providerAddr addrs.Provider, versionConstraints getproviders.VersionConstraints) {
	params := []any{providerAddr.ForDisplay(), getproviders.VersionConstraintsString(versionConstraints)}
	msg := s.prepareMessage(FindingMatchingVersionMessage, params...)
	s.log(msg)
}

// Implements ProviderInstaller interface.
func (s *StateMigrateHuman) LogFindingLatestVersion(providerAddr addrs.Provider) {
	params := []any{providerAddr.ForDisplay()}
	msg := s.prepareMessage(FindingLatestVersionMessage, params...)
	s.log(msg)
}

// Implements ProviderInstaller interface.
func (s *StateMigrateHuman) LogProviderVersionAlreadyInstalled(providerAddr addrs.Provider, version getproviders.Version) {
	params := []any{providerAddr.ForDisplay(), version}
	msg := s.prepareMessage(ProviderAlreadyInstalledMessage, params...)
	s.log(msg)
}

// Implements ProviderInstaller interface.
func (s *StateMigrateHuman) LogUsingProviderVersionFromCacheDir(providerAddr addrs.Provider, version getproviders.Version) {
	params := []any{providerAddr.ForDisplay(), version}
	msg := s.prepareMessage(UsingProviderFromCacheDirInfo, params...)
	s.log(msg)
}

// Implements ProviderInstaller interface.
func (s *StateMigrateHuman) LogBuiltInProviderAvailable(providerAddr addrs.Provider) {
	params := []any{providerAddr.ForDisplay()}
	msg := s.prepareMessage(BuiltInProviderAvailableMessage, params...)
	s.log(msg)
}

// Implements ProviderInstaller interface.
func (s *StateMigrateHuman) LogInstallingProviderVersion(providerAddr addrs.Provider, version getproviders.Version) {
	params := []any{providerAddr.ForDisplay(), version}
	msg := s.prepareMessage(InstallingProviderMessage, params...)
	s.log(msg)
}

// Implements ProviderInstaller interface.
func (s *StateMigrateHuman) LogReusingPreviousProviderVersion(providerAddr addrs.Provider) {
	params := []any{providerAddr.ForDisplay()}
	msg := s.prepareMessage(ReusingPreviousVersionInfo, params...)
	s.log(msg)
}

// Implements ProviderInstaller interface.
func (s *StateMigrateHuman) LogProviderVersionSuccess(providerAddr addrs.Provider, version getproviders.Version, auth *getproviders.PackageAuthenticationResult) {
	params := []any{providerAddr.ForDisplay(), version, auth, ""} // add empty key id to the end
	msg := s.prepareMessage(InstalledProviderVersionInfo, params...)
	s.log(msg)
}

// Implements ProviderInstaller interface.
func (s *StateMigrateHuman) LogProviderVersionSuccessWithKeyID(providerAddr addrs.Provider, version getproviders.Version, auth *getproviders.PackageAuthenticationResult, keyID string) {
	keyDetails := fmt.Sprintf(", key ID [reset][bold]%s[reset]", keyID) // key id needs to be formatted for human output
	params := []any{providerAddr.ForDisplay(), version, auth, keyDetails}

	msg := s.prepareMessage(InstalledProviderVersionInfo, params...)
	s.log(msg)
}

// Implements ProviderInstaller interface.
func (s *StateMigrateHuman) LogPartnerAndCommunityProviders() {
	msg := s.prepareMessage(PartnerAndCommunityProvidersMessage)
	s.log(msg)
}

// Implements ProviderInstaller interface.
func (s *StateMigrateHuman) prepareMessage(code InitMessageCode, params ...any) string {
	message, ok := MessageRegistry[code]
	if !ok {
		panic("missing message for init message code " + string(code))
	}

	if message.HumanValue == "" {
		// no need to apply colorization if the message is empty
		return message.HumanValue
	}

	return s.view.colorize.Color(strings.TrimSpace(fmt.Sprintf(message.HumanValue, params...)))
}

var _ Spacer = (*StateMigrateJSON)(nil)

type StateMigrateJSON struct {
	view *JSONView
}

// Implements Spacer
func (s *StateMigrateJSON) Spacer() {
	// no-op for JSON output, since we don't want to log empty messages in JSON
}

// Plain logging of messages
// Logged data includes by default:
// @level as "info"
// @module as "terraform.ui" (See NewJSONView)
// @timestamp formatted in the default way
// type as "log".
//
// No additional fields supplied.
func (s *StateMigrateJSON) Log(message string, params ...any) {
	msg := strings.TrimSpace(fmt.Sprintf(message, params...))
	s.view.log.Info(msg)
}
