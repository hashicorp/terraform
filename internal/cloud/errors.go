// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package cloud

import (
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

// String based errors
var (
	errApplyDiscarded                    = errors.New("Apply discarded.")
	errDestroyDiscarded                  = errors.New("Destroy discarded.")
	errRunApproved                       = errors.New("approved using the UI or API")
	errRunDiscarded                      = errors.New("discarded using the UI or API")
	errRunOverridden                     = errors.New("overridden using the UI or API")
	errApplyNeedsUIConfirmation          = errors.New("Cannot confirm apply due to -input=false. Please handle run confirmation in the UI.")
	errPolicyOverrideNeedsUIConfirmation = errors.New("Cannot override soft failed policy checks when -input=false. Please open the run in the UI to override.")
)

// Diagnostic error messages
var (
	invalidWorkspaceConfigMissingValues = tfdiags.AttributeValue(
		tfdiags.Error,
		"Invalid workspaces configuration",
		fmt.Sprintf("Missing workspace mapping strategy. Either workspace \"tags\" or \"name\" is required.\n\n%s", workspaceConfigurationHelp),
		cty.Path{cty.GetAttrStep{Name: "workspaces"}},
	)

	invalidWorkspaceConfigMisconfiguration = tfdiags.AttributeValue(
		tfdiags.Error,
		"Invalid workspaces configuration",
		fmt.Sprintf("Only one of workspace \"tags\" or \"name\" is allowed.\n\n%s", workspaceConfigurationHelp),
		cty.Path{cty.GetAttrStep{Name: "workspaces"}},
	)
)

const ignoreRemoteVersionHelp = "If you're sure you want to upgrade the state, you can force Terraform to continue using the -ignore-remote-version flag. This may result in an unusable workspace."

func missingConfigAttributeAndEnvVar(attribute string, envVar string) tfdiags.Diagnostic {
	detail := strings.TrimSpace(fmt.Sprintf("\"%s\" must be set in the cloud configuration or as an environment variable: %s.\n", attribute, envVar))
	return tfdiags.AttributeValue(
		tfdiags.Error,
		"Invalid or missing required argument",
		detail,
		cty.Path{cty.GetAttrStep{Name: attribute}})
}

func incompatibleWorkspaceTerraformVersion(message string, ignoreVersionConflict bool) tfdiags.Diagnostic {
	severity := tfdiags.Error
	suggestion := ignoreRemoteVersionHelp
	if ignoreVersionConflict {
		severity = tfdiags.Warning
		suggestion = ""
	}
	description := strings.TrimSpace(fmt.Sprintf("%s\n\n%s", message, suggestion))
	return tfdiags.Sourceless(severity, "Incompatible Terraform version", description)
}
