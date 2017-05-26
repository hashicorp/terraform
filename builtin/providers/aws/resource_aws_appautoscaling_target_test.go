package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/applicationautoscaling"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSAppautoScalingTarget_basic(t *testing.T) {
	var target applicationautoscaling.ScalableTarget

	randClusterName := fmt.Sprintf("cluster-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_appautoscaling_target.bar",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSAppautoscalingTargetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSAppautoscalingTargetConfig(randClusterName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAppautoscalingTargetExists("aws_appautoscaling_target.bar", &target),
					resource.TestCheckResourceAttr("aws_appautoscaling_target.bar", "service_namespace", "ecs"),
					resource.TestCheckResourceAttr("aws_appautoscaling_target.bar", "scalable_dimension", "ecs:service:DesiredCount"),
					resource.TestCheckResourceAttr("aws_appautoscaling_target.bar", "min_capacity", "1"),
					resource.TestCheckResourceAttr("aws_appautoscaling_target.bar", "max_capacity", "3"),
				),
			},

			{
				Config: testAccAWSAppautoscalingTargetConfigUpdate(randClusterName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAppautoscalingTargetExists("aws_appautoscaling_target.bar", &target),
					resource.TestCheckResourceAttr("aws_appautoscaling_target.bar", "min_capacity", "2"),
					resource.TestCheckResourceAttr("aws_appautoscaling_target.bar", "max_capacity", "8"),
				),
			},
		},
	})
}

func TestAccAWSAppautoScalingTarget_spotFleetRequest(t *testing.T) {
	var target applicationautoscaling.ScalableTarget

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_appautoscaling_target.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSAppautoscalingTargetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSAppautoscalingTargetSpotFleetRequestConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAppautoscalingTargetExists("aws_appautoscaling_target.test", &target),
					resource.TestCheckResourceAttr("aws_appautoscaling_target.test", "service_namespace", "ec2"),
					resource.TestCheckResourceAttr("aws_appautoscaling_target.test", "scalable_dimension", "ec2:spot-fleet-request:TargetCapacity"),
				),
			},
		},
	})
}

func TestAccAWSAppautoScalingTarget_emrCluster(t *testing.T) {
	var target applicationautoscaling.ScalableTarget
	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSAppautoscalingTargetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSAppautoscalingTargetEmrClusterConfig(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAppautoscalingTargetExists("aws_appautoscaling_target.bar", &target),
					resource.TestCheckResourceAttr("aws_appautoscaling_target.bar", "service_namespace", "elasticmapreduce"),
					resource.TestCheckResourceAttr("aws_appautoscaling_target.bar", "scalable_dimension", "elasticmapreduce:instancegroup:InstanceCount"),
				),
			},
		},
	})
}

func testAccCheckAWSAppautoscalingTargetDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).appautoscalingconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_appautoscaling_target" {
			continue
		}

		// Try to find the target
		describeTargets, err := conn.DescribeScalableTargets(
			&applicationautoscaling.DescribeScalableTargetsInput{
				ResourceIds:      []*string{aws.String(rs.Primary.ID)},
				ServiceNamespace: aws.String(rs.Primary.Attributes["service_namespace"]),
			},
		)

		if err == nil {
			if len(describeTargets.ScalableTargets) != 0 &&
				*describeTargets.ScalableTargets[0].ResourceId == rs.Primary.ID {
				return fmt.Errorf("Application AutoScaling Target still exists")
			}
		}

		// Verify error
		e, ok := err.(awserr.Error)
		if !ok {
			return err
		}
		if e.Code() != "" {
			return e
		}
	}

	return nil
}

func testAccCheckAWSAppautoscalingTargetExists(n string, target *applicationautoscaling.ScalableTarget) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Application AutoScaling Target ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).appautoscalingconn

		describeTargets, err := conn.DescribeScalableTargets(
			&applicationautoscaling.DescribeScalableTargetsInput{
				ResourceIds:      []*string{aws.String(rs.Primary.ID)},
				ServiceNamespace: aws.String(rs.Primary.Attributes["service_namespace"]),
			},
		)

		if err != nil {
			return err
		}

		if len(describeTargets.ScalableTargets) != 1 || *describeTargets.ScalableTargets[0].ResourceId != rs.Primary.ID {
			return fmt.Errorf("Application AutoScaling ResourceId not found")
		}

		target = describeTargets.ScalableTargets[0]

		return nil
	}
}

func testAccAWSAppautoscalingTargetConfig(
	randClusterName string) string {
	return fmt.Sprintf(`
resource "aws_iam_role" "autoscale_role" {
	name = "autoscalerole%s"
	path = "/"

	assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Service": "application-autoscaling.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
EOF
}

resource "aws_iam_role_policy" "autoscale_role_policy" {
	name = "autoscalepolicy%s"
	role = "${aws_iam_role.autoscale_role.id}"

	policy = <<EOF
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "ecs:DescribeServices",
                "ecs:UpdateService"
            ],
            "Resource": [
                "*"
            ]
        },
        {
            "Effect": "Allow",
            "Action": [
                "cloudwatch:DescribeAlarms"
            ],
            "Resource": [
                "*"
            ]
        }
    ]
}
EOF
}

resource "aws_ecs_cluster" "foo" {
	name = "%s"
}

resource "aws_ecs_task_definition" "task" {
	family = "foobar"
	container_definitions = <<EOF
[
    {
        "name": "busybox",
        "image": "busybox:latest",
        "cpu": 10,
        "memory": 128,
        "essential": true
    }
]
EOF
}

resource "aws_ecs_service" "service" {
	name = "foobar"
	cluster = "${aws_ecs_cluster.foo.id}"
	task_definition = "${aws_ecs_task_definition.task.arn}"
	desired_count = 1

	deployment_maximum_percent = 200
	deployment_minimum_healthy_percent = 50
}

resource "aws_appautoscaling_target" "bar" {
	service_namespace = "ecs"
	resource_id = "service/${aws_ecs_cluster.foo.name}/${aws_ecs_service.service.name}"
	scalable_dimension = "ecs:service:DesiredCount"
	role_arn = "${aws_iam_role.autoscale_role.arn}"
	min_capacity = 1
	max_capacity = 3
}
`, randClusterName, randClusterName, randClusterName)
}

