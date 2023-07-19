// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package json

type MessageType string

const (
	// Generic messages
	MessageVersion    MessageType = "version"
	MessageLog        MessageType = "log"
	MessageDiagnostic MessageType = "diagnostic"

	// Operation results
	MessageResourceDrift MessageType = "resource_drift"
	MessagePlannedChange MessageType = "planned_change"
	MessageChangeSummary MessageType = "change_summary"
	MessageOutputs       MessageType = "outputs"

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

	// Test messages
	MessageTestAbstract  MessageType = "test_abstract"
	MessageTestFile      MessageType = "test_file"
	MessageTestRun       MessageType = "test_run"
	MessageTestPlan      MessageType = "test_plan"
	MessageTestState     MessageType = "test_state"
	MessageTestSummary   MessageType = "test_summary"
	MessageTestCleanup   MessageType = "test_cleanup"
	MessageTestInterrupt MessageType = "test_interrupt"
)
