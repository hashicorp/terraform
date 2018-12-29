---
layout: "docs"
page_title: "Import: Resource Importability"
sidebar_current: "docs-import-importability"
description: |-
  Each resource in Terraform must implement some basic logic to become
  importable. As a result, not all Terraform resources are currently importable.
---

# Resource Importability

Each resource in Terraform must implement some basic logic to become
importable. As a result, not all Terraform resources are currently importable.
If you find a resource that you want to import and Terraform reports
that it isn't importable, please report an issue.

Converting a resource to be importable is also relatively simple, so if
you're interested in contributing that functionality, the Terraform team
would be grateful.

To make a resource importable, please see the
[plugin documentation on writing a resource](/docs/plugins/provider.html).

## Currently Available to Import

### AWS

* aws_api_gateway_account
* aws_api_gateway_api_key
* aws_autoscaling_group
* aws_cloudfront_distribution
* aws_cloudfront_origin_access_identity
* aws_cloudtrail
* aws_cloudwatch_event_rule
* aws_cloudwatch_log_group
* aws_cloudwatch_metric_alarm
* aws_customer_gateway
* aws_db_event_subscription
* aws_db_instance
* aws_db_option_group
* aws_db_parameter_group
* aws_db_security_group
* aws_db_subnet_group
* aws_dms_certificate
* aws_dms_endpoint
* aws_dms_replication_instance
* aws_dms_replication_subnet_group
* aws_dms_replication_task
* aws_dynamodb_table
* aws_ebs_volume
* aws_ecr_repository
* aws_efs_file_system
* aws_efs_mount_target
* aws_eip
* aws_elastic_beanstalk_application
* aws_elastic_beanstalk_environment
* aws_elasticache_cluster
* aws_elasticache_parameter_group
* aws_elasticache_subnet_group
* aws_elb
* aws_flow_log
* aws_glacier_vault
* aws_iam_account_password_policy
* aws_iam_group
* aws_iam_instance_profile
* aws_iam_role
* aws_iam_saml_provider
* aws_iam_server_certificate
* aws_iam_user
* aws_instance
* aws_internet_gateway
* aws_key_pair
* aws_kms_key
* aws_lambda_function
* aws_launch_configuration
* aws_nat_gateway
* aws_network_acl
* aws_network_interface
* aws_opsworks_custom_layer
* aws_opsworks_stack
* aws_placement_group
* aws_rds_cluster
* aws_rds_cluster_instance
* aws_rds_cluster_parameter_group
* aws_redshift_cluster
* aws_redshift_parameter_group
* aws_redshift_security_group
* aws_redshift_subnet_group
* aws_route53_delegation_set
* aws_route53_health_check
* aws_route53_zone
* aws_route_table
* aws_s3_bucket
* aws_security_group
* aws_ses_domain_identity
* aws_ses_receipt_filter
* aws_ses_receipt_rule_set
* aws_simpledb_domain
* aws_sns_topic
* aws_sns_topic_subscription
* aws_sqs_queue
* aws_subnet
* aws_vpc
* aws_vpc_dhcp_options
* aws_vpc_endpoint
* aws_vpc_peering_connection
* aws_vpn_connection
* aws_vpn_gateway


### Azure (Resource Manager)

* azurerm_availability_set
* azurerm_dns_zone
* azurerm_local_network_gateway
* azurerm_network_security_group
* azurerm_network_security_rule
* azurerm_public_ip
* azurerm_resource_group
* azurerm_sql_firewall_rule
* azurerm_storage_account
* azurerm_virtual_network

### Circonus

* circonus_check
* circonus_contact_group

### DigitalOcean

* digitalocean_domain
* digitalocean_droplet
* digitalocean_floating_ip
* digitalocean_ssh_key
* digitalocean_tag
* digitalocean_volume

### Fastly

* fastly_service_v1

### Google

* google_compute_autoscaler
* google_compute_firewall
* google_compute_forwarding_rule
* google_compute_http_health_check
* google_compute_instance_group_manager
* google_compute_instance_template
* google_compute_target_pool
* google_project

### OpenStack

* openstack_blockstorage_volume_v1
* openstack_blockstorage_volume_v2
* openstack_compute_floatingip_v2
* openstack_compute_keypair_v2
* openstack_compute_secgroup_v2
* openstack_compute_servergroup_v2
* openstack_fw_firewall_v1
* openstack_fw_policy_v1
* openstack_fw_rule_v1
* openstack_lb_member_v1
* openstack_lb_monitor_v1
* openstack_lb_pool_v1
* openstack_lb_vip_v1
* openstack_networking_floatingip_v2
* openstack_networking_network_v2
* openstack_networking_port_v2
* openstack_networking_secgroup_rule_v2
* openstack_networking_secgroup_v2
* openstack_networking_subnet_v2

### PostgreSQL

* postgresql_database

### Triton

* triton_key
* triton_firewall_rule
* triton_vlan
* triton_fabric
* triton_machine
