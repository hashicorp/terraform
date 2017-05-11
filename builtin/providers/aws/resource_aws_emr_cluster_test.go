package aws

import (
	"fmt"
	"log"
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/emr"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSEMRCluster_basic(t *testing.T) {
	var cluster emr.Cluster
	r := acctest.RandInt()
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSEmrDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSEmrClusterConfig(r),
				Check:  testAccCheckAWSEmrClusterExists("aws_emr_cluster.tf-test-cluster", &cluster),
			},
		},
	})
}

func TestAccAWSEMRCluster_security_config(t *testing.T) {
	var cluster emr.Cluster
	r := acctest.RandInt()
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSEmrDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSEmrClusterConfig_SecurityConfiguration(r),
				Check:  testAccCheckAWSEmrClusterExists("aws_emr_cluster.tf-test-cluster", &cluster),
			},
		},
	})
}

func TestAccAWSEMRCluster_bootstrap_ordering(t *testing.T) {
	var cluster emr.Cluster
	rName := acctest.RandomWithPrefix("tf-emr-bootstrap")
	argsInts := []string{
		"1",
		"2",
		"3",
		"4",
		"5",
		"6",
		"7",
		"8",
		"9",
		"10",
	}

	argsStrings := []string{
		"instance.isMaster=true",
		"echo running on master node",
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSEmrDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSEmrClusterConfig_bootstrap(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEmrClusterExists("aws_emr_cluster.test", &cluster),
					testAccCheck_bootstrap_order(&cluster, argsInts, argsStrings),
				),
			},
		},
	})
}

func TestAccAWSEMRCluster_terminationProtected(t *testing.T) {
	var cluster emr.Cluster
	r := acctest.RandInt()
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSEmrDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSEmrClusterConfig(r),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEmrClusterExists("aws_emr_cluster.tf-test-cluster", &cluster),
					resource.TestCheckResourceAttr(
						"aws_emr_cluster.tf-test-cluster", "termination_protection", "false"),
				),
			},
			{
				Config: testAccAWSEmrClusterConfigTerminationPolicyUpdated(r),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEmrClusterExists("aws_emr_cluster.tf-test-cluster", &cluster),
					resource.TestCheckResourceAttr(
						"aws_emr_cluster.tf-test-cluster", "termination_protection", "true"),
				),
			},
			{
				//Need to turn off termination_protection to allow the job to be deleted
				Config: testAccAWSEmrClusterConfig(r),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEmrClusterExists("aws_emr_cluster.tf-test-cluster", &cluster),
				),
			},
		},
	})
}

func TestAccAWSEMRCluster_visibleToAllUsers(t *testing.T) {
	var cluster emr.Cluster
	r := acctest.RandInt()
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSEmrDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSEmrClusterConfig(r),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEmrClusterExists("aws_emr_cluster.tf-test-cluster", &cluster),
					resource.TestCheckResourceAttr(
						"aws_emr_cluster.tf-test-cluster", "visible_to_all_users", "true"),
				),
			},
			{
				Config: testAccAWSEmrClusterConfigVisibleToAllUsersUpdated(r),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEmrClusterExists("aws_emr_cluster.tf-test-cluster", &cluster),
					resource.TestCheckResourceAttr(
						"aws_emr_cluster.tf-test-cluster", "visible_to_all_users", "false"),
				),
			},
		},
	})
}

func TestAccAWSEMRCluster_tags(t *testing.T) {
	var cluster emr.Cluster
	r := acctest.RandInt()
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSEmrDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSEmrClusterConfig(r),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEmrClusterExists("aws_emr_cluster.tf-test-cluster", &cluster),
					resource.TestCheckResourceAttr("aws_emr_cluster.tf-test-cluster", "tags.%", "4"),
					resource.TestCheckResourceAttr(
						"aws_emr_cluster.tf-test-cluster", "tags.role", "rolename"),
					resource.TestCheckResourceAttr(
						"aws_emr_cluster.tf-test-cluster", "tags.dns_zone", "env_zone"),
					resource.TestCheckResourceAttr(
						"aws_emr_cluster.tf-test-cluster", "tags.env", "env"),
					resource.TestCheckResourceAttr(
						"aws_emr_cluster.tf-test-cluster", "tags.name", "name-env")),
			},
			{
				Config: testAccAWSEmrClusterConfigUpdatedTags(r),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEmrClusterExists("aws_emr_cluster.tf-test-cluster", &cluster),
					resource.TestCheckResourceAttr("aws_emr_cluster.tf-test-cluster", "tags.%", "3"),
					resource.TestCheckResourceAttr(
						"aws_emr_cluster.tf-test-cluster", "tags.dns_zone", "new_zone"),
					resource.TestCheckResourceAttr(
						"aws_emr_cluster.tf-test-cluster", "tags.Env", "production"),
					resource.TestCheckResourceAttr(
						"aws_emr_cluster.tf-test-cluster", "tags.name", "name-env"),
				),
			},
		},
	})
}

