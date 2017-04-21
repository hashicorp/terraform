package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSEcsTaskDefinition_basic(t *testing.T) {
	var def ecs.TaskDefinition
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSEcsTaskDefinitionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSEcsTaskDefinition,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEcsTaskDefinitionExists("aws_ecs_task_definition.jenkins", &def),
				),
			},
			{
				Config: testAccAWSEcsTaskDefinitionModified,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEcsTaskDefinitionExists("aws_ecs_task_definition.jenkins", &def),
				),
			},
		},
	})
}

// Regression for https://github.com/hashicorp/terraform/issues/2370
func TestAccAWSEcsTaskDefinition_withScratchVolume(t *testing.T) {
	var def ecs.TaskDefinition
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSEcsTaskDefinitionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSEcsTaskDefinitionWithScratchVolume,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEcsTaskDefinitionExists("aws_ecs_task_definition.sleep", &def),
				),
			},
		},
	})
}

// Regression for https://github.com/hashicorp/terraform/issues/2694
func TestAccAWSEcsTaskDefinition_withEcsService(t *testing.T) {
	var def ecs.TaskDefinition
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSEcsTaskDefinitionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSEcsTaskDefinitionWithEcsService,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEcsTaskDefinitionExists("aws_ecs_task_definition.sleep", &def),
					testAccCheckAWSEcsServiceExists("aws_ecs_service.sleep-svc"),
				),
			},
			{
				Config: testAccAWSEcsTaskDefinitionWithEcsServiceModified,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEcsTaskDefinitionExists("aws_ecs_task_definition.sleep", &def),
					testAccCheckAWSEcsServiceExists("aws_ecs_service.sleep-svc"),
				),
			},
		},
	})
}

func TestAccAWSEcsTaskDefinition_withTaskRoleArn(t *testing.T) {
	var def ecs.TaskDefinition
	rInt := acctest.RandInt()
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSEcsTaskDefinitionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSEcsTaskDefinitionWithTaskRoleArn(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEcsTaskDefinitionExists("aws_ecs_task_definition.sleep", &def),
				),
			},
		},
	})
}

func TestAccAWSEcsTaskDefinition_withNetworkMode(t *testing.T) {
	var def ecs.TaskDefinition
	rInt := acctest.RandInt()
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSEcsTaskDefinitionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSEcsTaskDefinitionWithNetworkMode(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEcsTaskDefinitionExists("aws_ecs_task_definition.sleep", &def),
					resource.TestCheckResourceAttr(
						"aws_ecs_task_definition.sleep", "network_mode", "bridge"),
				),
			},
		},
	})
}

func TestAccAWSEcsTaskDefinition_constraint(t *testing.T) {
	var def ecs.TaskDefinition
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSEcsTaskDefinitionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSEcsTaskDefinition_constraint,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEcsTaskDefinitionExists("aws_ecs_task_definition.jenkins", &def),
					resource.TestCheckResourceAttr("aws_ecs_task_definition.jenkins", "placement_constraints.#", "1"),
					testAccCheckAWSTaskDefinitionConstraintsAttrs(&def),
				),
			},
		},
	})
}

func TestAccAWSEcsTaskDefinition_changeVolumesForcesNewResource(t *testing.T) {
	var before ecs.TaskDefinition
	var after ecs.TaskDefinition
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSEcsTaskDefinitionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSEcsTaskDefinition,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEcsTaskDefinitionExists("aws_ecs_task_definition.jenkins", &before),
				),
			},
			{
				Config: testAccAWSEcsTaskDefinitionUpdatedVolume,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEcsTaskDefinitionExists("aws_ecs_task_definition.jenkins", &after),
					testAccCheckEcsTaskDefinitionRecreated(t, &before, &after),
				),
			},
		},
	})
}

func testAccCheckEcsTaskDefinitionRecreated(t *testing.T,
	before, after *ecs.TaskDefinition) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if *before.Revision == *after.Revision {
			t.Fatalf("Expected change of TaskDefinition Revisions, but both were %v", before.Revision)
		}
		return nil
	}
}

