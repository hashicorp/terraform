// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package oci

import (
	"net"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
)

var (
	LongRetryTime   = 10 * time.Minute
	RetryableStatus = map[int]bool{
		429: true, // Too Many Requests
		500: true, // Internal Server Error
		503: true, // Service Unavailable
	}
)

func getDefaultRetryPolicy() *common.RetryPolicy {
	startTime := time.Now()
	return &common.RetryPolicy{
		MaximumNumberAttempts: 5,
		ShouldRetryOperation: func(response common.OCIOperationResponse) bool {
			return shouldRetry(response, startTime)
		},
		NextDuration: func(response common.OCIOperationResponse) time.Duration {
			return getRetryBackoffDuration(response, startTime)
		},
	}
}

func shouldRetry(response common.OCIOperationResponse, startTime time.Time) bool {
	if elapsed := time.Since(startTime); elapsed > LongRetryTime {
		return false
	}
	if response.Response == nil || response.Response.HTTPResponse() == nil {
		return false
	}

	statusCode := response.Response.HTTPResponse().StatusCode
	if RetryableStatus[statusCode] {
		return true
	}
	return response.Error != nil && isNetworkError(response.Error)
}

func getRetryBackoffDuration(response common.OCIOperationResponse, startTime time.Time) time.Duration {
	attempt := response.AttemptNumber
	if attempt > 5 {
		attempt = 5
	}
	return time.Duration(2*attempt*attempt) * time.Second
}

func isNetworkError(err error) bool {
	if netErr, ok := err.(net.Error); ok && (netErr.Timeout() || netErr.Temporary()) {
		return true
	}
	return strings.Contains(err.Error(), "i/o timeout")
}
