package clc

import (
	"fmt"
	"testing"

	clc "github.com/CenturyLinkCloud/clc-sdk"
	lb "github.com/CenturyLinkCloud/clc-sdk/lb"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

// things to test:
//   updates name/desc
//   toggles status
//   created w/o pool

const testAccDC = "WA1"

func TestAccLoadBalancerBasic(t *testing.T) {
	var resp lb.LoadBalancer
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckLBDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckLBConfigBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckLBExists("clc_load_balancer.acc_test_lb", &resp),
					resource.TestCheckResourceAttr("clc_load_balancer.acc_test_lb", "name", "acc_test_lb"),
					resource.TestCheckResourceAttr("clc_load_balancer.acc_test_lb", "data_center", testAccDC),
					resource.TestCheckResourceAttr("clc_load_balancer.acc_test_lb", "status", "enabled"),
				),
			},
			// update simple attrs
			resource.TestStep{
				Config: testAccCheckLBConfigNameDesc,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckLBExists("clc_load_balancer.acc_test_lb", &resp),
					resource.TestCheckResourceAttr("clc_load_balancer.acc_test_lb", "name", "foobar"),
					resource.TestCheckResourceAttr("clc_load_balancer.acc_test_lb", "description", "foobar"),
					resource.TestCheckResourceAttr("clc_load_balancer.acc_test_lb", "status", "disabled"),
				),
			},
		},
	})
}

func testAccCheckLBDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*clc.Client)
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "clc_load_balancer" {
			continue
		}
		_, err := client.LB.Get(testAccDC, rs.Primary.ID)
		if err == nil {
			return fmt.Errorf("LB still exists")
		}
	}
	return nil
}

func testAccCheckLBExists(n string, resp *lb.LoadBalancer) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}
		client := testAccProvider.Meta().(*clc.Client)
		l, err := client.LB.Get(testAccDC, rs.Primary.ID)
		if err != nil {
			return err
		}
		if l.ID != rs.Primary.ID {
			return fmt.Errorf("LB not found")
		}
		*resp = *l
		return nil
	}
}

const testAccCheckLBConfigBasic = `
resource "clc_load_balancer" "acc_test_lb" {
  data_center	= "WA1"
  name		= "acc_test_lb"
  description	= "load balancer test"
  status	= "enabled"
}`

const testAccCheckLBConfigNameDesc = `
resource "clc_load_balancer" "acc_test_lb" {
  data_center	= "WA1"
  name		= "foobar"
  description	= "foobar"
  status	= "disabled"
}`
