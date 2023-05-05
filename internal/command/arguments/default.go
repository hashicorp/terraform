// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package arguments

import (
	"flag"
	"io/ioutil"
)

// defaultFlagSet creates a FlagSet with the common settings to override
// the flag package's noisy defaults.
func defaultFlagSet(name string) *flag.FlagSet {
	f := flag.NewFlagSet(name, flag.ContinueOnError)
	f.SetOutput(ioutil.Discard)
	f.Usage = func() {}

	return f
}
