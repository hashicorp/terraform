package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSEcsCluster_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSEcsClusterDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSEcsCluster,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEcsClusterExists("aws_ecs_cluster.foo"),
				),
			},
		},
	})
}

func testAccCheckAWSEcsClusterDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).ecsconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_ecs_cluster" {
			continue
		}

		out, err := conn.DescribeClusters(&ecs.DescribeClustersInput{
			Clusters: []*string{aws.String(rs.Primary.ID)},
		})

		if err != nil {
			return err
		}

		for _, c := range out.Clusters {
			if *c.ClusterArn == rs.Primary.ID && *c.Status != "INACTIVE" {
				return fmt.Errorf("ECS cluster still exists:\n%s", c)
			}
		}
	}

	return nil
}

func testAccCheckAWSEcsClusterExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		_, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		return nil
	}
}

var testAccAWSEcsCluster = `
resource "aws_ecs_cluster" "foo" {
	name = "red-grapes"
}
`
