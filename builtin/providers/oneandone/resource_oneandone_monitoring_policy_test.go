package oneandone

import (
	"fmt"
	"testing"

	"github.com/1and1/oneandone-cloudserver-sdk-go"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"os"
	"time"
)

func TestAccOneandoneMonitoringPolicy_Basic(t *testing.T) {
	var mp oneandone.MonitoringPolicy

	name := "test"
	name_updated := "test1"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDOneandoneMonitoringPolicyDestroyCheck,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(testAccCheckOneandoneMonitoringPolicy_basic, name),
				Check: resource.ComposeTestCheckFunc(
					func(*terraform.State) error {
						time.Sleep(10 * time.Second)
						return nil
					},
					testAccCheckOneandoneMonitoringPolicyExists("oneandone_monitoring_policy.mp", &mp),
					testAccCheckOneandoneMonitoringPolicyAttributes("oneandone_monitoring_policy.mp", name),
					resource.TestCheckResourceAttr("oneandone_monitoring_policy.mp", "name", name),
				),
			},
			resource.TestStep{
				Config: fmt.Sprintf(testAccCheckOneandoneMonitoringPolicy_basic, name_updated),
				Check: resource.ComposeTestCheckFunc(
					func(*terraform.State) error {
						time.Sleep(10 * time.Second)
						return nil
					},
					testAccCheckOneandoneMonitoringPolicyExists("oneandone_monitoring_policy.mp", &mp),
					testAccCheckOneandoneMonitoringPolicyAttributes("oneandone_monitoring_policy.mp", name_updated),
					resource.TestCheckResourceAttr("oneandone_monitoring_policy.mp", "name", name_updated),
				),
			},
		},
	})
}

func testAccCheckDOneandoneMonitoringPolicyDestroyCheck(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "oneandone_monitoring_policy.mp" {
			continue
		}

		api := oneandone.New(os.Getenv("ONEANDONE_TOKEN"), oneandone.BaseUrl)

		_, err := api.GetMonitoringPolicy(rs.Primary.ID)

		if err == nil {
			return fmt.Errorf("MonitoringPolicy still exists %s %s", rs.Primary.ID, err.Error())
		}
	}

	return nil
}
func testAccCheckOneandoneMonitoringPolicyAttributes(n string, reverse_dns string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}
		if rs.Primary.Attributes["name"] != reverse_dns {
			return fmt.Errorf("Bad name: expected %s : found %s ", reverse_dns, rs.Primary.Attributes["name"])
		}

		return nil
	}
}

func testAccCheckOneandoneMonitoringPolicyExists(n string, fw_p *oneandone.MonitoringPolicy) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Record ID is set")
		}

		api := oneandone.New(os.Getenv("ONEANDONE_TOKEN"), oneandone.BaseUrl)

		found_fw, err := api.GetMonitoringPolicy(rs.Primary.ID)

		if err != nil {
			return fmt.Errorf("Error occured while fetching MonitoringPolicy: %s", rs.Primary.ID)
		}
		if found_fw.Id != rs.Primary.ID {
			return fmt.Errorf("Record not found")
		}
		fw_p = found_fw

		return nil
	}
}

const testAccCheckOneandoneMonitoringPolicy_basic = `
resource "oneandone_monitoring_policy" "mp" {
  name = "%s"
  agent = true
  email = "email@address.com"
  thresholds = {
    cpu = {
      warning = {
        value = 50,
        alert = false
      }
      critical = {
        value = 66,
        alert = false
      }
    }
    ram = {
      warning = {
        value = 70,
        alert = true
      }
      critical = {
        value = 80,
        alert = true
      }
    },
    ram = {
      warning = {
        value = 85,
        alert = true
      }
      critical = {
        value = 95,
        alert = true
      }
    },
    disk = {
      warning = {
        value = 84,
        alert = true
      }
      critical = {
        value = 94,
        alert = true
      }
    },
    transfer = {
      warning = {
        value = 1000,
        alert = true
      }
      critical = {
        value = 2000,
        alert = true
      }
    },
    internal_ping = {
      warning = {
        value = 3000,
        alert = true
      }
      critical = {
        value = 4000,
        alert = true
      }
    }
  }
  ports = [
    {
      email_notification = true
      port = 443
      protocol = "TCP"
      alert_if = "NOT_RESPONDING"
    },
    {
      email_notification = false
      port = 80
      protocol = "TCP"
      alert_if = "NOT_RESPONDING"
    },
    {
      email_notification = true
      port = 21
      protocol = "TCP"
      alert_if = "NOT_RESPONDING"
    }
  ]
  processes = [
    {
      email_notification = false
      process = "httpdeamon"
      alert_if = "RUNNING"
    },
    {
      process = "iexplorer",
      alert_if = "NOT_RUNNING"
      email_notification = true
    }]
}`
