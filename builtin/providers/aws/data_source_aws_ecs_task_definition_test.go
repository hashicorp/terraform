package aws

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAWSEcsDataSource_ecsTaskDefinition(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckAwsEcsTaskDefinitionDataSourceConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("data.aws_ecs_task_definition.mongo", "id", regexp.MustCompile("^arn:aws:ecs:us-west-2:[0-9]{12}:task-definition/mongodb:[1-9][0-9]*$")),
					resource.TestCheckResourceAttr("data.aws_ecs_task_definition.mongo", "family", "mongodb"),
					resource.TestCheckResourceAttr("data.aws_ecs_task_definition.mongo", "network_mode", "bridge"),
					resource.TestMatchResourceAttr("data.aws_ecs_task_definition.mongo", "revision", regexp.MustCompile("^[1-9][0-9]*$")),
					resource.TestCheckResourceAttr("data.aws_ecs_task_definition.mongo", "status", "ACTIVE"),
					resource.TestMatchResourceAttr("data.aws_ecs_task_definition.mongo", "task_role_arn", regexp.MustCompile("^arn:aws:iam::[0-9]{12}:role/mongo_role$")),
				),
			},
		},
	})
}

const testAccCheckAwsEcsTaskDefinitionDataSourceConfig = `
resource "aws_iam_role" "mongo_role" {
    name = "mongo_role"
    assume_role_policy = <<POLICY
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
POLICY
}

resource "aws_ecs_task_definition" "mongo" {
  family = "mongodb"
  task_role_arn = "${aws_iam_role.mongo_role.arn}"
  network_mode = "bridge"
  container_definitions = <<DEFINITION
[
  {
    "cpu": 128,
    "environment": [{
      "name": "SECRET",
      "value": "KEY"
    }],
    "essential": true,
    "image": "mongo:latest",
    "memory": 128,
    "memoryReservation": 64,
    "name": "mongodb"
  }
]
DEFINITION
}

data "aws_ecs_task_definition" "mongo" {
  task_definition = "${aws_ecs_task_definition.mongo.family}"
}
`
