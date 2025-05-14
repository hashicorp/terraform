// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"github.com/hashicorp/terraform/internal/backend/backendrun"
	"github.com/hashicorp/terraform/internal/cloud"
)

const failedToLoadSchemasMessage = `
Warning: Failed to update data for external integrations

Terraform was unable to generate a description of the updated
state for use with external integrations in HCP Terraform or Terraform Enterprise.
Any integrations configured for this workspace which depend on
information from the state may not work correctly when using the
result of this action.

This problem occurs when Terraform cannot read the schema for
one or more of the providers used in the state. The next successful
apply will correct the problem by re-generating the JSON description
of the state:
    terraform apply
`

func isCloudMode(b backendrun.OperationsBackend) bool {
	_, ok := b.(*cloud.Cloud)

	return ok
}
