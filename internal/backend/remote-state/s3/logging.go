// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package s3

import (
	"sync"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-uuid"
	"github.com/hashicorp/terraform/internal/logging"
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

const (
	logKeyBackendOperation       = "tf_backend.operation"
	logKeyBackendRequestId       = "tf_backend.req_id" // Using "req_id" to match pattern for provider logging
	logKeyBackendWorkspace       = "tf_backend.workspace"
	logKeyBackendWorkspacePrefix = "tf_backend.workspace-prefix"
)

const (
	// operationBackendConfigSchema    = "ConfigSchema"
	// operationBackendPrepareConfig   = "PrepareConfig"
	operationBackendConfigure = "Configure"
	// operationBackendStateMgr        = "StateMgr"
	operationBackendDeleteWorkspace = "DeleteWorkspace"
	operationBackendWorkspaces      = "Workspaces"
)

const (
	operationClientGet    = "Get"
	operationClientPut    = "Put"
	operationClientDelete = "Delete"
)

const (
	operationLockerLock   = "Lock"
	operationLockerUnlock = "Unlock"
)

var logger = sync.OnceValue(func() hclog.Logger {
	l := logging.HCLogger()
	return l.Named("backend-s3")
})

func logWithOperation(in hclog.Logger, operation string) hclog.Logger {
	log := in.With(
		logKeyBackendOperation, operation,
	)
	if id, err := uuid.GenerateUUID(); err == nil {
		log = log.With(
			logKeyBackendRequestId, id,
		)

	}
	return log
}

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
