package aws

import (
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
				Optional:    true,
				Description: descriptions["access_key"],
			},

			"secret_key": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: descriptions["secret_key"],
			},

			"security_token": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: descriptions["security_token"],
			},

			"credentials_provider": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				InputDefault: "detect",
				Description:  descriptions["credentials_provider"],
			},

			"credentials_file_path": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				InputDefault: "",
				Description:  descriptions["credentials_file_path"],
			},

			"credentials_file_profile": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				InputDefault: "",
				Description:  descriptions["credentials_file_profile"],
			},

			"region": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Description:  descriptions["region"],
				InputDefault: "us-east-1",
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"aws_autoscaling_group":            resourceAwsAutoscalingGroup(),
			"aws_db_instance":                  resourceAwsDbInstance(),
			"aws_db_parameter_group":           resourceAwsDbParameterGroup(),
			"aws_db_security_group":            resourceAwsDbSecurityGroup(),
			"aws_db_subnet_group":              resourceAwsDbSubnetGroup(),
			"aws_eip":                          resourceAwsEip(),
			"aws_elb":                          resourceAwsElb(),
			"aws_instance":                     resourceAwsInstance(),
			"aws_internet_gateway":             resourceAwsInternetGateway(),
			"aws_key_pair":                     resourceAwsKeyPair(),
			"aws_launch_configuration":         resourceAwsLaunchConfiguration(),
			"aws_main_route_table_association": resourceAwsMainRouteTableAssociation(),
			"aws_network_acl":                  resourceAwsNetworkAcl(),
			"aws_network_interface":            resourceAwsNetworkInterface(),
			"aws_route53_record":               resourceAwsRoute53Record(),
			"aws_route53_zone":                 resourceAwsRoute53Zone(),
			"aws_route_table":                  resourceAwsRouteTable(),
			"aws_route_table_association":      resourceAwsRouteTableAssociation(),
			"aws_s3_bucket":                    resourceAwsS3Bucket(),
			"aws_security_group":               resourceAwsSecurityGroup(),
			"aws_subnet":                       resourceAwsSubnet(),
			"aws_vpc":                          resourceAwsVpc(),
			"aws_vpc_peering_connection":       resourceAwsVpcPeeringConnection(),
			"aws_vpn_gateway":                  resourceAwsVpnGateway(),
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

		"security_token": "The temporary token from AWS STS service (if applicable)",

		"credentials_provider": "How to load credentials (static | iam | env | file)\n" +
			"Defaults to detect",

		"credentials_file_path": "Path to a file with credentials. Default\n" +
			"is ~/.aws/credentials. Implies credentials_provider=file",

		"credentials_file_profile": "Profile name in a credentials file." +
			"Default is 'default'. Implies credentials_provider=file",
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	config := Config{
		AccessKey:              d.Get("access_key").(string),
		SecretKey:              d.Get("secret_key").(string),
		Token:                  d.Get("security_token").(string),
		CredentialsFilePath:    d.Get("credentials_file_path").(string),
		CredentialsFileProfile: d.Get("credentials_file_profile").(string),
		Region:                 d.Get("region").(string),
	}

	return config.loadAndValidate(d.Get("credentials_provider").(string))
}
