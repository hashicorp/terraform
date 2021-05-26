package json

type MessageType string

const (
	// Generic messages
	MessageVersion    MessageType = "version"
	MessageLog        MessageType = "log"
	MessageDiagnostic MessageType = "diagnostic"

	// Operation results
	MessagePlannedChange MessageType = "planned_change"
	MessageChangeSummary MessageType = "change_summary"
	MessageDriftSummary  MessageType = "drift_summary"
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
)
