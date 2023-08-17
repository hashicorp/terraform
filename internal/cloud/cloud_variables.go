// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cloud

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/hashicorp/mnptu/internal/backend"
	"github.com/hashicorp/mnptu/internal/configs"
	"github.com/hashicorp/mnptu/internal/mnptu"
	"github.com/hashicorp/mnptu/internal/tfdiags"
)

func allowedSourceType(source mnptu.ValueSourceType) bool {
	return source == mnptu.ValueFromNamedFile || source == mnptu.ValueFromCLIArg || source == mnptu.ValueFromEnvVar
}

// ParseCloudRunVariables accepts a mapping of unparsed values and a mapping of variable
// declarations and returns a name/value variable map appropriate for an API run context,
// that is, containing variables only sourced from non-file inputs like CLI args
// and environment variables. However, all variable parsing diagnostics are returned
// in order to allow callers to short circuit cloud runs that contain variable
// declaration or parsing errors. The only exception is that missing required values are not
// considered errors because they may be defined within the cloud workspace.
func ParseCloudRunVariables(vv map[string]backend.UnparsedVariableValue, decls map[string]*configs.Variable) (map[string]string, tfdiags.Diagnostics) {
	declared, diags := backend.ParseDeclaredVariableValues(vv, decls)
	_, undedeclaredDiags := backend.ParseUndeclaredVariableValues(vv, decls)
	diags = diags.Append(undedeclaredDiags)

	ret := make(map[string]string, len(declared))

	// Even if there are parsing or declaration errors, populate the return map with the
	// variables that could be used for cloud runs
	for name, v := range declared {
		if !allowedSourceType(v.SourceType) {
			continue
		}

		// RunVariables are always expressed as HCL strings
		tokens := hclwrite.TokensForValue(v.Value)
		ret[name] = string(tokens.Bytes())
	}

	return ret, diags
}
