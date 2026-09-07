// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

type StateStoreProviderTrustLogger interface {
	LogInteractiveApproval()
	LogInteractiveRejection()
	LogAutomaticApproval()

	Spacer
}

// Human-readable messages, one version for human-readable output and the 'json' version for the
// human-readable @message field in the JSON output.
const (
	logInteractiveApprovalMessageHuman = "[reset][bold]The state store provider was approved by the user."
	logInteractiveApprovalMessageJSON  = "The state store provider was approved by the user."

	logInteractiveRejectionMessageHuman = "[reset][bold]The state store provider was rejected by the user."
	logInteractiveRejectionMessageJSON  = "The state store provider was rejected by the user."

	logInteractiveAutomaticApprovalMessageHuman = "[reset][bold]The state store provider was approved automatically."
	logInteractiveAutomaticApprovalMessageJSON  = "The state store provider was approved automatically."
)
