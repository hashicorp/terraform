## 1.8.0 (Unreleased)

ENHANCEMENTS:

* datasource/aws_kms_alias: Add target_key_arn attribute [GH-2551]
* resource/aws_appautoscaling_target: Support updating max_capacity, min_capacity, and role_arn attributes [GH-2950]
* resource/aws_elasticsearch_domain: Add support for encrypt_at_rest [GH-2632]
* resource/aws_kms_alias: Add target_key_arn attribute [GH-3096]
* resource/aws_route: Allow adding IPv6 routes to instances and network interfaces [GH-2265]
* resource/aws_vpn_connection: Add inside CIDR and pre-shared key attributes [GH-1862]
* resource/aws_api_gateway_integration: Allow update of content_handling attributes [GH-3123]
* resource/cognito_user_pool: support pre_token_generation in lambda_config [GH-3093]

BUG FIXES:

* resource/aws_ebs_snapshot: Fix `kms_key_id` attribute handling [GH-3085]
* resource/aws_eip_assocation: Retry association for pending instances [GH-3072]
* resource/aws_kinesis_firehose_delivery_stream: Prevent panic on missing S3 configuration prefix [GH-3073]
* resource/aws_sqs_queue_policy: Prevent missing policy error on read [GH-2739]

## 1.7.1 (January 19, 2018)

BUG FIXES:

