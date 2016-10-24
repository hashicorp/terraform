---
layout: "aws"
page_title: "AWS: aws_emr_cluster"
sidebar_current: "docs-aws-resource-emr-cluster"
description: |-
  Provides an Elastic MapReduce Cluster
---

# aws\_emr\_cluster

Provides an Elastic MapReduce Cluster, a web service that makes it easy to 
process large amounts of data efficiently. See [Amazon Elastic MapReduce Documentation](https://aws.amazon.com/documentation/elastic-mapreduce/) 
for more information. 

## Example Usage

```
resource "aws_emr_cluster" "emr-test-cluster" {
  name          = "emr-test-arn"
  release_label = "emr-4.6.0"
  applications  = ["Spark"]

  ec2_attributes {
    subnet_id                         = "${aws_subnet.main.id}"
    emr_managed_master_security_group = "${aws_security_group.sg.id}"
    emr_managed_slave_security_group  = "${aws_security_group.sg.id}"
    instance_profile                  = "${aws_iam_instance_profile.emr_profile.arn}"
  }

  master_instance_type = "m3.xlarge"
  core_instance_type   = "m3.xlarge"
  core_instance_count  = 1

  tags {
    role     = "rolename"
    env      = "env"
  }

  bootstrap_action {
    path = "s3://elasticmapreduce/bootstrap-actions/run-if"
    name = "runif"
    args = ["instance.isMaster=true", "echo running on master node"]
  }

  configurations = "test-fixtures/emr_configurations.json"

  service_role = "${aws_iam_role.iam_emr_service_role.arn}"
}
```

The `aws_emr_cluster` resource typically requires two IAM roles, one for the EMR Cluster
to use as a service, and another to place on your Cluster Instances to interact
with AWS from those instances. The suggested role policy template for the EMR service is `AmazonElasticMapReduceRole`,
and `AmazonElasticMapReduceforEC2Role` for the EC2 profile. See the [Getting
Started](https://docs.aws.amazon.com/ElasticMapReduce/latest/ManagementGuide/emr-gs-launch-sample-cluster.html) 
guide for more information on these IAM roles. There is also a fully-bootable
example Terraform configuration at the bottom of this page. 

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the job flow
* `release_label` - (Required) The release label for the Amazon EMR release
* `master_instance_type` - (Required) The EC2 instance type of the master node
* `core_instance_type` - (Optional) The EC2 instance type of the slave nodes
* `core_instance_count` - (Optional) number of Amazon EC2 instances used to execute the job flow. Default `0`
* `log_uri` - (Optional) S3 bucket to write the log files of the job flow. If a value
	is not provided, logs are not created
* `applications` - (Optional) A list of applications for the cluster. Valid values are: `Hadoop`, `Hive`,
	`Mahout`, `Pig`, and `Spark.` Case insensitive
* `ec2_attributes` - (Optional) attributes for the EC2 instances running the job
flow. Defined below
* `bootstrap_action` - (Optional) list of bootstrap actions that will be run before Hadoop is started on
	the cluster nodes. Defined below
* `configurations` - (Optional) list of configurations supplied for the EMR cluster you are creating
* `service_role` - (Optional) IAM role that will be assumed by the Amazon EMR service to access AWS resources
* `visible_to_all_users` - (Optional) Whether the job flow is visible to all IAM users of the AWS account associated with the job flow. Default `true`
* `tags` - (Optional) list of tags to apply to the EMR Cluster



## ec2\_attributes

Attributes for the Amazon EC2 instances running the job flow

* `key_name` - (Optional) Amazon EC2 key pair that can be used to ssh to the master
	node as the user called `hadoop`
* `subnet_id` - (Optional) VPC subnet id where you want the job flow to launch.
Cannot specify the `cc1.4xlarge` instance type for nodes of a job flow launched in a Amazon VPC
* `additional_master_security_groups` - (Optional) list of additional Amazon EC2 security group IDs for the master node
* `additional_slave_security_groups` - (Optional) list of additional Amazon EC2 security group IDs for the slave nodes
* `emr_managed_master_security_group` - (Optional) identifier of the Amazon EC2 security group for the master node
* `emr_managed_slave_security_group` - (Optional) identifier of the Amazon EC2 security group for the slave nodes
* `instance_profile` - (Optional) Instance Profile for EC2 instances of the cluster assume this role


## bootstrap\_action 

* `name` - (Required) name of the bootstrap action
* `path` - (Required) location of the script to run during a bootstrap action. Can be either a location in Amazon S3 or on a local file system
* `args` - (Optional) list of command line arguments to pass to the bootstrap action script

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the EMR Cluster
* `name` 
* `release_label` 
* `master_instance_type`
* `core_instance_type`
* `core_instance_count`
* `log_uri`
* `applications`
* `ec2_attributes`
* `bootstrap_action`
* `configurations`
* `service_role`
* `visible_to_all_users`
* `tags`


## Example bootable config

**NOTE:** This configuration demonstrates a minimal configuration needed to 
boot an example EMR Cluster. It is not meant to display best practices. Please
use at your own risk.


```
provider "aws" {
  region = "us-west-2"
}

resource "aws_emr_cluster" "tf-test-cluster" {
  name          = "emr-test-arn"
  release_label = "emr-4.6.0"
  applications  = ["Spark"]

  ec2_attributes {
    subnet_id                         = "${aws_subnet.main.id}"
    emr_managed_master_security_group = "${aws_security_group.allow_all.id}"
    emr_managed_slave_security_group  = "${aws_security_group.allow_all.id}"
    instance_profile                  = "${aws_iam_instance_profile.emr_profile.arn}"
  }

  master_instance_type = "m3.xlarge"
  core_instance_type   = "m3.xlarge"
  core_instance_count  = 1

  tags {
    role     = "rolename"
    dns_zone = "env_zone"
    env      = "env"
    name     = "name-env"
  }

  bootstrap_action {
    path = "s3://elasticmapreduce/bootstrap-actions/run-if"
    name = "runif"
    args = ["instance.isMaster=true", "echo running on master node"]
  }

  configurations = "test-fixtures/emr_configurations.json"

  service_role = "${aws_iam_role.iam_emr_service_role.arn}"
}

resource "aws_security_group" "allow_all" {
  name        = "allow_all"
  description = "Allow all inbound traffic"
  vpc_id      = "${aws_vpc.main.id}"

  ingress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  depends_on = ["aws_subnet.main"]

  lifecycle {
    ignore_changes = ["ingress", "egress"]
  }

  tags {
    name = "emr_test"
  }
}

resource "aws_vpc" "main" {
  cidr_block           = "168.31.0.0/16"
  enable_dns_hostnames = true

  tags {
    name = "emr_test"
  }
}

resource "aws_subnet" "main" {
  vpc_id     = "${aws_vpc.main.id}"
  cidr_block = "168.31.0.0/20"

  tags {
    name = "emr_test"
  }
}

resource "aws_internet_gateway" "gw" {
  vpc_id = "${aws_vpc.main.id}"
}

resource "aws_route_table" "r" {
  vpc_id = "${aws_vpc.main.id}"

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = "${aws_internet_gateway.gw.id}"
  }
}

resource "aws_main_route_table_association" "a" {
  vpc_id         = "${aws_vpc.main.id}"
  route_table_id = "${aws_route_table.r.id}"
}

###

# IAM Role setups

###

# IAM role for EMR Service
resource "aws_iam_role" "iam_emr_service_role" {
  name = "iam_emr_service_role"

  assume_role_policy = <<EOF
{
  "Version": "2008-10-17",
  "Statement": [
    {
      "Sid": "",
      "Effect": "Allow",
      "Principal": {
        "Service": "elasticmapreduce.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
EOF
}

resource "aws_iam_role_policy" "iam_emr_service_policy" {
  name = "iam_emr_service_policy"
  role = "${aws_iam_role.iam_emr_service_role.id}"

  policy = <<EOF
{
    "Version": "2012-10-17",
    "Statement": [{
        "Effect": "Allow",
        "Resource": "*",
        "Action": [
            "ec2:AuthorizeSecurityGroupEgress",
            "ec2:AuthorizeSecurityGroupIngress",
            "ec2:CancelSpotInstanceRequests",
            "ec2:CreateNetworkInterface",
            "ec2:CreateSecurityGroup",
            "ec2:CreateTags",
            "ec2:DeleteNetworkInterface",
            "ec2:DeleteSecurityGroup",
            "ec2:DeleteTags",
            "ec2:DescribeAvailabilityZones",
            "ec2:DescribeAccountAttributes",
            "ec2:DescribeDhcpOptions",
            "ec2:DescribeInstanceStatus",
            "ec2:DescribeInstances",
            "ec2:DescribeKeyPairs",
            "ec2:DescribeNetworkAcls",
            "ec2:DescribeNetworkInterfaces",
            "ec2:DescribePrefixLists",
            "ec2:DescribeRouteTables",
            "ec2:DescribeSecurityGroups",
            "ec2:DescribeSpotInstanceRequests",
            "ec2:DescribeSpotPriceHistory",
            "ec2:DescribeSubnets",
            "ec2:DescribeVpcAttribute",
            "ec2:DescribeVpcEndpoints",
            "ec2:DescribeVpcEndpointServices",
            "ec2:DescribeVpcs",
            "ec2:DetachNetworkInterface",
            "ec2:ModifyImageAttribute",
            "ec2:ModifyInstanceAttribute",
            "ec2:RequestSpotInstances",
            "ec2:RevokeSecurityGroupEgress",
            "ec2:RunInstances",
            "ec2:TerminateInstances",
            "ec2:DeleteVolume",
            "ec2:DescribeVolumeStatus",
            "ec2:DescribeVolumes",
            "ec2:DetachVolume",
            "iam:GetRole",
            "iam:GetRolePolicy",
            "iam:ListInstanceProfiles",
            "iam:ListRolePolicies",
            "iam:PassRole",
            "s3:CreateBucket",
            "s3:Get*",
            "s3:List*",
            "sdb:BatchPutAttributes",
            "sdb:Select",
            "sqs:CreateQueue",
            "sqs:Delete*",
            "sqs:GetQueue*",
            "sqs:PurgeQueue",
            "sqs:ReceiveMessage"
        ]
    }]
}
EOF
}

# IAM Role for EC2 Instance Profile
resource "aws_iam_role" "iam_emr_profile_role" {
  name = "iam_emr_profile_role"

  assume_role_policy = <<EOF
{
  "Version": "2008-10-17",
  "Statement": [
    {
      "Sid": "",
      "Effect": "Allow",
      "Principal": {
        "Service": "ec2.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
EOF
}

resource "aws_iam_instance_profile" "emr_profile" {
  name  = "emr_profile"
  roles = ["${aws_iam_role.iam_emr_profile_role.name}"]
}

resource "aws_iam_role_policy" "iam_emr_profile_policy" {
  name = "iam_emr_profile_policy"
  role = "${aws_iam_role.iam_emr_profile_role.id}"

  policy = <<EOF
{
    "Version": "2012-10-17",
    "Statement": [{
        "Effect": "Allow",
        "Resource": "*",
        "Action": [
            "cloudwatch:*",
            "dynamodb:*",
            "ec2:Describe*",
            "elasticmapreduce:Describe*",
            "elasticmapreduce:ListBootstrapActions",
            "elasticmapreduce:ListClusters",
            "elasticmapreduce:ListInstanceGroups",
            "elasticmapreduce:ListInstances",
            "elasticmapreduce:ListSteps",
            "kinesis:CreateStream",
            "kinesis:DeleteStream",
            "kinesis:DescribeStream",
            "kinesis:GetRecords",
            "kinesis:GetShardIterator",
            "kinesis:MergeShards",
            "kinesis:PutRecord",
            "kinesis:SplitShard",
            "rds:Describe*",
            "s3:*",
            "sdb:*",
            "sns:*",
            "sqs:*"
        ]
    }]
}
EOF
}
```
