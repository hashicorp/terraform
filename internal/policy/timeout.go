// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package policy

import "time"

const (
	// DefaultPerCallTimeout is the default maximum duration for a single
	// EvaluateResource RPC. If a single evaluation exceeds this duration,
	// the resource result is recorded as error and evaluation continues.
	DefaultPerCallTimeout = 30 * time.Second

	// DefaultOverallDeadline is the default maximum wall-clock duration for
	// an entire policy evaluation pass (e.g., all query resources). When this
	// deadline is reached, all remaining unevaluated resources are recorded as
	// error and the pass terminates.
	DefaultOverallDeadline = 10 * time.Minute
)

// EvalTimeouts holds the configurable timeout durations for policy evaluation.
// These timeouts protect against runaway policy evaluations.
type EvalTimeouts struct {
	// PerCall is the maximum duration for a single EvaluateResource RPC.
	// If zero, no per-call timeout is applied.
	PerCall time.Duration

	// Overall is the maximum wall-clock duration for an entire policy pass.
	// If zero, no overall deadline is applied.
	Overall time.Duration
}

// DefaultEvalTimeouts returns the production default timeout values.
func DefaultEvalTimeouts() EvalTimeouts {
	return EvalTimeouts{
		PerCall: DefaultPerCallTimeout,
		Overall: DefaultOverallDeadline,
	}
}
