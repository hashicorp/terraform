package cloud

import (
	"fmt"

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
