package shield

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"testing"
)

func TestShieldTarget_basic(t *testing.T) {
	var target Target

	testShieldTargetConfig := fmt.Sprintf(`
		resource "shield_target" "test_target" {
		  name = "Test-Target"
		  summary = "Terraform Test Target"
		  plugin = "mysql"
		  endpoint = "{\"mysql_user\":\"root\",\"mysql_password\":\"secure-pw\",\"mysql_host\": \"localhost\",\"mysql_port\": 3306}"
		  agent = "localhost:5444"
		}
	`)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testShieldTargetDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testShieldTargetConfig,
				Check: resource.ComposeTestCheckFunc(
					testShieldCheckTargetExists("shield_target.test_target", &target),
				),
			},
		},
	})
}

func testShieldTargetDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*ShieldClient)
	rs, ok := s.RootModule().Resources["shield_target.test_target"]
	if !ok {
		return fmt.Errorf("Not found %s", "shield_target.test_target")
	}

	response, err := client.Get(fmt.Sprintf("v1/target/%s", rs.Primary.Attributes["uuid"]))

	if err != nil {
		return err
	}

	if response.StatusCode != 404 {
		return fmt.Errorf("Target still exists")
	}

	return nil
}

func testShieldCheckTargetExists(n string, target *Target) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found %s", n)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No Target UUID is set")
		}
		return nil
	}
}
