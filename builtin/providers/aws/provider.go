package aws

import (
	"github.com/hashicorp/terraform/helper/schema"
)

// Provider returns a schema.Provider for AWS.
//
// NOTE: schema.Provider became available long after the AWS provider
// was started, so resources may not be converted to this new structure
// yet. This is a WIP. To assist with the migration, make sure any resources
// you migrate are acceptance tested, then perform the migration.
func Provider() *schema.Provider {
	// TODO: Move the validation to this, requires conditional schemas
	// TODO: Move the configuration to this, requires validation

	return &schema.Provider{
		ResourcesMap: map[string]*schema.Resource{
			"aws_eip":            resourceAwsEip(),
			"aws_security_group": resourceAwsSecurityGroup(),
		},
	}
}
