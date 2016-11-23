package newrelic

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	newrelic "github.com/paultyng/go-newrelic/api"
)

func TestAccNewRelicAlertPolicyChannel_Basic(t *testing.T) {
	rName := acctest.RandString(5)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckNewRelicAlertPolicyChannelDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckNewRelicAlertPolicyChannelConfig(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNewRelicAlertPolicyChannelExists("newrelic_alert_policy_channel.foo"),
				),
			},
			resource.TestStep{
				Config: testAccCheckNewRelicAlertPolicyChannelConfigUpdated(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNewRelicAlertPolicyChannelExists("newrelic_alert_policy_channel.foo"),
				),
			},
		},
	})
}

func testAccCheckNewRelicAlertPolicyChannelDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*newrelic.Client)
	for _, r := range s.RootModule().Resources {
		if r.Type != "newrelic_alert_policy_channel" {
			continue
		}

		ids, err := parseIDs(r.Primary.ID, 2)
		if err != nil {
			return err
		}

		policyID := ids[0]
		channelID := ids[1]

		exists, err := policyChannelExists(client, policyID, channelID)
		if err != nil {
			return err
		}

		if exists {
			return fmt.Errorf("Resource still exists")
		}
	}
	return nil
}

func testAccCheckNewRelicAlertPolicyChannelExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No resource ID is set")
		}

		client := testAccProvider.Meta().(*newrelic.Client)

		ids, err := parseIDs(rs.Primary.ID, 2)
		if err != nil {
			return err
		}

		policyID := ids[0]
		channelID := ids[1]

		exists, err := policyChannelExists(client, policyID, channelID)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("Resource not found: %v", rs.Primary.ID)
		}

		return nil
	}
}

func testAccCheckNewRelicAlertPolicyChannelConfig(rName string) string {
	return fmt.Sprintf(`
resource "newrelic_alert_policy" "foo" {
  name = "tf-test-%[1]s"
}

resource "newrelic_alert_channel" "foo" {
  name = "tf-test-%[1]s"
	type = "email"
	
	configuration = {
		recipients = "foo@example.com"
		include_json_attachment = "1"
	}
}

resource "newrelic_alert_policy_channel" "foo" {
  policy_id  = "${newrelic_alert_policy.foo.id}"
  channel_id = "${newrelic_alert_channel.foo.id}"
}
`, rName)
}

func testAccCheckNewRelicAlertPolicyChannelConfigUpdated(rName string) string {
	return fmt.Sprintf(`
resource "newrelic_alert_policy" "bar" {
  name = "tf-test-updated-%[1]s"
}

resource "newrelic_alert_channel" "foo" {
  name = "tf-test-updated-%[1]s"
	type = "email"
	
	configuration = {
		recipients = "bar@example.com"
		include_json_attachment = "0"
	}
}

resource "newrelic_alert_policy_channel" "foo" {
  policy_id  = "${newrelic_alert_policy.bar.id}"
  channel_id = "${newrelic_alert_channel.foo.id}"
}
`, rName)
}
