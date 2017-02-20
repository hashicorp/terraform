package shield

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"testing"
)

func TestShieldRetention_basic(t *testing.T) {
	var retention Retention

	testShieldRetentionConfig := fmt.Sprintf(`
		resource "shield_retention_policy" "test_retention" {
		  name = "Test-Retention"
		  summary = "Terraform Test Retention"
		  expires = 86400
		}
	`)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testShieldRetentionDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testShieldRetentionConfig,
				Check: resource.ComposeTestCheckFunc(
					testShieldCheckRetentionExists("shield_retention_policy.test_retention", &retention),
				),
			},
		},
	})
}

func testShieldRetentionDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*ShieldClient)
	rs, ok := s.RootModule().Resources["shield_retention_policy.test_retention"]
	if !ok {
		return fmt.Errorf("Not found %s", "shield_retention_policy.test_retention")
	}

	response, err := client.Get(fmt.Sprintf("v1/retention/%s", rs.Primary.Attributes["uuid"]))

	if err != nil {
		return err
	}

	if response.StatusCode != 404 {
		return fmt.Errorf("Retention still exists")
	}

	return nil
}

func testShieldCheckRetentionExists(n string, retention *Retention) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found %s", n)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No Retention UUID is set")
		}
		return nil
	}
}