func testAccCheckAWSTaskDefinitionConstraintsAttrs(def *ecs.TaskDefinition) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if len(def.PlacementConstraints) != 1 {
			return fmt.Errorf("Expected (1) placement_constraints, got (%d)", len(def.PlacementConstraints))
		}
		return nil
	}
}
func TestValidateAwsEcsTaskDefinitionNetworkMode(t *testing.T) {
	validNames := []string{
		"bridge",
		"host",
		"none",
	}
	for _, v := range validNames {
		_, errors := validateAwsEcsTaskDefinitionNetworkMode(v, "network_mode")
		if len(errors) != 0 {
			t.Fatalf("%q should be a valid AWS ECS Task Definition Network Mode: %q", v, errors)
		}
	}

	invalidNames := []string{
		"bridged",
		"-docker",
	}
	for _, v := range invalidNames {
		_, errors := validateAwsEcsTaskDefinitionNetworkMode(v, "network_mode")
		if len(errors) == 0 {
			t.Fatalf("%q should be an invalid AWS ECS Task Definition Network Mode", v)
		}
	}
}

func TestValidateAwsEcsTaskDefinitionContainerDefinitions(t *testing.T) {
	validDefinitions := []string{
		testValidateAwsEcsTaskDefinitionValidContainerDefinitions,
	}
	for _, v := range validDefinitions {
		_, errors := validateAwsEcsTaskDefinitionContainerDefinitions(v, "container_definitions")
		if len(errors) != 0 {
			t.Fatalf("%q should be a valid AWS ECS Task Definition Container Definitions: %q", v, errors)
		}
	}

	invalidDefinitions := []string{
		testValidateAwsEcsTaskDefinitionInvalidCommandContainerDefinitions,
	}
	for _, v := range invalidDefinitions {
		_, errors := validateAwsEcsTaskDefinitionContainerDefinitions(v, "container_definitions")
		if len(errors) == 0 {
			t.Fatalf("%q should be an invalid AWS ECS Task Definition Container Definitions", v)
		}
	}
}

func testAccCheckAWSEcsTaskDefinitionDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).ecsconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_ecs_task_definition" {
			continue
		}

		input := ecs.DescribeTaskDefinitionInput{
			TaskDefinition: aws.String(rs.Primary.Attributes["arn"]),
		}

		out, err := conn.DescribeTaskDefinition(&input)

		if err != nil {
			return err
		}

		if out.TaskDefinition != nil && *out.TaskDefinition.Status != "INACTIVE" {
			return fmt.Errorf("ECS task definition still exists:\n%#v", *out.TaskDefinition)
		}
	}

	return nil
}

func testAccCheckAWSEcsTaskDefinitionExists(name string, def *ecs.TaskDefinition) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		conn := testAccProvider.Meta().(*AWSClient).ecsconn

		out, err := conn.DescribeTaskDefinition(&ecs.DescribeTaskDefinitionInput{
			TaskDefinition: aws.String(rs.Primary.Attributes["arn"]),
		})
		if err != nil {
			return err
		}

		*def = *out.TaskDefinition

		return nil
	}
}

var testAccAWSEcsTaskDefinition_constraint = `
resource "aws_ecs_task_definition" "jenkins" {
  family = "terraform-acc-test"
  container_definitions = <<TASK_DEFINITION
[
	{
		"cpu": 10,
		"command": ["sleep", "10"],
		"entryPoint": ["/"],
		"environment": [
			{"name": "VARNAME", "value": "VARVAL"}
		],
		"essential": true,
		"image": "jenkins",
		"links": ["mongodb"],
		"memory": 128,
		"name": "jenkins",
		"portMappings": [
			{
				"containerPort": 80,
				"hostPort": 8080
			}
		]
	},
	{
		"cpu": 10,
		"command": ["sleep", "10"],
		"entryPoint": ["/"],
		"essential": true,
		"image": "mongodb",
		"memory": 128,
		"name": "mongodb",
		"portMappings": [
			{
				"containerPort": 28017,
				"hostPort": 28017
			}
		]
	}
]
TASK_DEFINITION

  volume {
    name = "jenkins-home"
    host_path = "/ecs/jenkins-home"
  }

	placement_constraints {
		type = "memberOf"
		expression = "attribute:ecs.availability-zone in [us-west-2a, us-west-2b]"
	}
}
`

