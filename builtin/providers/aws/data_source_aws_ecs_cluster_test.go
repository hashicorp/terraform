package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAWSEcsDataSource_ecsCluster(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckAwsEcsClusterDataSourceConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.aws_ecs_cluster.default", "status", "ACTIVE"),
					resource.TestCheckResourceAttr("data.aws_ecs_cluster.default", "pending_tasks_count", "0"),
					resource.TestCheckResourceAttr("data.aws_ecs_cluster.default", "running_tasks_count", "0"),
					resource.TestCheckResourceAttr("data.aws_ecs_cluster.default", "registered_container_instances_count", "0"),
				),
			},
		},
	})
}

var testAccCheckAwsEcsClusterDataSourceConfig = fmt.Sprintf(`
resource "aws_ecs_cluster" "default" {
  name = "default-%d"
}

resource "aws_ecs_task_definition" "mongo" {
  family = "mongodb"
  container_definitions = <<DEFINITION
[
  {
    "cpu": 128,
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

data "aws_ecs_cluster" "default" {
  cluster_name = "${aws_ecs_cluster.default.name}"
}
`, acctest.RandInt())
