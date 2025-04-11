// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package oci

import (
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/terraform/internal/logging"
)

var (
	logger = NewBackendLogger(logging.NewLogger("tf-backend-oci"))
)

type backendLogger struct {
	hclog.Logger
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