func testAccCheck_bootstrap_order(cluster *emr.Cluster, argsInts, argsStrings []string) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		emrconn := testAccProvider.Meta().(*AWSClient).emrconn
		req := emr.ListBootstrapActionsInput{
			ClusterId: cluster.Id,
		}

		resp, err := emrconn.ListBootstrapActions(&req)
		if err != nil {
			return fmt.Errorf("[ERR] Error listing boostrap actions in test: %s", err)
		}

		// make sure we actually checked something
		var ran bool
		for _, ba := range resp.BootstrapActions {
			// assume name matches the config
			rArgs := aws.StringValueSlice(ba.Args)
			if *ba.Name == "test" {
				ran = true
				if !reflect.DeepEqual(argsInts, rArgs) {
					return fmt.Errorf("Error matching Bootstrap args:\n\texpected: %#v\n\tgot: %#v", argsInts, rArgs)
				}
			} else if *ba.Name == "runif" {
				ran = true
				if !reflect.DeepEqual(argsStrings, rArgs) {
					return fmt.Errorf("Error matching Bootstrap args:\n\texpected: %#v\n\tgot: %#v", argsStrings, rArgs)
				}
			}
		}

		if !ran {
			return fmt.Errorf("Expected to compare bootstrap actions, but no checks were ran")
		}

		return nil
	}
}

func testAccCheckAWSEmrDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).emrconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_emr_cluster" {
			continue
		}

		params := &emr.DescribeClusterInput{
			ClusterId: aws.String(rs.Primary.ID),
		}

		describe, err := conn.DescribeCluster(params)

		if err == nil {
			if describe.Cluster != nil &&
				*describe.Cluster.Status.State == "WAITING" {
				return fmt.Errorf("EMR Cluster still exists")
			}
		}

		providerErr, ok := err.(awserr.Error)
		if !ok {
			return err
		}

		log.Printf("[ERROR] %v", providerErr)
	}

	return nil
}

func testAccCheckAWSEmrClusterExists(n string, v *emr.Cluster) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No cluster id set")
		}
		conn := testAccProvider.Meta().(*AWSClient).emrconn
		describe, err := conn.DescribeCluster(&emr.DescribeClusterInput{
			ClusterId: aws.String(rs.Primary.ID),
		})
		if err != nil {
			return fmt.Errorf("EMR error: %v", err)
		}

		if describe.Cluster != nil &&
			*describe.Cluster.Id != rs.Primary.ID {
			return fmt.Errorf("EMR cluser not found")
		}

		*v = *describe.Cluster

		if describe.Cluster != nil &&
			*describe.Cluster.Status.State != "WAITING" {
			return fmt.Errorf("EMR cluser is not up yet")
		}

		return nil
	}
}

