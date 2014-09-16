package aws

import (
	"os"

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
		Schema: map[string]*schema.Schema{
			"region": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: envDefaultFunc("AWS_REGION"),
			},

			"access_key": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: envDefaultFunc("AWS_ACCESS_KEY"),
			},

			"secret_key": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: envDefaultFunc("AWS_SECRET_KEY"),
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"aws_eip":             resourceAwsEip(),
			"aws_instance":        resourceAwsInstance(),
			"aws_security_group":  resourceAwsSecurityGroup(),
			"aws_db_subnet_group": resourceAwsDbSubnetGroup(),
		},
	}
}

func envDefaultFunc(k string) schema.SchemaDefaultFunc {
	return func() (interface{}, error) {
		if v := os.Getenv(k); v != "" {
			return v, nil
		}

		return nil, nil
	}
}
