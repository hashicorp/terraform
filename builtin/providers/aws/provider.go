package aws

import (
	"os"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	goamz "github.com/mitchellh/goamz/aws"
)

// Provider returns a terraform.ResourceProvider.
func Provider() terraform.ResourceProvider {
	// TODO: Move the validation to this, requires conditional schemas
	// TODO: Move the configuration to this, requires validation

	auth := awsAuthSource{}

	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"access_key": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: auth.accessKeyResolver(),
				Description: descriptions["access_key"],
			},

			"secret_key": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: auth.secretKeyResolver(),
				Description: descriptions["secret_key"],
			},

			"region": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				DefaultFunc: schema.MultiEnvDefaultFunc([]string{
					"AWS_REGION",
					"AWS_DEFAULT_REGION",
				}, nil),
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
			"aws_route53_record":               resourceAwsRoute53Record(),
			"aws_route53_zone":                 resourceAwsRoute53Zone(),
			"aws_route_table":                  resourceAwsRouteTable(),
			"aws_route_table_association":      resourceAwsRouteTableAssociation(),
			"aws_s3_bucket":                    resourceAwsS3Bucket(),
			"aws_security_group":               resourceAwsSecurityGroup(),
			"aws_subnet":                       resourceAwsSubnet(),
			"aws_vpc":                          resourceAwsVpc(),
			"aws_vpc_peering_connection":       resourceAwsVpcPeeringConnection(),
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

type awsAuthSource struct {
	conventionalAuth          goamz.Auth
	attemptedConventionalAuth bool
}

func (a *awsAuthSource) awsSourcedAuth() goamz.Auth {
	if a.attemptedConventionalAuth == false {
		auth, err := goamz.EnvAuth()

		if err != nil {
			auth, _ = goamz.SharedAuth()
		}

		a.conventionalAuth = auth
		a.attemptedConventionalAuth = true
	}

	return a.conventionalAuth
}

func (a *awsAuthSource) accessKeyResolver() schema.SchemaDefaultFunc {
	return func() (interface{}, error) {
		// NOTE: Do not remove this until AWS_ACCESS_KEY support has been removed
		// https://github.com/hashicorp/terraform/issues/866
		if accessKey := os.Getenv("AWS_ACCESS_KEY"); accessKey != "" {
			return accessKey, nil
		}

		if auth := a.awsSourcedAuth(); auth.AccessKey != "" {
			return auth.AccessKey, nil
		}

		return nil, nil
	}
}

func (a *awsAuthSource) secretKeyResolver() schema.SchemaDefaultFunc {
	return func() (interface{}, error) {
		// NOTE: Do not remove this until AWS_SECRET_KEY support has been removed
		// https://github.com/hashicorp/terraform/issues/866
		if secretKey := os.Getenv("AWS_SECRET_KEY"); secretKey != "" {
			return secretKey, nil
		}

		if auth := a.awsSourcedAuth(); auth.SecretKey != "" {
			return auth.SecretKey, nil
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
