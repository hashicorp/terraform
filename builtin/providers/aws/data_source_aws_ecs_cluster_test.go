package aws

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAWSEcsDataSource_ecsCluster(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
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

const testAccCheckAwsEcsClusterDataSourceConfig = `
resource "aws_ecs_cluster" "default" {
  name = "default"
}

data "aws_ecs_cluster" "default" {
  cluster_name = "${aws_ecs_cluster.default.name}"
}
`
