// Code generated by private/model/cli/gen-api/main.go. DO NOT EDIT.

package dynamodb

const (

	// ErrCodeBackupInUseException for service response error code
	// "BackupInUseException".
	//
	// There is another ongoing conflicting backup control plane operation on the
	// table. The backups is either being created, deleted or restored to a table.
	ErrCodeBackupInUseException = "BackupInUseException"

	// ErrCodeBackupNotFoundException for service response error code
	// "BackupNotFoundException".
	//
	// Backup not found for the given BackupARN.
	ErrCodeBackupNotFoundException = "BackupNotFoundException"

	// ErrCodeConditionalCheckFailedException for service response error code
	// "ConditionalCheckFailedException".
	//
	// A condition specified in the operation could not be evaluated.
	ErrCodeConditionalCheckFailedException = "ConditionalCheckFailedException"

	// ErrCodeContinuousBackupsUnavailableException for service response error code
	// "ContinuousBackupsUnavailableException".
	//
	// Backups have not yet been enabled for this table.
	ErrCodeContinuousBackupsUnavailableException = "ContinuousBackupsUnavailableException"

	// ErrCodeGlobalTableAlreadyExistsException for service response error code
	// "GlobalTableAlreadyExistsException".
	//
	// The specified global table already exists.
	ErrCodeGlobalTableAlreadyExistsException = "GlobalTableAlreadyExistsException"

	// ErrCodeGlobalTableNotFoundException for service response error code
	// "GlobalTableNotFoundException".
	//
	// The specified global table does not exist.
	ErrCodeGlobalTableNotFoundException = "GlobalTableNotFoundException"

	// ErrCodeInternalServerError for service response error code
	// "InternalServerError".
	//
	// An error occurred on the server side.
	ErrCodeInternalServerError = "InternalServerError"

	// ErrCodeItemCollectionSizeLimitExceededException for service response error code
	// "ItemCollectionSizeLimitExceededException".
	//
	// An item collection is too large. This exception is only returned for tables
	// that have one or more local secondary indexes.
	ErrCodeItemCollectionSizeLimitExceededException = "ItemCollectionSizeLimitExceededException"

	// ErrCodeLimitExceededException for service response error code
	// "LimitExceededException".
	//
	// The number of concurrent table requests (cumulative number of tables in the
	// CREATING, DELETING or UPDATING state) exceeds the maximum allowed of 10.
	//
	// Also, for tables with secondary indexes, only one of those tables can be
	// in the CREATING state at any point in time. Do not attempt to create more
	// than one such table simultaneously.
	//
	// The total limit of tables in the ACTIVE state is 250.
	ErrCodeLimitExceededException = "LimitExceededException"

	// ErrCodeProvisionedThroughputExceededException for service response error code
	// "ProvisionedThroughputExceededException".
	//
	// Your request rate is too high. The AWS SDKs for DynamoDB automatically retry
	// requests that receive this exception. Your request is eventually successful,
	// unless your retry queue is too large to finish. Reduce the frequency of requests
	// and use exponential backoff. For more information, go to Error Retries and
	// Exponential Backoff (http://docs.aws.amazon.com/amazondynamodb/latest/developerguide/Programming.Errors.html#Programming.Errors.RetryAndBackoff)
	// in the Amazon DynamoDB Developer Guide.
	ErrCodeProvisionedThroughputExceededException = "ProvisionedThroughputExceededException"

	// ErrCodeReplicaAlreadyExistsException for service response error code
	// "ReplicaAlreadyExistsException".
	//
	// The specified replica is already part of the global table.
	ErrCodeReplicaAlreadyExistsException = "ReplicaAlreadyExistsException"

	// ErrCodeReplicaNotFoundException for service response error code
	// "ReplicaNotFoundException".
	//
	// The specified replica is no longer part of the global table.
	ErrCodeReplicaNotFoundException = "ReplicaNotFoundException"

	// ErrCodeResourceInUseException for service response error code
	// "ResourceInUseException".
	//
	// The operation conflicts with the resource's availability. For example, you
	// attempted to recreate an existing table, or tried to delete a table currently
	// in the CREATING state.
	ErrCodeResourceInUseException = "ResourceInUseException"

	// ErrCodeResourceNotFoundException for service response error code
	// "ResourceNotFoundException".
	//
	// The operation tried to access a nonexistent table or index. The resource
	// might not be specified correctly, or its status might not be ACTIVE.
	ErrCodeResourceNotFoundException = "ResourceNotFoundException"

	// ErrCodeTableAlreadyExistsException for service response error code
	// "TableAlreadyExistsException".
	//
	// A table with the name already exists.
	ErrCodeTableAlreadyExistsException = "TableAlreadyExistsException"

	// ErrCodeTableInUseException for service response error code
	// "TableInUseException".
	//
	// A table by that name is either being created or deleted.
	ErrCodeTableInUseException = "TableInUseException"

	// ErrCodeTableNotFoundException for service response error code
	// "TableNotFoundException".
	//
	// A table with the name TableName does not currently exist within the subscriber's
	// account.
	ErrCodeTableNotFoundException = "TableNotFoundException"
)
