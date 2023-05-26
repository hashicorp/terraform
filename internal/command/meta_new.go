// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package command

import (
	"os"
	"strconv"

	"github.com/hashicorp/terraform/internal/plans/planfile"
)

// NOTE: Temporary file until this branch is cleaned up.

// Input returns whether or not input asking is enabled.
func (m *Meta) Input() bool {
	if test || !m.input {
		return false
	}

	if envVar := os.Getenv(InputModeEnvVar); envVar != "" {
		if v, err := strconv.ParseBool(envVar); err == nil && !v {
			return false
		}
	}

	return true
}

// PlanFile loads the plan file at the given path, which might be either a local
// or cloud plan.
//
// If the return value and error are both nil, the given path exists but seems
// to be a configuration directory instead.
//
// Error will be non-nil if path refers to something which looks like a plan
// file and loading the file fails.
func (m *Meta) PlanFile(path string) (*planfile.WrappedPlanFile, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	if fi.IsDir() {
		// Looks like a configuration directory.
		return nil, nil
	}

	return planfile.OpenWrapped(path)
}
