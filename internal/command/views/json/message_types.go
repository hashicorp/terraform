// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package json

type MessageType string

const (
	// Generic messages
	MessageVersion    MessageType = "version"
	MessageLog        MessageType = "log"
	MessageDiagnostic MessageType = "diagnostic"

	// Operation results
	MessageResourceDrift           MessageType = "resource_drift"
	MessagePlannedChange           MessageType = "planned_change"
	MessageDeferredChange          MessageType = "deferred_change"
	MessagePlannedActionInvocation MessageType = "planned_action_invocation"
	MessageChangeSummary           MessageType = "change_summary"
	MessageOutputs                 MessageType = "outputs"

	// Hook-driven messages
	MessageApplyStart        MessageType = "apply_start"
	MessageApplyProgress     MessageType = "apply_progress"
	MessageApplyComplete     MessageType = "apply_complete"
	MessageApplyErrored      MessageType = "apply_errored"
	MessageProvisionStart    MessageType = "provision_start"
	MessageProvisionProgress MessageType = "provision_progress"
	MessageProvisionComplete MessageType = "provision_complete"
	MessageProvisionErrored  MessageType = "provision_errored"
	MessageRefreshStart      MessageType = "refresh_start"
	MessageRefreshComplete   MessageType = "refresh_complete"

	// Ephemeral operation messages
	MessageEphemeralOpStart    MessageType = "ephemeral_op_start"
	MessageEphemeralOpProgress MessageType = "ephemeral_op_progress"
	MessageEphemeralOpComplete MessageType = "ephemeral_op_complete"
	MessageEphemeralOpErrored  MessageType = "ephemeral_op_errored"

	// Test messages
	MessageTestAbstract  MessageType = "test_abstract"
	MessageTestFile      MessageType = "test_file"
	MessageTestRun       MessageType = "test_run"
	MessageTestPlan      MessageType = "test_plan"
	MessageTestState     MessageType = "test_state"
	MessageTestSummary   MessageType = "test_summary"
	MessageTestCleanup   MessageType = "test_cleanup"
	MessageTestInterrupt MessageType = "test_interrupt"
	MessageTestStatus    MessageType = "test_status"
	MessageTestRetry     MessageType = "test_retry"

	// List messages
	MessageListStart         MessageType = "list_start"
	MessageListResourceFound MessageType = "list_resource_found"
	MessageListComplete      MessageType = "list_complete"

	// Action messages
	MessageActionStart    MessageType = "action_start"
	MessageActionProgress MessageType = "action_progress"
	MessageActionComplete MessageType = "action_complete"
	MessageActionErrored  MessageType = "action_errored"

	// Policy messages
	MessagePolicyInfo             MessageType = "policy_info"
	MessagePolicyDiagnostic       MessageType = "policy_diagnostic"
	MessagePolicyEvaluationResult MessageType = "policy_result"
	MessagePolicyQuerySummary     MessageType = "policy_query_summary"

	// Provider installation messages
	LogProviderInstallationStart           MessageType = "provider_installation_start"
	LogStateStoreProviderInstallationStart MessageType = "state_store_provider_installation_start"
	LogProviderQueryUsePreviousVersion     MessageType = "provider_query_use_previous_version"
	LogProviderQueryUseConstraints         MessageType = "provider_query_use_constraints"
	LogProviderQueryUseLatest              MessageType = "provider_query_use_latest"
	LogProviderVersionAlreadyInstalled     MessageType = "provider_version_already_installed"
	LogProviderVersionFoundInCacheDir      MessageType = "provider_version_found_in_cache_dir"
	LogProviderVersionInstallationStart    MessageType = "provider_version_installation_start"
	LogProviderVersionInstallationComplete MessageType = "provider_version_installation_complete"
	LogBuiltInProviderAvailable            MessageType = "built_in_provider_available"
	LogThirdPartyProvidersInstalled        MessageType = "third_party_providers_installed"

	// Dependency lock file messages
	// Uses provider-oriented language in case we add a module lock file in future.
	LogProviderLockfileCreated MessageType = "provider_lockfile_created"
	LogProviderLockfileUpdated MessageType = "provider_lockfile_updated"

	// Provider trust-related message (currently used only in PSS context)
	LogProviderInteractiveApproval  MessageType = "provider_interactive_approval"
	LogProviderInteractiveRejection MessageType = "provider_interactive_rejection"
	LogProviderAutomaticApproval    MessageType = "provider_automatic_approval"

	// State migration-related messages
	LogMigrationStart                             MessageType = "migration_start"
	LogMigrationComplete                          MessageType = "migration_complete"
	LogMigrationErrored                           MessageType = "migration_errored"
	LogMigrationFinalized                         MessageType = "migration_finalized"
	LogMigrationSourceInitializationStart         MessageType = "migration_source_initialization_start"
	LogMigrationSourceInitializationComplete      MessageType = "migration_source_initialization_complete"
	LogMigrationDestinationInitializationStart    MessageType = "migration_destination_initialization_start"
	LogMigrationDestinationInitializationComplete MessageType = "migration_destination_initialization_complete"

	// Init-specific messages
	//
	// NOTE: These are not used to set the `type` field in the JSON output from the init command.
	// Instead, the init command's JSON output was implemented so that some messages are logged with
	// `"type": "init_output"` and a `message_code` field that takes the const values below.
	// In a future major version we should make init's JSON output align with the conventions used
	// elsewhere in the CLI. For now these consts are here to demonstrate that they're public-facing
	// and changes are potentially breaking.
	CopyingConfigurationMessage       MessageType = "copying_configuration_message"
)
