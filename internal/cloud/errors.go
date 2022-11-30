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

type multiErrors []error

// Append errs to e only if the individual error in errs is not nil
//
// If any of the errs is itself multiErrors, each individual error in errs is appended.
func (e *multiErrors) Append(errs ...error) {
	for _, err := range errs {
		if err == nil {
			continue
		}
		if errs, ok := err.(multiErrors); ok {
			*e = append(*e, errs...)
		} else {
			*e = append(*e, err)
		}
	}
}

// multiErrors returns an error string by joining
// all of its nonnil errors with colon separator.
func (e multiErrors) Error() string {
	if len(e) == 0 {
		return ""
	}
	es := make([]string, 0, len(e))
	for _, err := range e {
		if err != nil {
			es = append(es, err.Error())
		}
	}
	return strings.Join(es, ":")
}

// Err returns e as an error or returns nil if no errors were collected.
func (e multiErrors) Err() error {
	// Only return self if we have at least one nonnil error.
	for _, err := range e {
		if err != nil {
			return e
		}
	}
	return nil
}
