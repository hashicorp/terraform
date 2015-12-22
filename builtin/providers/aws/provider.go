package aws

import (
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/mutexkv"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

// Provider returns a terraform.ResourceProvider.
func Provider() terraform.ResourceProvider {
	// TODO: Move the validation to this, requires conditional schemas
	// TODO: Move the configuration to this, requires validation

	// The actual provider
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"access_key": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: descriptions["access_key"],
			},

			"secret_key": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: descriptions["secret_key"],
			},

			"token": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: descriptions["token"],
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

			"max_retries": &schema.Schema{
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     11,
				Description: descriptions["max_retries"],
			},

			"allowed_account_ids": &schema.Schema{
				Type:          schema.TypeSet,
				Elem:          &schema.Schema{Type: schema.TypeString},
				Optional:      true,
				ConflictsWith: []string{"forbidden_account_ids"},
				Set: func(v interface{}) int {
					return hashcode.String(v.(string))
				},
			},

			"forbidden_account_ids": &schema.Schema{
				Type:          schema.TypeSet,
				Elem:          &schema.Schema{Type: schema.TypeString},
				Optional:      true,
				ConflictsWith: []string{"allowed_account_ids"},
				Set: func(v interface{}) int {
					return hashcode.String(v.(string))
				},
			},

			"dynamodb_endpoint": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: descriptions["dynamodb_endpoint"],
			},

			"kinesis_endpoint": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: descriptions["kinesis_endpoint"],
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"aws_ami":                              resourceAwsAmi(),
			"aws_ami_copy":                         resourceAwsAmiCopy(),
			"aws_ami_from_instance":                resourceAwsAmiFromInstance(),
			"aws_app_cookie_stickiness_policy":     resourceAwsAppCookieStickinessPolicy(),
			"aws_autoscaling_group":                resourceAwsAutoscalingGroup(),
			"aws_autoscaling_notification":         resourceAwsAutoscalingNotification(),
			"aws_autoscaling_policy":               resourceAwsAutoscalingPolicy(),
			"aws_autoscaling_schedule":             resourceAwsAutoscalingSchedule(),
			"aws_cloudformation_stack":             resourceAwsCloudFormationStack(),
			"aws_cloudtrail":                       resourceAwsCloudTrail(),
			"aws_cloudwatch_log_group":             resourceAwsCloudWatchLogGroup(),
			"aws_autoscaling_lifecycle_hook":       resourceAwsAutoscalingLifecycleHook(),
			"aws_cloudwatch_metric_alarm":          resourceAwsCloudWatchMetricAlarm(),
			"aws_codedeploy_app":                   resourceAwsCodeDeployApp(),
			"aws_codedeploy_deployment_group":      resourceAwsCodeDeployDeploymentGroup(),
			"aws_codecommit_repository":            resourceAwsCodeCommitRepository(),
			"aws_customer_gateway":                 resourceAwsCustomerGateway(),
			"aws_db_instance":                      resourceAwsDbInstance(),
			"aws_db_parameter_group":               resourceAwsDbParameterGroup(),
			"aws_db_security_group":                resourceAwsDbSecurityGroup(),
			"aws_db_subnet_group":                  resourceAwsDbSubnetGroup(),
			"aws_directory_service_directory":      resourceAwsDirectoryServiceDirectory(),
			"aws_dynamodb_table":                   resourceAwsDynamoDbTable(),
			"aws_ebs_volume":                       resourceAwsEbsVolume(),
			"aws_ecr_repository":                   resourceAwsEcrRepository(),
			"aws_ecr_repository_policy":            resourceAwsEcrRepositoryPolicy(),
			"aws_ecs_cluster":                      resourceAwsEcsCluster(),
			"aws_ecs_service":                      resourceAwsEcsService(),
			"aws_ecs_task_definition":              resourceAwsEcsTaskDefinition(),
			"aws_efs_file_system":                  resourceAwsEfsFileSystem(),
			"aws_efs_mount_target":                 resourceAwsEfsMountTarget(),
			"aws_eip":                              resourceAwsEip(),
			"aws_elasticache_cluster":              resourceAwsElasticacheCluster(),
			"aws_elasticache_parameter_group":      resourceAwsElasticacheParameterGroup(),
			"aws_elasticache_security_group":       resourceAwsElasticacheSecurityGroup(),
			"aws_elasticache_subnet_group":         resourceAwsElasticacheSubnetGroup(),
			"aws_elasticsearch_domain":             resourceAwsElasticSearchDomain(),
			"aws_elb":                              resourceAwsElb(),
			"aws_flow_log":                         resourceAwsFlowLog(),
			"aws_glacier_vault":                    resourceAwsGlacierVault(),
			"aws_iam_access_key":                   resourceAwsIamAccessKey(),
			"aws_iam_group_policy":                 resourceAwsIamGroupPolicy(),
			"aws_iam_group":                        resourceAwsIamGroup(),
			"aws_iam_group_membership":             resourceAwsIamGroupMembership(),
			"aws_iam_instance_profile":             resourceAwsIamInstanceProfile(),
			"aws_iam_policy":                       resourceAwsIamPolicy(),
			"aws_iam_policy_attachment":            resourceAwsIamPolicyAttachment(),
			"aws_iam_role_policy":                  resourceAwsIamRolePolicy(),
			"aws_iam_role":                         resourceAwsIamRole(),
			"aws_iam_saml_provider":                resourceAwsIamSamlProvider(),
			"aws_iam_server_certificate":           resourceAwsIAMServerCertificate(),
			"aws_iam_user_policy":                  resourceAwsIamUserPolicy(),
			"aws_iam_user":                         resourceAwsIamUser(),
			"aws_instance":                         resourceAwsInstance(),
			"aws_internet_gateway":                 resourceAwsInternetGateway(),
			"aws_key_pair":                         resourceAwsKeyPair(),
			"aws_kinesis_firehose_delivery_stream": resourceAwsKinesisFirehoseDeliveryStream(),
			"aws_kinesis_stream":                   resourceAwsKinesisStream(),
			"aws_lambda_function":                  resourceAwsLambdaFunction(),
			"aws_lambda_event_source_mapping":      resourceAwsLambdaEventSourceMapping(),
			"aws_launch_configuration":             resourceAwsLaunchConfiguration(),
			"aws_lb_cookie_stickiness_policy":      resourceAwsLBCookieStickinessPolicy(),
			"aws_main_route_table_association":     resourceAwsMainRouteTableAssociation(),
			"aws_nat_gateway":                      resourceAwsNatGateway(),
			"aws_network_acl":                      resourceAwsNetworkAcl(),
			"aws_network_acl_rule":                 resourceAwsNetworkAclRule(),
			"aws_network_interface":                resourceAwsNetworkInterface(),
			"aws_opsworks_stack":                   resourceAwsOpsworksStack(),
			"aws_opsworks_java_app_layer":          resourceAwsOpsworksJavaAppLayer(),
			"aws_opsworks_haproxy_layer":           resourceAwsOpsworksHaproxyLayer(),
			"aws_opsworks_static_web_layer":        resourceAwsOpsworksStaticWebLayer(),
			"aws_opsworks_php_app_layer":           resourceAwsOpsworksPhpAppLayer(),
			"aws_opsworks_rails_app_layer":         resourceAwsOpsworksRailsAppLayer(),
			"aws_opsworks_nodejs_app_layer":        resourceAwsOpsworksNodejsAppLayer(),
			"aws_opsworks_memcached_layer":         resourceAwsOpsworksMemcachedLayer(),
			"aws_opsworks_mysql_layer":             resourceAwsOpsworksMysqlLayer(),
			"aws_opsworks_ganglia_layer":           resourceAwsOpsworksGangliaLayer(),
			"aws_opsworks_custom_layer":            resourceAwsOpsworksCustomLayer(),
			"aws_placement_group":                  resourceAwsPlacementGroup(),
			"aws_proxy_protocol_policy":            resourceAwsProxyProtocolPolicy(),
			"aws_rds_cluster":                      resourceAwsRDSCluster(),
			"aws_rds_cluster_instance":             resourceAwsRDSClusterInstance(),
			"aws_route53_delegation_set":           resourceAwsRoute53DelegationSet(),
			"aws_route53_record":                   resourceAwsRoute53Record(),
			"aws_route53_zone_association":         resourceAwsRoute53ZoneAssociation(),
			"aws_route53_zone":                     resourceAwsRoute53Zone(),
			"aws_route53_health_check":             resourceAwsRoute53HealthCheck(),
			"aws_route":                            resourceAwsRoute(),
			"aws_route_table":                      resourceAwsRouteTable(),
			"aws_route_table_association":          resourceAwsRouteTableAssociation(),
			"aws_s3_bucket":                        resourceAwsS3Bucket(),
			"aws_s3_bucket_object":                 resourceAwsS3BucketObject(),
			"aws_security_group":                   resourceAwsSecurityGroup(),
			"aws_security_group_rule":              resourceAwsSecurityGroupRule(),
			"aws_spot_instance_request":            resourceAwsSpotInstanceRequest(),
			"aws_sqs_queue":                        resourceAwsSqsQueue(),
			"aws_sns_topic":                        resourceAwsSnsTopic(),
			"aws_sns_topic_subscription":           resourceAwsSnsTopicSubscription(),
			"aws_subnet":                           resourceAwsSubnet(),
			"aws_volume_attachment":                resourceAwsVolumeAttachment(),
			"aws_vpc_dhcp_options_association":     resourceAwsVpcDhcpOptionsAssociation(),
			"aws_vpc_dhcp_options":                 resourceAwsVpcDhcpOptions(),
			"aws_vpc_peering_connection":           resourceAwsVpcPeeringConnection(),
			"aws_vpc":                              resourceAwsVpc(),
			"aws_vpc_endpoint":                     resourceAwsVpcEndpoint(),
			"aws_vpn_connection":                   resourceAwsVpnConnection(),
			"aws_vpn_connection_route":             resourceAwsVpnConnectionRoute(),
			"aws_vpn_gateway":                      resourceAwsVpnGateway(),
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

		"token": "session token. A session token is only required if you are\n" +
			"using temporary security credentials.",

		"max_retries": "The maximum number of times an AWS API request is\n" +
			"being executed. If the API request still fails, an error is\n" +
			"thrown.",

		"dynamodb_endpoint": "Use this to override the default endpoint URL constructed from the `region`.\n" +
			"It's typically used to connect to dynamodb-local.",

		"kinesis_endpoint": "Use this to override the default endpoint URL constructed from the `region`.\n" +
			"It's typically used to connect to kinesalite.",
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	config := Config{
		AccessKey:        d.Get("access_key").(string),
		SecretKey:        d.Get("secret_key").(string),
		Token:            d.Get("token").(string),
		Region:           d.Get("region").(string),
		MaxRetries:       d.Get("max_retries").(int),
		DynamoDBEndpoint: d.Get("dynamodb_endpoint").(string),
		KinesisEndpoint:  d.Get("kinesis_endpoint").(string),
	}

	if v, ok := d.GetOk("allowed_account_ids"); ok {
		config.AllowedAccountIds = v.(*schema.Set).List()
	}

	if v, ok := d.GetOk("forbidden_account_ids"); ok {
		config.ForbiddenAccountIds = v.(*schema.Set).List()
	}

	return config.Client()
}

// This is a global MutexKV for use within this plugin.
var awsMutexKV = mutexkv.NewMutexKV()
