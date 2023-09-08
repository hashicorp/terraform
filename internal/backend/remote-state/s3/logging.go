package s3

import (
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/terraform/internal/states/statemgr"
)

const (
	logKeyBucket = "tf_backend.s3.bucket"
	logKeyPath   = "tf_backend.s3.path"
)

const (
	logKeyBackendLockId        = "tf_backend.lock.id"
	logKeyBackendLockOperation = "tf_backend.lock.operation"
	logKeyBackendLockInfo      = "tf_backend.lock.info"
	logKeyBackendLockWho       = "tf_backend.lock.who"
	logKeyBackendLockVersion   = "tf_backend.lock.version"
	logKeyBackendLockPath      = "tf_backend.lock.path"
)

func logWithLockInfo(in hclog.Logger, info *statemgr.LockInfo) hclog.Logger {
	return in.With(
		logKeyBackendLockId, info.ID,
		logKeyBackendLockOperation, info.Operation,
		logKeyBackendLockInfo, info.Info,
		logKeyBackendLockWho, info.Who,
		logKeyBackendLockVersion, info.Version,
		logKeyBackendLockPath, info.Path,
	)
}

func logWithLockID(in hclog.Logger, id string) hclog.Logger {
	return in.With(
		logKeyBackendLockId, id,
	)
}
