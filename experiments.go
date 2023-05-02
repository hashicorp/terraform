// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

// experimentsAllowed can be set to any non-empty string using Go linker
// arguments in order to enable the use of experimental features for a
// particular Terraform build:
//
//	go install -ldflags="-X 'main.experimentsAllowed=yes'"
//
// By default this variable is initialized as empty, in which case
// experimental features are not available.
//
// The Terraform release process should arrange for this variable to be
// set for alpha releases and development snapshots, but _not_ for
// betas, release candidates, or final releases.
//
// (NOTE: Some experimental features predate the rule that experiments
// are available only for alpha/dev builds, and so intentionally do not
// make use of this setting to avoid retracting a previously-documented
// open experiment.)
var experimentsAllowed string

func ExperimentsAllowed() bool {
	return experimentsAllowed != ""
}