func testAccAWSAppautoscalingTargetConfigUpdate(
	randClusterName string) string {
	return fmt.Sprintf(`
resource "aws_iam_role" "autoscale_role" {
	name = "autoscalerole%s"
	path = "/"

	assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Service": "application-autoscaling.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
EOF
}

resource "aws_iam_role_policy" "autoscale_role_policy" {
	name = "autoscalepolicy%s"
	role = "${aws_iam_role.autoscale_role.id}"

	policy = <<EOF
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "ecs:DescribeServices",
                "ecs:UpdateService"
            ],
            "Resource": [
                "*"
            ]
        },
        {
            "Effect": "Allow",
            "Action": [
                "cloudwatch:DescribeAlarms"
            ],
            "Resource": [
                "*"
            ]
        }
    ]
}
EOF
}

resource "aws_ecs_cluster" "foo" {
	name = "%s"
}

resource "aws_ecs_task_definition" "task" {
	family = "foobar"
	container_definitions = <<EOF
[
    {
        "name": "busybox",
        "image": "busybox:latest",
        "cpu": 10,
        "memory": 128,
        "essential": true
    }
]
EOF
}

resource "aws_ecs_service" "service" {
	name = "foobar"
	cluster = "${aws_ecs_cluster.foo.id}"
	task_definition = "${aws_ecs_task_definition.task.arn}"
	desired_count = 2

	deployment_maximum_percent = 200
	deployment_minimum_healthy_percent = 50
}

resource "aws_appautoscaling_target" "bar" {
	service_namespace = "ecs"
	resource_id = "service/${aws_ecs_cluster.foo.name}/${aws_ecs_service.service.name}"
	scalable_dimension = "ecs:service:DesiredCount"
	role_arn = "${aws_iam_role.autoscale_role.arn}"
	min_capacity = 2
	max_capacity = 8
}
`, randClusterName, randClusterName, randClusterName)
}

func testAccAWSAppautoscalingTargetEmrClusterConfig(rInt int) string {
	return fmt.Sprintf(`
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
  core_instance_count  = 2

  tags {
    role     = "rolename"
    dns_zone = "env_zone"
    env      = "env"
    name     = "name-env"
  }

  keep_job_flow_alive_when_no_steps = true

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

resource "aws_emr_instance_group" "task" {
    cluster_id     = "${aws_emr_cluster.tf-test-cluster.id}"
    instance_count = 1
    instance_type  = "m3.xlarge"
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

resource "aws_appautoscaling_target" "bar" {
	service_namespace = "elasticmapreduce"
	resource_id = "instancegroup/${aws_emr_cluster.tf-test-cluster.id}/${aws_emr_instance_group.task.id}"
	scalable_dimension = "elasticmapreduce:instancegroup:InstanceCount"
	role_arn = "${aws_iam_role.emr-autoscaling-role.arn}"
	min_capacity = 1
	max_capacity = 8
}

`, rInt, rInt, rInt, rInt, rInt, rInt, rInt, rInt, rInt, rInt)
}

var testAccAWSAppautoscalingTargetSpotFleetRequestConfig = fmt.Sprintf(`
resource "aws_iam_role" "fleet_role" {
  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Service": [
          "spotfleet.amazonaws.com",
          "ec2.amazonaws.com"
        ]
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
EOF
}

resource "aws_iam_role_policy_attachment" "fleet_role_policy" {
  role = "${aws_iam_role.fleet_role.name}"
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonEC2SpotFleetRole"
}

resource "aws_spot_fleet_request" "test" {
  iam_fleet_role = "${aws_iam_role.fleet_role.arn}"
  spot_price = "0.005"
  target_capacity = 2
  valid_until = "2019-11-04T20:44:20Z"
  terminate_instances_with_expiration = true

  launch_specification {
    instance_type = "m3.medium"
    ami = "ami-d06a90b0"
  }
}

resource "aws_iam_role" "autoscale_role" {
  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Service": "application-autoscaling.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
EOF
}

resource "aws_iam_role_policy_attachment" "autoscale_role_policy_a" {
  role = "${aws_iam_role.autoscale_role.name}"
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonEC2SpotFleetRole"
}

resource "aws_iam_role_policy_attachment" "autoscale_role_policy_b" {
  role = "${aws_iam_role.autoscale_role.name}"
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonEC2SpotFleetAutoscaleRole"
}

resource "aws_appautoscaling_target" "test" {
  service_namespace = "ec2"
  resource_id = "spot-fleet-request/${aws_spot_fleet_request.test.id}"
  scalable_dimension = "ec2:spot-fleet-request:TargetCapacity"
  role_arn = "${aws_iam_role.autoscale_role.arn}"
  min_capacity = 1
  max_capacity = 3
}
`)