var testAccAWSEcsTaskDefinition = `
resource "aws_ecs_task_definition" "jenkins" {
  family = "terraform-acc-test"
  container_definitions = <<TASK_DEFINITION
[
	{
		"cpu": 10,
		"command": ["sleep", "10"],
		"entryPoint": ["/"],
		"environment": [
			{"name": "VARNAME", "value": "VARVAL"}
		],
		"essential": true,
		"image": "jenkins",
		"links": ["mongodb"],
		"memory": 128,
		"name": "jenkins",
		"portMappings": [
			{
				"containerPort": 80,
				"hostPort": 8080
			}
		]
	},
	{
		"cpu": 10,
		"command": ["sleep", "10"],
		"entryPoint": ["/"],
		"essential": true,
		"image": "mongodb",
		"memory": 128,
		"name": "mongodb",
		"portMappings": [
			{
				"containerPort": 28017,
				"hostPort": 28017
			}
		]
	}
]
TASK_DEFINITION

  volume {
    name = "jenkins-home"
    host_path = "/ecs/jenkins-home"
  }
}
`

var testAccAWSEcsTaskDefinitionUpdatedVolume = `
resource "aws_ecs_task_definition" "jenkins" {
  family = "terraform-acc-test"
  container_definitions = <<TASK_DEFINITION
[
	{
		"cpu": 10,
		"command": ["sleep", "10"],
		"entryPoint": ["/"],
		"environment": [
			{"name": "VARNAME", "value": "VARVAL"}
		],
		"essential": true,
		"image": "jenkins",
		"links": ["mongodb"],
		"memory": 128,
		"name": "jenkins",
		"portMappings": [
			{
				"containerPort": 80,
				"hostPort": 8080
			}
		]
	},
	{
		"cpu": 10,
		"command": ["sleep", "10"],
		"entryPoint": ["/"],
		"essential": true,
		"image": "mongodb",
		"memory": 128,
		"name": "mongodb",
		"portMappings": [
			{
				"containerPort": 28017,
				"hostPort": 28017
			}
		]
	}
]
TASK_DEFINITION

  volume {
    name = "jenkins-home"
    host_path = "/ecs/jenkins"
  }
}
`

var testAccAWSEcsTaskDefinitionWithScratchVolume = `
resource "aws_ecs_task_definition" "sleep" {
  family = "terraform-acc-sc-volume-test"
  container_definitions = <<TASK_DEFINITION
[
  {
    "name": "sleep",
    "image": "busybox",
    "cpu": 10,
    "command": ["sleep","360"],
    "memory": 10,
    "essential": true
  }
]
TASK_DEFINITION

  volume {
    name = "database_scratch"
  }
}
`

func testAccAWSEcsTaskDefinitionWithTaskRoleArn(rInt int) string {
	return fmt.Sprintf(`
	resource "aws_iam_role" "role_test" {
		name = "tf_old_name-%d"
		path = "/test/"
		assume_role_policy = <<EOF
{
	"Version": "2012-10-17",
	"Statement": [
		{
			"Action": "sts:AssumeRole",
			"Principal": {
				"Service": "ec2.amazonaws.com"
			},
			"Effect": "Allow",
			"Sid": ""
		}
	]
}
EOF
	}

	resource "aws_iam_role_policy" "role_test" {
		name = "role_update_test-%d"
		role = "${aws_iam_role.role_test.id}"
		policy = <<EOF
{
	"Version": "2012-10-17",
	"Statement": [
		{
			"Effect": "Allow",
			"Action": [
				"s3:GetBucketLocation",
				"s3:ListAllMyBuckets"
			],
			"Resource": "arn:aws:s3:::*"
		}
	]
}
EOF
	}

	resource "aws_ecs_task_definition" "sleep" {
		family = "terraform-acc-sc-volume-test"
		task_role_arn = "${aws_iam_role.role_test.arn}"
		container_definitions = <<TASK_DEFINITION
[
	{
		"name": "sleep",
		"image": "busybox",
		"cpu": 10,
		"command": ["sleep","360"],
		"memory": 10,
		"essential": true
	}
]
TASK_DEFINITION
		volume {
		name = "database_scratch"
	}
}`, rInt, rInt)
}

