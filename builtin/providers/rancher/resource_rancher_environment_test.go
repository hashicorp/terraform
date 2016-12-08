package rancher

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	rancherClient "github.com/rancher/go-rancher/client"
)

func TestAccRancherEnvironment(t *testing.T) {
	var environment rancherClient.Project

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckRancherEnvironmentDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccRancherEnvironmentConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRancherEnvironmentExists("rancher_environment.foo", &environment),
					resource.TestCheckResourceAttr("rancher_environment.foo", "name", "foo"),
					resource.TestCheckResourceAttr("rancher_environment.foo", "description", "Terraform acc test group"),
					resource.TestCheckResourceAttr("rancher_environment.foo", "orchestration", "cattle"),
				),
			},
			resource.TestStep{
				Config: testAccRancherEnvironmentUpdateConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRancherEnvironmentExists("rancher_environment.foo", &environment),
					resource.TestCheckResourceAttr("rancher_environment.foo", "name", "foo2"),
					resource.TestCheckResourceAttr("rancher_environment.foo", "description", "Terraform acc test group - updated"),
					resource.TestCheckResourceAttr("rancher_environment.foo", "orchestration", "swarm"),
				),
			},
		},
	})
}

func testAccCheckRancherEnvironmentExists(n string, env *rancherClient.Project) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No App Name is set")
		}

		client := testAccProvider.Meta().(*Config)

		foundEnv, err := client.Project.ById(rs.Primary.ID)
		if err != nil {
			return err
		}

		if foundEnv.Resource.Id != rs.Primary.ID {
			return fmt.Errorf("Environment not found")
		}

		*env = *foundEnv

		return nil
	}
}

func testAccCheckRancherEnvironmentDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*Config)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "rancher_environment" {
			continue
		}
		env, err := client.Project.ById(rs.Primary.ID)

		if err == nil {
			if env != nil &&
				env.Resource.Id == rs.Primary.ID &&
				env.State != "removed" {
				return fmt.Errorf("Environment still exists")
			}
		}

		return nil
	}
	return nil
}

const testAccRancherEnvironmentConfig = `
resource "rancher_environment" "foo" {
	name = "foo"
	description = "Terraform acc test group"
	orchestration = "cattle"
}
`

const testAccRancherEnvironmentUpdateConfig = `
resource "rancher_environment" "foo" {
	name = "foo2"
	description = "Terraform acc test group - updated"
	orchestration = "swarm"
}
`
