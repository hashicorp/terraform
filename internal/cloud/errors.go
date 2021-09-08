package cloud

import (
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

var (
	invalidOrganizationConfigMissingValue = tfdiags.AttributeValue(
		tfdiags.Error,
		"Invalid organization value",
		`The "organization" attribute value must not be empty.`,
		cty.Path{cty.GetAttrStep{Name: "organization"}},
	)

	invalidWorkspaceConfigMissingValues = tfdiags.AttributeValue(
		tfdiags.Error,
		"Invalid workspaces configuration",
		`Either workspace "name" or "prefix" is required.`,
		cty.Path{cty.GetAttrStep{Name: "workspaces"}},
	)

	invalidWorkspaceConfigMisconfiguration = tfdiags.AttributeValue(
		tfdiags.Error,
		"Invalid workspaces configuration",
		`Only one of workspace "name" or "prefix" is allowed.`,
		cty.Path{cty.GetAttrStep{Name: "workspaces"}},
	)
)
