package command

import (
	"os"
	"strconv"

	"github.com/hashicorp/terraform/plans/planfile"
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

// PlanFile returns a reader for the plan file at the given path.
//
// If the return value and error are both nil, the given path exists but seems
// to be a configuration directory instead.
//
// Error will be non-nil if path refers to something which looks like a plan
// file and loading the file fails.
func (m *Meta) PlanFile(path string) (*planfile.Reader, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	if fi.IsDir() {
		// Looks like a configuration directory.
		return nil, nil
	}

	return planfile.Open(path)
}
