// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package moduletest

import (
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type CommandMode int

const (
	// NormalMode is the default mode for running terraform test.
	NormalMode CommandMode = iota
	// CleanupMode is used when running terraform test cleanup.
	// In this mode, the graph will be built with the intention of cleaning up
	// the state, rather than applying changes.
	CleanupMode
)

type Suite struct {
	Status      Status
	CommandMode CommandMode

	Files map[string]*File
}

type TestSuiteRunner interface {
	Test(experimentsAllowed bool) (Status, tfdiags.Diagnostics)
	Stop()
	Cancel()

	// IsStopped allows code outside the moduletest package to confirm the suite was stopped
	// when handling a graceful exit scenario
	IsStopped() bool
}
