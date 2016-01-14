package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSEcsTaskDefinition_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSEcsTaskDefinitionDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSEcsTaskDefinition,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEcsTaskDefinitionExists("aws_ecs_task_definition.jenkins"),
				),
			},
			resource.TestStep{
				Config: testAccAWSEcsTaskDefinitionModified,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEcsTaskDefinitionExists("aws_ecs_task_definition.jenkins"),
				),
			},
		},
	})
}

// Regression for https://github.com/hashicorp/terraform/issues/2370
func TestAccAWSEcsTaskDefinition_withScratchVolume(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSEcsTaskDefinitionDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSEcsTaskDefinitionWithScratchVolume,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEcsTaskDefinitionExists("aws_ecs_task_definition.sleep"),
				),
			},
		},
	})
}

// Regression for https://github.com/hashicorp/terraform/issues/2694
func TestAccAWSEcsTaskDefinition_withEcsService(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSEcsTaskDefinitionDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSEcsTaskDefinitionWithEcsService,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEcsTaskDefinitionExists("aws_ecs_task_definition.sleep"),
					testAccCheckAWSEcsServiceExists("aws_ecs_service.sleep-svc"),
				),
			},
			resource.TestStep{
				Config: testAccAWSEcsTaskDefinitionWithEcsServiceModified,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEcsTaskDefinitionExists("aws_ecs_task_definition.sleep"),
					testAccCheckAWSEcsServiceExists("aws_ecs_service.sleep-svc"),
				),
			},
		},
	})
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

func testAccCheckAWSEcsTaskDefinitionExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		_, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		return nil
	}
}

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
