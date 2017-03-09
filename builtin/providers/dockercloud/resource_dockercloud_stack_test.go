package dockercloud

import (
	"fmt"
	"testing"

	"github.com/docker/go-dockercloud/dockercloud"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccCheckDockercloudStack_Basic(t *testing.T) {
	var stack dockercloud.Stack

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDockercloudStackDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckDockercloudStackConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDockercloudStackExists("dockercloud_stack.foobar", &stack),
					testAccCheckDockercloudStackAttributes(&stack),
					resource.TestCheckResourceAttr(
						"dockercloud_stack.foobar", "name", "foobar-test-terraform"),
				),
			},
		},
	})
}

func testAccCheckDockercloudStackDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "dockercloud_stack" {
			continue
		}

		stack, err := dockercloud.GetStack(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("Error retrieving stack: %s", err)
		}

		if stack.State != "Terminated" {
			return fmt.Errorf("Stack still running")
		}
	}

	return nil
}

func testAccCheckDockercloudStackExists(n string, stack *dockercloud.Stack) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Record ID is set")
		}

		retrieveStack, err := dockercloud.GetStack(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("Error retrieving stack: %s", err)
		}

		if retrieveStack.Uuid != rs.Primary.ID {
			return fmt.Errorf("Stack not found")
		}

		*stack = retrieveStack

		return nil
	}
}

func testAccCheckDockercloudStackAttributes(stack *dockercloud.Stack) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if stack.Name != "foobar-test-terraform" {
			return fmt.Errorf("Bad name: %s", stack.Name)
		}

		return nil
	}
}

const testAccCheckDockercloudStackConfig_basic = `
resource "dockercloud_node_cluster" "foobar" {
    name = "foobar-test-terraform"
    node_provider = "aws"
    size = "t2.nano"
    region = "us-east-1"
}

resource "dockercloud_stack" "foobar" {
    name = "foobar-test-terraform"
    depends_on = ["dockercloud_node_cluster.foobar"]
}`
