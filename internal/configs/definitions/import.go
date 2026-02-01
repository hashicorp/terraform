// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package definitions

import (
	"github.com/hashicorp/hcl/v2"

	"github.com/hashicorp/terraform/internal/addrs"
)

// Import represents an "import" block in a module or file.
type Import struct {
	ID hcl.Expression

	Identity hcl.Expression

	To hcl.Expression
	// The To address may not be resolvable immediately if it contains dynamic
	// index expressions, so we will extract the ConfigResource address and
	// store it here for reference.
	ToResource addrs.ConfigResource

	ForEach hcl.Expression

	ProviderConfigRef *ProviderConfigRef
	Provider          addrs.Provider

	DeclRange         hcl.Range
	ProviderDeclRange hcl.Range
}
