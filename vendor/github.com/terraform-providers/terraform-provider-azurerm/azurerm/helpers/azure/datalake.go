package azure

import (
	"regexp"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

//store and analytic account names are the same
func ValidateDataLakeAccountName() schema.SchemaValidateFunc {
	return validation.StringMatch(
		regexp.MustCompile(`\A([a-z0-9]{3,24})\z`),
		"Name can only consist of lowercase letters and numbers and must be between 3 and 24 characters long",
	)
}

func ValidateDataLakeFirewallRuleName() schema.SchemaValidateFunc {
	return validation.StringMatch(
		regexp.MustCompile(`\A([-_a-zA-Z0-9]{3,50})\z`),
		"Name can only consist of letters, numbers, underscores and hyphens and must be between 3 and 50 characters long",
	)
}
