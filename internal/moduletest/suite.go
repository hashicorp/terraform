// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package moduletest

import "github.com/hashicorp/terraform/internal/tfdiags"

type Suite struct {
	Status Status

	Files map[string]*File
}

type TestSuiteRunner interface {
	Test() (Status, tfdiags.Diagnostics)
	Stop()
	Cancel()

	// IsStopped allows code outside the moduletest package to confirm the suite was stopped
	// when handling a graceful exit scenario
	IsStopped() bool
}
