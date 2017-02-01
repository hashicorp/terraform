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

func TestAccNewRelicAlertChannel_Basic(t *testing.T) {
	rName := acctest.RandString(5)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckNewRelicAlertChannelDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckNewRelicAlertChannelConfig(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNewRelicAlertChannelExists("newrelic_alert_channel.foo"),
					resource.TestCheckResourceAttr(
						"newrelic_alert_channel.foo", "name", fmt.Sprintf("tf-test-%s", rName)),
					resource.TestCheckResourceAttr(
						"newrelic_alert_channel.foo", "type", "email"),
					resource.TestCheckResourceAttr(
						"newrelic_alert_channel.foo", "configuration.recipients", "foo@example.com"),
					resource.TestCheckResourceAttr(
						"newrelic_alert_channel.foo", "configuration.include_json_attachment", "1"),
				),
			},
			resource.TestStep{
				Config: testAccCheckNewRelicAlertChannelConfigUpdated(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNewRelicAlertChannelExists("newrelic_alert_channel.foo"),
					resource.TestCheckResourceAttr(
						"newrelic_alert_channel.foo", "name", fmt.Sprintf("tf-test-updated-%s", rName)),
					resource.TestCheckResourceAttr(
						"newrelic_alert_channel.foo", "type", "email"),
					resource.TestCheckResourceAttr(
						"newrelic_alert_channel.foo", "configuration.recipients", "bar@example.com"),
					resource.TestCheckResourceAttr(
						"newrelic_alert_channel.foo", "configuration.include_json_attachment", "0"),
				),
			},
		},
	})
}

func testAccCheckNewRelicAlertChannelDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*newrelic.Client)
	for _, r := range s.RootModule().Resources {
		if r.Type != "newrelic_alert_channel" {
			continue
		}

		id, err := strconv.ParseInt(r.Primary.ID, 10, 32)
		if err != nil {
			return err
		}

		_, err = client.GetAlertChannel(int(id))

		if err == nil {
			return fmt.Errorf("Alert channel still exists")
		}

	}
	return nil
}

func testAccCheckNewRelicAlertChannelExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No channel ID is set")
		}

		client := testAccProvider.Meta().(*newrelic.Client)

		id, err := strconv.ParseInt(rs.Primary.ID, 10, 32)
		if err != nil {
			return err
		}

		found, err := client.GetAlertChannel(int(id))
		if err != nil {
			return err
		}

		if strconv.Itoa(found.ID) != rs.Primary.ID {
			return fmt.Errorf("Channel not found: %v - %v", rs.Primary.ID, found)
		}

		return nil
	}
}

func testAccCheckNewRelicAlertChannelConfig(rName string) string {
	return fmt.Sprintf(`
resource "newrelic_alert_channel" "foo" {
  name = "tf-test-%s"
	type = "email"
	
	configuration = {
		recipients = "foo@example.com"
		include_json_attachment = "1"
	}
}
`, rName)
}

func testAccCheckNewRelicAlertChannelConfigUpdated(rName string) string {
	return fmt.Sprintf(`
resource "newrelic_alert_channel" "foo" {
  name = "tf-test-updated-%s"
	type = "email"
	
	configuration = {
		recipients = "bar@example.com"
		include_json_attachment = "0"
	}
}
`, rName)
}