func testAccAWSEmrClusterConfig_bootstrap(r string) string {
	return fmt.Sprintf(`
resource "aws_emr_cluster" "test" {
  count                = 1
  name                 = "%s"
  release_label        = "emr-5.0.0"
  applications         = ["Hadoop", "Hive"]
  log_uri              = "s3n://terraform/testlog/"
  master_instance_type = "m4.large"
  core_instance_type   = "m1.small"
  core_instance_count  = 1
  service_role         = "${aws_iam_role.iam_emr_default_role.arn}"

  depends_on = ["aws_main_route_table_association.a"]

  ec2_attributes {
    subnet_id = "${aws_subnet.main.id}"

    emr_managed_master_security_group = "${aws_security_group.allow_all.id}"
    emr_managed_slave_security_group  = "${aws_security_group.allow_all.id}"
    instance_profile                  = "${aws_iam_instance_profile.emr_profile.arn}"
  }

  bootstrap_action {
    path = "s3://elasticmapreduce/bootstrap-actions/run-if"
    name = "runif"
    args = ["instance.isMaster=true", "echo running on master node"]
  }

  bootstrap_action = [
    {
      path = "s3://${aws_s3_bucket.tester.bucket}/testscript.sh"
      name = "test"

      args = ["1",
        "2",
        "3",
        "4",
        "5",
        "6",
        "7",
        "8",
        "9",
        "10",
      ]
    },
  ]
}

resource "aws_iam_instance_profile" "emr_profile" {
  name = "%s_profile"
  role = "${aws_iam_role.iam_emr_profile_role.name}"
}

resource "aws_iam_role" "iam_emr_default_role" {
  name = "%s_default_role"

  assume_role_policy = <<EOT
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
EOT
}

resource "aws_iam_role" "iam_emr_profile_role" {
  name = "%s_profile_role"

  assume_role_policy = <<EOT
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
EOT
}

resource "aws_iam_role_policy_attachment" "profile-attach" {
  role       = "${aws_iam_role.iam_emr_profile_role.id}"
  policy_arn = "${aws_iam_policy.iam_emr_profile_policy.arn}"
}

resource "aws_iam_role_policy_attachment" "service-attach" {
  role       = "${aws_iam_role.iam_emr_default_role.id}"
  policy_arn = "${aws_iam_policy.iam_emr_default_policy.arn}"
}

resource "aws_iam_policy" "iam_emr_default_policy" {
  name = "%s_emr"

  policy = <<EOT
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
EOT
}

resource "aws_iam_policy" "iam_emr_profile_policy" {
  name = "%s_profile"

  policy = <<EOT
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
EOT
}

resource "aws_vpc" "main" {
  cidr_block           = "168.31.0.0/16"
  enable_dns_hostnames = true

  tags {
    name = "emr_test_cts"
  }
}

resource "aws_subnet" "main" {
  vpc_id     = "${aws_vpc.main.id}"
  cidr_block = "168.31.0.0/20"

  tags {
    name = "emr_test_cts"
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

output "cluser_id" {
  value = "${aws_emr_cluster.test.id}"
}

resource "aws_s3_bucket" "tester" {
  bucket = "%s"
  acl    = "public-read"
}

resource "aws_s3_bucket_object" "testobject" {
  bucket = "${aws_s3_bucket.tester.bucket}"
  key    = "testscript.sh"

  #source = "testscript.sh"
  content = "${data.template_file.testscript.rendered}"
  acl     = "public-read"
}

data "template_file" "testscript" {
  template = <<POLICY
#!/bin/bash

echo $@
POLICY
}`, r, r, r, r, r, r, r)
}

