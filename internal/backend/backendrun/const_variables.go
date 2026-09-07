// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package backendrun

import (
	"context"

	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// ConstVariableSupplier is an optional interface that backends can implement
// to supply variable values from their remote storage. This is used to fetch
// const variable values that are needed during early configuration loading
// (e.g., for module source resolution), before a full operation is started.
type ConstVariableSupplier interface {
	// FetchVariables retrieves Terraform variable values stored in the
	// backend for the given workspace. Only variables that are relevant to
	// Terraform (as opposed to environment variables or other categories)
	// should be returned.
	FetchVariables(ctx context.Context, workspace string) (map[string]arguments.UnparsedVariableValue, tfdiags.Diagnostics)
}
