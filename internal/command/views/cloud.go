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
	msg, elapsed := retryLogMessage(attemptNum, resp, &v.lastRetry)
	// retryLogMessage returns an empty string for the first attempt or for rate-limited responses (HTTP 429)
	if msg != "" {
		if elapsed != nil {
			v.output(msg, elapsed) // subsequent retry message
		} else {
			v.output(msg)            // initial retry message
			v.view.streams.Println() // ensures a newline between messages
		}
	}
}

func (v *CloudHuman) output(messageCode CloudMessageCode, params ...any) {
	v.view.streams.Println(v.prepareMessage(messageCode, params...))
}

func (v *CloudHuman) prepareMessage(messageCode CloudMessageCode, params ...any) string {
	message, ok := CloudMessageRegistry[messageCode]
	if !ok {
		// display the message code as fallback if not found in the message registry
		return string(messageCode)
	}

	if message.HumanValue == "" {
		// no need to apply colorization if the message is empty
		return message.HumanValue
	}

	output := strings.TrimSpace(fmt.Sprintf(message.HumanValue, params...))
	if v.view.colorize != nil {
		return v.view.colorize.Color(output)
	}

	return output
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
	msg, elapsed := retryLogMessage(attemptNum, resp, &v.lastRetry)
	// retryLogMessage returns an empty string for the first attempt or for rate-limited responses (HTTP 429)
	if msg != "" {
		if elapsed != nil {
			v.output(msg, elapsed) // subsequent retry message
		} else {
			v.output(msg) // initial retry message
		}
	}
}

func (v *CloudJSON) output(messageCode CloudMessageCode, params ...any) {
	// don't add empty messages to json output
	preppedMessage := v.prepareMessage(messageCode, params...)
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

func (v *CloudJSON) prepareMessage(messageCode CloudMessageCode, params ...any) string {
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
		HumanValue: repeatedRetryError,
		JSONValue:  repeatedRetryErrorJSON,
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

const repeatedRetryError = `[reset][yellow]Still trying to restore the connection... (%s elapsed)[reset]`
const repeatedRetryErrorJSON = `Still trying to restore the connection... (%s elapsed)`

func retryLogMessage(attemptNum int, resp *http.Response, lastRetry *time.Time) (CloudMessageCode, *time.Duration) {
	// Skips logging for the first attempt or for rate-limited requests (HTTP 429)
	if attemptNum == 0 || (resp != nil && resp.StatusCode == 429) {
		*lastRetry = time.Now() // Update the retry timestamp for subsequent attempts
		return "", nil
	}

	// Logs initial retry message on the first retry attempt
	if attemptNum == 1 {
		return InitialRetryErrorMessage, nil
	}

	// Logs repeated retry message on subsequent attempts with elapsed time
	elapsed := time.Since(*lastRetry).Round(time.Second)
	return RepeatedRetryErrorMessage, &elapsed
}
