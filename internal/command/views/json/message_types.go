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

	// Provider installation messages
	InitializingStateStoreProviderStart MessageType = "state_store_provider_initialization_start"

	// PSS messages
	StateStoreProviderInteractiveApproval  MessageType = "state_store_provider_interactive_approval"
	StateStoreProviderInteractiveRejection MessageType = "state_store_provider_interactive_rejection"
	StateStoreProviderAutomaticApproval    MessageType = "state_store_provider_automatic_approval"

	// State Migration messages (`state migrate` command)
	StateMigrationStart                             MessageType = "migration_start"
	StateMigrationComplete                          MessageType = "migration_complete"
	StateMigrationErrored                           MessageType = "migration_errored"
	StateMigrationFinalized                         MessageType = "migration_finalized"
	StateMigrationSourceInitializationStart         MessageType = "migration_source_initialization_start"
	StateMigrationSourceInitializationComplete      MessageType = "migration_source_initialization_complete"
	StateMigrationDestinationInitializationStart    MessageType = "migration_destination_initialization_start"
	StateMigrationDestinationInitializationComplete MessageType = "migration_destination_initialization_complete"
)
