package cloud

import (
	"fmt"
	"strings"

	tfe "github.com/hashicorp/go-tfe"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

var (
	invalidOrganizationConfigMissingValue = tfdiags.AttributeValue(
		tfdiags.Error,
		"Invalid organization value",
		`The "organization" attribute value must not be empty.\n\n%s`,
		cty.Path{cty.GetAttrStep{Name: "organization"}},
	)

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

func terraformMismatchDiagnostic(ignoreVersionConflict bool, organization string, workspace *tfe.Workspace, tfversion string) tfdiags.Diagnostic {
	severity := tfdiags.Error
	if ignoreVersionConflict {
		severity = tfdiags.Warning
	}

	suggestion := "If you're sure you want to upgrade the state, you can force Terraform to continue using the -ignore-remote-version flag. This may result in an unusable workspace."
	if ignoreVersionConflict {
		suggestion = ""
	}

	description := fmt.Sprintf(
		"The local Terraform version (%s) does not meet the version requirements for remote workspace %s/%s (%s).\n\n%s",
		tfversion,
		organization,
		workspace.Name,
		workspace.TerraformVersion,
		suggestion,
	)
	description = strings.TrimSpace(description)
	return tfdiags.Sourceless(severity, "Terraform version mismatch", description)
}

func terraformInvalidVersionOrConstraint(ignoreVersionConflict bool, tfversion string) tfdiags.Diagnostic {
	severity := tfdiags.Error
	if ignoreVersionConflict {
		severity = tfdiags.Warning
	}

	suggestion := "If you're sure you want to upgrade the state, you can force Terraform to continue using the -ignore-remote-version flag. This may result in an unusable workspace."
	if ignoreVersionConflict {
		suggestion = ""
	}

	description := fmt.Sprintf("The remote workspace specified an invalid Terraform version or version constraint: %s\n\n%s", tfversion, suggestion)
	description = strings.TrimSpace(description)
	return tfdiags.Sourceless(severity, "Terraform version error", description)
}