func testAccAWSEcsTaskDefinitionWithNetworkMode(rInt int) string {
	return fmt.Sprintf(`
 resource "aws_iam_role" "role_test" {
	 name = "tf_old_name-%d"
	 path = "/test/"
	 assume_role_policy = <<EOF
{
 "Version": "2012-10-17",
 "Statement": [
	 {
		 "Action": "sts:AssumeRole",
		 "Principal": {
			 "Service": "ec2.amazonaws.com"
		 },
		 "Effect": "Allow",
		 "Sid": ""
	 }
 ]
}
EOF
 }

 resource "aws_iam_role_policy" "role_test" {
	 name = "role_update_test-%d"
	 role = "${aws_iam_role.role_test.id}"
	 policy = <<EOF
{
 "Version": "2012-10-17",
 "Statement": [
	 {
		 "Effect": "Allow",
		 "Action": [
			 "s3:GetBucketLocation",
			 "s3:ListAllMyBuckets"
		 ],
		 "Resource": "arn:aws:s3:::*"
	 }
 ]
}
 EOF
 }

 resource "aws_ecs_task_definition" "sleep" {
	 family = "terraform-acc-sc-volume-test-network-mode"
	 task_role_arn = "${aws_iam_role.role_test.arn}"
	 network_mode = "bridge"
	 container_definitions = <<TASK_DEFINITION
[
 {
	 "name": "sleep",
	 "image": "busybox",
	 "cpu": 10,
	 "command": ["sleep","360"],
	 "memory": 10,
	 "essential": true
 }
]
TASK_DEFINITION

	 volume {
		 name = "database_scratch"
	 }
 }`, rInt, rInt)
}

var testAccAWSEcsTaskDefinitionWithEcsService = `
resource "aws_ecs_cluster" "default" {
  name = "terraform-acc-test"
}

resource "aws_ecs_service" "sleep-svc" {
  name = "tf-acc-ecs-svc"
  cluster = "${aws_ecs_cluster.default.id}"
  task_definition = "${aws_ecs_task_definition.sleep.arn}"
  desired_count = 1
}

resource "aws_ecs_task_definition" "sleep" {
  family = "terraform-acc-sc-volume-test"
  container_definitions = <<TASK_DEFINITION
[
  {
    "name": "sleep",
    "image": "busybox",
    "cpu": 10,
    "command": ["sleep","360"],
    "memory": 10,
    "essential": true
  }
]
TASK_DEFINITION

  volume {
    name = "database_scratch"
  }
}
`
var testAccAWSEcsTaskDefinitionWithEcsServiceModified = `
resource "aws_ecs_cluster" "default" {
  name = "terraform-acc-test"
}

resource "aws_ecs_service" "sleep-svc" {
  name = "tf-acc-ecs-svc"
  cluster = "${aws_ecs_cluster.default.id}"
  task_definition = "${aws_ecs_task_definition.sleep.arn}"
  desired_count = 1
}

resource "aws_ecs_task_definition" "sleep" {
  family = "terraform-acc-sc-volume-test"
  container_definitions = <<TASK_DEFINITION
[
  {
    "name": "sleep",
    "image": "busybox",
    "cpu": 20,
    "command": ["sleep","360"],
    "memory": 50,
    "essential": true
  }
]
TASK_DEFINITION

  volume {
    name = "database_scratch"
  }
}
`

var testAccAWSEcsTaskDefinitionModified = `
resource "aws_ecs_task_definition" "jenkins" {
  family = "terraform-acc-test"
  container_definitions = <<TASK_DEFINITION
[
	{
		"cpu": 10,
		"command": ["sleep", "10"],
		"entryPoint": ["/"],
		"environment": [
			{"name": "VARNAME", "value": "VARVAL"}
		],
		"essential": true,
		"image": "jenkins",
		"links": ["mongodb"],
		"memory": 128,
		"name": "jenkins",
		"portMappings": [
			{
				"containerPort": 80,
				"hostPort": 8080
			}
		]
	},
	{
		"cpu": 20,
		"command": ["sleep", "10"],
		"entryPoint": ["/"],
		"essential": true,
		"image": "mongodb",
		"memory": 128,
		"name": "mongodb",
		"portMappings": [
			{
				"containerPort": 28017,
				"hostPort": 28017
			}
		]
	}
]
TASK_DEFINITION

  volume {
    name = "jenkins-home"
    host_path = "/ecs/jenkins-home"
  }
}
`

var testValidateAwsEcsTaskDefinitionValidContainerDefinitions = `
[
  {
    "name": "sleep",
    "image": "busybox",
    "cpu": 10,
    "command": ["sleep","360"],
    "memory": 10,
    "essential": true
  }
]
`

var testValidateAwsEcsTaskDefinitionInvalidCommandContainerDefinitions = `
[
  {
    "name": "sleep",
    "image": "busybox",
    "cpu": 10,
    "command": "sleep 360",
    "memory": 10,
    "essential": true
  }
]
`