func testAccAWSEmrClusterConfig(r int) string {
	return fmt.Sprintf(`
provider "aws" {
  region = "us-west-2"
}

resource "aws_emr_cluster" "tf-test-cluster" {
  name          = "emr-test-%d"
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

  keep_job_flow_alive_when_no_steps = true
  termination_protection = false

  bootstrap_action {
    path = "s3://elasticmapreduce/bootstrap-actions/run-if"
    name = "runif"
    args = ["instance.isMaster=true", "echo running on master node"]
  }

  configurations = "test-fixtures/emr_configurations.json"

  depends_on = ["aws_main_route_table_association.a"]

  service_role = "${aws_iam_role.iam_emr_default_role.arn}"
  autoscaling_role = "${aws_iam_role.emr-autoscaling-role.arn}"
}

resource "aws_security_group" "allow_all" {
  name        = "allow_all_%d"
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
    name = "emr_test_%d"
  }
}

resource "aws_subnet" "main" {
  vpc_id     = "${aws_vpc.main.id}"
  cidr_block = "168.31.0.0/20"

  tags {
    name = "emr_test_%d"
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

# IAM things

###

# IAM role for EMR Service
resource "aws_iam_role" "iam_emr_default_role" {
  name = "iam_emr_default_role_%d"

  assume_role_policy = <<EOT
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
EOT
}

resource "aws_iam_role_policy_attachment" "service-attach" {
  role       = "${aws_iam_role.iam_emr_default_role.id}"
  policy_arn = "${aws_iam_policy.iam_emr_default_policy.arn}"
}

resource "aws_iam_policy" "iam_emr_default_policy" {
  name = "iam_emr_default_policy_%d"

  policy = <<EOT
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
EOT
}

# IAM Role for EC2 Instance Profile
resource "aws_iam_role" "iam_emr_profile_role" {
  name = "iam_emr_profile_role_%d"

  assume_role_policy = <<EOT
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
EOT
}

resource "aws_iam_instance_profile" "emr_profile" {
  name  = "emr_profile_%d"
  roles = ["${aws_iam_role.iam_emr_profile_role.name}"]
}

resource "aws_iam_role_policy_attachment" "profile-attach" {
  role       = "${aws_iam_role.iam_emr_profile_role.id}"
  policy_arn = "${aws_iam_policy.iam_emr_profile_policy.arn}"
}

resource "aws_iam_policy" "iam_emr_profile_policy" {
  name = "iam_emr_profile_policy_%d"

  policy = <<EOT
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
EOT
}

# IAM Role for autoscaling
resource "aws_iam_role" "emr-autoscaling-role" {
  name               = "EMR_AutoScaling_DefaultRole_%d"
  assume_role_policy = "${data.aws_iam_policy_document.emr-autoscaling-role-policy.json}"
}

data "aws_iam_policy_document" "emr-autoscaling-role-policy" {
  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRole"]
    principals = {
      type        = "Service"
      identifiers = ["elasticmapreduce.amazonaws.com","application-autoscaling.amazonaws.com"]
    }
  }
}

resource "aws_iam_role_policy_attachment" "emr-autoscaling-role" {
  role       = "${aws_iam_role.emr-autoscaling-role.name}"
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonElasticMapReduceforAutoScalingRole"
}
`, r, r, r, r, r, r, r, r, r, r)
}

