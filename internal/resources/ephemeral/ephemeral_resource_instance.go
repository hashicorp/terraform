// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package ephemeral

import (
	"context"

	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// ResourceInstance is an interface that must be implemented for each
// active ephemeral resource instance to determine how it should be renewed
// and eventually closed.
type ResourceInstance interface {
	// Renew attempts to extend the life of the remote object associated with
	// this resource instance, optionally returning a new renewal request to be
	// passed to a subsequent call to this method.
	//
	// If the object's life is not extended successfully then Renew returns
	// error diagnostics explaining why not, and future requests that might
	// have made use of the object will fail.
	Renew(ctx context.Context, req providers.EphemeralRenew) (nextRenew *providers.EphemeralRenew, diags tfdiags.Diagnostics)

	// Close proactively ends the life of the remote object associated with
	// this resource instance, if possible. For example, if the remote object
	// is a temporary lease for a dynamically-generated secret then this
	// might end that lease and thus cause the secret to be promptly revoked.
	Close(ctx context.Context) tfdiags.Diagnostics
}
