package cloud

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

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
