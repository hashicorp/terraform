package newrelic

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	newrelic "github.com/paultyng/go-newrelic/api"
)

func TestAccNewRelicAlertPolicy_Basic(t *testing.T) {
	rName := acctest.RandString(5)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckNewRelicAlertPolicyDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckNewRelicAlertPolicyConfig(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNewRelicAlertPolicyExists("newrelic_alert_policy.foo"),
					resource.TestCheckResourceAttr(
						"newrelic_alert_policy.foo", "name", fmt.Sprintf("tf-test-%s", rName)),
					resource.TestCheckResourceAttr(
						"newrelic_alert_policy.foo", "incident_preference", "PER_POLICY"),
				),
			},
			resource.TestStep{
				Config: testAccCheckNewRelicAlertPolicyConfigUpdated(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNewRelicAlertPolicyExists("newrelic_alert_policy.foo"),
					resource.TestCheckResourceAttr(
						"newrelic_alert_policy.foo", "name", fmt.Sprintf("tf-test-updated-%s", rName)),
					resource.TestCheckResourceAttr(
						"newrelic_alert_policy.foo", "incident_preference", "PER_CONDITION"),
				),
			},
		},
	})
}

func testAccCheckNewRelicAlertPolicyDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*newrelic.Client)
	for _, r := range s.RootModule().Resources {
		if r.Type != "newrelic_alert_policy" {
			continue
		}

		id, err := strconv.ParseInt(r.Primary.ID, 10, 32)
		if err != nil {
			return err
		}

		_, err = client.GetAlertPolicy(int(id))

		if err == nil {
			return fmt.Errorf("Policy still exists")
		}

	}
	return nil
}

func testAccCheckNewRelicAlertPolicyExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No policy ID is set")
		}

		client := testAccProvider.Meta().(*newrelic.Client)

		id, err := strconv.ParseInt(rs.Primary.ID, 10, 32)
		if err != nil {
			return err
		}

		found, err := client.GetAlertPolicy(int(id))
		if err != nil {
			return err
		}

		if strconv.Itoa(found.ID) != rs.Primary.ID {
			return fmt.Errorf("Policy not found: %v - %v", rs.Primary.ID, found)
		}

		return nil
	}
}

func testAccCheckNewRelicAlertPolicyConfig(rName string) string {
	return fmt.Sprintf(`
resource "newrelic_alert_policy" "foo" {
  name = "tf-test-%s"
}
`, rName)
}

func testAccCheckNewRelicAlertPolicyConfigUpdated(rName string) string {
	return fmt.Sprintf(`
resource "newrelic_alert_policy" "foo" {
  name                = "tf-test-updated-%s"
  incident_preference = "PER_CONDITION"
}
`, rName)
}
