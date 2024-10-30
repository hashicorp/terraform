// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform/internal/terraform"
)

type RetryLogHook struct {
	terraform.NilHook

	view *View

	lastRetry time.Time
}

var _ terraform.Hook = (*UiHook)(nil)

func NewRetryLoghook(view *View) *RetryLogHook {
	return &RetryLogHook{
		view: view,
	}
}

// RetryLogHook returns a string providing an update about a request from the
// client being retried.
//
// If colorize is true, then the value returned by this function should be
// processed by a colorizer.
func (hook *RetryLogHook) RetryLogHook(attemptNum int, resp *http.Response, colorize bool) string {
	// Ignore the first retry to make sure any delayed output will
	// be written to the console before we start logging retries.
	//
	// The retry logic in the TFE client will retry both rate limited
	// requests and server errors, but in the cloud backend we only
	// care about server errors so we ignore rate limit (429) errors.
	if attemptNum == 0 || (resp != nil && resp.StatusCode == 429) {
		hook.lastRetry = time.Now()
		return ""
	}

	var msg string
	if attemptNum == 1 {
		msg = initialRetryError
	} else {
		msg = fmt.Sprintf(repeatedRetryError, time.Since(hook.lastRetry).Round(time.Second))
	}

	if colorize {
		return strings.TrimSpace(fmt.Sprintf("[reset][yellow]%s[reset]", msg))
	}
	return hook.view.colorize.Color(strings.TrimSpace(msg))
}

// The newline in this error is to make it look good in the CLI!
const initialRetryError = `
There was an error connecting to HCP Terraform. Please do not exit
Terraform to prevent data loss! Trying to restore the connection...
`

const repeatedRetryError = "Still trying to restore the connection... (%s elapsed)"