* data-source/aws_db_snapshot: Prevent crash on unfinished snapshots ([#2960](https://github.com/terraform-providers/terraform-provider-aws/issues/2960))
* resource/aws_cloudfront_distribution: Retry deletion on DistributionNotDisabled ([#3034](https://github.com/terraform-providers/terraform-provider-aws/issues/3034))
* resource/aws_codebuild_project: Prevent crash on empty source buildspec and location ([#3011](https://github.com/terraform-providers/terraform-provider-aws/issues/3011))
* resource/aws_codepipeline: Prevent crash on empty artifacts ([#2998](https://github.com/terraform-providers/terraform-provider-aws/issues/2998))
* resource/aws_appautoscaling_policy: Match correct policy when multiple policies with same name and service ([#3012](https://github.com/terraform-providers/terraform-provider-aws/issues/3012))
* resource/aws_eip: Do not disassociate EIP on tags-only update ([#2975](https://github.com/terraform-providers/terraform-provider-aws/issues/2975))
* resource/aws_elastic_beanstalk_application: Retry DescribeApplication after creation ([#3064](https://github.com/terraform-providers/terraform-provider-aws/issues/3064))
* resource/aws_emr_cluster: Retry creation on `ValidationException` (IAM) ([#3027](https://github.com/terraform-providers/terraform-provider-aws/issues/3027))
* resource/aws_emr_cluster: Retry creation on `AccessDeniedException` (IAM) ([#3050](https://github.com/terraform-providers/terraform-provider-aws/issues/3050))
* resource/aws_iam_instance_profile: Allow cleanup during destruction without refresh ([#2983](https://github.com/terraform-providers/terraform-provider-aws/issues/2983))
* resource/aws_iam_role: Prevent missing attached policy results ([#2857](https://github.com/terraform-providers/terraform-provider-aws/issues/2857))
* resource/aws_iam_user: Prevent state removal during name attribute update ([#2979](https://github.com/terraform-providers/terraform-provider-aws/issues/2979))
* resource/aws_iam_user: Allow path attribute update ([#2940](https://github.com/terraform-providers/terraform-provider-aws/issues/2940))
* resource/aws_iam_user_policy: Fix updates with generated policy names and validate JSON ([#3031](https://github.com/terraform-providers/terraform-provider-aws/issues/3031))
* resource/aws_instance: Retry IAM instance profile (re)association for eventual consistency on update ([#3055](https://github.com/terraform-providers/terraform-provider-aws/issues/3055))
* resource/aws_lambda_function: Make EC2 rate limit errors retryable on update ([#2964](https://github.com/terraform-providers/terraform-provider-aws/issues/2964))
* resource/aws_lambda_function: Retry creation on EC2 throttle error ([#3062](https://github.com/terraform-providers/terraform-provider-aws/issues/3062))
* resource/aws_lb_target_group: Allow a blank health check path, for TCP healthchecks ([#2980](https://github.com/terraform-providers/terraform-provider-aws/issues/2980))
* resource/aws_sns_topic_subscription: Prevent crash on subscription attribute update ([#2967](https://github.com/terraform-providers/terraform-provider-aws/issues/2967))
* resource/aws_kinesis_firehose_delivery_stream: Fix import for S3 destinations ([#2970](https://github.com/terraform-providers/terraform-provider-aws/issues/2970))
* resource/aws_kinesis_firehose_delivery_stream: Prevent crash on empty Redshift's S3 Backup Description ([#2970](https://github.com/terraform-providers/terraform-provider-aws/issues/2970))
* resource/aws_kinesis_firehose_delivery_stream: Detect drifts in `processing_configuration` ([#2970](https://github.com/terraform-providers/terraform-provider-aws/issues/2970))
* resource/aws_kinesis_firehose_delivery_stream: Prevent crash on empty CloudWatch logging opts ([#3052](https://github.com/terraform-providers/terraform-provider-aws/issues/3052))

## 1.7.0 (January 12, 2018)

FEATURES:

* **New Resource:** `aws_api_gateway_documentation_part` ([#2893](https://github.com/terraform-providers/terraform-provider-aws/issues/2893))
* **New Resource:** `aws_cloudwatch_event_permission` ([#2888](https://github.com/terraform-providers/terraform-provider-aws/issues/2888))
* **New Resource:** `aws_cognito_user_pool_client` ([#1803](https://github.com/terraform-providers/terraform-provider-aws/issues/1803))
* **New Resource:** `aws_cognito_user_pool_domain` ([#2325](https://github.com/terraform-providers/terraform-provider-aws/issues/2325))
* **New Resource:** `aws_glue_catalog_database` ([#2175](https://github.com/terraform-providers/terraform-provider-aws/issues/2175))
* **New Resource:** `aws_guardduty_detector` ([#2524](https://github.com/terraform-providers/terraform-provider-aws/issues/2524))
* **New Resource:** `aws_guardduty_member` ([#2911](https://github.com/terraform-providers/terraform-provider-aws/issues/2911))
* **New Resource:** `aws_route53_query_log` ([#2770](https://github.com/terraform-providers/terraform-provider-aws/issues/2770))
* **New Resource:** `aws_service_discovery_service` ([#2613](https://github.com/terraform-providers/terraform-provider-aws/issues/2613))

ENHANCEMENTS:

* provider: `eu-west-3` is now supported ([#2707](https://github.com/terraform-providers/terraform-provider-aws/issues/2707))
* provider: Endpoints can now be specified for ACM, ECR, ECS, STS and Route 53 ([#2795](https://github.com/terraform-providers/terraform-provider-aws/issues/2795))
* provider: Endpoints can now be specified for API Gateway and Lambda ([#2641](https://github.com/terraform-providers/terraform-provider-aws/issues/2641))
* data-source/aws_iam_server_certificate: Add support for retrieving public key ([#2749](https://github.com/terraform-providers/terraform-provider-aws/issues/2749))
* data-source/aws_vpc_peering_connection: Add support for cross-region VPC peering ([#2508](https://github.com/terraform-providers/terraform-provider-aws/issues/2508))
* data-source/aws_ssm_parameter: Support returning raw encrypted SecureString value ([#2777](https://github.com/terraform-providers/terraform-provider-aws/issues/2777))
* resource/aws_kinesis_firehose_delivery_stream: Import is now supported ([#2082](https://github.com/terraform-providers/terraform-provider-aws/issues/2082))
* resource/aws_cognito_user_pool: The ARN for the pool is now computed and exposed as an attribute ([#2723](https://github.com/terraform-providers/terraform-provider-aws/issues/2723))
* resource/aws_directory_service_directory: Add `security_group_id` field ([#2688](https://github.com/terraform-providers/terraform-provider-aws/issues/2688))
* resource/aws_rds_cluster_instance: Support Performance Insights ([#2331](https://github.com/terraform-providers/terraform-provider-aws/issues/2331))
* resource/aws_rds_cluster_instance: Set `db_subnet_group_name` in state on read if available ([#2606](https://github.com/terraform-providers/terraform-provider-aws/issues/2606))
* resource/aws_eip: Tagging is now supported ([#2768](https://github.com/terraform-providers/terraform-provider-aws/issues/2768))
* resource/aws_codepipeline: ARN is now exposed as an attribute ([#2773](https://github.com/terraform-providers/terraform-provider-aws/issues/2773))
* resource/aws_appautoscaling_scheduled_action: `min_capacity` argument is now honoured ([#2794](https://github.com/terraform-providers/terraform-provider-aws/issues/2794))
* resource/aws_rds_cluster: Clusters in the `resetting-master-credentials` state no longer cause an error ([#2791](https://github.com/terraform-providers/terraform-provider-aws/issues/2791))
* resource/aws_cloudwatch_metric_alarm: Support optional datapoints_to_alarm configuration ([#2609](https://github.com/terraform-providers/terraform-provider-aws/issues/2609))
* resource/aws_ses_event_destination: Add support for SNS destinations ([#1737](https://github.com/terraform-providers/terraform-provider-aws/issues/1737))
* resource/aws_iam_role: Delete inline policies when `force_detach_policies = true` ([#2388](https://github.com/terraform-providers/terraform-provider-aws/issues/2388))
* resource/aws_lb_target_group: Improve `health_check` validation ([#2580](https://github.com/terraform-providers/terraform-provider-aws/issues/2580))
* resource/aws_ecs_service: Add `health_check_grace_period_seconds` attribute ([#2788](https://github.com/terraform-providers/terraform-provider-aws/issues/2788))
* resource/aws_vpc_peering_connection: Add support for cross-region VPC peering ([#2508](https://github.com/terraform-providers/terraform-provider-aws/issues/2508))
* resource/aws_vpc_peering_connection_accepter: Add support for cross-region VPC peering ([#2508](https://github.com/terraform-providers/terraform-provider-aws/issues/2508))
* resource/aws_elasticsearch_domain: export kibana endpoint ([#2804](https://github.com/terraform-providers/terraform-provider-aws/issues/2804))
* resource/aws_ssm_association: Allow for multiple targets ([#2297](https://github.com/terraform-providers/terraform-provider-aws/issues/2297))
* resource/aws_instance: Add computed field for volume_id of block device ([#1489](https://github.com/terraform-providers/terraform-provider-aws/issues/1489))
* resource/aws_api_gateway_integration: Allow update of URI attributes ([#2834](https://github.com/terraform-providers/terraform-provider-aws/issues/2834))
* resource/aws_ecs_cluster: Support resource import ([#2762](https://github.com/terraform-providers/terraform-provider-aws/issues/2762))

BUG FIXES:

* resource/aws_cognito_user_pool: Update Cognito email message length to 20,000 ([#2692](https://github.com/terraform-providers/terraform-provider-aws/issues/2692))
* resource/aws_volume_attachment: Changing device name without changing volume or instance ID now correctly produces a diff ([#2720](https://github.com/terraform-providers/terraform-provider-aws/issues/2720))
* resource/aws_s3_bucket_object: Object tagging is now supported in GovCloud ([#2665](https://github.com/terraform-providers/terraform-provider-aws/issues/2665))
* resource/aws_elasticsearch_domain: Fixed a crash when no Cloudwatch log group is configured ([#2787](https://github.com/terraform-providers/terraform-provider-aws/issues/2787))
* resource/aws_s3_bucket_policy: Set the resource ID after successful creation ([#2820](https://github.com/terraform-providers/terraform-provider-aws/issues/2820))
* resource/aws_db_event_subscription: Set the source type when updating categories ([#2833](https://github.com/terraform-providers/terraform-provider-aws/issues/2833))
* resource/aws_db_parameter_group: Remove group from state if it's gone ([#2868](https://github.com/terraform-providers/terraform-provider-aws/issues/2868))
* resource/aws_appautoscaling_target: Make `role_arn` optional & computed ([#2889](https://github.com/terraform-providers/terraform-provider-aws/issues/2889))
* resource/aws_ssm_maintenance_window: Respect `enabled` during updates ([#2818](https://github.com/terraform-providers/terraform-provider-aws/issues/2818))
* resource/aws_lb_target_group: Fix max prefix length check ([#2790](https://github.com/terraform-providers/terraform-provider-aws/issues/2790))
* resource/aws_config_delivery_channel: Retry deletion ([#2910](https://github.com/terraform-providers/terraform-provider-aws/issues/2910))
* resource/aws_lb+aws_elb: Fix regression with undefined `name` ([#2939](https://github.com/terraform-providers/terraform-provider-aws/issues/2939))
* resource/aws_lb_target_group: Fix validation rules for LB's healthcheck ([#2906](https://github.com/terraform-providers/terraform-provider-aws/issues/2906))
* provider: Fix regression affecting empty Optional+Computed fields ([#2348](https://github.com/terraform-providers/terraform-provider-aws/issues/2348))

## 1.6.0 (December 18, 2017)

FEATURES:

* **New Data Source:** `aws_network_interface` ([#2316](https://github.com/terraform-providers/terraform-provider-aws/issues/2316))
* **New Data Source:** `aws_elb` ([#2004](https://github.com/terraform-providers/terraform-provider-aws/issues/2004))
* **New Resource:** `aws_dx_connection_association` ([#2360](https://github.com/terraform-providers/terraform-provider-aws/issues/2360))
* **New Resource:** `aws_appautoscaling_scheduled_action` ([#2231](https://github.com/terraform-providers/terraform-provider-aws/issues/2231))
* **New Resource:** `aws_cloudwatch_log_resource_policy` ([#2243](https://github.com/terraform-providers/terraform-provider-aws/issues/2243))
* **New Resource:** `aws_media_store_container` ([#2448](https://github.com/terraform-providers/terraform-provider-aws/issues/2448))
* **New Resource:** `aws_service_discovery_public_dns_namespace` ([#2569](https://github.com/terraform-providers/terraform-provider-aws/issues/2569))
* **New Resource:** `aws_service_discovery_private_dns_namespace` ([#2589](https://github.com/terraform-providers/terraform-provider-aws/issues/2589))

IMPROVEMENTS:

* resource/aws_ssm_association: Add `association_name` ([#2257](https://github.com/terraform-providers/terraform-provider-aws/issues/2257))
* resource/aws_ecs_service: Add `network_configuration` ([#2299](https://github.com/terraform-providers/terraform-provider-aws/issues/2299))
* resource/aws_lambda_function: Add `reserved_concurrent_executions` ([#2504](https://github.com/terraform-providers/terraform-provider-aws/issues/2504))
* resource/aws_ecs_service: Add `launch_type` (Fargate support) ([#2483](https://github.com/terraform-providers/terraform-provider-aws/issues/2483))
* resource/aws_ecs_task_definition: Add `cpu`, `memory`, `execution_role_arn` & `requires_compatibilities` (Fargate support) ([#2483](https://github.com/terraform-providers/terraform-provider-aws/issues/2483))
* resource/aws_ecs_cluster: Add arn attribute ([#2552](https://github.com/terraform-providers/terraform-provider-aws/issues/2552))
* resource/aws_elasticache_security_group: Add import support ([#2277](https://github.com/terraform-providers/terraform-provider-aws/issues/2277))
* resource/aws_sqs_queue_policy: Support import by queue URL ([#2544](https://github.com/terraform-providers/terraform-provider-aws/issues/2544))
* resource/aws_elasticsearch_domain: Add `log_publishing_options` ([#2285](https://github.com/terraform-providers/terraform-provider-aws/issues/2285))
* resource/aws_athena_database: Add `force_destroy` field ([#2363](https://github.com/terraform-providers/terraform-provider-aws/issues/2363))
* resource/aws_elasticache_replication_group: Add support for Redis auth, in-transit and at-rest encryption ([#2090](https://github.com/terraform-providers/terraform-provider-aws/issues/2090))
* resource/aws_s3_bucket: Add `server_side_encryption_configuration` block ([#2472](https://github.com/terraform-providers/terraform-provider-aws/issues/2472))

BUG FIXES:

* data-source/aws_instance: Set `placement_group` if available ([#2400](https://github.com/terraform-providers/terraform-provider-aws/issues/2400))
* resource/aws_elasticache_parameter_group: Add StateFunc to make name lowercase ([#2426](https://github.com/terraform-providers/terraform-provider-aws/issues/2426))
* resource/aws_elasticache_replication_group: Modify validation, make replication_group_id lowercase ([#2432](https://github.com/terraform-providers/terraform-provider-aws/issues/2432))
* resource/aws_db_instance: Treat `storage-optimization` as valid state ([#2409](https://github.com/terraform-providers/terraform-provider-aws/issues/2409))
* resource/aws_dynamodb_table: Ensure `ttl` is properly read ([#2452](https://github.com/terraform-providers/terraform-provider-aws/issues/2452))
* resource/aws_lb_target_group: fixes to behavior based on protocol type ([#2380](https://github.com/terraform-providers/terraform-provider-aws/issues/2380))
* resource/aws_mq_broker: Fix crash in hashing function ([#2598](https://github.com/terraform-providers/terraform-provider-aws/issues/2598))
* resource/aws_ebs_volume_attachment: Allow attachments to instances which are stopped ([#1444](https://github.com/terraform-providers/terraform-provider-aws/issues/1444))
* resource/aws_ssm_parameter: Path names with a leading '/' no longer generate incorrect ARNs ([#2604](https://github.com/terraform-providers/terraform-provider-aws/issues/2604))

## 1.5.0 (November 29, 2017)

FEATURES:

* **New Resource:** `aws_mq_broker` ([#2466](https://github.com/terraform-providers/terraform-provider-aws/issues/2466))
* **New Resource:** `aws_mq_configuration` ([#2466](https://github.com/terraform-providers/terraform-provider-aws/issues/2466))

## 1.4.0 (November 29, 2017)

BUG FIXES:

* resource/aws_cognito_user_pool: Fix `email_subject_by_link` ([#2395](https://github.com/terraform-providers/terraform-provider-aws/issues/2395))
* resource/aws_api_gateway_method_response: Fix conflict exception in API gateway method response ([#2393](https://github.com/terraform-providers/terraform-provider-aws/issues/2393))
* resource/aws_api_gateway_method: Fix typo `authorization_type` -> `authorization` ([#2430](https://github.com/terraform-providers/terraform-provider-aws/issues/2430))

IMPROVEMENTS:

* data-source/aws_nat_gateway: Add missing address attributes to the schema ([#2209](https://github.com/terraform-providers/terraform-provider-aws/issues/2209))
* resource/aws_ssm_maintenance_window_target: Change MaxItems of targets ([#2361](https://github.com/terraform-providers/terraform-provider-aws/issues/2361))
* resource/aws_sfn_state_machine: Support Update State machine call ([#2349](https://github.com/terraform-providers/terraform-provider-aws/issues/2349))
* resource/aws_instance: Set placement_group in state on read if available ([#2398](https://github.com/terraform-providers/terraform-provider-aws/issues/2398))

## 1.3.1 (November 20, 2017)

BUG FIXES:

* resource/aws_ecs_task_definition: Fix equivalency comparator ([#2339](https://github.com/terraform-providers/terraform-provider-aws/issues/2339))
* resource/aws_batch_job_queue: Return errors correctly if deletion fails ([#2322](https://github.com/terraform-providers/terraform-provider-aws/issues/2322))
* resource/aws_security_group_rule: Parse `description` correctly ([#1959](https://github.com/terraform-providers/terraform-provider-aws/issues/1959))
* Fixed Cognito Lambda Config Validation for optional ARN configurations ([#2370](https://github.com/terraform-providers/terraform-provider-aws/issues/2370))
* resource/aws_cognito_identity_pool_roles_attachment: Fix typo "authenticated" -> "unauthenticated" ([#2358](https://github.com/terraform-providers/terraform-provider-aws/issues/2358))

## 1.3.0 (November 16, 2017)

NOTES:

* resource/aws_redshift_cluster: Field `enable_logging`, `bucket_name` and `s3_key_prefix` were deprecated in favour of a new `logging` block ([#2230](https://github.com/terraform-providers/terraform-provider-aws/issues/2230))
* resource/aws_lb_target_group: We no longer provide defaults for `health_check`'s `path` nor `matcher` in order to support network load balancers where these arguments aren't valid. Creating _new_ ALB will therefore require you to specify these two arguments. Existing deployments are unaffected. ([#2251](https://github.com/terraform-providers/terraform-provider-aws/issues/2251))

FEATURES:

* **New Data Source:** `aws_rds_cluster` ([#2070](https://github.com/terraform-providers/terraform-provider-aws/issues/2070))
* **New Data Source:** `aws_elasticache_replication_group` ([#2124](https://github.com/terraform-providers/terraform-provider-aws/issues/2124))
* **New Data Source:** `aws_instances` ([#2266](https://github.com/terraform-providers/terraform-provider-aws/issues/2266))
* **New Resource:** `aws_ses_template` ([#2003](https://github.com/terraform-providers/terraform-provider-aws/issues/2003))
* **New Resource:** `aws_dx_lag` ([#2154](https://github.com/terraform-providers/terraform-provider-aws/issues/2154))
* **New Resource:** `aws_dx_connection` ([#2173](https://github.com/terraform-providers/terraform-provider-aws/issues/2173))
* **New Resource:** `aws_athena_database` ([#1922](https://github.com/terraform-providers/terraform-provider-aws/issues/1922))
* **New Resource:** `aws_athena_named_query` ([#1893](https://github.com/terraform-providers/terraform-provider-aws/issues/1893))
* **New Resource:** `aws_ssm_resource_data_sync` ([#1895](https://github.com/terraform-providers/terraform-provider-aws/issues/1895))
* **New Resource:** `aws_cognito_user_pool` ([#1419](https://github.com/terraform-providers/terraform-provider-aws/issues/1419))

IMPROVEMENTS:

* provider: Add support for assuming roles via profiles defined in `~/.aws/config` ([#1608](https://github.com/terraform-providers/terraform-provider-aws/issues/1608))
* data-source/efs_file_system: Added dns_name ([#2105](https://github.com/terraform-providers/terraform-provider-aws/issues/2105))
* data-source/aws_ssm_parameter: Add `arn` attribute ([#2273](https://github.com/terraform-providers/terraform-provider-aws/issues/2273))
* data-source/aws_ebs_volume: Add `arn` attribute ([#2271](https://github.com/terraform-providers/terraform-provider-aws/issues/2271))
* resource/aws_batch_job_queue: Add validation for `name` ([#2159](https://github.com/terraform-providers/terraform-provider-aws/issues/2159))
* resource/aws_batch_compute_environment: Improve validation for `compute_environment_name` ([#2159](https://github.com/terraform-providers/terraform-provider-aws/issues/2159))
* resource/aws_ssm_parameter: Add support for import ([#2234](https://github.com/terraform-providers/terraform-provider-aws/issues/2234))
* resource/aws_redshift_cluster: Add support for `snapshot_copy` ([#2238](https://github.com/terraform-providers/terraform-provider-aws/issues/2238))
* resource/aws_ecs_task_definition: Print `container_definitions` as JSON instead of checksum ([#1195](https://github.com/terraform-providers/terraform-provider-aws/issues/1195))
* resource/aws_ssm_parameter: Add `arn` attribute ([#2273](https://github.com/terraform-providers/terraform-provider-aws/issues/2273))
* resource/aws_elb: Add listener `ssl_certificate_id` ARN validation ([#2276](https://github.com/terraform-providers/terraform-provider-aws/issues/2276))
* resource/aws_cloudformation_stack: Support updating `tags` ([#2262](https://github.com/terraform-providers/terraform-provider-aws/issues/2262))
* resource/aws_elb: Add `arn` attribute ([#2272](https://github.com/terraform-providers/terraform-provider-aws/issues/2272))
* resource/aws_ebs_volume: Add `arn` attribute ([#2271](https://github.com/terraform-providers/terraform-provider-aws/issues/2271))

BUG FIXES:

* resource/aws_appautoscaling_policy: Retry putting policy on invalid token ([#2135](https://github.com/terraform-providers/terraform-provider-aws/issues/2135))
* resource/aws_batch_compute_environment: `compute_environment_name` allows hyphens ([#2126](https://github.com/terraform-providers/terraform-provider-aws/issues/2126))
* resource/aws_batch_job_definition: `name` allows hyphens ([#2126](https://github.com/terraform-providers/terraform-provider-aws/issues/2126))
* resource/aws_elasticache_parameter_group: Raise timeout for retry on pending changes ([#2134](https://github.com/terraform-providers/terraform-provider-aws/issues/2134))
* resource/aws_kms_key: Retry GetKeyRotationStatus on NotFoundException ([#2133](https://github.com/terraform-providers/terraform-provider-aws/issues/2133))
* resource/aws_lb_target_group: Fix issue that prevented using `aws_lb_target_group` with 
  Network type load balancers ([#2251](https://github.com/terraform-providers/terraform-provider-aws/issues/2251))
* resource/aws_lb: mark subnets as `ForceNew` for network load balancers ([#2310](https://github.com/terraform-providers/terraform-provider-aws/issues/2310))
* resource/aws_redshift_cluster: Make master_username ForceNew ([#2202](https://github.com/terraform-providers/terraform-provider-aws/issues/2202))
* resource/aws_cloudwatch_log_metric_filter: Fix pattern length check ([#2107](https://github.com/terraform-providers/terraform-provider-aws/issues/2107))
* resource/aws_cloudwatch_log_group: Use ID as name ([#2190](https://github.com/terraform-providers/terraform-provider-aws/issues/2190))
* resource/aws_elasticsearch_domain: Added ForceNew to vpc_options ([#2157](https://github.com/terraform-providers/terraform-provider-aws/issues/2157))
* resource/aws_redshift_cluster: Make snapshot identifiers `ForceNew` ([#2212](https://github.com/terraform-providers/terraform-provider-aws/issues/2212))
* resource/aws_elasticsearch_domain_policy: Fix typo in err code ([#2249](https://github.com/terraform-providers/terraform-provider-aws/issues/2249))
* resource/aws_appautoscaling_policy: Retry PutScalingPolicy on rate exceeded message ([#2275](https://github.com/terraform-providers/terraform-provider-aws/issues/2275))
* resource/aws_dynamodb_table: Retry creation on `LimitExceededException` w/ different error message ([#2274](https://github.com/terraform-providers/terraform-provider-aws/issues/2274))

## 1.2.0 (October 31, 2017)

INTERNAL:

* Remove `id` fields from schema definitions ([#1626](https://github.com/terraform-providers/terraform-provider-aws/issues/1626))

FEATURES:

* **New Resource:** `aws_servicecatalog_portfolio` ([#1694](https://github.com/terraform-providers/terraform-provider-aws/issues/1694))
* **New Resource:** `aws_ses_domain_dkim` ([#1786](https://github.com/terraform-providers/terraform-provider-aws/issues/1786))
* **New Resource:** `aws_cognito_identity_pool_roles_attachment` ([#863](https://github.com/terraform-providers/terraform-provider-aws/issues/863))
* **New Resource:** `aws_ecr_lifecycle_policy` ([#2096](https://github.com/terraform-providers/terraform-provider-aws/issues/2096))
* **New Data Source:** `aws_nat_gateway` ([#1294](https://github.com/terraform-providers/terraform-provider-aws/issues/1294))
* **New Data Source:** `aws_dynamodb_table` ([#2062](https://github.com/terraform-providers/terraform-provider-aws/issues/2062))
* **New Data Source:** `aws_cloudtrail_service_account` ([#1774](https://github.com/terraform-providers/terraform-provider-aws/issues/1774))

IMPROVEMENTS:

* resource/aws_ami: Support configurable timeouts ([#1811](https://github.com/terraform-providers/terraform-provider-aws/issues/1811))
* resource/ami_copy: Support configurable timeouts ([#1811](https://github.com/terraform-providers/terraform-provider-aws/issues/1811))
* resource/ami_from_instance: Support configurable timeouts ([#1811](https://github.com/terraform-providers/terraform-provider-aws/issues/1811))
* data-source/aws_security_group: add description ([#1943](https://github.com/terraform-providers/terraform-provider-aws/issues/1943))
* resource/aws_cloudfront_distribution: Change the default minimum_protocol_version to TLSv1 ([#1856](https://github.com/terraform-providers/terraform-provider-aws/issues/1856))
* resource/aws_sns_topic: Support SMS in protocols ([#1813](https://github.com/terraform-providers/terraform-provider-aws/issues/1813))
* resource/aws_spot_fleet_request: Add support for `tags` ([#2042](https://github.com/terraform-providers/terraform-provider-aws/issues/2042))
* resource/aws_kinesis_firehose_delivery_stream: Add `s3_backup_mode` option ([#1830](https://github.com/terraform-providers/terraform-provider-aws/issues/1830))
* resource/aws_elasticsearch_domain: Support VPC configuration ([#1958](https://github.com/terraform-providers/terraform-provider-aws/issues/1958))
* resource/aws_alb_target_group: Add support for `target_type` ([#1589](https://github.com/terraform-providers/terraform-provider-aws/issues/1589))
* resource/aws_sqs_queue: Add support for `tags` ([#1987](https://github.com/terraform-providers/terraform-provider-aws/issues/1987))
* resource/aws_security_group: Add `revoke_rules_on_delete` option to force a security group to revoke 
  rules before deleting the grou ([#2074](https://github.com/terraform-providers/terraform-provider-aws/issues/2074))
* resource/aws_cloudwatch_log_metric_filter: Add support for DefaultValue ([#1578](https://github.com/terraform-providers/terraform-provider-aws/issues/1578))
* resource/aws_emr_cluster: Expose error on `TERMINATED_WITH_ERRORS` ([#2081](https://github.com/terraform-providers/terraform-provider-aws/issues/2081))

BUG FIXES:

* resource/aws_elasticache_parameter_group: Add missing return to retry logic ([#1891](https://github.com/terraform-providers/terraform-provider-aws/issues/1891))
* resource/aws_batch_job_queue: Wait for update completion when disabling ([#1892](https://github.com/terraform-providers/terraform-provider-aws/issues/1892))
* resource/aws_snapshot_create_volume_permission: Raise creation timeout to 10mins ([#1894](https://github.com/terraform-providers/terraform-provider-aws/issues/1894))
* resource/aws_snapshot_create_volume_permission: Raise creation timeout to 20mins ([#2049](https://github.com/terraform-providers/terraform-provider-aws/issues/2049))
* resource/aws_kms_alias: Retry creation on `NotFoundException` ([#1896](https://github.com/terraform-providers/terraform-provider-aws/issues/1896))
* resource/aws_kms_key: Retry reading tags on `NotFoundException` ([#1900](https://github.com/terraform-providers/terraform-provider-aws/issues/1900))
* resource/aws_db_snapshot: Raise creation timeout to 20mins ([#1905](https://github.com/terraform-providers/terraform-provider-aws/issues/1905))
* resource/aws_lb: Allow assigning EIP to network LB ([#1956](https://github.com/terraform-providers/terraform-provider-aws/issues/1956))
* resource/aws_s3_bucket: Retry tagging on OperationAborted ([#2008](https://github.com/terraform-providers/terraform-provider-aws/issues/2008))
* resource/aws_cognito_identity_pool: Fixed refresh of providers ([#2015](https://github.com/terraform-providers/terraform-provider-aws/issues/2015))
* resource/aws_elasticache_replication_group: Raise creation timeout to 50mins ([#2048](https://github.com/terraform-providers/terraform-provider-aws/issues/2048))
* resource/aws_api_gateway_usag_plan: Fixed setting of rate_limit ([#2076](https://github.com/terraform-providers/terraform-provider-aws/issues/2076))
* resource/aws_elastic_beanstalk_application: Expose error leading to failed deletion ([#2080](https://github.com/terraform-providers/terraform-provider-aws/issues/2080))
* resource/aws_s3_bucket: Accept query strings in redirect hosts ([#2059](https://github.com/terraform-providers/terraform-provider-aws/issues/2059))

## 1.1.0 (October 16, 2017)

NOTES:

* resource/aws_alb_* & data-source/aws_alb_*: In order to support network LBs, ALBs were renamed to `aws_lb_*` due to the way APIs "new" (non-Classic) load balancers are structured in AWS. All existing ALB functionality remains untouched and new resources work the same way. `aws_alb_*` resources are still in place as "aliases", but documentation will only mention `aws_lb_*`.
`aws_alb_*` aliases will be removed in future major version. ([#1806](https://github.com/terraform-providers/terraform-provider-aws/issues/1806))
* Deprecated:
  * data-source/aws_alb
  * data-source/aws_alb_listener
  * data-source/aws_alb_target_group
  * resource/aws_alb
  * resource/aws_alb_listener
  * resource/aws_alb_listener_rule
  * resource/aws_alb_target_group
  * resource/aws_alb_target_group_attachment

FEATURES:

* **New Resource:** `aws_batch_job_definition` ([#1710](https://github.com/terraform-providers/terraform-provider-aws/issues/1710))
* **New Resource:** `aws_batch_job_queue` ([#1710](https://github.com/terraform-providers/terraform-provider-aws/issues/1710))
* **New Resource:** `aws_lb` ([#1806](https://github.com/terraform-providers/terraform-provider-aws/issues/1806))
* **New Resource:** `aws_lb_listener` ([#1806](https://github.com/terraform-providers/terraform-provider-aws/issues/1806))
* **New Resource:** `aws_lb_listener_rule` ([#1806](https://github.com/terraform-providers/terraform-provider-aws/issues/1806))
* **New Resource:** `aws_lb_target_group` ([#1806](https://github.com/terraform-providers/terraform-provider-aws/issues/1806))
* **New Resource:** `aws_lb_target_group_attachment` ([#1806](https://github.com/terraform-providers/terraform-provider-aws/issues/1806))
* **New Data Source:** `aws_lb` ([#1806](https://github.com/terraform-providers/terraform-provider-aws/issues/1806))
* **New Data Source:** `aws_lb_listener` ([#1806](https://github.com/terraform-providers/terraform-provider-aws/issues/1806))
* **New Data Source:** `aws_lb_target_group` ([#1806](https://github.com/terraform-providers/terraform-provider-aws/issues/1806))
* **New Data Source:** `aws_iam_user` ([#1805](https://github.com/terraform-providers/terraform-provider-aws/issues/1805))
* **New Data Source:** `aws_s3_bucket` ([#1505](https://github.com/terraform-providers/terraform-provider-aws/issues/1505))

IMPROVEMENTS:

* data-source/aws_redshift_service_account: Add `arn` attribute ([#1775](https://github.com/terraform-providers/terraform-provider-aws/issues/1775))
* data-source/aws_vpc_endpoint: Expose `prefix_list_id` ([#1733](https://github.com/terraform-providers/terraform-provider-aws/issues/1733))
* resource/aws_kinesis_stream: Add support for encryption ([#1139](https://github.com/terraform-providers/terraform-provider-aws/issues/1139))
* resource/aws_cloudwatch_log_group: Add support for encryption via `kms_key_id` ([#1751](https://github.com/terraform-providers/terraform-provider-aws/issues/1751))
* resource/aws_spot_instance_request: Add support for `instance_interruption_behaviour` ([#1735](https://github.com/terraform-providers/terraform-provider-aws/issues/1735))
* resource/aws_ses_event_destination: Add support for `open` & `click` event types ([#1773](https://github.com/terraform-providers/terraform-provider-aws/issues/1773))
* resource/aws_efs_file_system: Expose `dns_name` ([#1825](https://github.com/terraform-providers/terraform-provider-aws/issues/1825))
* resource/aws_security_group+aws_security_group_rule: Add support for rule description ([#1587](https://github.com/terraform-providers/terraform-provider-aws/issues/1587))
* resource/aws_emr_cluster: enable configuration of ebs root volume size ([#1375](https://github.com/terraform-providers/terraform-provider-aws/issues/1375))
* resource/aws_ami: Add `root_snapshot_id` attribute ([#1572](https://github.com/terraform-providers/terraform-provider-aws/issues/1572))
* resource/aws_vpn_connection: Mark preshared keys as sensitive ([#1850](https://github.com/terraform-providers/terraform-provider-aws/issues/1850))
* resource/aws_codedeploy_deployment_group: Support blue/green and in-place deployments with traffic control ([#1162](https://github.com/terraform-providers/terraform-provider-aws/issues/1162))
* resource/aws_elb: Update ELB idle timeout to 4000s ([#1861](https://github.com/terraform-providers/terraform-provider-aws/issues/1861))
* resource/aws_spot_fleet_request: Add support for instance_interruption_behaviour ([#1847](https://github.com/terraform-providers/terraform-provider-aws/issues/1847))
* resource/aws_kinesis_firehose_delivery_stream: Specify kinesis stream as the source of a aws_kinesis_firehose_delivery_stream ([#1605](https://github.com/terraform-providers/terraform-provider-aws/issues/1605))
* resource/aws_kinesis_firehose_delivery_stream: Output complete error when creation fails ([#1881](https://github.com/terraform-providers/terraform-provider-aws/issues/1881))

BUG FIXES:

* data-source/aws_db_instance: Make `db_instance_arn` expose ARN instead of identifier (use `db_cluster_identifier` for identifier) ([#1766](https://github.com/terraform-providers/terraform-provider-aws/issues/1766))
* data-source/aws_db_snapshot: Expose `storage_type` (was not exposed) ([#1833](https://github.com/terraform-providers/terraform-provider-aws/issues/1833))
* data-source/aws_ami: Update the `tags` structure for easier referencing ([#1706](https://github.com/terraform-providers/terraform-provider-aws/issues/1706))
* data-source/aws_ebs_snapshot: Update the `tags` structure for easier referencing ([#1706](https://github.com/terraform-providers/terraform-provider-aws/issues/1706))
* data-source/aws_ebs_volume: Update the `tags` structure for easier referencing ([#1706](https://github.com/terraform-providers/terraform-provider-aws/issues/1706))
* data-source/aws_instance: Update the `tags` structure for easier referencing ([#1706](https://github.com/terraform-providers/terraform-provider-aws/issues/1706))
* resource/aws_spot_instance_request: Handle `closed` request correctly ([#1903](https://github.com/terraform-providers/terraform-provider-aws/issues/1903))
* resource/aws_cloudtrail: Raise update retry timeout ([#1820](https://github.com/terraform-providers/terraform-provider-aws/issues/1820))
* resource/aws_elasticache_parameter_group: Retry resetting group on pending changes ([#1821](https://github.com/terraform-providers/terraform-provider-aws/issues/1821))
* resource/aws_kms_key: Retry getting rotation status ([#1818](https://github.com/terraform-providers/terraform-provider-aws/issues/1818))
* resource/aws_kms_key: Retry getting key policy ([#1854](https://github.com/terraform-providers/terraform-provider-aws/issues/1854))
* resource/aws_vpn_connection: Raise timeout to 40mins ([#1819](https://github.com/terraform-providers/terraform-provider-aws/issues/1819))
* resource/aws_kinesis_firehose_delivery_stream: Fix crash caused by missing `processing_configuration` ([#1738](https://github.com/terraform-providers/terraform-provider-aws/issues/1738))
* resource/aws_rds_cluster_instance: Treat `configuring-enhanced-monitoring` as pending state ([#1744](https://github.com/terraform-providers/terraform-provider-aws/issues/1744))
* resource/aws_rds_cluster_instance: Treat more states as pending ([#1790](https://github.com/terraform-providers/terraform-provider-aws/issues/1790))
* resource/aws_route_table: Increase number of not-found checks/retries after creation ([#1791](https://github.com/terraform-providers/terraform-provider-aws/issues/1791))
* resource/aws_batch_compute_environment: Fix ARN attribute name/value (`ecc_cluster_arn` -> `ecs_cluster_arn`) ([#1809](https://github.com/terraform-providers/terraform-provider-aws/issues/1809))
* resource/aws_kinesis_stream: Retry creation of the stream on `LimitExceededException` (handle throttling) ([#1339](https://github.com/terraform-providers/terraform-provider-aws/issues/1339))
* resource/aws_vpn_connection_route: Treat route in state `deleted` as deleted ([#1848](https://github.com/terraform-providers/terraform-provider-aws/issues/1848))
* resource/aws_eip: Avoid disassociating if there's no association ([#1683](https://github.com/terraform-providers/terraform-provider-aws/issues/1683))
* resource/aws_elasticache_cluster: Allow scaling up cluster by modifying `az_mode` (avoid recreation) ([#1758](https://github.com/terraform-providers/terraform-provider-aws/issues/1758))
* resource/aws_lambda_function: Fix Lambda Function Updates When Published ([#1797](https://github.com/terraform-providers/terraform-provider-aws/issues/1797))
* resource/aws_appautoscaling_*: Use dimension to uniquely identify target/policy ([#1808](https://github.com/terraform-providers/terraform-provider-aws/issues/1808))
* resource/aws_vpn_connection_route: Wait until route is available/deleted ([#1849](https://github.com/terraform-providers/terraform-provider-aws/issues/1849))
* resource/aws_cloudfront_distribution: Ignore `minimum_protocol_version` if default certificate is used ([#1785](https://github.com/terraform-providers/terraform-provider-aws/issues/1785))
* resource/aws_security_group: Using `self = false` with `cidr_blocks` should be allowed ([#1839](https://github.com/terraform-providers/terraform-provider-aws/issues/1839))
* resource/aws_instance: Check VPC array size to avoid crashes on Eucalyptus Cloud ([#1882](https://github.com/terraform-providers/terraform-provider-aws/issues/1882))

## 1.0.0 (September 27, 2017)

NOTES:

* resource/aws_appautoscaling_policy: Nest step scaling policy fields, deprecate 1st level fields ([#1620](https://github.com/terraform-providers/terraform-provider-aws/issues/1620))

FEATURES:

* **New Resource:** `aws_waf_rate_based_rule` ([#1606](https://github.com/terraform-providers/terraform-provider-aws/issues/1606))
* **New Resource:** `aws_batch_compute_environment` ([#1048](https://github.com/terraform-providers/terraform-provider-aws/issues/1048))

IMPROVEMENTS:

* provider: Expand shared_credentials_file ([#1511](https://github.com/terraform-providers/terraform-provider-aws/issues/1511))
* provider: Add support for Task Roles when running on ECS or CodeBuild ([#1425](https://github.com/terraform-providers/terraform-provider-aws/issues/1425))
* resource/aws_instance: New `user_data_base64` attribute that allows non-UTF8 data (such as gzip) to be assigned to user-data without corruption ([#850](https://github.com/terraform-providers/terraform-provider-aws/issues/850))
* data-source/aws_vpc: Expose enable_dns_* in aws_vpc data_source ([#1373](https://github.com/terraform-providers/terraform-provider-aws/issues/1373))
* resource/aws_appautoscaling_policy: Add support for DynamoDB ([#1650](https://github.com/terraform-providers/terraform-provider-aws/issues/1650))
* resource/aws_directory_service_directory: Add support for `tags` ([#1398](https://github.com/terraform-providers/terraform-provider-aws/issues/1398))
* resource/aws_rds_cluster: Allow setting of rds cluster engine ([#1415](https://github.com/terraform-providers/terraform-provider-aws/issues/1415))
* resource/aws_ssm_association: now supports update for `parameters`, `schedule_expression`,`output_location` ([#1421](https://github.com/terraform-providers/terraform-provider-aws/issues/1421))
* resource/aws_ssm_patch_baseline: now supports update for multiple attributes ([#1421](https://github.com/terraform-providers/terraform-provider-aws/issues/1421))
* resource/aws_cloudformation_stack: Add support for Import ([#1432](https://github.com/terraform-providers/terraform-provider-aws/issues/1432))
* resource/aws_rds_cluster_instance: Expose availability_zone attribute ([#1439](https://github.com/terraform-providers/terraform-provider-aws/issues/1439))
* resource/aws_efs_file_system: Add support for encryption ([#1420](https://github.com/terraform-providers/terraform-provider-aws/issues/1420))
* resource/aws_db_parameter_group: Allow underscores in names ([#1460](https://github.com/terraform-providers/terraform-provider-aws/issues/1460))
* resource/aws_elasticsearch_domain: Assign tags right after creation ([#1399](https://github.com/terraform-providers/terraform-provider-aws/issues/1399))
* resource/aws_route53_record: Allow CAA record type ([#1467](https://github.com/terraform-providers/terraform-provider-aws/issues/1467))
* resource/aws_codebuild_project: Allowed for BITBUCKET source type ([#1468](https://github.com/terraform-providers/terraform-provider-aws/issues/1468))
* resource/aws_emr_cluster: Add `instance_group` parameter for EMR clusters ([#1071](https://github.com/terraform-providers/terraform-provider-aws/issues/1071))
* resource/aws_alb_listener_rule: Populate `listener_arn` field ([#1303](https://github.com/terraform-providers/terraform-provider-aws/issues/1303))
* resource/aws_api_gateway_rest_api: Add a body property to API Gateway RestAPI for Swagger import support ([#1197](https://github.com/terraform-providers/terraform-provider-aws/issues/1197))
* resource/aws_opsworks_stack: Add support for tags ([#1523](https://github.com/terraform-providers/terraform-provider-aws/issues/1523))
* Add retries for AppScaling policies throttling exceptions ([#1430](https://github.com/terraform-providers/terraform-provider-aws/issues/1430))
* resource/aws_ssm_patch_baseline: Add compliance level to patch approval rules ([#1531](https://github.com/terraform-providers/terraform-provider-aws/issues/1531))
* resource/aws_ssm_activation: Export ssm activation activation_code ([#1570](https://github.com/terraform-providers/terraform-provider-aws/issues/1570))
* resource/aws_network_interface: Added private_dns_name to network_interface ([#1599](https://github.com/terraform-providers/terraform-provider-aws/issues/1599))
* data-source/aws_redshift_service_account: updated with latest redshift service account ID's ([#1614](https://github.com/terraform-providers/terraform-provider-aws/issues/1614))
* resource/aws_ssm_parameter: Refresh from state on 404 ([#1436](https://github.com/terraform-providers/terraform-provider-aws/issues/1436))
* resource/aws_api_gateway_rest_api: Allow binary media types to be updated ([#1600](https://github.com/terraform-providers/terraform-provider-aws/issues/1600))
* resource/aws_waf_rule: Make `predicates`' `data_id` required (it always was on the API's side, it's just reflected in the schema) ([#1606](https://github.com/terraform-providers/terraform-provider-aws/issues/1606))
* resource/aws_waf_web_acl: Introduce new `type` field in `rules` to allow referencing `RATE_BASED` type ([#1606](https://github.com/terraform-providers/terraform-provider-aws/issues/1606))
* resource/aws_ssm_association: Migrate the schema to use association_id ([#1579](https://github.com/terraform-providers/terraform-provider-aws/issues/1579))
* resource/aws_ssm_document: Added name validation ([#1638](https://github.com/terraform-providers/terraform-provider-aws/issues/1638))
* resource/aws_nat_gateway: Add tags support ([#1625](https://github.com/terraform-providers/terraform-provider-aws/issues/1625))
* resource/aws_route53_record: Add support for Route53 multi-value answer routing policy ([#1686](https://github.com/terraform-providers/terraform-provider-aws/issues/1686))
* resource/aws_instance: Read iops only when volume type is io1 ([#1573](https://github.com/terraform-providers/terraform-provider-aws/issues/1573))
* resource/aws_rds_cluster(+_instance) Allow specifying the engine ([#1591](https://github.com/terraform-providers/terraform-provider-aws/issues/1591))
* resource/aws_cloudwatch_event_target: Add Input transformer for Cloudwatch Events ([#1343](https://github.com/terraform-providers/terraform-provider-aws/issues/1343))
* resource/aws_directory_service_directory: Support Import functionality ([#1732](https://github.com/terraform-providers/terraform-provider-aws/issues/1732))

BUG FIXES:

* resource/aws_instance: Fix `associate_public_ip_address` ([#1340](https://github.com/terraform-providers/terraform-provider-aws/issues/1340))
* resource/aws_instance: Fix import in EC2 Classic ([#1453](https://github.com/terraform-providers/terraform-provider-aws/issues/1453))
* resource/aws_emr_cluster: Avoid spurious diff of `log_uri` ([#1374](https://github.com/terraform-providers/terraform-provider-aws/issues/1374))
* resource/aws_cloudwatch_log_subscription_filter: Add support for ResourceNotFound ([#1414](https://github.com/terraform-providers/terraform-provider-aws/issues/1414))
* resource/aws_sns_topic_subscription: Prevent duplicate (un)subscribe during initial creation ([#1480](https://github.com/terraform-providers/terraform-provider-aws/issues/1480))
* resource/aws_alb: Cleanup ENIs after deleting ALB ([#1427](https://github.com/terraform-providers/terraform-provider-aws/issues/1427))
* resource/aws_s3_bucket: Wrap s3 calls in retry to avoid race during creation ([#891](https://github.com/terraform-providers/terraform-provider-aws/issues/891))
* resource/aws_eip: Remove from state on deletion ([#1551](https://github.com/terraform-providers/terraform-provider-aws/issues/1551))
* resource/aws_security_group: Adding second scenario where IPv6 is not supported ([#880](https://github.com/terraform-providers/terraform-provider-aws/issues/880))

## 0.1.4 (August 08, 2017)

FEATURES:

* **New Resource:** `aws_cloudwatch_dashboard` ([#1172](https://github.com/terraform-providers/terraform-provider-aws/issues/1172))
* **New Data Source:** `aws_internet_gateway` ([#1196](https://github.com/terraform-providers/terraform-provider-aws/issues/1196))
* **New Data Source:** `aws_efs_mount_target` ([#1255](https://github.com/terraform-providers/terraform-provider-aws/issues/1255))

IMPROVEMENTS:

* AWS SDK to log extra debug details on request errors ([#1210](https://github.com/terraform-providers/terraform-provider-aws/issues/1210))
* resource/aws_spot_fleet_request: Add support for  `wait_for_fulfillment` ([#1241](https://github.com/terraform-providers/terraform-provider-aws/issues/1241))
* resource/aws_autoscaling_schedule: Allow empty value ([#1268](https://github.com/terraform-providers/terraform-provider-aws/issues/1268))
* resource/aws_ssm_association: Add support for OutputLocation and Schedule Expression ([#1253](https://github.com/terraform-providers/terraform-provider-aws/issues/1253))
* resource/aws_ssm_patch_baseline: Update support for Operating System ([#1260](https://github.com/terraform-providers/terraform-provider-aws/issues/1260))
* resource/aws_db_instance: Expose db_instance ca_cert_identifier ([#1256](https://github.com/terraform-providers/terraform-provider-aws/issues/1256))
* resource/aws_rds_cluster: Add support for iam_roles to rds_cluster ([#1258](https://github.com/terraform-providers/terraform-provider-aws/issues/1258))
* resource/aws_rds_cluster_parameter_group: Support > 20 parameters ([#1298](https://github.com/terraform-providers/terraform-provider-aws/issues/1298))
* data-source/aws_iam_role: Normalize the IAM role data source ([#1330](https://github.com/terraform-providers/terraform-provider-aws/issues/1330))
* resource/aws_kinesis_stream: Increase Timeouts, add Timeout Support ([#1345](https://github.com/terraform-providers/terraform-provider-aws/issues/1345))

BUG FIXES:

* resource/aws_instance: Guard check for aws_instance UserData to prevent panic ([#1288](https://github.com/terraform-providers/terraform-provider-aws/issues/1288))
* resource/aws_config: Set AWS Config Configuration recorder & Delivery channel names as ForceNew ([#1247](https://github.com/terraform-providers/terraform-provider-aws/issues/1247))
* resource/aws_cloudtrail: Retry if IAM role isn't propagated yet ([#1312](https://github.com/terraform-providers/terraform-provider-aws/issues/1312))
* resource/aws_cloudtrail: Fix CloudWatch role ARN/group updates ([#1357](https://github.com/terraform-providers/terraform-provider-aws/issues/1357))
* resource/aws_eip_association: Avoid crash in EC2 Classic ([#1344](https://github.com/terraform-providers/terraform-provider-aws/issues/1344))
* resource/aws_elasticache_parameter_group: Allow removing parameters ([#1309](https://github.com/terraform-providers/terraform-provider-aws/issues/1309))
* resource/aws_kinesis: add retries for Kinesis throttling exceptions ([#1085](https://github.com/terraform-providers/terraform-provider-aws/issues/1085))
* resource/aws_kinesis_firehose: adding support for `ExtendedS3DestinationConfiguration` ([#1015](https://github.com/terraform-providers/terraform-provider-aws/issues/1015))
* resource/aws_spot_fleet_request: Ignore empty `key_name` ([#1203](https://github.com/terraform-providers/terraform-provider-aws/issues/1203))
* resource/aws_emr_instance_group: fix crash when changing `instance_group.count` ([#1287](https://github.com/terraform-providers/terraform-provider-aws/issues/1287))
* resource/aws_elasticsearch_domain: Fix updating config when update doesn't involve EBS ([#1131](https://github.com/terraform-providers/terraform-provider-aws/issues/1131))
* resource/aws_s3_bucket: Avoid crashing when no lifecycle rule is defined ([#1316](https://github.com/terraform-providers/terraform-provider-aws/issues/1316))
* resource/elastic_transcoder_preset: Fix provider validation ([#1338](https://github.com/terraform-providers/terraform-provider-aws/issues/1338))
* resource/aws_s3_bucket: Avoid crashing when `filter` is not set ([#1350](https://github.com/terraform-providers/terraform-provider-aws/issues/1350))

## 0.1.3 (July 25, 2017)

FEATURES:

* **New Data Source:** `aws_iam_instance_profile` ([#1024](https://github.com/terraform-providers/terraform-provider-aws/issues/1024))
* **New Data Source:** `aws_alb_target_group` ([#1037](https://github.com/terraform-providers/terraform-provider-aws/issues/1037))
* **New Data Source:** `aws_iam_group` ([#1140](https://github.com/terraform-providers/terraform-provider-aws/issues/1140))
* **New Resource:** `aws_api_gateway_request_validator` ([#1064](https://github.com/terraform-providers/terraform-provider-aws/issues/1064))
* **New Resource:** `aws_api_gateway_gateway_response` ([#1168](https://github.com/terraform-providers/terraform-provider-aws/issues/1168))
* **New Resource:** `aws_iot_policy` ([#986](https://github.com/terraform-providers/terraform-provider-aws/issues/986))
* **New Resource:** `aws_iot_certificate` ([#1225](https://github.com/terraform-providers/terraform-provider-aws/issues/1225))

IMPROVEMENTS:

* resource/aws_sqs_queue: Add support for Server-Side Encryption ([#962](https://github.com/terraform-providers/terraform-provider-aws/issues/962))
* resource/aws_vpc: Add support for classiclink_dns_support ([#1079](https://github.com/terraform-providers/terraform-provider-aws/issues/1079))
* resource/aws_lambda_function: Add support for lambda_function vpc_config update ([#1080](https://github.com/terraform-providers/terraform-provider-aws/issues/1080))
* resource/aws_lambda_function: Add support for lambda_function dead_letter_config update ([#1080](https://github.com/terraform-providers/terraform-provider-aws/issues/1080))
* resource/aws_route53_health_check: add support for health_check regions ([#1116](https://github.com/terraform-providers/terraform-provider-aws/issues/1116))
* resource/aws_spot_instance_request: add support for request launch group ([#1097](https://github.com/terraform-providers/terraform-provider-aws/issues/1097))
* resource/aws_rds_cluster_instance: Export the RDI Resource ID for the instance ([#1142](https://github.com/terraform-providers/terraform-provider-aws/issues/1142))
* resource/aws_sns_topic_subscription: Support password-protected HTTPS endpoints ([#861](https://github.com/terraform-providers/terraform-provider-aws/issues/861))

BUG FIXES:

* provider: Remove assumeRoleHash ([#1227](https://github.com/terraform-providers/terraform-provider-aws/issues/1227))
* resource/aws_ami: Retry on `InvalidAMIID.NotFound` ([#1035](https://github.com/terraform-providers/terraform-provider-aws/issues/1035))
* resource/aws_iam_server_certificate: Fix restriction on length of `name_prefix` ([#1217](https://github.com/terraform-providers/terraform-provider-aws/issues/1217))
* resource/aws_autoscaling_group: Fix handling of empty `vpc_zone_identifier` (EC2 classic & default VPC) ([#1191](https://github.com/terraform-providers/terraform-provider-aws/issues/1191))
* resource/aws_ecr_repository_policy: Add retry logic to work around IAM eventual consistency ([#1165](https://github.com/terraform-providers/terraform-provider-aws/issues/1165))
* resource/aws_ecs_service: Fixes normalization issues in placement_strategy ([#1025](https://github.com/terraform-providers/terraform-provider-aws/issues/1025))
* resource/aws_eip: Retry reading EIPs on creation ([#1053](https://github.com/terraform-providers/terraform-provider-aws/issues/1053))
* resource/aws_elastic_beanstalk_environment: Avoid spurious diffs of JSON-based `setting`s ([#901](https://github.com/terraform-providers/terraform-provider-aws/issues/901))
* resource/aws_opsworks_permission: Fix 'set permissions' failing to set ssh access ([#1038](https://github.com/terraform-providers/terraform-provider-aws/issues/1038))
* resource/aws_s3_bucket_notification: Fix missing `bucket` field after import ([#978](https://github.com/terraform-providers/terraform-provider-aws/issues/978))
* resource/aws_sfn_state_machine: Handle another NotFound exception type ([#1062](https://github.com/terraform-providers/terraform-provider-aws/issues/1062))
* resource/aws_ssm_parameter: ForceNew on ssm_parameter rename ([#1022](https://github.com/terraform-providers/terraform-provider-aws/issues/1022))
* resource/aws_instance: Update SourceDestCheck modification on new resources ([#1065](https://github.com/terraform-providers/terraform-provider-aws/issues/1065))
* resource/aws_spot_instance_request: fixed and issue with network interfaces configuration ([#1070](https://github.com/terraform-providers/terraform-provider-aws/issues/1070))
* resource/aws_rds_cluster: Modify RDS Cluster after restoring from snapshot, if required ([#926](https://github.com/terraform-providers/terraform-provider-aws/issues/926))
* resource/aws_kms_alias: Retry lookups after creation ([#1040](https://github.com/terraform-providers/terraform-provider-aws/issues/1040))
* resource/aws_internet_gateway: Retry deletion properly on `DependencyViolation` ([#1021](https://github.com/terraform-providers/terraform-provider-aws/issues/1021))
* resource/aws_elb: Cleanup ENIs after deleting ELB ([#1036](https://github.com/terraform-providers/terraform-provider-aws/issues/1036))
* resource/aws_kms_key: Retry lookups after creation ([#1039](https://github.com/terraform-providers/terraform-provider-aws/issues/1039))
* resource/aws_dms_replication_instance: Add modifying as a pending creation state ([#1114](https://github.com/terraform-providers/terraform-provider-aws/issues/1114))
* resource/aws_redshift_cluster: Trigger ForceNew aws_redshift_cluster on encrypted change ([#1120](https://github.com/terraform-providers/terraform-provider-aws/issues/1120))
* resource/aws_default_network_acl: Add support for ipv6_cidr_block ([#1113](https://github.com/terraform-providers/terraform-provider-aws/issues/1113))
* resource/aws_autoscaling_group: Suppress diffs when an empty set is specified for `availability_zones` ([#1190](https://github.com/terraform-providers/terraform-provider-aws/issues/1190))
* resource/aws_vpc: Ignore ClassicLink DNS support in unsupported regions ([#1176](https://github.com/terraform-providers/terraform-provider-aws/issues/1176))
* resource/elastic_beanstalk_configuration_template: Handle missing platform ([#1222](https://github.com/terraform-providers/terraform-provider-aws/issues/1222))
* r/elasticache_parameter_group: support more than 20 parameters ([#1221](https://github.com/terraform-providers/terraform-provider-aws/issues/1221))
* data-source/aws_db_instance: Fix the output of subnet_group_name ([#1141](https://github.com/terraform-providers/terraform-provider-aws/issues/1141))
* data-source/aws_iam_server_certificate: Fix restriction on length of `name_prefix` ([#1217](https://github.com/terraform-providers/terraform-provider-aws/issues/1217))

## 0.1.2 (June 30, 2017)

FEATURES:

* **New Resource**: `aws_network_interface_sg_attachment` ([#860](https://github.com/terraform-providers/terraform-provider-aws/issues/860))
* **New Data Source**: `aws_ecr_repository` ([#944](https://github.com/terraform-providers/terraform-provider-aws/issues/944))

IMPROVEMENTS:

* Added ability to change the deadline for the EC2 metadata API endpoint ([#950](https://github.com/terraform-providers/terraform-provider-aws/issues/950))
* resource/aws_api_gateway_integration: Add support for specifying cache key parameters ([#893](https://github.com/terraform-providers/terraform-provider-aws/issues/893))
* resource/aws_cloudwatch_event_target: Add ecs_target ([#977](https://github.com/terraform-providers/terraform-provider-aws/issues/977))
* resource/aws_vpn_connection: Add BGP related information on aws_vpn_connection ([#973](https://github.com/terraform-providers/terraform-provider-aws/issues/973))
* resource/aws_cloudformation_stack: Add timeout support ([#994](https://github.com/terraform-providers/terraform-provider-aws/issues/994))
* resource/aws_ssm_parameter: Add support for ssm parameter overwrite ([#1006](https://github.com/terraform-providers/terraform-provider-aws/issues/1006))
* resource/aws_codebuild_project: Add support for environment privileged_mode [GH1009]
* resource/aws_dms_endpoint: Add support for dynamodb as an endpoint target ([#1002](https://github.com/terraform-providers/terraform-provider-aws/issues/1002))
* resource/aws_s3_bucket: Support lifecycle tags filter ([#899](https://github.com/terraform-providers/terraform-provider-aws/issues/899))
* resource/aws_s3_bucket_object: Allow to set WebsiteRedirect on S3 object ([#1020](https://github.com/terraform-providers/terraform-provider-aws/issues/1020))

BUG FIXES:

* resource/aws_waf: Only set FieldToMatch.Data if not empty ([#953](https://github.com/terraform-providers/terraform-provider-aws/issues/953))
* resource/aws_elastic_beanstalk_application_version: Scope labels to application ([#956](https://github.com/terraform-providers/terraform-provider-aws/issues/956))
* resource/aws_s3_bucket: Allow use of `days = 0` with lifecycle transition ([#957](https://github.com/terraform-providers/terraform-provider-aws/issues/957))
* resource/aws_ssm_maintenance_window_task: Make task_parameters updateable on aws_ssm_maintenance_window_task resource ([#965](https://github.com/terraform-providers/terraform-provider-aws/issues/965))
* resource/aws_kinesis_stream: don't force stream destroy on shard_count update ([#894](https://github.com/terraform-providers/terraform-provider-aws/issues/894))
* resource/aws_cloudfront_distribution: Remove validation from custom_origin params ([#987](https://github.com/terraform-providers/terraform-provider-aws/issues/987))
* resource_aws_route53_record: Allow import of Route 53 records with underscores in the name ([#14717](https://github.com/hashicorp/terraform/pull/14717))
* d/aws_db_snapshot: Id was being set incorrectly ([#992](https://github.com/terraform-providers/terraform-provider-aws/issues/992))
* resource/aws_spot_fleet_request: Raise the create timeout to be 10m ([#993](https://github.com/terraform-providers/terraform-provider-aws/issues/993))
* d/aws_ecs_cluster: Add ARN as an exported param for aws_ecs_cluster ([#991](https://github.com/terraform-providers/terraform-provider-aws/issues/991))
* resource/aws_ebs_volume: Not setting the state for ebs_volume correctly ([#999](https://github.com/terraform-providers/terraform-provider-aws/issues/999))
* resource/aws_network_acl: Make action in ingress / egress case insensitive ([#1000](https://github.com/terraform-providers/terraform-provider-aws/issues/1000))

## 0.1.1 (June 21, 2017)

BUG FIXES:

* Fixing malformed ARN attribute for aws_security_group data source ([#910](https://github.com/terraform-providers/terraform-provider-aws/issues/910))

## 0.1.0 (June 20, 2017)

BACKWARDS INCOMPATIBILITIES / NOTES:

FEATURES:

* **New Resource:** `aws_vpn_gateway_route_propagation` [[#15137](https://github.com/terraform-providers/terraform-provider-aws/issues/15137)](https://github.com/hashicorp/terraform/pull/15137)

IMPROVEMENTS:

* resource/ebs_snapshot: Add support for tags ([#3](https://github.com/terraform-providers/terraform-provider-aws/issues/3))
* resource/aws_elasticsearch_domain: now retries on IAM role association failure ([#12](https://github.com/terraform-providers/terraform-provider-aws/issues/12))
* resource/codebuild_project: Increase timeout for creation retry (IAM) ([#904](https://github.com/terraform-providers/terraform-provider-aws/issues/904))
* resource/dynamodb_table: Expose stream_label attribute ([#20](https://github.com/terraform-providers/terraform-provider-aws/issues/20))
* resource/opsworks: Add support for configurable timeouts in AWS OpsWorks Instances. ([#857](https://github.com/terraform-providers/terraform-provider-aws/issues/857))
* Fix handling of AdRoll's hologram clients ([#17](https://github.com/terraform-providers/terraform-provider-aws/issues/17))
* resource/sqs_queue: Add support for name_prefix to aws_sqs_queue ([#855](https://github.com/terraform-providers/terraform-provider-aws/issues/855))
* resource/iam_role: Add support for iam_role tp force_detach_policies ([#890](https://github.com/terraform-providers/terraform-provider-aws/issues/890))

BUG FIXES:

* fix aws cidr validation error [[#15158](https://github.com/terraform-providers/terraform-provider-aws/issues/15158)](https://github.com/hashicorp/terraform/pull/15158)
* resource/elasticache_parameter_group: Retry deletion on InvalidCacheParameterGroupState ([#8](https://github.com/terraform-providers/terraform-provider-aws/issues/8))
* resource/security_group: Raise creation timeout ([#9](https://github.com/terraform-providers/terraform-provider-aws/issues/9))
* resource/rds_cluster: Retry modification on InvalidDBClusterStateFault ([#18](https://github.com/terraform-providers/terraform-provider-aws/issues/18))
* resource/lambda: Fix incorrect GovCloud regexes ([#16](https://github.com/terraform-providers/terraform-provider-aws/issues/16))
* Allow ipv6_cidr_block to be assigned to peering_connection ([#879](https://github.com/terraform-providers/terraform-provider-aws/issues/879))
* resource/rds_db_instance: Correctly create cross-region encrypted replica ([#865](https://github.com/terraform-providers/terraform-provider-aws/issues/865))
* resource/eip: dissociate EIP on update ([#878](https://github.com/terraform-providers/terraform-provider-aws/issues/878))
* resource/iam_server_certificate: Increase deletion timeout ([#907](https://github.com/terraform-providers/terraform-provider-aws/issues/907))
