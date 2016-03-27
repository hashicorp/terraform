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
//   basic create/delete
//   update nodes
//   works for 80 and 443 together

func TestAccLBPoolBasic(t *testing.T) {
	var pool lb.Pool
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckLBPDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckLBPConfigBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckLBPExists("clc_load_balancer_pool.acc_test_pool", &pool),
					resource.TestCheckResourceAttr("clc_load_balancer_pool.acc_test_pool", "port", "80"),
				),
			},
			resource.TestStep{
				Config: testAccCheckLBPConfigUpdates,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckLBPExists("clc_load_balancer_pool.acc_test_pool", &pool),
					resource.TestCheckResourceAttr("clc_load_balancer.acc_test_lbp", "description", "description modified"),
					resource.TestCheckResourceAttr("clc_load_balancer.acc_test_lbp", "status", "disabled"),
					resource.TestCheckResourceAttr("clc_load_balancer_pool.acc_test_pool", "nodes.0.privatePort", "8080"),
				),
			},
		},
	})
}

func testAccCheckLBPDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*clc.Client)
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "clc_load_balancer_pool" {
			continue
		}
		lbid := rs.Primary.Attributes["load_balancer"]
		if _, err := client.LB.Get(testAccDC, rs.Primary.ID); err != nil {
			return nil // parent LB already gone
		}
		if _, err := client.LB.GetPool(testAccDC, lbid, rs.Primary.ID); err == nil {
			return fmt.Errorf("LB still exists")
		}
	}
	return nil
}

func testAccCheckLBPExists(n string, resp *lb.Pool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}
		lbid := rs.Primary.Attributes["load_balancer"]
		client := testAccProvider.Meta().(*clc.Client)
		p, err := client.LB.GetPool(testAccDC, lbid, rs.Primary.ID)
		if err != nil {
			return err
		}
		if p.ID != rs.Primary.ID {
			return fmt.Errorf("Pool not found")
		}
		*resp = *p
		return nil
	}
}

const testAccCheckLBPConfigBasic = `
variable "dc" { default = "IL1" }

resource "clc_group" "acc_test_lbp_group" {
  location_id		= "${var.dc}"
  name			= "acc_test_lbp_group"
  parent		= "Default Group"
}

# need a server here because we need to reference an ip owned by this account
resource "clc_server" "acc_test_lbp_server" {
  name_template		= "node"
  description		= "load balanced"
  source_server_id	= "UBUNTU-14-64-TEMPLATE"
  type			= "standard"
  group_id		= "${clc_group.acc_test_lbp_group.id}"
  cpu			= 1
  memory_mb		= 1024
  password		= "Green123$"
  power_state		= "started"

}

resource "clc_load_balancer" "acc_test_lbp" {
  data_center		= "${var.dc}"
  name			= "acc_test_lb"
  description		= "load balancer test"
  status		= "enabled"
  depends_on            = ["clc_server.acc_test_lbp_server"]
}

resource "clc_load_balancer_pool" "acc_test_pool" {
  port			= 80
  data_center		= "${var.dc}"
  load_balancer		= "${clc_load_balancer.acc_test_lbp.id}"
  nodes
    {
      status		= "enabled"
      ipAddress		= "${clc_server.acc_test_lbp_server.private_ip_address}"
      privatePort	= 80
    }
  depends_on            = ["clc_server.acc_test_lbp_server"]
}
`

const testAccCheckLBPConfigUpdates = `
variable "dc" { default = "IL1" }

resource "clc_group" "acc_test_lbp_group" {
  location_id		= "${var.dc}"
  name			= "acc_test_lbp_group"
  parent		= "Default Group"
}

# need a server here because we need to reference an ip owned by this account
resource "clc_server" "acc_test_lbp_server" {
  name_template		= "node"
  description		= "load balanced"
  source_server_id	= "UBUNTU-14-64-TEMPLATE"
  type			= "standard"
  group_id		= "${clc_group.acc_test_lbp_group.id}"
  cpu			= 1
  memory_mb		= 1024
  password		= "Green123$"
  power_state		= "started"

}

resource "clc_load_balancer" "acc_test_lbp" {
  data_center		= "${var.dc}"
  name			= "acc_test_lb"
  description		= "description modified"
  status		= "disabled"
  depends_on            = ["clc_server.acc_test_lbp_server"]
}

resource "clc_load_balancer_pool" "acc_test_pool" {
  port			= 80
  data_center		= "${var.dc}"
  load_balancer		= "${clc_load_balancer.acc_test_lbp.id}"
  nodes
    {
      status		= "enabled"
      ipAddress		= "${clc_server.acc_test_lbp_server.private_ip_address}"
      privatePort	= 8080
    }
  depends_on            = ["clc_server.acc_test_lbp_server"]
}
`