func testAccAWSEmrClusterConfig_SecurityConfiguration(r int) string {
	return fmt.Sprintf(`
provider "aws" {
  region = "us-west-2"
}

resource "aws_emr_cluster" "tf-test-cluster" {
  name          = "emr-test-%d"
  release_label = "emr-5.5.0"
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

  security_configuration = "${aws_emr_security_configuration.foo.name}"

  tags {
    role     = "rolename"
    dns_zone = "env_zone"
    env      = "env"
    name     = "name-env"
  }

  keep_job_flow_alive_when_no_steps = true
  termination_protection = false

  bootstrap_action {
    path = "s3://elasticmapreduce/bootstrap-actions/run-if"
    name = "runif"
    args = ["instance.isMaster=true", "echo running on master node"]
  }

  configurations = "test-fixtures/emr_configurations.json"

  depends_on = ["aws_main_route_table_association.a"]

  service_role = "${aws_iam_role.iam_emr_default_role.arn}"
  autoscaling_role = "${aws_iam_role.emr-autoscaling-role.arn}"
}

resource "aws_security_group" "allow_all" {
  name        = "allow_all_%d"
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
    name = "emr_test_%d"
  }
}

resource "aws_subnet" "main" {
  vpc_id     = "${aws_vpc.main.id}"
  cidr_block = "168.31.0.0/20"

  tags {
    name = "emr_test_%d"
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

# IAM things

###

# IAM role for EMR Service
resource "aws_iam_role" "iam_emr_default_role" {
  name = "iam_emr_default_role_%d"

  assume_role_policy = <<EOT
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
EOT
}

resource "aws_iam_role_policy_attachment" "service-attach" {
  role       = "${aws_iam_role.iam_emr_default_role.id}"
  policy_arn = "${aws_iam_policy.iam_emr_default_policy.arn}"
}

resource "aws_iam_policy" "iam_emr_default_policy" {
  name = "iam_emr_default_policy_%d"

  policy = <<EOT
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
EOT
}

# IAM Role for EC2 Instance Profile
resource "aws_iam_role" "iam_emr_profile_role" {
  name = "iam_emr_profile_role_%d"

  assume_role_policy = <<EOT
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
EOT
}

resource "aws_iam_instance_profile" "emr_profile" {
  name  = "emr_profile_%d"
  roles = ["${aws_iam_role.iam_emr_profile_role.name}"]
}

resource "aws_iam_role_policy_attachment" "profile-attach" {
  role       = "${aws_iam_role.iam_emr_profile_role.id}"
  policy_arn = "${aws_iam_policy.iam_emr_profile_policy.arn}"
}

resource "aws_iam_policy" "iam_emr_profile_policy" {
  name = "iam_emr_profile_policy_%d"

  policy = <<EOT
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
EOT
}

# IAM Role for autoscaling
resource "aws_iam_role" "emr-autoscaling-role" {
  name               = "EMR_AutoScaling_DefaultRole_%d"
  assume_role_policy = "${data.aws_iam_policy_document.emr-autoscaling-role-policy.json}"
}

data "aws_iam_policy_document" "emr-autoscaling-role-policy" {
  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRole"]
    principals = {
      type        = "Service"
      identifiers = ["elasticmapreduce.amazonaws.com","application-autoscaling.amazonaws.com"]
    }
  }
}

resource "aws_iam_role_policy_attachment" "emr-autoscaling-role" {
  role       = "${aws_iam_role.emr-autoscaling-role.name}"
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonElasticMapReduceforAutoScalingRole"
}

resource "aws_emr_security_configuration" "foo" {
	configuration = <<EOF
{
  "EncryptionConfiguration": {
    "AtRestEncryptionConfiguration": {
      "S3EncryptionConfiguration": {
        "EncryptionMode": "SSE-S3"
      },
      "LocalDiskEncryptionConfiguration": {
        "EncryptionKeyProviderType": "AwsKms",
        "AwsKmsKey": "${aws_kms_key.foo.arn}"
      }
    },
    "EnableInTransitEncryption": false,
    "EnableAtRestEncryption": true
  }
}
EOF
}

resource "aws_kms_key" "foo" {
    description = "Terraform acc test %d"
    deletion_window_in_days = 7
    policy = <<POLICY
{
  "Version": "2012-10-17",
  "Id": "kms-tf-1",
  "Statement": [
    {
      "Sid": "Enable IAM User Permissions",
      "Effect": "Allow",
      "Principal": {
        "AWS": "*"
      },
      "Action": "kms:*",
      "Resource": "*"
    }
  ]
}
POLICY
}
`, r, r, r, r, r, r, r, r, r, r, r)
}

func testAccAWSEmrClusterConfigTerminationPolicyUpdated(r int) string {
	return fmt.Sprintf(`
provider "aws" {
  region = "us-west-2"
}

resource "aws_emr_cluster" "tf-test-cluster" {
  name          = "emr-test-%d"
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

  keep_job_flow_alive_when_no_steps = true
  termination_protection = true

  bootstrap_action {
    path = "s3://elasticmapreduce/bootstrap-actions/run-if"
    name = "runif"
    args = ["instance.isMaster=true", "echo running on master node"]
  }

  configurations = "test-fixtures/emr_configurations.json"

  depends_on = ["aws_main_route_table_association.a"]

  service_role = "${aws_iam_role.iam_emr_default_role.arn}"
  autoscaling_role = "${aws_iam_role.emr-autoscaling-role.arn}"
}

resource "aws_security_group" "allow_all" {
  name        = "allow_all_%d"
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
    name = "emr_test_%d"
  }
}

resource "aws_subnet" "main" {
  vpc_id     = "${aws_vpc.main.id}"
  cidr_block = "168.31.0.0/20"

  tags {
    name = "emr_test_%d"
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

# IAM things

###

# IAM role for EMR Service
resource "aws_iam_role" "iam_emr_default_role" {
  name = "iam_emr_default_role_%d"

  assume_role_policy = <<EOT
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
EOT
}

resource "aws_iam_role_policy_attachment" "service-attach" {
  role       = "${aws_iam_role.iam_emr_default_role.id}"
  policy_arn = "${aws_iam_policy.iam_emr_default_policy.arn}"
}

resource "aws_iam_policy" "iam_emr_default_policy" {
  name = "iam_emr_default_policy_%d"

  policy = <<EOT
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
EOT
}

# IAM Role for EC2 Instance Profile
resource "aws_iam_role" "iam_emr_profile_role" {
  name = "iam_emr_profile_role_%d"

  assume_role_policy = <<EOT
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
EOT
}

resource "aws_iam_instance_profile" "emr_profile" {
  name  = "emr_profile_%d"
  roles = ["${aws_iam_role.iam_emr_profile_role.name}"]
}

resource "aws_iam_role_policy_attachment" "profile-attach" {
  role       = "${aws_iam_role.iam_emr_profile_role.id}"
  policy_arn = "${aws_iam_policy.iam_emr_profile_policy.arn}"
}

resource "aws_iam_policy" "iam_emr_profile_policy" {
  name = "iam_emr_profile_policy_%d"

  policy = <<EOT
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
EOT
}

# IAM Role for autoscaling
resource "aws_iam_role" "emr-autoscaling-role" {
  name               = "EMR_AutoScaling_DefaultRole_%d"
  assume_role_policy = "${data.aws_iam_policy_document.emr-autoscaling-role-policy.json}"
}

data "aws_iam_policy_document" "emr-autoscaling-role-policy" {
  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRole"]

    principals = {
      type        = "Service"
      identifiers = ["elasticmapreduce.amazonaws.com","application-autoscaling.amazonaws.com"]
    }
  }
}

resource "aws_iam_role_policy_attachment" "emr-autoscaling-role" {
  role       = "${aws_iam_role.emr-autoscaling-role.name}"
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonElasticMapReduceforAutoScalingRole"
}
`, r, r, r, r, r, r, r, r, r, r)
}

