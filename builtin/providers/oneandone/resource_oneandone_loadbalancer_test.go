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

func TestAccOneandoneLoadbalancer_Basic(t *testing.T) {
	var lb oneandone.LoadBalancer

	name := "test_loadbalancer"
	name_updated := "test_loadbalancer_renamed"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDOneandoneLoadbalancerDestroyCheck,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(testAccCheckOneandoneLoadbalancer_basic, name),
				Check: resource.ComposeTestCheckFunc(
					func(*terraform.State) error {
						time.Sleep(10 * time.Second)
						return nil
					},
					testAccCheckOneandoneLoadbalancerExists("oneandone_loadbalancer.lb", &lb),
					testAccCheckOneandoneLoadbalancerAttributes("oneandone_loadbalancer.lb", name),
					resource.TestCheckResourceAttr("oneandone_loadbalancer.lb", "name", name),
				),
			},
			resource.TestStep{
				Config: fmt.Sprintf(testAccCheckOneandoneLoadbalancer_update, name_updated),
				Check: resource.ComposeTestCheckFunc(
					func(*terraform.State) error {
						time.Sleep(10 * time.Second)
						return nil
					},
					testAccCheckOneandoneLoadbalancerExists("oneandone_loadbalancer.lb", &lb),
					testAccCheckOneandoneLoadbalancerAttributes("oneandone_loadbalancer.lb", name_updated),
					resource.TestCheckResourceAttr("oneandone_loadbalancer.lb", "name", name_updated),
				),
			},
		},
	})
}

func testAccCheckDOneandoneLoadbalancerDestroyCheck(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "oneandone_loadbalancer.lb" {
			continue
		}

		api := oneandone.New(os.Getenv("ONEANDONE_TOKEN"), oneandone.BaseUrl)

		_, err := api.GetLoadBalancer(rs.Primary.ID)

		if err == nil {
			return fmt.Errorf("Loadbalancer still exists %s %s", rs.Primary.ID, err.Error())
		}
	}

	return nil
}
func testAccCheckOneandoneLoadbalancerAttributes(n string, reverse_dns string) resource.TestCheckFunc {
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

func testAccCheckOneandoneLoadbalancerExists(n string, fw_p *oneandone.LoadBalancer) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Record ID is set")
		}

		api := oneandone.New(os.Getenv("ONEANDONE_TOKEN"), oneandone.BaseUrl)

		found_fw, err := api.GetLoadBalancer(rs.Primary.ID)

		if err != nil {
			return fmt.Errorf("Error occured while fetching Loadbalancer: %s", rs.Primary.ID)
		}
		if found_fw.Id != rs.Primary.ID {
			return fmt.Errorf("Record not found")
		}
		fw_p = found_fw

		return nil
	}
}

const testAccCheckOneandoneLoadbalancer_basic = `
resource "oneandone_loadbalancer" "lb" {
  name = "%s"
  method = "ROUND_ROBIN"
  persistence = true
  persistence_time = 60
  health_check_test = "TCP"
  health_check_interval = 300
  datacenter = "US"
  rules = [
    {
      protocol = "TCP"
      port_balancer = 8080
      port_server = 8089
      source_ip = "0.0.0.0"
    },
    {
      protocol = "TCP"
      port_balancer = 9090
      port_server = 9099
      source_ip = "0.0.0.0"
    }
  ]
}`

const testAccCheckOneandoneLoadbalancer_update = `
resource "oneandone_loadbalancer" "lb" {
  name = "%s"
  method = "ROUND_ROBIN"
  persistence = true
  persistence_time = 60
  health_check_test = "TCP"
  health_check_interval = 300
  datacenter = "US"
  rules = [
    {
      protocol = "TCP"
      port_balancer = 8080
      port_server = 8089
      source_ip = "0.0.0.0"
    }
  ]
}`
