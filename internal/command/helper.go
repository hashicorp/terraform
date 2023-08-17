// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"github.com/hashicorp/mnptu/internal/backend"
	"github.com/hashicorp/mnptu/internal/cloud"
)

const failedToLoadSchemasMessage = `
Warning: Failed to update data for external integrations

mnptu was unable to generate a description of the updated
state for use with external integrations in mnptu Cloud.
Any integrations configured for this workspace which depend on
information from the state may not work correctly when using the
result of this action.

This problem occurs when mnptu cannot read the schema for
one or more of the providers used in the state. The next successful
apply will correct the problem by re-generating the JSON description
of the state:
    mnptu apply
`

func isCloudMode(b backend.Enhanced) bool {
	_, ok := b.(*cloud.Cloud)

	return ok
}
