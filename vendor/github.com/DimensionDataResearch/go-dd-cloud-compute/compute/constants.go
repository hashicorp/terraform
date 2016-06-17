package compute

const (
	// ResourceStatusNormal indicates that a resource is active.
	ResourceStatusNormal = "NORMAL"

	// ResourceStatusPendingAdd indicates that an add operation is pending for the resource.
	ResourceStatusPendingAdd = "PENDING_ADD"

	// ResourceStatusPendingChange indicates that a change operation is pending for the resource.
	ResourceStatusPendingChange = "PENDING_CHANGE"

	// ResourceStatusPendingDelete indicates that a delete operation is pending for the resource.
	ResourceStatusPendingDelete = "PENDING_DELETE"
)
