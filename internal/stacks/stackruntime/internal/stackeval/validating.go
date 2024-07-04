// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"context"

	"github.com/hashicorp/terraform/internal/depsfile"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type ValidateOpts struct {
	ProviderFactories ProviderFactories
	DependencyLocks   depsfile.Locks
}

// Validateable is implemented by objects that can participate in validation.
type Validatable interface {
	// Validate returns diagnostics for any part of the reciever which
	// has an invalid configuration.
	//
	// Validate implementations should be shallow, which is to say that
	// in particular they _must not_ call the Validate method of other
	// objects that implement Validatable, and should also think very
	// hard about calling any validation-related methods of other objects,
	// so as to avoid generating duplicate diagnostics via two different
	// return paths.
	//
	// In general, assume that _all_ objects that implement Validatable will
	// have their Validate methods called at some point during validation, and
	// so it's unnecessary and harmful to try to handle validation on behalf of
	// some other related object.
	Validate(ctx context.Context) tfdiags.Diagnostics

	// Our general async validation helper relies on this to name its
	// tracing span.
	tracingNamer
}
