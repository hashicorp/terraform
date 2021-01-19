package stressgen

import (
	"github.com/hashicorp/terraform/internal/stresstest/internal/stressaddr"
)

// ConfigSeries is the top-level type describing a test case. It consists of
// one or more configurations that the test harness must plan, apply, and then
// verify in order.
//
// ConfigSeries is intended to allow us to find defects that appear only when
// updating or replacing objects. If we limited all of our tests to only a
// single step then we would, in effect, be testing only "create" operations.
//
// Each ConfigSeries also has an implied final step of destroying everything
// that remains after the last explicit step. That destroy step runs against
// the most recently-applied configuration, so it will still have access to
// any provider configurations that are needed to destroy the final set of
// objects.
type ConfigSeries struct {
	// Addr is an identifier for this particular generated series, which
	// a caller can use to rebuild the same series as long as nothing
	// in the config generator code has changed in the meantime.
	Addr stressaddr.ConfigSeries

	Steps []*Config
}
