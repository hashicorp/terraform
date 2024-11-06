// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// The Cloud view is used for operations that are specific to cloud operations.
type Cloud interface {
	RetryLog(attemptNum int, resp *http.Response)
	Diagnostics(diags tfdiags.Diagnostics)
	Output(messageCode CloudMessageCode, params ...any)
	PrepareMessage(messageCode CloudMessageCode, params ...any) string
}

// NewCloud returns Cloud implementation for the given ViewType.
func NewCloud(vt arguments.ViewType, view *View) Cloud {
	switch vt {
	case arguments.ViewJSON:
		return &CloudJSON{
			view: NewJSONView(view),
		}
	case arguments.ViewHuman:
		return &CloudHuman{
			view: view,
		}
	default:
		panic(fmt.Sprintf("unknown view type %v", vt))
	}
}

// The CloudHuman implementation renders human-readable text logs, suitable for
// a scrolling terminal.
type CloudHuman struct {
	view *View

	lastRetry time.Time
}

var _ Cloud = (*CloudHuman)(nil)

func (v *CloudHuman) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}

func (v *CloudHuman) RetryLog(attemptNum int, resp *http.Response) {
	msg := retryLogMessage(attemptNum, resp, &v.lastRetry)
	// retryLogMessage returns an empty string for the first attempt or rate-limited responses
	if msg != "" {
		v.Output(msg)
	}
}

func (v *CloudHuman) Output(messageCode CloudMessageCode, params ...any) {
	v.view.streams.Println(v.PrepareMessage(messageCode, params...))
}

func (v *CloudHuman) PrepareMessage(messageCode CloudMessageCode, params ...any) string {
	message, ok := CloudMessageRegistry[messageCode]
	if !ok {
		// display the message code as fallback if not found in the message registry
		return string(messageCode)
	}

	if message.HumanValue == "" {
		// no need to apply colorization if the message is empty
		return message.HumanValue
	}

	return v.view.colorize.Color(strings.TrimSpace(fmt.Sprintf(message.HumanValue, params...)))
}

// The CloudJSON implementation renders streaming JSON logs, suitable for
// integrating with other software.
type CloudJSON struct {
	view *JSONView

	lastRetry time.Time
}

var _ Cloud = (*CloudJSON)(nil)

func (v *CloudJSON) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}

func (v *CloudJSON) RetryLog(attemptNum int, resp *http.Response) {
	msg := retryLogMessage(attemptNum, resp, &v.lastRetry)
	// retryLogMessage returns an empty string for the first attempt or rate-limited responses
	if msg != "" {
		v.Output(msg)
	}
}

func (v *CloudJSON) Output(messageCode CloudMessageCode, params ...any) {
	// don't add empty messages to json output
	preppedMessage := v.PrepareMessage(messageCode, params...)
	if preppedMessage == "" {
		return
	}

	current_timestamp := time.Now().UTC().Format(time.RFC3339)
	json_data := map[string]string{
		"@level":       "info",
		"@message":     preppedMessage,
		"@module":      "terraform.ui",
		"@timestamp":   current_timestamp,
		"type":         "cloud_output",
		"message_code": string(messageCode),
	}

	cloud_output, _ := json.Marshal(json_data)
	v.view.view.streams.Println(string(cloud_output))
}

func (v *CloudJSON) PrepareMessage(messageCode CloudMessageCode, params ...any) string {
	message, ok := CloudMessageRegistry[messageCode]
	if !ok {
		// display the message code as fallback if not found in the message registry
		return string(messageCode)
	}

	return strings.TrimSpace(fmt.Sprintf(message.JSONValue, params...))
}

// CloudMessage represents a message string in both json and human decorated text format.
type CloudMessage struct {
	HumanValue string
	JSONValue  string
}

var CloudMessageRegistry map[CloudMessageCode]CloudMessage = map[CloudMessageCode]CloudMessage{
	"initial_retry_error_message": {
		HumanValue: initialRetryError,
		JSONValue:  initialRetryErrorJSON,
	},
	"repeated_retry_error_message": {
		HumanValue: repeatdRetryError,
		JSONValue:  repeatdRetryErrorJSON,
	},
}

type CloudMessageCode string

const (
	InitialRetryErrorMessage  CloudMessageCode = "initial_retry_error_message"
	RepeatedRetryErrorMessage CloudMessageCode = "repeated_retry_error_message"
)

const initialRetryError = `[reset][yellow]
There was an error connecting to HCP Terraform. Please do not exit
Terraform to prevent data loss! Trying to restore the connection...[reset]
`
const initialRetryErrorJSON = `
There was an error connecting to HCP Terraform. Please do not exit
Terraform to prevent data loss! Trying to restore the connection...
`

const repeatdRetryError = `[reset][yellow]Still trying to restore the connection... (%s elapsed)[reset]`
const repeatdRetryErrorJSON = `Still trying to restore the connection... (%s elapsed)`

// retryLogMessage determines the appropriate retry message based on the attempt number
// and the response status, and updates the `lastRetry` timestamp. It returns a
// message code that indicates whether a retry message should be logged.
//
// The `RetryLog` function uses the returned message to decide if the retry message
// should be logged, based on the retry attempt and response status.
//
// The function:
// - Skips logging for the first retry or if the response indicates a rate-limited request (HTTP 429).
// - Logs an initial retry message on the first retry attempt.
// - Logs a repeated retry message on subsequent attempts, including the elapsed time since the last retry.
func retryLogMessage(attemptNum int, resp *http.Response, lastRetry *time.Time) CloudMessageCode {
	// Ignore the first retry to ensure any delayed output is written before logging retries.
	// Skip rate-limited requests (HTTP 429), which are handled differently.
	if attemptNum == 0 || (resp != nil && resp.StatusCode == 429) {
		*lastRetry = time.Now() // Update the retry timestamp
		return ""               // No message to log
	}

	// Return the initial retry message on the first retry attempt
	if attemptNum == 1 {
		return InitialRetryErrorMessage
	}

	// Return a repeated retry message with the time since the last retry for subsequent attempts
	return CloudMessageCode(fmt.Sprint(RepeatedRetryErrorMessage, time.Since(*lastRetry).Round(time.Second)))
}