func testAccAWSEmrClusterConfigVisibleToAllUsersUpdated(r int) string {
	return fmt.Sprintf(`
provider "aws" {
  region = "us-west-2"
}

resource "aws_emr_cluster" "tf-test-cluster" {
  name          = "emr-test-%d"
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

  keep_job_flow_alive_when_no_steps = true
  visible_to_all_users = false

  bootstrap_action {
    path = "s3://elasticmapreduce/bootstrap-actions/run-if"
    name = "runif"
    args = ["instance.isMaster=true", "echo running on master node"]
  }

  configurations = "test-fixtures/emr_configurations.json"

  depends_on = ["aws_main_route_table_association.a"]

  service_role = "${aws_iam_role.iam_emr_default_role.arn}"
  autoscaling_role = "${aws_iam_role.emr-autoscaling-role.arn}"
}

resource "aws_security_group" "allow_all" {
  name        = "allow_all_%d"
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
    name = "emr_test_%d"
  }
}

resource "aws_subnet" "main" {
  vpc_id     = "${aws_vpc.main.id}"
  cidr_block = "168.31.0.0/20"

  tags {
    name = "emr_test_%d"
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

# IAM things

###

# IAM role for EMR Service
resource "aws_iam_role" "iam_emr_default_role" {
  name = "iam_emr_default_role_%d"

  assume_role_policy = <<EOT
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
EOT
}

resource "aws_iam_role_policy_attachment" "service-attach" {
  role       = "${aws_iam_role.iam_emr_default_role.id}"
  policy_arn = "${aws_iam_policy.iam_emr_default_policy.arn}"
}

resource "aws_iam_policy" "iam_emr_default_policy" {
  name = "iam_emr_default_policy_%d"

  policy = <<EOT
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
EOT
}

# IAM Role for EC2 Instance Profile
resource "aws_iam_role" "iam_emr_profile_role" {
  name = "iam_emr_profile_role_%d"

  assume_role_policy = <<EOT
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
EOT
}

resource "aws_iam_instance_profile" "emr_profile" {
  name  = "emr_profile_%d"
  roles = ["${aws_iam_role.iam_emr_profile_role.name}"]
}

resource "aws_iam_role_policy_attachment" "profile-attach" {
  role       = "${aws_iam_role.iam_emr_profile_role.id}"
  policy_arn = "${aws_iam_policy.iam_emr_profile_policy.arn}"
}

resource "aws_iam_policy" "iam_emr_profile_policy" {
  name = "iam_emr_profile_policy_%d"

  policy = <<EOT
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
EOT
}

# IAM Role for autoscaling
resource "aws_iam_role" "emr-autoscaling-role" {
  name               = "EMR_AutoScaling_DefaultRole_%d"
  assume_role_policy = "${data.aws_iam_policy_document.emr-autoscaling-role-policy.json}"
}

data "aws_iam_policy_document" "emr-autoscaling-role-policy" {
  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRole"]

    principals = {
      type        = "Service"
      identifiers = ["elasticmapreduce.amazonaws.com","application-autoscaling.amazonaws.com"]
    }
  }
}

resource "aws_iam_role_policy_attachment" "emr-autoscaling-role" {
  role       = "${aws_iam_role.emr-autoscaling-role.name}"
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonElasticMapReduceforAutoScalingRole"
}
`, r, r, r, r, r, r, r, r, r, r)
}

