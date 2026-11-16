// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package configs

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
)

// ProviderRequirementExpr is a container for deferred HCL expressions for a
// required provider's source and/or version.
type ProviderRequirementExpr struct {
	Name        string
	SourceExpr  hcl.Expression
	VersionExpr hcl.Expression

	ConfigAliases []addrs.LocalProviderConfig

	DeclRange hcl.Range
}

func (e *ProviderRequirementExpr) IsEmpty() bool {
	return e.SourceExpr == nil && e.VersionExpr == nil
}
