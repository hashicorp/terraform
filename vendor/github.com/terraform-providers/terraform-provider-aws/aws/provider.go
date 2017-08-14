package aws

import (
	"bytes"
	"fmt"
	"log"

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
			"access_key": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: descriptions["access_key"],
			},

			"secret_key": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: descriptions["secret_key"],
			},

			"profile": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: descriptions["profile"],
			},

			"assume_role": assumeRoleSchema(),

			"shared_credentials_file": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: descriptions["shared_credentials_file"],
			},

			"token": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: descriptions["token"],
			},

			"region": {
				Type:     schema.TypeString,
				Required: true,
				DefaultFunc: schema.MultiEnvDefaultFunc([]string{
					"AWS_REGION",
					"AWS_DEFAULT_REGION",
				}, nil),
				Description:  descriptions["region"],
				InputDefault: "us-east-1",
			},

			"max_retries": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     25,
				Description: descriptions["max_retries"],
			},

			"allowed_account_ids": {
				Type:          schema.TypeSet,
				Elem:          &schema.Schema{Type: schema.TypeString},
				Optional:      true,
				ConflictsWith: []string{"forbidden_account_ids"},
				Set:           schema.HashString,
			},

			"forbidden_account_ids": {
				Type:          schema.TypeSet,
				Elem:          &schema.Schema{Type: schema.TypeString},
				Optional:      true,
				ConflictsWith: []string{"allowed_account_ids"},
				Set:           schema.HashString,
			},

			"dynamodb_endpoint": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: descriptions["dynamodb_endpoint"],
				Removed:     "Use `dynamodb` inside `endpoints` block instead",
			},

			"kinesis_endpoint": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: descriptions["kinesis_endpoint"],
				Removed:     "Use `kinesis` inside `endpoints` block instead",
			},

			"endpoints": endpointsSchema(),

			"insecure": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: descriptions["insecure"],
			},

			"skip_credentials_validation": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: descriptions["skip_credentials_validation"],
			},

			"skip_get_ec2_platforms": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: descriptions["skip_get_ec2_platforms"],
			},

			"skip_region_validation": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: descriptions["skip_region_validation"],
			},

			"skip_requesting_account_id": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: descriptions["skip_requesting_account_id"],
			},

			"skip_metadata_api_check": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: descriptions["skip_metadata_api_check"],
			},

			"s3_force_path_style": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: descriptions["s3_force_path_style"],
			},
		},

		DataSourcesMap: map[string]*schema.Resource{
			"aws_acm_certificate":          dataSourceAwsAcmCertificate(),
			"aws_alb":                      dataSourceAwsAlb(),
			"aws_alb_listener":             dataSourceAwsAlbListener(),
			"aws_alb_target_group":         dataSourceAwsAlbTargetGroup(),
			"aws_ami":                      dataSourceAwsAmi(),
			"aws_ami_ids":                  dataSourceAwsAmiIds(),
			"aws_autoscaling_groups":       dataSourceAwsAutoscalingGroups(),
			"aws_availability_zone":        dataSourceAwsAvailabilityZone(),
			"aws_availability_zones":       dataSourceAwsAvailabilityZones(),
			"aws_billing_service_account":  dataSourceAwsBillingServiceAccount(),
			"aws_caller_identity":          dataSourceAwsCallerIdentity(),
			"aws_canonical_user_id":        dataSourceAwsCanonicalUserId(),
			"aws_cloudformation_stack":     dataSourceAwsCloudFormationStack(),
			"aws_db_instance":              dataSourceAwsDbInstance(),
			"aws_db_snapshot":              dataSourceAwsDbSnapshot(),
			"aws_ebs_snapshot":             dataSourceAwsEbsSnapshot(),
			"aws_ebs_snapshot_ids":         dataSourceAwsEbsSnapshotIds(),
			"aws_ebs_volume":               dataSourceAwsEbsVolume(),
			"aws_ecr_repository":           dataSourceAwsEcrRepository(),
			"aws_ecs_cluster":              dataSourceAwsEcsCluster(),
			"aws_ecs_container_definition": dataSourceAwsEcsContainerDefinition(),
			"aws_ecs_task_definition":      dataSourceAwsEcsTaskDefinition(),
			"aws_efs_file_system":          dataSourceAwsEfsFileSystem(),
			"aws_efs_mount_target":         dataSourceAwsEfsMountTarget(),
			"aws_eip":                      dataSourceAwsEip(),
			"aws_elastic_beanstalk_solution_stack": dataSourceAwsElasticBeanstalkSolutionStack(),
			"aws_elasticache_cluster":              dataSourceAwsElastiCacheCluster(),
			"aws_elb_hosted_zone_id":               dataSourceAwsElbHostedZoneId(),
			"aws_elb_service_account":              dataSourceAwsElbServiceAccount(),
			"aws_iam_account_alias":                dataSourceAwsIamAccountAlias(),
			"aws_iam_group":                        dataSourceAwsIAMGroup(),
			"aws_iam_instance_profile":             dataSourceAwsIAMInstanceProfile(),
			"aws_iam_policy_document":              dataSourceAwsIamPolicyDocument(),
			"aws_iam_role":                         dataSourceAwsIAMRole(),
			"aws_iam_server_certificate":           dataSourceAwsIAMServerCertificate(),
			"aws_internet_gateway":                 dataSourceAwsInternetGateway(),
			"aws_instance":                         dataSourceAwsInstance(),
			"aws_ip_ranges":                        dataSourceAwsIPRanges(),
			"aws_kinesis_stream":                   dataSourceAwsKinesisStream(),
			"aws_kms_alias":                        dataSourceAwsKmsAlias(),
			"aws_kms_ciphertext":                   dataSourceAwsKmsCiphetext(),
			"aws_kms_secret":                       dataSourceAwsKmsSecret(),
			"aws_partition":                        dataSourceAwsPartition(),
			"aws_prefix_list":                      dataSourceAwsPrefixList(),
			"aws_redshift_service_account":         dataSourceAwsRedshiftServiceAccount(),
			"aws_region":                           dataSourceAwsRegion(),
			"aws_route_table":                      dataSourceAwsRouteTable(),
			"aws_route53_zone":                     dataSourceAwsRoute53Zone(),
			"aws_s3_bucket_object":                 dataSourceAwsS3BucketObject(),
			"aws_sns_topic":                        dataSourceAwsSnsTopic(),
			"aws_ssm_parameter":                    dataSourceAwsSsmParameter(),
			"aws_subnet":                           dataSourceAwsSubnet(),
			"aws_subnet_ids":                       dataSourceAwsSubnetIDs(),
			"aws_security_group":                   dataSourceAwsSecurityGroup(),
			"aws_vpc":                              dataSourceAwsVpc(),
			"aws_vpc_endpoint":                     dataSourceAwsVpcEndpoint(),
			"aws_vpc_endpoint_service":             dataSourceAwsVpcEndpointService(),
			"aws_vpc_peering_connection":           dataSourceAwsVpcPeeringConnection(),
			"aws_vpn_gateway":                      dataSourceAwsVpnGateway(),
		},

		ResourcesMap: map[string]*schema.Resource{
			"aws_alb":                                      resourceAwsAlb(),
			"aws_alb_listener":                             resourceAwsAlbListener(),
			"aws_alb_listener_rule":                        resourceAwsAlbListenerRule(),
			"aws_alb_target_group":                         resourceAwsAlbTargetGroup(),
			"aws_alb_target_group_attachment":              resourceAwsAlbTargetGroupAttachment(),
			"aws_ami":                                      resourceAwsAmi(),
			"aws_ami_copy":                                 resourceAwsAmiCopy(),
			"aws_ami_from_instance":                        resourceAwsAmiFromInstance(),
			"aws_ami_launch_permission":                    resourceAwsAmiLaunchPermission(),
			"aws_api_gateway_account":                      resourceAwsApiGatewayAccount(),
			"aws_api_gateway_api_key":                      resourceAwsApiGatewayApiKey(),
			"aws_api_gateway_authorizer":                   resourceAwsApiGatewayAuthorizer(),
			"aws_api_gateway_base_path_mapping":            resourceAwsApiGatewayBasePathMapping(),
			"aws_api_gateway_client_certificate":           resourceAwsApiGatewayClientCertificate(),
			"aws_api_gateway_deployment":                   resourceAwsApiGatewayDeployment(),
			"aws_api_gateway_domain_name":                  resourceAwsApiGatewayDomainName(),
			"aws_api_gateway_gateway_response":             resourceAwsApiGatewayGatewayResponse(),
			"aws_api_gateway_integration":                  resourceAwsApiGatewayIntegration(),
			"aws_api_gateway_integration_response":         resourceAwsApiGatewayIntegrationResponse(),
			"aws_api_gateway_method":                       resourceAwsApiGatewayMethod(),
			"aws_api_gateway_method_response":              resourceAwsApiGatewayMethodResponse(),
			"aws_api_gateway_method_settings":              resourceAwsApiGatewayMethodSettings(),
			"aws_api_gateway_model":                        resourceAwsApiGatewayModel(),
			"aws_api_gateway_request_validator":            resourceAwsApiGatewayRequestValidator(),
			"aws_api_gateway_resource":                     resourceAwsApiGatewayResource(),
			"aws_api_gateway_rest_api":                     resourceAwsApiGatewayRestApi(),
			"aws_api_gateway_stage":                        resourceAwsApiGatewayStage(),
			"aws_api_gateway_usage_plan":                   resourceAwsApiGatewayUsagePlan(),
			"aws_api_gateway_usage_plan_key":               resourceAwsApiGatewayUsagePlanKey(),
			"aws_app_cookie_stickiness_policy":             resourceAwsAppCookieStickinessPolicy(),
			"aws_appautoscaling_target":                    resourceAwsAppautoscalingTarget(),
			"aws_appautoscaling_policy":                    resourceAwsAppautoscalingPolicy(),
			"aws_autoscaling_attachment":                   resourceAwsAutoscalingAttachment(),
			"aws_autoscaling_group":                        resourceAwsAutoscalingGroup(),
			"aws_autoscaling_notification":                 resourceAwsAutoscalingNotification(),
			"aws_autoscaling_policy":                       resourceAwsAutoscalingPolicy(),
			"aws_autoscaling_schedule":                     resourceAwsAutoscalingSchedule(),
			"aws_cloudformation_stack":                     resourceAwsCloudFormationStack(),
			"aws_cloudfront_distribution":                  resourceAwsCloudFrontDistribution(),
			"aws_cloudfront_origin_access_identity":        resourceAwsCloudFrontOriginAccessIdentity(),
			"aws_cloudtrail":                               resourceAwsCloudTrail(),
			"aws_cloudwatch_event_rule":                    resourceAwsCloudWatchEventRule(),
			"aws_cloudwatch_event_target":                  resourceAwsCloudWatchEventTarget(),
			"aws_cloudwatch_log_destination":               resourceAwsCloudWatchLogDestination(),
			"aws_cloudwatch_log_destination_policy":        resourceAwsCloudWatchLogDestinationPolicy(),
			"aws_cloudwatch_log_group":                     resourceAwsCloudWatchLogGroup(),
			"aws_cloudwatch_log_metric_filter":             resourceAwsCloudWatchLogMetricFilter(),
			"aws_cloudwatch_log_stream":                    resourceAwsCloudWatchLogStream(),
			"aws_cloudwatch_log_subscription_filter":       resourceAwsCloudwatchLogSubscriptionFilter(),
			"aws_config_config_rule":                       resourceAwsConfigConfigRule(),
			"aws_config_configuration_recorder":            resourceAwsConfigConfigurationRecorder(),
			"aws_config_configuration_recorder_status":     resourceAwsConfigConfigurationRecorderStatus(),
			"aws_config_delivery_channel":                  resourceAwsConfigDeliveryChannel(),
			"aws_cognito_identity_pool":                    resourceAwsCognitoIdentityPool(),
			"aws_autoscaling_lifecycle_hook":               resourceAwsAutoscalingLifecycleHook(),
			"aws_cloudwatch_metric_alarm":                  resourceAwsCloudWatchMetricAlarm(),
			"aws_cloudwatch_dashboard":                     resourceAwsCloudWatchDashboard(),
			"aws_codedeploy_app":                           resourceAwsCodeDeployApp(),
			"aws_codedeploy_deployment_config":             resourceAwsCodeDeployDeploymentConfig(),
			"aws_codedeploy_deployment_group":              resourceAwsCodeDeployDeploymentGroup(),
			"aws_codecommit_repository":                    resourceAwsCodeCommitRepository(),
			"aws_codecommit_trigger":                       resourceAwsCodeCommitTrigger(),
			"aws_codebuild_project":                        resourceAwsCodeBuildProject(),
			"aws_codepipeline":                             resourceAwsCodePipeline(),
			"aws_customer_gateway":                         resourceAwsCustomerGateway(),
			"aws_db_event_subscription":                    resourceAwsDbEventSubscription(),
			"aws_db_instance":                              resourceAwsDbInstance(),
			"aws_db_option_group":                          resourceAwsDbOptionGroup(),
			"aws_db_parameter_group":                       resourceAwsDbParameterGroup(),
			"aws_db_security_group":                        resourceAwsDbSecurityGroup(),
			"aws_db_snapshot":                              resourceAwsDbSnapshot(),
			"aws_db_subnet_group":                          resourceAwsDbSubnetGroup(),
			"aws_devicefarm_project":                       resourceAwsDevicefarmProject(),
			"aws_directory_service_directory":              resourceAwsDirectoryServiceDirectory(),
			"aws_dms_certificate":                          resourceAwsDmsCertificate(),
			"aws_dms_endpoint":                             resourceAwsDmsEndpoint(),
			"aws_dms_replication_instance":                 resourceAwsDmsReplicationInstance(),
			"aws_dms_replication_subnet_group":             resourceAwsDmsReplicationSubnetGroup(),
			"aws_dms_replication_task":                     resourceAwsDmsReplicationTask(),
			"aws_dynamodb_table":                           resourceAwsDynamoDbTable(),
			"aws_ebs_snapshot":                             resourceAwsEbsSnapshot(),
			"aws_ebs_volume":                               resourceAwsEbsVolume(),
			"aws_ecr_repository":                           resourceAwsEcrRepository(),
			"aws_ecr_repository_policy":                    resourceAwsEcrRepositoryPolicy(),
			"aws_ecs_cluster":                              resourceAwsEcsCluster(),
			"aws_ecs_service":                              resourceAwsEcsService(),
			"aws_ecs_task_definition":                      resourceAwsEcsTaskDefinition(),
			"aws_efs_file_system":                          resourceAwsEfsFileSystem(),
			"aws_efs_mount_target":                         resourceAwsEfsMountTarget(),
			"aws_egress_only_internet_gateway":             resourceAwsEgressOnlyInternetGateway(),
			"aws_eip":                                      resourceAwsEip(),
			"aws_eip_association":                          resourceAwsEipAssociation(),
			"aws_elasticache_cluster":                      resourceAwsElasticacheCluster(),
			"aws_elasticache_parameter_group":              resourceAwsElasticacheParameterGroup(),
			"aws_elasticache_replication_group":            resourceAwsElasticacheReplicationGroup(),
			"aws_elasticache_security_group":               resourceAwsElasticacheSecurityGroup(),
			"aws_elasticache_subnet_group":                 resourceAwsElasticacheSubnetGroup(),
			"aws_elastic_beanstalk_application":            resourceAwsElasticBeanstalkApplication(),
			"aws_elastic_beanstalk_application_version":    resourceAwsElasticBeanstalkApplicationVersion(),
			"aws_elastic_beanstalk_configuration_template": resourceAwsElasticBeanstalkConfigurationTemplate(),
			"aws_elastic_beanstalk_environment":            resourceAwsElasticBeanstalkEnvironment(),
			"aws_elasticsearch_domain":                     resourceAwsElasticSearchDomain(),
			"aws_elasticsearch_domain_policy":              resourceAwsElasticSearchDomainPolicy(),
			"aws_elastictranscoder_pipeline":               resourceAwsElasticTranscoderPipeline(),
			"aws_elastictranscoder_preset":                 resourceAwsElasticTranscoderPreset(),
			"aws_elb":                                      resourceAwsElb(),
			"aws_elb_attachment":                           resourceAwsElbAttachment(),
			"aws_emr_cluster":                              resourceAwsEMRCluster(),
			"aws_emr_instance_group":                       resourceAwsEMRInstanceGroup(),
			"aws_emr_security_configuration":               resourceAwsEMRSecurityConfiguration(),
			"aws_flow_log":                                 resourceAwsFlowLog(),
			"aws_glacier_vault":                            resourceAwsGlacierVault(),
			"aws_iam_access_key":                           resourceAwsIamAccessKey(),
			"aws_iam_account_alias":                        resourceAwsIamAccountAlias(),
			"aws_iam_account_password_policy":              resourceAwsIamAccountPasswordPolicy(),
			"aws_iam_group_policy":                         resourceAwsIamGroupPolicy(),
			"aws_iam_group":                                resourceAwsIamGroup(),
			"aws_iam_group_membership":                     resourceAwsIamGroupMembership(),
			"aws_iam_group_policy_attachment":              resourceAwsIamGroupPolicyAttachment(),
			"aws_iam_instance_profile":                     resourceAwsIamInstanceProfile(),
			"aws_iam_openid_connect_provider":              resourceAwsIamOpenIDConnectProvider(),
			"aws_iam_policy":                               resourceAwsIamPolicy(),
			"aws_iam_policy_attachment":                    resourceAwsIamPolicyAttachment(),
			"aws_iam_role_policy_attachment":               resourceAwsIamRolePolicyAttachment(),
			"aws_iam_role_policy":                          resourceAwsIamRolePolicy(),
			"aws_iam_role":                                 resourceAwsIamRole(),
			"aws_iam_saml_provider":                        resourceAwsIamSamlProvider(),
			"aws_iam_server_certificate":                   resourceAwsIAMServerCertificate(),
			"aws_iam_user_policy_attachment":               resourceAwsIamUserPolicyAttachment(),
			"aws_iam_user_policy":                          resourceAwsIamUserPolicy(),
			"aws_iam_user_ssh_key":                         resourceAwsIamUserSshKey(),
			"aws_iam_user":                                 resourceAwsIamUser(),
			"aws_iam_user_login_profile":                   resourceAwsIamUserLoginProfile(),
			"aws_inspector_assessment_target":              resourceAWSInspectorAssessmentTarget(),
			"aws_inspector_assessment_template":            resourceAWSInspectorAssessmentTemplate(),
			"aws_inspector_resource_group":                 resourceAWSInspectorResourceGroup(),
			"aws_instance":                                 resourceAwsInstance(),
			"aws_internet_gateway":                         resourceAwsInternetGateway(),
			"aws_iot_certificate":                          resourceAwsIotCertificate(),
			"aws_iot_policy":                               resourceAwsIotPolicy(),
			"aws_key_pair":                                 resourceAwsKeyPair(),
			"aws_kinesis_firehose_delivery_stream":         resourceAwsKinesisFirehoseDeliveryStream(),
			"aws_kinesis_stream":                           resourceAwsKinesisStream(),
			"aws_kms_alias":                                resourceAwsKmsAlias(),
			"aws_kms_key":                                  resourceAwsKmsKey(),
			"aws_lambda_function":                          resourceAwsLambdaFunction(),
			"aws_lambda_event_source_mapping":              resourceAwsLambdaEventSourceMapping(),
			"aws_lambda_alias":                             resourceAwsLambdaAlias(),
			"aws_lambda_permission":                        resourceAwsLambdaPermission(),
			"aws_launch_configuration":                     resourceAwsLaunchConfiguration(),
			"aws_lightsail_domain":                         resourceAwsLightsailDomain(),
			"aws_lightsail_instance":                       resourceAwsLightsailInstance(),
			"aws_lightsail_key_pair":                       resourceAwsLightsailKeyPair(),
			"aws_lightsail_static_ip":                      resourceAwsLightsailStaticIp(),
			"aws_lightsail_static_ip_attachment":           resourceAwsLightsailStaticIpAttachment(),
			"aws_lb_cookie_stickiness_policy":              resourceAwsLBCookieStickinessPolicy(),
			"aws_load_balancer_policy":                     resourceAwsLoadBalancerPolicy(),
			"aws_load_balancer_backend_server_policy":      resourceAwsLoadBalancerBackendServerPolicies(),
			"aws_load_balancer_listener_policy":            resourceAwsLoadBalancerListenerPolicies(),
			"aws_lb_ssl_negotiation_policy":                resourceAwsLBSSLNegotiationPolicy(),
			"aws_main_route_table_association":             resourceAwsMainRouteTableAssociation(),
			"aws_nat_gateway":                              resourceAwsNatGateway(),
			"aws_network_acl":                              resourceAwsNetworkAcl(),
			"aws_default_network_acl":                      resourceAwsDefaultNetworkAcl(),
			"aws_network_acl_rule":                         resourceAwsNetworkAclRule(),
			"aws_network_interface":                        resourceAwsNetworkInterface(),
			"aws_network_interface_attachment":             resourceAwsNetworkInterfaceAttachment(),
			"aws_opsworks_application":                     resourceAwsOpsworksApplication(),
			"aws_opsworks_stack":                           resourceAwsOpsworksStack(),
			"aws_opsworks_java_app_layer":                  resourceAwsOpsworksJavaAppLayer(),
			"aws_opsworks_haproxy_layer":                   resourceAwsOpsworksHaproxyLayer(),
			"aws_opsworks_static_web_layer":                resourceAwsOpsworksStaticWebLayer(),
			"aws_opsworks_php_app_layer":                   resourceAwsOpsworksPhpAppLayer(),
			"aws_opsworks_rails_app_layer":                 resourceAwsOpsworksRailsAppLayer(),
			"aws_opsworks_nodejs_app_layer":                resourceAwsOpsworksNodejsAppLayer(),
			"aws_opsworks_memcached_layer":                 resourceAwsOpsworksMemcachedLayer(),
			"aws_opsworks_mysql_layer":                     resourceAwsOpsworksMysqlLayer(),
			"aws_opsworks_ganglia_layer":                   resourceAwsOpsworksGangliaLayer(),
			"aws_opsworks_custom_layer":                    resourceAwsOpsworksCustomLayer(),
			"aws_opsworks_instance":                        resourceAwsOpsworksInstance(),
			"aws_opsworks_user_profile":                    resourceAwsOpsworksUserProfile(),
			"aws_opsworks_permission":                      resourceAwsOpsworksPermission(),
			"aws_opsworks_rds_db_instance":                 resourceAwsOpsworksRdsDbInstance(),
			"aws_placement_group":                          resourceAwsPlacementGroup(),
			"aws_proxy_protocol_policy":                    resourceAwsProxyProtocolPolicy(),
			"aws_rds_cluster":                              resourceAwsRDSCluster(),
			"aws_rds_cluster_instance":                     resourceAwsRDSClusterInstance(),
			"aws_rds_cluster_parameter_group":              resourceAwsRDSClusterParameterGroup(),
			"aws_redshift_cluster":                         resourceAwsRedshiftCluster(),
			"aws_redshift_security_group":                  resourceAwsRedshiftSecurityGroup(),
			"aws_redshift_parameter_group":                 resourceAwsRedshiftParameterGroup(),
			"aws_redshift_subnet_group":                    resourceAwsRedshiftSubnetGroup(),
			"aws_route53_delegation_set":                   resourceAwsRoute53DelegationSet(),
			"aws_route53_record":                           resourceAwsRoute53Record(),
			"aws_route53_zone_association":                 resourceAwsRoute53ZoneAssociation(),
			"aws_route53_zone":                             resourceAwsRoute53Zone(),
			"aws_route53_health_check":                     resourceAwsRoute53HealthCheck(),
			"aws_route":                                    resourceAwsRoute(),
			"aws_route_table":                              resourceAwsRouteTable(),
			"aws_default_route_table":                      resourceAwsDefaultRouteTable(),
			"aws_route_table_association":                  resourceAwsRouteTableAssociation(),
			"aws_ses_active_receipt_rule_set":              resourceAwsSesActiveReceiptRuleSet(),
			"aws_ses_domain_identity":                      resourceAwsSesDomainIdentity(),
			"aws_ses_receipt_filter":                       resourceAwsSesReceiptFilter(),
			"aws_ses_receipt_rule":                         resourceAwsSesReceiptRule(),
			"aws_ses_receipt_rule_set":                     resourceAwsSesReceiptRuleSet(),
			"aws_ses_configuration_set":                    resourceAwsSesConfigurationSet(),
			"aws_ses_event_destination":                    resourceAwsSesEventDestination(),
			"aws_s3_bucket":                                resourceAwsS3Bucket(),
			"aws_s3_bucket_policy":                         resourceAwsS3BucketPolicy(),
			"aws_s3_bucket_object":                         resourceAwsS3BucketObject(),
			"aws_s3_bucket_notification":                   resourceAwsS3BucketNotification(),
			"aws_security_group":                           resourceAwsSecurityGroup(),
			"aws_network_interface_sg_attachment":          resourceAwsNetworkInterfaceSGAttachment(),
			"aws_default_security_group":                   resourceAwsDefaultSecurityGroup(),
			"aws_security_group_rule":                      resourceAwsSecurityGroupRule(),
			"aws_simpledb_domain":                          resourceAwsSimpleDBDomain(),
			"aws_ssm_activation":                           resourceAwsSsmActivation(),
			"aws_ssm_association":                          resourceAwsSsmAssociation(),
			"aws_ssm_document":                             resourceAwsSsmDocument(),
			"aws_ssm_maintenance_window":                   resourceAwsSsmMaintenanceWindow(),
			"aws_ssm_maintenance_window_target":            resourceAwsSsmMaintenanceWindowTarget(),
			"aws_ssm_maintenance_window_task":              resourceAwsSsmMaintenanceWindowTask(),
			"aws_ssm_patch_baseline":                       resourceAwsSsmPatchBaseline(),
			"aws_ssm_patch_group":                          resourceAwsSsmPatchGroup(),
			"aws_ssm_parameter":                            resourceAwsSsmParameter(),
			"aws_spot_datafeed_subscription":               resourceAwsSpotDataFeedSubscription(),
			"aws_spot_instance_request":                    resourceAwsSpotInstanceRequest(),
			"aws_spot_fleet_request":                       resourceAwsSpotFleetRequest(),
			"aws_sqs_queue":                                resourceAwsSqsQueue(),
			"aws_sqs_queue_policy":                         resourceAwsSqsQueuePolicy(),
			"aws_snapshot_create_volume_permission":        resourceAwsSnapshotCreateVolumePermission(),
			"aws_sns_topic":                                resourceAwsSnsTopic(),
			"aws_sns_topic_policy":                         resourceAwsSnsTopicPolicy(),
			"aws_sns_topic_subscription":                   resourceAwsSnsTopicSubscription(),
			"aws_sfn_activity":                             resourceAwsSfnActivity(),
			"aws_sfn_state_machine":                        resourceAwsSfnStateMachine(),
			"aws_default_subnet":                           resourceAwsDefaultSubnet(),
			"aws_subnet":                                   resourceAwsSubnet(),
			"aws_volume_attachment":                        resourceAwsVolumeAttachment(),
			"aws_vpc_dhcp_options_association":             resourceAwsVpcDhcpOptionsAssociation(),
			"aws_default_vpc_dhcp_options":                 resourceAwsDefaultVpcDhcpOptions(),
			"aws_vpc_dhcp_options":                         resourceAwsVpcDhcpOptions(),
			"aws_vpc_peering_connection":                   resourceAwsVpcPeeringConnection(),
			"aws_vpc_peering_connection_accepter":          resourceAwsVpcPeeringConnectionAccepter(),
			"aws_default_vpc":                              resourceAwsDefaultVpc(),
			"aws_vpc":                                      resourceAwsVpc(),
			"aws_vpc_endpoint":                             resourceAwsVpcEndpoint(),
			"aws_vpc_endpoint_route_table_association":     resourceAwsVpcEndpointRouteTableAssociation(),
			"aws_vpn_connection":                           resourceAwsVpnConnection(),
			"aws_vpn_connection_route":                     resourceAwsVpnConnectionRoute(),
			"aws_vpn_gateway":                              resourceAwsVpnGateway(),
			"aws_vpn_gateway_attachment":                   resourceAwsVpnGatewayAttachment(),
			"aws_vpn_gateway_route_propagation":            resourceAwsVpnGatewayRoutePropagation(),
			"aws_waf_byte_match_set":                       resourceAwsWafByteMatchSet(),
			"aws_waf_ipset":                                resourceAwsWafIPSet(),
			"aws_waf_rule":                                 resourceAwsWafRule(),
			"aws_waf_size_constraint_set":                  resourceAwsWafSizeConstraintSet(),
			"aws_waf_web_acl":                              resourceAwsWafWebAcl(),
			"aws_waf_xss_match_set":                        resourceAwsWafXssMatchSet(),
			"aws_waf_sql_injection_match_set":              resourceAwsWafSqlInjectionMatchSet(),
			"aws_wafregional_byte_match_set":               resourceAwsWafRegionalByteMatchSet(),
			"aws_wafregional_ipset":                        resourceAwsWafRegionalIPSet(),
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

		"profile": "The profile for API operations. If not set, the default profile\n" +
			"created with `aws configure` will be used.",

		"shared_credentials_file": "The path to the shared credentials file. If not set\n" +
			"this defaults to ~/.aws/credentials.",

		"token": "session token. A session token is only required if you are\n" +
			"using temporary security credentials.",

		"max_retries": "The maximum number of times an AWS API request is\n" +
			"being executed. If the API request still fails, an error is\n" +
			"thrown.",

		"cloudformation_endpoint": "Use this to override the default endpoint URL constructed from the `region`.\n",

		"cloudwatch_endpoint": "Use this to override the default endpoint URL constructed from the `region`.\n",

		"cloudwatchevents_endpoint": "Use this to override the default endpoint URL constructed from the `region`.\n",

		"cloudwatchlogs_endpoint": "Use this to override the default endpoint URL constructed from the `region`.\n",

		"devicefarm_endpoint": "Use this to override the default endpoint URL constructed from the `region`.\n",

		"dynamodb_endpoint": "Use this to override the default endpoint URL constructed from the `region`.\n" +
			"It's typically used to connect to dynamodb-local.",

		"kinesis_endpoint": "Use this to override the default endpoint URL constructed from the `region`.\n" +
			"It's typically used to connect to kinesalite.",

		"kms_endpoint": "Use this to override the default endpoint URL constructed from the `region`.\n",

		"iam_endpoint": "Use this to override the default endpoint URL constructed from the `region`.\n",

		"ec2_endpoint": "Use this to override the default endpoint URL constructed from the `region`.\n",

		"elb_endpoint": "Use this to override the default endpoint URL constructed from the `region`.\n",

		"rds_endpoint": "Use this to override the default endpoint URL constructed from the `region`.\n",

		"s3_endpoint": "Use this to override the default endpoint URL constructed from the `region`.\n",

		"sns_endpoint": "Use this to override the default endpoint URL constructed from the `region`.\n",

		"sqs_endpoint": "Use this to override the default endpoint URL constructed from the `region`.\n",

		"insecure": "Explicitly allow the provider to perform \"insecure\" SSL requests. If omitted," +
			"default value is `false`",

		"skip_credentials_validation": "Skip the credentials validation via STS API. " +
			"Used for AWS API implementations that do not have STS available/implemented.",

		"skip_get_ec2_platforms": "Skip getting the supported EC2 platforms. " +
			"Used by users that don't have ec2:DescribeAccountAttributes permissions.",

		"skip_region_validation": "Skip static validation of region name. " +
			"Used by users of alternative AWS-like APIs or users w/ access to regions that are not public (yet).",

		"skip_requesting_account_id": "Skip requesting the account ID. " +
			"Used for AWS API implementations that do not have IAM/STS API and/or metadata API.",

		"skip_medatadata_api_check": "Skip the AWS Metadata API check. " +
			"Used for AWS API implementations that do not have a metadata api endpoint.",

		"s3_force_path_style": "Set this to true to force the request to use path-style addressing,\n" +
			"i.e., http://s3.amazonaws.com/BUCKET/KEY. By default, the S3 client will\n" +
			"use virtual hosted bucket addressing when possible\n" +
			"(http://BUCKET.s3.amazonaws.com/KEY). Specific to the Amazon S3 service.",

		"assume_role_role_arn": "The ARN of an IAM role to assume prior to making API calls.",

		"assume_role_session_name": "The session name to use when assuming the role. If omitted," +
			" no session name is passed to the AssumeRole call.",

		"assume_role_external_id": "The external ID to use when assuming the role. If omitted," +
			" no external ID is passed to the AssumeRole call.",

		"assume_role_policy": "The permissions applied when assuming a role. You cannot use," +
			" this policy to grant further permissions that are in excess to those of the, " +
			" role that is being assumed.",
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	config := Config{
		AccessKey:               d.Get("access_key").(string),
		SecretKey:               d.Get("secret_key").(string),
		Profile:                 d.Get("profile").(string),
		CredsFilename:           d.Get("shared_credentials_file").(string),
		Token:                   d.Get("token").(string),
		Region:                  d.Get("region").(string),
		MaxRetries:              d.Get("max_retries").(int),
		Insecure:                d.Get("insecure").(bool),
		SkipCredsValidation:     d.Get("skip_credentials_validation").(bool),
		SkipGetEC2Platforms:     d.Get("skip_get_ec2_platforms").(bool),
		SkipRegionValidation:    d.Get("skip_region_validation").(bool),
		SkipRequestingAccountId: d.Get("skip_requesting_account_id").(bool),
		SkipMetadataApiCheck:    d.Get("skip_metadata_api_check").(bool),
		S3ForcePathStyle:        d.Get("s3_force_path_style").(bool),
	}

	assumeRoleList := d.Get("assume_role").(*schema.Set).List()
	if len(assumeRoleList) == 1 {
		assumeRole := assumeRoleList[0].(map[string]interface{})
		config.AssumeRoleARN = assumeRole["role_arn"].(string)
		config.AssumeRoleSessionName = assumeRole["session_name"].(string)
		config.AssumeRoleExternalID = assumeRole["external_id"].(string)

		if v := assumeRole["policy"].(string); v != "" {
			config.AssumeRolePolicy = v
		}

		log.Printf("[INFO] assume_role configuration set: (ARN: %q, SessionID: %q, ExternalID: %q, Policy: %q)",
			config.AssumeRoleARN, config.AssumeRoleSessionName, config.AssumeRoleExternalID, config.AssumeRolePolicy)
	} else {
		log.Printf("[INFO] No assume_role block read from configuration")
	}

	endpointsSet := d.Get("endpoints").(*schema.Set)

	for _, endpointsSetI := range endpointsSet.List() {
		endpoints := endpointsSetI.(map[string]interface{})
		config.CloudFormationEndpoint = endpoints["cloudformation"].(string)
		config.CloudWatchEndpoint = endpoints["cloudwatch"].(string)
		config.CloudWatchEventsEndpoint = endpoints["cloudwatchevents"].(string)
		config.CloudWatchLogsEndpoint = endpoints["cloudwatchlogs"].(string)
		config.DeviceFarmEndpoint = endpoints["devicefarm"].(string)
		config.DynamoDBEndpoint = endpoints["dynamodb"].(string)
		config.Ec2Endpoint = endpoints["ec2"].(string)
		config.ElbEndpoint = endpoints["elb"].(string)
		config.IamEndpoint = endpoints["iam"].(string)
		config.KinesisEndpoint = endpoints["kinesis"].(string)
		config.KmsEndpoint = endpoints["kms"].(string)
		config.RdsEndpoint = endpoints["rds"].(string)
		config.S3Endpoint = endpoints["s3"].(string)
		config.SnsEndpoint = endpoints["sns"].(string)
		config.SqsEndpoint = endpoints["sqs"].(string)
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

func assumeRoleSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeSet,
		Optional: true,
		MaxItems: 1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"role_arn": {
					Type:        schema.TypeString,
					Optional:    true,
					Description: descriptions["assume_role_role_arn"],
				},

				"session_name": {
					Type:        schema.TypeString,
					Optional:    true,
					Description: descriptions["assume_role_session_name"],
				},

				"external_id": {
					Type:        schema.TypeString,
					Optional:    true,
					Description: descriptions["assume_role_external_id"],
				},

				"policy": {
					Type:        schema.TypeString,
					Optional:    true,
					Description: descriptions["assume_role_policy"],
				},
			},
		},
	}
}

func endpointsSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeSet,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"cloudwatch": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["cloudwatch_endpoint"],
				},
				"cloudwatchevents": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["cloudwatchevents_endpoint"],
				},
				"cloudwatchlogs": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["cloudwatchlogs_endpoint"],
				},
				"cloudformation": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["cloudformation_endpoint"],
				},
				"devicefarm": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["devicefarm_endpoint"],
				},
				"dynamodb": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["dynamodb_endpoint"],
				},
				"iam": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["iam_endpoint"],
				},

				"ec2": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["ec2_endpoint"],
				},

				"elb": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["elb_endpoint"],
				},
				"kinesis": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["kinesis_endpoint"],
				},
				"kms": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["kms_endpoint"],
				},
				"rds": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["rds_endpoint"],
				},
				"s3": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["s3_endpoint"],
				},
				"sns": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["sns_endpoint"],
				},
				"sqs": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["sqs_endpoint"],
				},
			},
		},
		Set: endpointsToHash,
	}
}

func endpointsToHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", m["cloudwatch"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["cloudwatchevents"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["cloudwatchlogs"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["cloudformation"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["devicefarm"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["dynamodb"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["iam"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["ec2"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["elb"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["kinesis"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["kms"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["rds"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["s3"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["sns"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["sqs"].(string)))

	return hashcode.String(buf.String())
}
