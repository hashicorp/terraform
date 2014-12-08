package aws

import (
	"os"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

// Provider returns a terraform.ResourceProvider.
func Provider() terraform.ResourceProvider {
	// TODO: Move the validation to this, requires conditional schemas
	// TODO: Move the configuration to this, requires validation

	return &schema.Provider{
		Schema: map[string]*schema.Schema{
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

			"region": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				DefaultFunc:  envDefaultFunc("AWS_REGION"),
				Description:  descriptions["region"],
				InputDefault: "us-east-1",
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"aws_autoscaling_group":       resourceAwsAutoscalingGroup(),
			"aws_db_instance":             resourceAwsDbInstance(),
			"aws_db_parameter_group":      resourceAwsDbParameterGroup(),
			"aws_db_security_group":       resourceAwsDbSecurityGroup(),
			"aws_db_subnet_group":         resourceAwsDbSubnetGroup(),
			"aws_eip":                     resourceAwsEip(),
			"aws_elb":                     resourceAwsElb(),
			"aws_instance":                resourceAwsInstance(),
			"aws_internet_gateway":        resourceAwsInternetGateway(),
			"aws_launch_configuration":    resourceAwsLaunchConfiguration(),
			"aws_network_acl":             resourceAwsNetworkAcl(),
			"aws_route53_record":          resourceAwsRoute53Record(),
			"aws_route53_zone":            resourceAwsRoute53Zone(),
			"aws_route_table":             resourceAwsRouteTable(),
			"aws_route_table_association": resourceAwsRouteTableAssociation(),
			"aws_s3_bucket":               resourceAwsS3Bucket(),
			"aws_security_group":          resourceAwsSecurityGroup(),
			"aws_subnet":                  resourceAwsSubnet(),
			"aws_vpc":                     resourceAwsVpc(),
		},

		ConfigureFunc: providerConfigure,
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

func envDefaultFunc(k string) schema.SchemaDefaultFunc {
	return func() (interface{}, error) {
		if v := os.Getenv(k); v != "" {
			return v, nil
		}

		return nil, nil
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	config := Config{
		AccessKey: d.Get("access_key").(string),
		SecretKey: d.Get("secret_key").(string),
		Region:    d.Get("region").(string),
	}

	return config.Client()
}
