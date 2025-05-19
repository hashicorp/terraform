// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

// CloudHooks provides functions that help with integrating directly into
// the go-tfe tfe.Client struct.
type CloudHooks struct {
	// lastRetry is set to the last time a request was retried.
	lastRetry time.Time
}

// RetryLogHook returns a string providing an update about a request from the
// client being retried.
//
// If colorize is true, then the value returned by this function should be
// processed by a colorizer.
func (hooks *CloudHooks) RetryLogHook(attemptNum int, resp *http.Response, colorize bool) string {
	// Ignore the first retry to make sure any delayed output will
	// be written to the console before we start logging retries.
	//
	// The retry logic in the TFE client will retry both rate limited
	// requests and server errors, but in the cloud backend we only
	// care about server errors so we ignore rate limit (429) errors.
	if attemptNum == 0 || (resp != nil && resp.StatusCode == 429) {
		hooks.lastRetry = time.Now()
		return ""
	}

	var msg string
	if attemptNum == 1 {
		msg = initialRetryError
	} else {
		msg = fmt.Sprintf(repeatedRetryError, time.Since(hooks.lastRetry).Round(time.Second))
	}

	if colorize {
		return strings.TrimSpace(fmt.Sprintf("[reset][yellow]%s[reset]", msg))
	}
	return strings.TrimSpace(msg)
}

// The newline in this error is to make it look good in the CLI!
const initialRetryError = `
There was an error connecting to HCP Terraform. Please do not exit
Terraform to prevent data loss! Trying to restore the connection...
`

const repeatedRetryError = "Still trying to restore the connection... (%s elapsed)"
