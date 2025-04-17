// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package oci

import (
	"sync"

	hclog "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-uuid"
	"github.com/hashicorp/terraform/internal/logging"
	"github.com/oracle/oci-go-sdk/v65/common"
)

var (
	loggerFunc = sync.OnceValue(func() hclog.Logger {
		l := logging.HCLogger()
		return l.Named("backend-oracle_oci")
	})
)

type backendLogger struct {
	hclog.Logger
}

func setSDKLogger() {
	sdklogger := NewBackendLogger(loggerFunc().With("component", "oci-go-sdk"))
	common.SetSDKLogger(sdklogger)
}
func NewBackendLogger(l hclog.Logger) backendLogger {
	return backendLogger{l}
}

// This fuction is needed for oci-go-sdk
func (l backendLogger) LogLevel() int {
	return int(l.Logger.GetLevel())
}
func (l backendLogger) Log(logLevel int, format string, v ...interface{}) error {
	l.Logger.Log(hclog.Level(logLevel), format, v...)
	return nil
}
func logWithOperation(operation string) hclog.Logger {
	log := loggerFunc().With(
		"operation", operation,
	)
	if id, err := uuid.GenerateUUID(); err == nil {
		log = log.With(
			"req_id", id,
		)

	}
	return log
}