func testAccAWSEmrClusterConfigUpdatedTags(r int) string {
	return fmt.Sprintf(`
provider "aws" {
  region = "us-west-2"
}

resource "aws_emr_cluster" "tf-test-cluster" {
  name          = "emr-test-%d"
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
    dns_zone = "new_zone"
    Env      = "production"
    name     = "name-env"
  }

  keep_job_flow_alive_when_no_steps = true
  termination_protection = false

  bootstrap_action {
    path = "s3://elasticmapreduce/bootstrap-actions/run-if"
    name = "runif"
    args = ["instance.isMaster=true", "echo running on master node"]
  }

  configurations = "test-fixtures/emr_configurations.json"

  depends_on = ["aws_main_route_table_association.a"]

  service_role = "${aws_iam_role.iam_emr_default_role.arn}"
  autoscaling_role = "${aws_iam_role.emr-autoscaling-role.arn}"
}

resource "aws_security_group" "allow_all" {
  name        = "allow_all_%d"
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
    name = "emr_test_%d"
  }
}

resource "aws_subnet" "main" {
  vpc_id     = "${aws_vpc.main.id}"
  cidr_block = "168.31.0.0/20"

  tags {
    name = "emr_test_%d"
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

# IAM things

###

# IAM role for EMR Service
resource "aws_iam_role" "iam_emr_default_role" {
  name = "iam_emr_default_role_%d"

  assume_role_policy = <<EOT
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
EOT
}

resource "aws_iam_role_policy_attachment" "service-attach" {
  role       = "${aws_iam_role.iam_emr_default_role.id}"
  policy_arn = "${aws_iam_policy.iam_emr_default_policy.arn}"
}

resource "aws_iam_policy" "iam_emr_default_policy" {
  name = "iam_emr_default_policy_%d"

  policy = <<EOT
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
EOT
}

# IAM Role for EC2 Instance Profile
resource "aws_iam_role" "iam_emr_profile_role" {
  name = "iam_emr_profile_role_%d"

  assume_role_policy = <<EOT
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
EOT
}

resource "aws_iam_instance_profile" "emr_profile" {
  name  = "emr_profile_%d"
  roles = ["${aws_iam_role.iam_emr_profile_role.name}"]
}

resource "aws_iam_role_policy_attachment" "profile-attach" {
  role       = "${aws_iam_role.iam_emr_profile_role.id}"
  policy_arn = "${aws_iam_policy.iam_emr_profile_policy.arn}"
}

resource "aws_iam_policy" "iam_emr_profile_policy" {
  name = "iam_emr_profile_policy_%d"

  policy = <<EOT
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
EOT
}

# IAM Role for autoscaling
resource "aws_iam_role" "emr-autoscaling-role" {
  name               = "EMR_AutoScaling_DefaultRole_%d"
  assume_role_policy = "${data.aws_iam_policy_document.emr-autoscaling-role-policy.json}"
}

data "aws_iam_policy_document" "emr-autoscaling-role-policy" {
  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRole"]

    principals = {
      type        = "Service"
      identifiers = ["elasticmapreduce.amazonaws.com","application-autoscaling.amazonaws.com"]
    }
  }
}

resource "aws_iam_role_policy_attachment" "emr-autoscaling-role" {
  role       = "${aws_iam_role.emr-autoscaling-role.name}"
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonElasticMapReduceforAutoScalingRole"
}
`, r, r, r, r, r, r, r, r, r, r)
}
