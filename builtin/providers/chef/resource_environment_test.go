package chef

import (
	"fmt"
	"reflect"
	"testing"

	chefc "github.com/go-chef/chef"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccEnvironment_basic(t *testing.T) {
	var env chefc.Environment

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccEnvironmentCheckDestroy(&env),
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccEnvironmentConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccEnvironmentCheckExists("chef_environment.test", &env),
					func(s *terraform.State) error {

						if expected := "terraform-acc-test-basic"; env.Name != expected {
							return fmt.Errorf("wrong name; expected %v, got %v", expected, env.Name)
						}
						if expected := "Terraform Acceptance Tests"; env.Description != expected {
							return fmt.Errorf("wrong description; expected %v, got %v", expected, env.Description)
						}

						expectedConstraints := map[string]string{
							"terraform": "= 1.0.0",
						}
						if !reflect.DeepEqual(env.CookbookVersions, expectedConstraints) {
							return fmt.Errorf("wrong cookbook constraints; expected %#v, got %#v", expectedConstraints, env.CookbookVersions)
						}

						var expectedAttributes interface{}
						expectedAttributes = map[string]interface{}{
							"terraform_acc_test": true,
						}
						if !reflect.DeepEqual(env.DefaultAttributes, expectedAttributes) {
							return fmt.Errorf("wrong default attributes; expected %#v, got %#v", expectedAttributes, env.DefaultAttributes)
						}
						if !reflect.DeepEqual(env.OverrideAttributes, expectedAttributes) {
							return fmt.Errorf("wrong override attributes; expected %#v, got %#v", expectedAttributes, env.OverrideAttributes)
						}

						return nil
					},
				),
			},
		},
	})
}

func testAccEnvironmentCheckExists(rn string, env *chefc.Environment) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("resource not found: %s", rn)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("environment id not set")
		}

		client := testAccProvider.Meta().(*chefc.Client)
		gotEnv, err := client.Environments.Get(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("error getting environment: %s", err)
		}

		*env = *gotEnv

		return nil
	}
}

func testAccEnvironmentCheckDestroy(env *chefc.Environment) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*chefc.Client)
		_, err := client.Environments.Get(env.Name)
		if err == nil {
			return fmt.Errorf("environment still exists")
		}
		if _, ok := err.(*chefc.ErrorResponse); !ok {
			// A more specific check is tricky because Chef Server can return
			// a few different error codes in this case depending on which
			// part of its stack catches the error.
			return fmt.Errorf("got something other than an HTTP error (%v) when getting environment", err)
		}

		return nil
	}
}

const testAccEnvironmentConfig_basic = `
resource "chef_environment" "test" {
  name = "terraform-acc-test-basic"
  description = "Terraform Acceptance Tests"
  default_attributes_json = <<EOT
{
     "terraform_acc_test": true
}
EOT
  override_attributes_json = <<EOT
{
     "terraform_acc_test": true
}
EOT
  cookbook_constraints = {
    "terraform" = "= 1.0.0"
  }
}
`
