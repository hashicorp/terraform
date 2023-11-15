// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package dynrpcserver

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const unavailableMsg = "must call Setup.Handshake first"

var unavailableErr error = status.Error(codes.Unavailable, unavailableMsg)
