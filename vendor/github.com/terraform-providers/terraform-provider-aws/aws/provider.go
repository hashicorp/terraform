package aws

import (
	"bytes"
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/mutexkv"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	homedir "github.com/mitchellh/go-homedir"
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
			"aws_acm_certificate":                  dataSourceAwsAcmCertificate(),
			"aws_acmpca_certificate_authority":     dataSourceAwsAcmpcaCertificateAuthority(),
			"aws_ami":                              dataSourceAwsAmi(),
			"aws_ami_ids":                          dataSourceAwsAmiIds(),
			"aws_api_gateway_resource":             dataSourceAwsApiGatewayResource(),
			"aws_api_gateway_rest_api":             dataSourceAwsApiGatewayRestApi(),
			"aws_arn":                              dataSourceAwsArn(),
			"aws_autoscaling_groups":               dataSourceAwsAutoscalingGroups(),
			"aws_availability_zone":                dataSourceAwsAvailabilityZone(),
			"aws_availability_zones":               dataSourceAwsAvailabilityZones(),
			"aws_batch_compute_environment":        dataSourceAwsBatchComputeEnvironment(),
			"aws_batch_job_queue":                  dataSourceAwsBatchJobQueue(),
			"aws_billing_service_account":          dataSourceAwsBillingServiceAccount(),
			"aws_caller_identity":                  dataSourceAwsCallerIdentity(),
			"aws_canonical_user_id":                dataSourceAwsCanonicalUserId(),
			"aws_cloudformation_export":            dataSourceAwsCloudFormationExport(),
			"aws_cloudformation_stack":             dataSourceAwsCloudFormationStack(),
			"aws_cloudhsm_v2_cluster":              dataSourceCloudHsm2Cluster(),
			"aws_cloudtrail_service_account":       dataSourceAwsCloudTrailServiceAccount(),
			"aws_cloudwatch_log_group":             dataSourceAwsCloudwatchLogGroup(),
			"aws_cognito_user_pools":               dataSourceAwsCognitoUserPools(),
			"aws_codecommit_repository":            dataSourceAwsCodeCommitRepository(),
			"aws_db_cluster_snapshot":              dataSourceAwsDbClusterSnapshot(),
			"aws_db_event_categories":              dataSourceAwsDbEventCategories(),
			"aws_db_instance":                      dataSourceAwsDbInstance(),
			"aws_db_snapshot":                      dataSourceAwsDbSnapshot(),
			"aws_dx_gateway":                       dataSourceAwsDxGateway(),
			"aws_dynamodb_table":                   dataSourceAwsDynamoDbTable(),
			"aws_ebs_snapshot":                     dataSourceAwsEbsSnapshot(),
			"aws_ebs_snapshot_ids":                 dataSourceAwsEbsSnapshotIds(),
			"aws_ebs_volume":                       dataSourceAwsEbsVolume(),
			"aws_ecr_repository":                   dataSourceAwsEcrRepository(),
			"aws_ecs_cluster":                      dataSourceAwsEcsCluster(),
			"aws_ecs_container_definition":         dataSourceAwsEcsContainerDefinition(),
			"aws_ecs_service":                      dataSourceAwsEcsService(),
			"aws_ecs_task_definition":              dataSourceAwsEcsTaskDefinition(),
			"aws_efs_file_system":                  dataSourceAwsEfsFileSystem(),
			"aws_efs_mount_target":                 dataSourceAwsEfsMountTarget(),
			"aws_eip":                              dataSourceAwsEip(),
			"aws_eks_cluster":                      dataSourceAwsEksCluster(),
			"aws_elastic_beanstalk_hosted_zone":    dataSourceAwsElasticBeanstalkHostedZone(),
			"aws_elastic_beanstalk_solution_stack": dataSourceAwsElasticBeanstalkSolutionStack(),
			"aws_elasticache_cluster":              dataSourceAwsElastiCacheCluster(),
			"aws_elb":                              dataSourceAwsElb(),
			"aws_elasticache_replication_group":    dataSourceAwsElasticacheReplicationGroup(),
			"aws_elb_hosted_zone_id":               dataSourceAwsElbHostedZoneId(),
			"aws_elb_service_account":              dataSourceAwsElbServiceAccount(),
			"aws_glue_script":                      dataSourceAwsGlueScript(),
			"aws_iam_account_alias":                dataSourceAwsIamAccountAlias(),
			"aws_iam_group":                        dataSourceAwsIAMGroup(),
			"aws_iam_instance_profile":             dataSourceAwsIAMInstanceProfile(),
			"aws_iam_policy":                       dataSourceAwsIAMPolicy(),
			"aws_iam_policy_document":              dataSourceAwsIamPolicyDocument(),
			"aws_iam_role":                         dataSourceAwsIAMRole(),
			"aws_iam_server_certificate":           dataSourceAwsIAMServerCertificate(),
			"aws_iam_user":                         dataSourceAwsIAMUser(),
			"aws_internet_gateway":                 dataSourceAwsInternetGateway(),
			"aws_iot_endpoint":                     dataSourceAwsIotEndpoint(),
			"aws_inspector_rules_packages":         dataSourceAwsInspectorRulesPackages(),
			"aws_instance":                         dataSourceAwsInstance(),
			"aws_instances":                        dataSourceAwsInstances(),
			"aws_ip_ranges":                        dataSourceAwsIPRanges(),
			"aws_kinesis_stream":                   dataSourceAwsKinesisStream(),
			"aws_kms_alias":                        dataSourceAwsKmsAlias(),
			"aws_kms_ciphertext":                   dataSourceAwsKmsCiphertext(),
			"aws_kms_key":                          dataSourceAwsKmsKey(),
			"aws_kms_secret":                       dataSourceAwsKmsSecret(),
			"aws_kms_secrets":                      dataSourceAwsKmsSecrets(),
			"aws_lambda_function":                  dataSourceAwsLambdaFunction(),
			"aws_lambda_invocation":                dataSourceAwsLambdaInvocation(),
			"aws_launch_configuration":             dataSourceAwsLaunchConfiguration(),
			"aws_launch_template":                  dataSourceAwsLaunchTemplate(),
			"aws_mq_broker":                        dataSourceAwsMqBroker(),
			"aws_nat_gateway":                      dataSourceAwsNatGateway(),
			"aws_network_acls":                     dataSourceAwsNetworkAcls(),
			"aws_network_interface":                dataSourceAwsNetworkInterface(),
			"aws_network_interfaces":               dataSourceAwsNetworkInterfaces(),
			"aws_partition":                        dataSourceAwsPartition(),
			"aws_prefix_list":                      dataSourceAwsPrefixList(),
			"aws_pricing_product":                  dataSourceAwsPricingProduct(),
			"aws_rds_cluster":                      dataSourceAwsRdsCluster(),
			"aws_redshift_cluster":                 dataSourceAwsRedshiftCluster(),
			"aws_redshift_service_account":         dataSourceAwsRedshiftServiceAccount(),
			"aws_region":                           dataSourceAwsRegion(),
			"aws_route":                            dataSourceAwsRoute(),
			"aws_route_table":                      dataSourceAwsRouteTable(),
			"aws_route_tables":                     dataSourceAwsRouteTables(),
			"aws_route53_zone":                     dataSourceAwsRoute53Zone(),
			"aws_s3_bucket":                        dataSourceAwsS3Bucket(),
			"aws_s3_bucket_object":                 dataSourceAwsS3BucketObject(),
			"aws_secretsmanager_secret":            dataSourceAwsSecretsManagerSecret(),
			"aws_secretsmanager_secret_version":    dataSourceAwsSecretsManagerSecretVersion(),
			"aws_sns_topic":                        dataSourceAwsSnsTopic(),
			"aws_sqs_queue":                        dataSourceAwsSqsQueue(),
			"aws_ssm_parameter":                    dataSourceAwsSsmParameter(),
			"aws_storagegateway_local_disk":        dataSourceAwsStorageGatewayLocalDisk(),
			"aws_subnet":                           dataSourceAwsSubnet(),
			"aws_subnet_ids":                       dataSourceAwsSubnetIDs(),
			"aws_vpcs":                             dataSourceAwsVpcs(),
			"aws_security_group":                   dataSourceAwsSecurityGroup(),
			"aws_security_groups":                  dataSourceAwsSecurityGroups(),
			"aws_vpc":                              dataSourceAwsVpc(),
			"aws_vpc_dhcp_options":                 dataSourceAwsVpcDhcpOptions(),
			"aws_vpc_endpoint":                     dataSourceAwsVpcEndpoint(),
			"aws_vpc_endpoint_service":             dataSourceAwsVpcEndpointService(),
			"aws_vpc_peering_connection":           dataSourceAwsVpcPeeringConnection(),
			"aws_vpn_gateway":                      dataSourceAwsVpnGateway(),
			"aws_workspaces_bundle":                dataSourceAwsWorkspaceBundle(),

			// Adding the Aliases for the ALB -> LB Rename
			"aws_lb":               dataSourceAwsLb(),
			"aws_alb":              dataSourceAwsLb(),
			"aws_lb_listener":      dataSourceAwsLbListener(),
			"aws_alb_listener":     dataSourceAwsLbListener(),
			"aws_lb_target_group":  dataSourceAwsLbTargetGroup(),
			"aws_alb_target_group": dataSourceAwsLbTargetGroup(),
		},

		ResourcesMap: map[string]*schema.Resource{
			"aws_acm_certificate":                              resourceAwsAcmCertificate(),
			"aws_acm_certificate_validation":                   resourceAwsAcmCertificateValidation(),
			"aws_acmpca_certificate_authority":                 resourceAwsAcmpcaCertificateAuthority(),
			"aws_ami":                                          resourceAwsAmi(),
			"aws_ami_copy":                                     resourceAwsAmiCopy(),
			"aws_ami_from_instance":                            resourceAwsAmiFromInstance(),
			"aws_ami_launch_permission":                        resourceAwsAmiLaunchPermission(),
			"aws_api_gateway_account":                          resourceAwsApiGatewayAccount(),
			"aws_api_gateway_api_key":                          resourceAwsApiGatewayApiKey(),
			"aws_api_gateway_authorizer":                       resourceAwsApiGatewayAuthorizer(),
			"aws_api_gateway_base_path_mapping":                resourceAwsApiGatewayBasePathMapping(),
			"aws_api_gateway_client_certificate":               resourceAwsApiGatewayClientCertificate(),
			"aws_api_gateway_deployment":                       resourceAwsApiGatewayDeployment(),
			"aws_api_gateway_documentation_part":               resourceAwsApiGatewayDocumentationPart(),
			"aws_api_gateway_documentation_version":            resourceAwsApiGatewayDocumentationVersion(),
			"aws_api_gateway_domain_name":                      resourceAwsApiGatewayDomainName(),
			"aws_api_gateway_gateway_response":                 resourceAwsApiGatewayGatewayResponse(),
			"aws_api_gateway_integration":                      resourceAwsApiGatewayIntegration(),
			"aws_api_gateway_integration_response":             resourceAwsApiGatewayIntegrationResponse(),
			"aws_api_gateway_method":                           resourceAwsApiGatewayMethod(),
			"aws_api_gateway_method_response":                  resourceAwsApiGatewayMethodResponse(),
			"aws_api_gateway_method_settings":                  resourceAwsApiGatewayMethodSettings(),
			"aws_api_gateway_model":                            resourceAwsApiGatewayModel(),
			"aws_api_gateway_request_validator":                resourceAwsApiGatewayRequestValidator(),
			"aws_api_gateway_resource":                         resourceAwsApiGatewayResource(),
			"aws_api_gateway_rest_api":                         resourceAwsApiGatewayRestApi(),
			"aws_api_gateway_stage":                            resourceAwsApiGatewayStage(),
			"aws_api_gateway_usage_plan":                       resourceAwsApiGatewayUsagePlan(),
			"aws_api_gateway_usage_plan_key":                   resourceAwsApiGatewayUsagePlanKey(),
			"aws_api_gateway_vpc_link":                         resourceAwsApiGatewayVpcLink(),
			"aws_app_cookie_stickiness_policy":                 resourceAwsAppCookieStickinessPolicy(),
			"aws_appautoscaling_target":                        resourceAwsAppautoscalingTarget(),
			"aws_appautoscaling_policy":                        resourceAwsAppautoscalingPolicy(),
			"aws_appautoscaling_scheduled_action":              resourceAwsAppautoscalingScheduledAction(),
			"aws_appsync_api_key":                              resourceAwsAppsyncApiKey(),
			"aws_appsync_datasource":                           resourceAwsAppsyncDatasource(),
			"aws_appsync_graphql_api":                          resourceAwsAppsyncGraphqlApi(),
			"aws_athena_database":                              resourceAwsAthenaDatabase(),
			"aws_athena_named_query":                           resourceAwsAthenaNamedQuery(),
			"aws_autoscaling_attachment":                       resourceAwsAutoscalingAttachment(),
			"aws_autoscaling_group":                            resourceAwsAutoscalingGroup(),
			"aws_autoscaling_lifecycle_hook":                   resourceAwsAutoscalingLifecycleHook(),
			"aws_autoscaling_notification":                     resourceAwsAutoscalingNotification(),
			"aws_autoscaling_policy":                           resourceAwsAutoscalingPolicy(),
			"aws_autoscaling_schedule":                         resourceAwsAutoscalingSchedule(),
			"aws_budgets_budget":                               resourceAwsBudgetsBudget(),
			"aws_cloud9_environment_ec2":                       resourceAwsCloud9EnvironmentEc2(),
			"aws_cloudformation_stack":                         resourceAwsCloudFormationStack(),
			"aws_cloudfront_distribution":                      resourceAwsCloudFrontDistribution(),
			"aws_cloudfront_origin_access_identity":            resourceAwsCloudFrontOriginAccessIdentity(),
			"aws_cloudfront_public_key":                        resourceAwsCloudFrontPublicKey(),
			"aws_cloudtrail":                                   resourceAwsCloudTrail(),
			"aws_cloudwatch_event_permission":                  resourceAwsCloudWatchEventPermission(),
			"aws_cloudwatch_event_rule":                        resourceAwsCloudWatchEventRule(),
			"aws_cloudwatch_event_target":                      resourceAwsCloudWatchEventTarget(),
			"aws_cloudwatch_log_destination":                   resourceAwsCloudWatchLogDestination(),
			"aws_cloudwatch_log_destination_policy":            resourceAwsCloudWatchLogDestinationPolicy(),
			"aws_cloudwatch_log_group":                         resourceAwsCloudWatchLogGroup(),
			"aws_cloudwatch_log_metric_filter":                 resourceAwsCloudWatchLogMetricFilter(),
			"aws_cloudwatch_log_resource_policy":               resourceAwsCloudWatchLogResourcePolicy(),
			"aws_cloudwatch_log_stream":                        resourceAwsCloudWatchLogStream(),
			"aws_cloudwatch_log_subscription_filter":           resourceAwsCloudwatchLogSubscriptionFilter(),
			"aws_config_aggregate_authorization":               resourceAwsConfigAggregateAuthorization(),
			"aws_config_config_rule":                           resourceAwsConfigConfigRule(),
			"aws_config_configuration_aggregator":              resourceAwsConfigConfigurationAggregator(),
			"aws_config_configuration_recorder":                resourceAwsConfigConfigurationRecorder(),
			"aws_config_configuration_recorder_status":         resourceAwsConfigConfigurationRecorderStatus(),
			"aws_config_delivery_channel":                      resourceAwsConfigDeliveryChannel(),
			"aws_cognito_identity_pool":                        resourceAwsCognitoIdentityPool(),
			"aws_cognito_identity_pool_roles_attachment":       resourceAwsCognitoIdentityPoolRolesAttachment(),
			"aws_cognito_identity_provider":                    resourceAwsCognitoIdentityProvider(),
			"aws_cognito_user_group":                           resourceAwsCognitoUserGroup(),
			"aws_cognito_user_pool":                            resourceAwsCognitoUserPool(),
			"aws_cognito_user_pool_client":                     resourceAwsCognitoUserPoolClient(),
			"aws_cognito_user_pool_domain":                     resourceAwsCognitoUserPoolDomain(),
			"aws_cloudhsm_v2_cluster":                          resourceAwsCloudHsm2Cluster(),
			"aws_cloudhsm_v2_hsm":                              resourceAwsCloudHsm2Hsm(),
			"aws_cognito_resource_server":                      resourceAwsCognitoResourceServer(),
			"aws_cloudwatch_metric_alarm":                      resourceAwsCloudWatchMetricAlarm(),
			"aws_cloudwatch_dashboard":                         resourceAwsCloudWatchDashboard(),
			"aws_codedeploy_app":                               resourceAwsCodeDeployApp(),
			"aws_codedeploy_deployment_config":                 resourceAwsCodeDeployDeploymentConfig(),
			"aws_codedeploy_deployment_group":                  resourceAwsCodeDeployDeploymentGroup(),
			"aws_codecommit_repository":                        resourceAwsCodeCommitRepository(),
			"aws_codecommit_trigger":                           resourceAwsCodeCommitTrigger(),
			"aws_codebuild_project":                            resourceAwsCodeBuildProject(),
			"aws_codebuild_webhook":                            resourceAwsCodeBuildWebhook(),
			"aws_codepipeline":                                 resourceAwsCodePipeline(),
			"aws_codepipeline_webhook":                         resourceAwsCodePipelineWebhook(),
			"aws_customer_gateway":                             resourceAwsCustomerGateway(),
			"aws_dax_cluster":                                  resourceAwsDaxCluster(),
			"aws_dax_parameter_group":                          resourceAwsDaxParameterGroup(),
			"aws_dax_subnet_group":                             resourceAwsDaxSubnetGroup(),
			"aws_db_cluster_snapshot":                          resourceAwsDbClusterSnapshot(),
			"aws_db_event_subscription":                        resourceAwsDbEventSubscription(),
			"aws_db_instance":                                  resourceAwsDbInstance(),
			"aws_db_option_group":                              resourceAwsDbOptionGroup(),
			"aws_db_parameter_group":                           resourceAwsDbParameterGroup(),
			"aws_db_security_group":                            resourceAwsDbSecurityGroup(),
			"aws_db_snapshot":                                  resourceAwsDbSnapshot(),
			"aws_db_subnet_group":                              resourceAwsDbSubnetGroup(),
			"aws_devicefarm_project":                           resourceAwsDevicefarmProject(),
			"aws_directory_service_directory":                  resourceAwsDirectoryServiceDirectory(),
			"aws_directory_service_conditional_forwarder":      resourceAwsDirectoryServiceConditionalForwarder(),
			"aws_dms_certificate":                              resourceAwsDmsCertificate(),
			"aws_dms_endpoint":                                 resourceAwsDmsEndpoint(),
			"aws_dms_replication_instance":                     resourceAwsDmsReplicationInstance(),
			"aws_dms_replication_subnet_group":                 resourceAwsDmsReplicationSubnetGroup(),
			"aws_dms_replication_task":                         resourceAwsDmsReplicationTask(),
			"aws_dx_bgp_peer":                                  resourceAwsDxBgpPeer(),
			"aws_dx_connection":                                resourceAwsDxConnection(),
			"aws_dx_connection_association":                    resourceAwsDxConnectionAssociation(),
			"aws_dx_gateway":                                   resourceAwsDxGateway(),
			"aws_dx_gateway_association":                       resourceAwsDxGatewayAssociation(),
			"aws_dx_hosted_private_virtual_interface":          resourceAwsDxHostedPrivateVirtualInterface(),
			"aws_dx_hosted_private_virtual_interface_accepter": resourceAwsDxHostedPrivateVirtualInterfaceAccepter(),
			"aws_dx_hosted_public_virtual_interface":           resourceAwsDxHostedPublicVirtualInterface(),
			"aws_dx_hosted_public_virtual_interface_accepter":  resourceAwsDxHostedPublicVirtualInterfaceAccepter(),
			"aws_dx_lag":                                       resourceAwsDxLag(),
			"aws_dx_private_virtual_interface":                 resourceAwsDxPrivateVirtualInterface(),
			"aws_dx_public_virtual_interface":                  resourceAwsDxPublicVirtualInterface(),
			"aws_dynamodb_table":                               resourceAwsDynamoDbTable(),
			"aws_dynamodb_table_item":                          resourceAwsDynamoDbTableItem(),
			"aws_dynamodb_global_table":                        resourceAwsDynamoDbGlobalTable(),
			"aws_ec2_fleet":                                    resourceAwsEc2Fleet(),
			"aws_ebs_snapshot":                                 resourceAwsEbsSnapshot(),
			"aws_ebs_snapshot_copy":                            resourceAwsEbsSnapshotCopy(),
			"aws_ebs_volume":                                   resourceAwsEbsVolume(),
			"aws_ecr_lifecycle_policy":                         resourceAwsEcrLifecyclePolicy(),
			"aws_ecr_repository":                               resourceAwsEcrRepository(),
			"aws_ecr_repository_policy":                        resourceAwsEcrRepositoryPolicy(),
			"aws_ecs_cluster":                                  resourceAwsEcsCluster(),
			"aws_ecs_service":                                  resourceAwsEcsService(),
			"aws_ecs_task_definition":                          resourceAwsEcsTaskDefinition(),
			"aws_efs_file_system":                              resourceAwsEfsFileSystem(),
			"aws_efs_mount_target":                             resourceAwsEfsMountTarget(),
			"aws_egress_only_internet_gateway":                 resourceAwsEgressOnlyInternetGateway(),
			"aws_eip":                                          resourceAwsEip(),
			"aws_eip_association":                              resourceAwsEipAssociation(),
			"aws_eks_cluster":                                  resourceAwsEksCluster(),
			"aws_elasticache_cluster":                          resourceAwsElasticacheCluster(),
			"aws_elasticache_parameter_group":                  resourceAwsElasticacheParameterGroup(),
			"aws_elasticache_replication_group":                resourceAwsElasticacheReplicationGroup(),
			"aws_elasticache_security_group":                   resourceAwsElasticacheSecurityGroup(),
			"aws_elasticache_subnet_group":                     resourceAwsElasticacheSubnetGroup(),
			"aws_elastic_beanstalk_application":                resourceAwsElasticBeanstalkApplication(),
			"aws_elastic_beanstalk_application_version":        resourceAwsElasticBeanstalkApplicationVersion(),
			"aws_elastic_beanstalk_configuration_template":     resourceAwsElasticBeanstalkConfigurationTemplate(),
			"aws_elastic_beanstalk_environment":                resourceAwsElasticBeanstalkEnvironment(),
			"aws_elasticsearch_domain":                         resourceAwsElasticSearchDomain(),
			"aws_elasticsearch_domain_policy":                  resourceAwsElasticSearchDomainPolicy(),
			"aws_elastictranscoder_pipeline":                   resourceAwsElasticTranscoderPipeline(),
			"aws_elastictranscoder_preset":                     resourceAwsElasticTranscoderPreset(),
			"aws_elb":                                          resourceAwsElb(),
			"aws_elb_attachment":                               resourceAwsElbAttachment(),
			"aws_emr_cluster":                                  resourceAwsEMRCluster(),
			"aws_emr_instance_group":                           resourceAwsEMRInstanceGroup(),
			"aws_emr_security_configuration":                   resourceAwsEMRSecurityConfiguration(),
			"aws_flow_log":                                     resourceAwsFlowLog(),
			"aws_gamelift_alias":                               resourceAwsGameliftAlias(),
			"aws_gamelift_build":                               resourceAwsGameliftBuild(),
			"aws_gamelift_fleet":                               resourceAwsGameliftFleet(),
			"aws_glacier_vault":                                resourceAwsGlacierVault(),
			"aws_glue_catalog_database":                        resourceAwsGlueCatalogDatabase(),
			"aws_glue_catalog_table":                           resourceAwsGlueCatalogTable(),
			"aws_glue_classifier":                              resourceAwsGlueClassifier(),
			"aws_glue_connection":                              resourceAwsGlueConnection(),
			"aws_glue_crawler":                                 resourceAwsGlueCrawler(),
			"aws_glue_job":                                     resourceAwsGlueJob(),
			"aws_glue_trigger":                                 resourceAwsGlueTrigger(),
			"aws_guardduty_detector":                           resourceAwsGuardDutyDetector(),
			"aws_guardduty_ipset":                              resourceAwsGuardDutyIpset(),
			"aws_guardduty_member":                             resourceAwsGuardDutyMember(),
			"aws_guardduty_threatintelset":                     resourceAwsGuardDutyThreatintelset(),
			"aws_iam_access_key":                               resourceAwsIamAccessKey(),
			"aws_iam_account_alias":                            resourceAwsIamAccountAlias(),
			"aws_iam_account_password_policy":                  resourceAwsIamAccountPasswordPolicy(),
			"aws_iam_group_policy":                             resourceAwsIamGroupPolicy(),
			"aws_iam_group":                                    resourceAwsIamGroup(),
			"aws_iam_group_membership":                         resourceAwsIamGroupMembership(),
			"aws_iam_group_policy_attachment":                  resourceAwsIamGroupPolicyAttachment(),
			"aws_iam_instance_profile":                         resourceAwsIamInstanceProfile(),
			"aws_iam_openid_connect_provider":                  resourceAwsIamOpenIDConnectProvider(),
			"aws_iam_policy":                                   resourceAwsIamPolicy(),
			"aws_iam_policy_attachment":                        resourceAwsIamPolicyAttachment(),
			"aws_iam_role_policy_attachment":                   resourceAwsIamRolePolicyAttachment(),
			"aws_iam_role_policy":                              resourceAwsIamRolePolicy(),
			"aws_iam_role":                                     resourceAwsIamRole(),
			"aws_iam_saml_provider":                            resourceAwsIamSamlProvider(),
			"aws_iam_server_certificate":                       resourceAwsIAMServerCertificate(),
			"aws_iam_service_linked_role":                      resourceAwsIamServiceLinkedRole(),
			"aws_iam_user_group_membership":                    resourceAwsIamUserGroupMembership(),
			"aws_iam_user_policy_attachment":                   resourceAwsIamUserPolicyAttachment(),
			"aws_iam_user_policy":                              resourceAwsIamUserPolicy(),
			"aws_iam_user_ssh_key":                             resourceAwsIamUserSshKey(),
			"aws_iam_user":                                     resourceAwsIamUser(),
			"aws_iam_user_login_profile":                       resourceAwsIamUserLoginProfile(),
			"aws_inspector_assessment_target":                  resourceAWSInspectorAssessmentTarget(),
			"aws_inspector_assessment_template":                resourceAWSInspectorAssessmentTemplate(),
			"aws_inspector_resource_group":                     resourceAWSInspectorResourceGroup(),
			"aws_instance":                                     resourceAwsInstance(),
			"aws_internet_gateway":                             resourceAwsInternetGateway(),
			"aws_iot_certificate":                              resourceAwsIotCertificate(),
			"aws_iot_policy":                                   resourceAwsIotPolicy(),
			"aws_iot_thing":                                    resourceAwsIotThing(),
			"aws_iot_thing_type":                               resourceAwsIotThingType(),
			"aws_iot_topic_rule":                               resourceAwsIotTopicRule(),
			"aws_key_pair":                                     resourceAwsKeyPair(),
			"aws_kinesis_firehose_delivery_stream":             resourceAwsKinesisFirehoseDeliveryStream(),
			"aws_kinesis_stream":                               resourceAwsKinesisStream(),
			"aws_kms_alias":                                    resourceAwsKmsAlias(),
			"aws_kms_grant":                                    resourceAwsKmsGrant(),
			"aws_kms_key":                                      resourceAwsKmsKey(),
			"aws_lambda_function":                              resourceAwsLambdaFunction(),
			"aws_lambda_event_source_mapping":                  resourceAwsLambdaEventSourceMapping(),
			"aws_lambda_alias":                                 resourceAwsLambdaAlias(),
			"aws_lambda_permission":                            resourceAwsLambdaPermission(),
			"aws_launch_configuration":                         resourceAwsLaunchConfiguration(),
			"aws_launch_template":                              resourceAwsLaunchTemplate(),
			"aws_lightsail_domain":                             resourceAwsLightsailDomain(),
			"aws_lightsail_instance":                           resourceAwsLightsailInstance(),
			"aws_lightsail_key_pair":                           resourceAwsLightsailKeyPair(),
			"aws_lightsail_static_ip":                          resourceAwsLightsailStaticIp(),
			"aws_lightsail_static_ip_attachment":               resourceAwsLightsailStaticIpAttachment(),
			"aws_lb_cookie_stickiness_policy":                  resourceAwsLBCookieStickinessPolicy(),
			"aws_load_balancer_policy":                         resourceAwsLoadBalancerPolicy(),
			"aws_load_balancer_backend_server_policy":          resourceAwsLoadBalancerBackendServerPolicies(),
			"aws_load_balancer_listener_policy":                resourceAwsLoadBalancerListenerPolicies(),
			"aws_lb_ssl_negotiation_policy":                    resourceAwsLBSSLNegotiationPolicy(),
			"aws_macie_member_account_association":             resourceAwsMacieMemberAccountAssociation(),
			"aws_macie_s3_bucket_association":                  resourceAwsMacieS3BucketAssociation(),
			"aws_main_route_table_association":                 resourceAwsMainRouteTableAssociation(),
			"aws_mq_broker":                                    resourceAwsMqBroker(),
			"aws_mq_configuration":                             resourceAwsMqConfiguration(),
			"aws_media_store_container":                        resourceAwsMediaStoreContainer(),
			"aws_media_store_container_policy":                 resourceAwsMediaStoreContainerPolicy(),
			"aws_nat_gateway":                                  resourceAwsNatGateway(),
			"aws_network_acl":                                  resourceAwsNetworkAcl(),
			"aws_default_network_acl":                          resourceAwsDefaultNetworkAcl(),
			"aws_neptune_cluster":                              resourceAwsNeptuneCluster(),
			"aws_neptune_cluster_instance":                     resourceAwsNeptuneClusterInstance(),
			"aws_neptune_cluster_parameter_group":              resourceAwsNeptuneClusterParameterGroup(),
			"aws_neptune_cluster_snapshot":                     resourceAwsNeptuneClusterSnapshot(),
			"aws_neptune_event_subscription":                   resourceAwsNeptuneEventSubscription(),
			"aws_neptune_parameter_group":                      resourceAwsNeptuneParameterGroup(),
			"aws_neptune_subnet_group":                         resourceAwsNeptuneSubnetGroup(),
			"aws_network_acl_rule":                             resourceAwsNetworkAclRule(),
			"aws_network_interface":                            resourceAwsNetworkInterface(),
			"aws_network_interface_attachment":                 resourceAwsNetworkInterfaceAttachment(),
			"aws_opsworks_application":                         resourceAwsOpsworksApplication(),
			"aws_opsworks_stack":                               resourceAwsOpsworksStack(),
			"aws_opsworks_java_app_layer":                      resourceAwsOpsworksJavaAppLayer(),
			"aws_opsworks_haproxy_layer":                       resourceAwsOpsworksHaproxyLayer(),
			"aws_opsworks_static_web_layer":                    resourceAwsOpsworksStaticWebLayer(),
			"aws_opsworks_php_app_layer":                       resourceAwsOpsworksPhpAppLayer(),
			"aws_opsworks_rails_app_layer":                     resourceAwsOpsworksRailsAppLayer(),
			"aws_opsworks_nodejs_app_layer":                    resourceAwsOpsworksNodejsAppLayer(),
			"aws_opsworks_memcached_layer":                     resourceAwsOpsworksMemcachedLayer(),
			"aws_opsworks_mysql_layer":                         resourceAwsOpsworksMysqlLayer(),
			"aws_opsworks_ganglia_layer":                       resourceAwsOpsworksGangliaLayer(),
			"aws_opsworks_custom_layer":                        resourceAwsOpsworksCustomLayer(),
			"aws_opsworks_instance":                            resourceAwsOpsworksInstance(),
			"aws_opsworks_user_profile":                        resourceAwsOpsworksUserProfile(),
			"aws_opsworks_permission":                          resourceAwsOpsworksPermission(),
			"aws_opsworks_rds_db_instance":                     resourceAwsOpsworksRdsDbInstance(),
			"aws_organizations_organization":                   resourceAwsOrganizationsOrganization(),
			"aws_organizations_account":                        resourceAwsOrganizationsAccount(),
			"aws_organizations_policy":                         resourceAwsOrganizationsPolicy(),
			"aws_organizations_policy_attachment":              resourceAwsOrganizationsPolicyAttachment(),
			"aws_placement_group":                              resourceAwsPlacementGroup(),
			"aws_proxy_protocol_policy":                        resourceAwsProxyProtocolPolicy(),
			"aws_rds_cluster":                                  resourceAwsRDSCluster(),
			"aws_rds_cluster_instance":                         resourceAwsRDSClusterInstance(),
			"aws_rds_cluster_parameter_group":                  resourceAwsRDSClusterParameterGroup(),
			"aws_redshift_cluster":                             resourceAwsRedshiftCluster(),
			"aws_redshift_security_group":                      resourceAwsRedshiftSecurityGroup(),
			"aws_redshift_parameter_group":                     resourceAwsRedshiftParameterGroup(),
			"aws_redshift_subnet_group":                        resourceAwsRedshiftSubnetGroup(),
			"aws_redshift_snapshot_copy_grant":                 resourceAwsRedshiftSnapshotCopyGrant(),
			"aws_redshift_event_subscription":                  resourceAwsRedshiftEventSubscription(),
			"aws_route53_delegation_set":                       resourceAwsRoute53DelegationSet(),
			"aws_route53_query_log":                            resourceAwsRoute53QueryLog(),
			"aws_route53_record":                               resourceAwsRoute53Record(),
			"aws_route53_zone_association":                     resourceAwsRoute53ZoneAssociation(),
			"aws_route53_zone":                                 resourceAwsRoute53Zone(),
			"aws_route53_health_check":                         resourceAwsRoute53HealthCheck(),
			"aws_route":                                        resourceAwsRoute(),
			"aws_route_table":                                  resourceAwsRouteTable(),
			"aws_default_route_table":                          resourceAwsDefaultRouteTable(),
			"aws_route_table_association":                      resourceAwsRouteTableAssociation(),
			"aws_secretsmanager_secret":                        resourceAwsSecretsManagerSecret(),
			"aws_secretsmanager_secret_version":                resourceAwsSecretsManagerSecretVersion(),
			"aws_ses_active_receipt_rule_set":                  resourceAwsSesActiveReceiptRuleSet(),
			"aws_ses_domain_identity":                          resourceAwsSesDomainIdentity(),
			"aws_ses_domain_identity_verification":             resourceAwsSesDomainIdentityVerification(),
			"aws_ses_domain_dkim":                              resourceAwsSesDomainDkim(),
			"aws_ses_domain_mail_from":                         resourceAwsSesDomainMailFrom(),
			"aws_ses_receipt_filter":                           resourceAwsSesReceiptFilter(),
			"aws_ses_receipt_rule":                             resourceAwsSesReceiptRule(),
			"aws_ses_receipt_rule_set":                         resourceAwsSesReceiptRuleSet(),
			"aws_ses_configuration_set":                        resourceAwsSesConfigurationSet(),
			"aws_ses_event_destination":                        resourceAwsSesEventDestination(),
			"aws_ses_identity_notification_topic":              resourceAwsSesNotificationTopic(),
			"aws_ses_template":                                 resourceAwsSesTemplate(),
			"aws_s3_bucket":                                    resourceAwsS3Bucket(),
			"aws_s3_bucket_policy":                             resourceAwsS3BucketPolicy(),
			"aws_s3_bucket_object":                             resourceAwsS3BucketObject(),
			"aws_s3_bucket_notification":                       resourceAwsS3BucketNotification(),
			"aws_s3_bucket_metric":                             resourceAwsS3BucketMetric(),
			"aws_s3_bucket_inventory":                          resourceAwsS3BucketInventory(),
			"aws_security_group":                               resourceAwsSecurityGroup(),
			"aws_network_interface_sg_attachment":              resourceAwsNetworkInterfaceSGAttachment(),
			"aws_default_security_group":                       resourceAwsDefaultSecurityGroup(),
			"aws_security_group_rule":                          resourceAwsSecurityGroupRule(),
			"aws_servicecatalog_portfolio":                     resourceAwsServiceCatalogPortfolio(),
			"aws_service_discovery_private_dns_namespace":      resourceAwsServiceDiscoveryPrivateDnsNamespace(),
			"aws_service_discovery_public_dns_namespace":       resourceAwsServiceDiscoveryPublicDnsNamespace(),
			"aws_service_discovery_service":                    resourceAwsServiceDiscoveryService(),
			"aws_simpledb_domain":                              resourceAwsSimpleDBDomain(),
			"aws_ssm_activation":                               resourceAwsSsmActivation(),
			"aws_ssm_association":                              resourceAwsSsmAssociation(),
			"aws_ssm_document":                                 resourceAwsSsmDocument(),
			"aws_ssm_maintenance_window":                       resourceAwsSsmMaintenanceWindow(),
			"aws_ssm_maintenance_window_target":                resourceAwsSsmMaintenanceWindowTarget(),
			"aws_ssm_maintenance_window_task":                  resourceAwsSsmMaintenanceWindowTask(),
			"aws_ssm_patch_baseline":                           resourceAwsSsmPatchBaseline(),
			"aws_ssm_patch_group":                              resourceAwsSsmPatchGroup(),
			"aws_ssm_parameter":                                resourceAwsSsmParameter(),
			"aws_ssm_resource_data_sync":                       resourceAwsSsmResourceDataSync(),
			"aws_storagegateway_cache":                         resourceAwsStorageGatewayCache(),
			"aws_storagegateway_cached_iscsi_volume":           resourceAwsStorageGatewayCachedIscsiVolume(),
			"aws_storagegateway_gateway":                       resourceAwsStorageGatewayGateway(),
			"aws_storagegateway_nfs_file_share":                resourceAwsStorageGatewayNfsFileShare(),
			"aws_storagegateway_smb_file_share":                resourceAwsStorageGatewaySmbFileShare(),
			"aws_storagegateway_upload_buffer":                 resourceAwsStorageGatewayUploadBuffer(),
			"aws_storagegateway_working_storage":               resourceAwsStorageGatewayWorkingStorage(),
			"aws_spot_datafeed_subscription":                   resourceAwsSpotDataFeedSubscription(),
			"aws_spot_instance_request":                        resourceAwsSpotInstanceRequest(),
			"aws_spot_fleet_request":                           resourceAwsSpotFleetRequest(),
			"aws_sqs_queue":                                    resourceAwsSqsQueue(),
			"aws_sqs_queue_policy":                             resourceAwsSqsQueuePolicy(),
			"aws_snapshot_create_volume_permission":            resourceAwsSnapshotCreateVolumePermission(),
			"aws_sns_platform_application":                     resourceAwsSnsPlatformApplication(),
			"aws_sns_sms_preferences":                          resourceAwsSnsSmsPreferences(),
			"aws_sns_topic":                                    resourceAwsSnsTopic(),
			"aws_sns_topic_policy":                             resourceAwsSnsTopicPolicy(),
			"aws_sns_topic_subscription":                       resourceAwsSnsTopicSubscription(),
			"aws_sfn_activity":                                 resourceAwsSfnActivity(),
			"aws_sfn_state_machine":                            resourceAwsSfnStateMachine(),
			"aws_default_subnet":                               resourceAwsDefaultSubnet(),
			"aws_subnet":                                       resourceAwsSubnet(),
			"aws_swf_domain":                                   resourceAwsSwfDomain(),
			"aws_volume_attachment":                            resourceAwsVolumeAttachment(),
			"aws_vpc_dhcp_options_association":                 resourceAwsVpcDhcpOptionsAssociation(),
			"aws_default_vpc_dhcp_options":                     resourceAwsDefaultVpcDhcpOptions(),
			"aws_vpc_dhcp_options":                             resourceAwsVpcDhcpOptions(),
			"aws_vpc_peering_connection":                       resourceAwsVpcPeeringConnection(),
			"aws_vpc_peering_connection_accepter":              resourceAwsVpcPeeringConnectionAccepter(),
			"aws_vpc_peering_connection_options":               resourceAwsVpcPeeringConnectionOptions(),
			"aws_default_vpc":                                  resourceAwsDefaultVpc(),
			"aws_vpc":                                          resourceAwsVpc(),
			"aws_vpc_endpoint":                                 resourceAwsVpcEndpoint(),
			"aws_vpc_endpoint_connection_notification":         resourceAwsVpcEndpointConnectionNotification(),
			"aws_vpc_endpoint_route_table_association":         resourceAwsVpcEndpointRouteTableAssociation(),
			"aws_vpc_endpoint_subnet_association":              resourceAwsVpcEndpointSubnetAssociation(),
			"aws_vpc_endpoint_service":                         resourceAwsVpcEndpointService(),
			"aws_vpc_endpoint_service_allowed_principal":       resourceAwsVpcEndpointServiceAllowedPrincipal(),
			"aws_vpc_ipv4_cidr_block_association":              resourceAwsVpcIpv4CidrBlockAssociation(),
			"aws_vpn_connection":                               resourceAwsVpnConnection(),
			"aws_vpn_connection_route":                         resourceAwsVpnConnectionRoute(),
			"aws_vpn_gateway":                                  resourceAwsVpnGateway(),
			"aws_vpn_gateway_attachment":                       resourceAwsVpnGatewayAttachment(),
			"aws_vpn_gateway_route_propagation":                resourceAwsVpnGatewayRoutePropagation(),
			"aws_waf_byte_match_set":                           resourceAwsWafByteMatchSet(),
			"aws_waf_ipset":                                    resourceAwsWafIPSet(),
			"aws_waf_rate_based_rule":                          resourceAwsWafRateBasedRule(),
			"aws_waf_regex_match_set":                          resourceAwsWafRegexMatchSet(),
			"aws_waf_regex_pattern_set":                        resourceAwsWafRegexPatternSet(),
			"aws_waf_rule":                                     resourceAwsWafRule(),
			"aws_waf_rule_group":                               resourceAwsWafRuleGroup(),
			"aws_waf_size_constraint_set":                      resourceAwsWafSizeConstraintSet(),
			"aws_waf_web_acl":                                  resourceAwsWafWebAcl(),
			"aws_waf_xss_match_set":                            resourceAwsWafXssMatchSet(),
			"aws_waf_sql_injection_match_set":                  resourceAwsWafSqlInjectionMatchSet(),
			"aws_waf_geo_match_set":                            resourceAwsWafGeoMatchSet(),
			"aws_wafregional_byte_match_set":                   resourceAwsWafRegionalByteMatchSet(),
			"aws_wafregional_geo_match_set":                    resourceAwsWafRegionalGeoMatchSet(),
			"aws_wafregional_ipset":                            resourceAwsWafRegionalIPSet(),
			"aws_wafregional_rate_based_rule":                  resourceAwsWafRegionalRateBasedRule(),
			"aws_wafregional_regex_match_set":                  resourceAwsWafRegionalRegexMatchSet(),
			"aws_wafregional_regex_pattern_set":                resourceAwsWafRegionalRegexPatternSet(),
			"aws_wafregional_rule":                             resourceAwsWafRegionalRule(),
			"aws_wafregional_rule_group":                       resourceAwsWafRegionalRuleGroup(),
			"aws_wafregional_size_constraint_set":              resourceAwsWafRegionalSizeConstraintSet(),
			"aws_wafregional_sql_injection_match_set":          resourceAwsWafRegionalSqlInjectionMatchSet(),
			"aws_wafregional_xss_match_set":                    resourceAwsWafRegionalXssMatchSet(),
			"aws_wafregional_web_acl":                          resourceAwsWafRegionalWebAcl(),
			"aws_wafregional_web_acl_association":              resourceAwsWafRegionalWebAclAssociation(),
			"aws_batch_compute_environment":                    resourceAwsBatchComputeEnvironment(),
			"aws_batch_job_definition":                         resourceAwsBatchJobDefinition(),
			"aws_batch_job_queue":                              resourceAwsBatchJobQueue(),
			"aws_pinpoint_app":                                 resourceAwsPinpointApp(),
			"aws_pinpoint_adm_channel":                         resourceAwsPinpointADMChannel(),
			"aws_pinpoint_apns_channel":                        resourceAwsPinpointAPNSChannel(),
			"aws_pinpoint_baidu_channel":                       resourceAwsPinpointBaiduChannel(),
			"aws_pinpoint_email_channel":                       resourceAwsPinpointEmailChannel(),
			"aws_pinpoint_event_stream":                        resourceAwsPinpointEventStream(),
			"aws_pinpoint_gcm_channel":                         resourceAwsPinpointGCMChannel(),
			"aws_pinpoint_sms_channel":                         resourceAwsPinpointSMSChannel(),

			// ALBs are actually LBs because they can be type `network` or `application`
			// To avoid regressions, we will add a new resource for each and they both point
			// back to the old ALB version. IF the Terraform supported aliases for resources
			// this would be a whole lot simpler
			"aws_alb":                         resourceAwsLb(),
			"aws_lb":                          resourceAwsLb(),
			"aws_alb_listener":                resourceAwsLbListener(),
			"aws_lb_listener":                 resourceAwsLbListener(),
			"aws_alb_listener_certificate":    resourceAwsLbListenerCertificate(),
			"aws_lb_listener_certificate":     resourceAwsLbListenerCertificate(),
			"aws_alb_listener_rule":           resourceAwsLbbListenerRule(),
			"aws_lb_listener_rule":            resourceAwsLbbListenerRule(),
			"aws_alb_target_group":            resourceAwsLbTargetGroup(),
			"aws_lb_target_group":             resourceAwsLbTargetGroup(),
			"aws_alb_target_group_attachment": resourceAwsLbTargetGroupAttachment(),
			"aws_lb_target_group_attachment":  resourceAwsLbTargetGroupAttachment(),
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

		"apigateway_endpoint": "Use this to override the default endpoint URL constructed from the `region`.\n",

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

		"lambda_endpoint": "Use this to override the default endpoint URL constructed from the `region`\n",

		"ec2_endpoint": "Use this to override the default endpoint URL constructed from the `region`.\n",

		"autoscaling_endpoint": "Use this to override the default endpoint URL constructed from the `region`.\n",

		"efs_endpoint": "Use this to override the default endpoint URL constructed from the `region`.\n",

		"elb_endpoint": "Use this to override the default endpoint URL constructed from the `region`.\n",

		"es_endpoint": "Use this to override the default endpoint URL constructed from the `region`.\n",

		"rds_endpoint": "Use this to override the default endpoint URL constructed from the `region`.\n",

		"s3_endpoint": "Use this to override the default endpoint URL constructed from the `region`.\n",

		"sns_endpoint": "Use this to override the default endpoint URL constructed from the `region`.\n",

		"sqs_endpoint": "Use this to override the default endpoint URL constructed from the `region`.\n",

		"ssm_endpoint": "Use this to override the default endpoint URL constructed from the `region`.\n",

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

	// Set CredsFilename, expanding home directory
	credsPath, err := homedir.Expand(d.Get("shared_credentials_file").(string))
	if err != nil {
		return nil, err
	}
	config.CredsFilename = credsPath

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
		config.AcmEndpoint = endpoints["acm"].(string)
		config.ApigatewayEndpoint = endpoints["apigateway"].(string)
		config.CloudFormationEndpoint = endpoints["cloudformation"].(string)
		config.CloudWatchEndpoint = endpoints["cloudwatch"].(string)
		config.CloudWatchEventsEndpoint = endpoints["cloudwatchevents"].(string)
		config.CloudWatchLogsEndpoint = endpoints["cloudwatchlogs"].(string)
		config.DeviceFarmEndpoint = endpoints["devicefarm"].(string)
		config.DynamoDBEndpoint = endpoints["dynamodb"].(string)
		config.Ec2Endpoint = endpoints["ec2"].(string)
		config.AutoscalingEndpoint = endpoints["autoscaling"].(string)
		config.EcrEndpoint = endpoints["ecr"].(string)
		config.EcsEndpoint = endpoints["ecs"].(string)
		config.EfsEndpoint = endpoints["efs"].(string)
		config.ElbEndpoint = endpoints["elb"].(string)
		config.EsEndpoint = endpoints["es"].(string)
		config.IamEndpoint = endpoints["iam"].(string)
		config.KinesisEndpoint = endpoints["kinesis"].(string)
		config.KmsEndpoint = endpoints["kms"].(string)
		config.LambdaEndpoint = endpoints["lambda"].(string)
		config.R53Endpoint = endpoints["r53"].(string)
		config.RdsEndpoint = endpoints["rds"].(string)
		config.S3Endpoint = endpoints["s3"].(string)
		config.SnsEndpoint = endpoints["sns"].(string)
		config.SqsEndpoint = endpoints["sqs"].(string)
		config.StsEndpoint = endpoints["sts"].(string)
		config.SsmEndpoint = endpoints["ssm"].(string)
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
				"acm": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["acm_endpoint"],
				},
				"apigateway": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["apigateway_endpoint"],
				},
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

				"autoscaling": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["autoscaling_endpoint"],
				},

				"ecr": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["ecr_endpoint"],
				},

				"ecs": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["ecs_endpoint"],
				},

				"efs": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["efs_endpoint"],
				},

				"elb": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["elb_endpoint"],
				},
				"es": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["es_endpoint"],
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
				"lambda": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["lambda_endpoint"],
				},
				"r53": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["r53_endpoint"],
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
				"sts": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["sts_endpoint"],
				},
				"ssm": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["ssm_endpoint"],
				},
			},
		},
		Set: endpointsToHash,
	}
}

func endpointsToHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", m["apigateway"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["cloudwatch"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["cloudwatchevents"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["cloudwatchlogs"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["cloudformation"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["devicefarm"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["dynamodb"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["iam"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["ec2"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["autoscaling"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["efs"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["elb"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["kinesis"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["kms"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["lambda"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["rds"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["s3"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["sns"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["sqs"].(string)))

	return hashcode.String(buf.String())
}
