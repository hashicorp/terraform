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
				Type:         schema.TypeString,
				Required:     true,
				DefaultFunc:  envDefaultFunc("AWS_REGION"),
				Description:  descriptions["region"],
				InputDefault: "us-east-1",
			},

			"access_key": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: envDefaultFunc("AWS_ACCESS_KEY"),
				Description: descriptions["access_key"],
			},

			"secret_key": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: envDefaultFunc("AWS_SECRET_KEY"),
				Description: descriptions["secret_key"],
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

var descriptions map[string]string

func init() {
	descriptions = map[string]string{
		"region": "The region where AWS operations will take place. Examples\n" +
			"are us-east-1, us-west-2, etc.",

		"access_key": "The access key for API operations. You can retrieve this\n" +
			"from the 'Security & Credentials' section of the AWS console.",

		"secret_key": "The secret key for API operations. You can retrieve this\n" +
			"from the 'Security & Credentials' section of the AWS console.",
	}
}
