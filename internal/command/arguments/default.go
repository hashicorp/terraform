// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"flag"
	"io"
)

// defaultFlagSet creates a FlagSet with the common settings to override
// the flag package's noisy defaults.
func defaultFlagSet(name string) *flag.FlagSet {
	f := flag.NewFlagSet(name, flag.ContinueOnError)
	f.SetOutput(io.Discard)
	f.Usage = func() {}

	return f
}
