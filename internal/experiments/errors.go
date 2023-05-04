// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package experiments

import (
	"fmt"
)

// UnavailableError is the error type returned by GetCurrent when the requested
// experiment is not recognized at all.
type UnavailableError struct {
	ExperimentName string
}

func (e UnavailableError) Error() string {
	return fmt.Sprintf("no current experiment is named %q", e.ExperimentName)
}

// ConcludedError is the error type returned by GetCurrent when the requested
// experiment is recognized as concluded.
type ConcludedError struct {
	ExperimentName string
	Message        string
}

func (e ConcludedError) Error() string {
	return fmt.Sprintf("experiment %q has concluded: %s", e.ExperimentName, e.Message)
}
