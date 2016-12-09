// experiment package contains helper functions for tracking experimental
// features throughout Terraform.
//
// This package should be used for creating, enabling, querying, and deleting
// experimental features. By unifying all of that onto a single interface,
// we can have the Go compiler help us by enforcing every place we touch
// an experimental feature.
//
// To create a new experiment:
//
//   1. Add the experiment to the global vars list below, prefixed with X_
//
//   2. Add the experiment variable to the All listin the init() function
//
//   3. Use it!
//
// To remove an experiment:
//
//   1. Delete the experiment global var.
//
//   2. Try to compile and fix all the places where the var was referenced.
//
// To use an experiment:
//
//   1. Use Flag() if you want the experiment to be available from the CLI.
//
//   2. Use Enabled() to check whether it is enabled.
//
// As a general user:
//
//   1. The `-Xexperiment-name` flag
//   2. The `TF_X_<experiment-name>` env var.
//   3. The `TF_X_FORCE` env var can be set to force an experimental feature
//      without human verifications.
//
package experiment

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
)

// The experiments that are available are listed below. Any package in
// Terraform defining an experiment should define the experiments below.
// By keeping them all within the experiment package we force a single point
// of definition and use. This allows the compiler to enforce references
// so it becomes easy to remove the features.
var (
	// Reuse the old graphs from TF 0.7.x. These will be removed at some point.
	X_legacyGraph = newBasicID("legacy-graph", "LEGACY_GRAPH", false)

	// Shadow graph. This is already on by default. Disabling it will be
	// allowed for awhile in order for it to not block operations.
	X_shadow = newBasicID("shadow", "SHADOW", true)
)

// Global variables this package uses because we are a package
// with global state.
var (
	// all is the list of all experiements. Do not modify this.
	All []ID

	// enabled keeps track of what flags have been enabled
	enabled     map[string]bool
	enabledLock sync.Mutex

	// Hidden "experiment" that forces all others to be on without verification
	x_force = newBasicID("force", "FORCE", false)
)

func init() {
	// The list of all experiments, update this when an experiment is added.
	All = []ID{
		X_legacyGraph,
		X_shadow,
		x_force,
	}

	// Load
	reload()
}

// reload is used by tests to reload the global state. This is called by
// init publicly.
func reload() {
	// Initialize
	enabledLock.Lock()
	enabled = make(map[string]bool)
	enabledLock.Unlock()

	// Set defaults and check env vars
	for _, id := range All {
		// Get the default value
		def := id.Default()

		// If we set it in the env var, default it to true
		key := fmt.Sprintf("TF_X_%s", strings.ToUpper(id.Env()))
		if v := os.Getenv(key); v != "" {
			def = v != "0"
		}

		// Set the default
		SetEnabled(id, def)
	}
}

// Enabled returns whether an experiment has been enabled or not.
func Enabled(id ID) bool {
	enabledLock.Lock()
	defer enabledLock.Unlock()
	return enabled[id.Flag()]
}

// SetEnabled sets an experiment to enabled/disabled. Please check with
// the experiment docs for when calling this actually affects the experiment.
func SetEnabled(id ID, v bool) {
	enabledLock.Lock()
	defer enabledLock.Unlock()
	enabled[id.Flag()] = v
}

// Force returns true if the -Xforce of TF_X_FORCE flag is present, which
// advises users of this package to not verify with the user that they want
// experimental behavior and to just continue with it.
func Force() bool {
	return Enabled(x_force)
}

// Flag configures the given FlagSet with the flags to configure
// all active experiments.
func Flag(fs *flag.FlagSet) {
	for _, id := range All {
		desc := id.Flag()
		key := fmt.Sprintf("X%s", id.Flag())
		fs.Var(&idValue{X: id}, key, desc)
	}
}

// idValue implements flag.Value for setting the enabled/disabled state
// of an experiment from the CLI.
type idValue struct {
	X ID
}

func (v *idValue) IsBoolFlag() bool { return true }
func (v *idValue) String() string   { return strconv.FormatBool(Enabled(v.X)) }
func (v *idValue) Set(raw string) error {
	b, err := strconv.ParseBool(raw)
	if err == nil {
		SetEnabled(v.X, b)
	}

	return err
}
