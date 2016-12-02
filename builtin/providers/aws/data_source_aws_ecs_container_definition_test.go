package aws

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAWSEcsDataSource_ecsContainerDefinition(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckAwsEcsContainerDefinitionDataSourceConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.aws_ecs_container_definition.mongo", "image", "mongo:latest"),
					resource.TestCheckResourceAttr("data.aws_ecs_container_definition.mongo", "image_digest", "latest"),
					resource.TestCheckResourceAttr("data.aws_ecs_container_definition.mongo", "memory", "128"),
					resource.TestCheckResourceAttr("data.aws_ecs_container_definition.mongo", "memory_reservation", "64"),
					resource.TestCheckResourceAttr("data.aws_ecs_container_definition.mongo", "cpu", "128"),
					resource.TestCheckResourceAttr("data.aws_ecs_container_definition.mongo", "environment.SECRET", "KEY"),
				),
			},
		},
	})
}

const testAccCheckAwsEcsContainerDefinitionDataSourceConfig = `
resource "aws_ecs_cluster" "default" {
  name = "terraformecstest1"
}

resource "aws_ecs_task_definition" "mongo" {
  family = "mongodb"
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

resource "aws_ecs_service" "mongo" {
  name = "mongodb"
  cluster = "${aws_ecs_cluster.default.id}"
  task_definition = "${aws_ecs_task_definition.mongo.arn}"
  desired_count = 1
}

data "aws_ecs_container_definition" "mongo" {
  task_definition = "${aws_ecs_task_definition.mongo.id}"
  container_name = "mongodb"
}
`
