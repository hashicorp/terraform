package openstack

import (
	"fmt"
	"testing"
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/fwaas/firewalls"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccFWFirewallV1_basic(t *testing.T) {
	var policyID *string

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckFWFirewallV1Destroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccFWFirewallV1_basic_1,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFWFirewallV1("openstack_fw_firewall_v1.fw_1", "", "", policyID),
				),
			},
			resource.TestStep{
				Config: testAccFWFirewallV1_basic_2,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFWFirewallV1(
						"openstack_fw_firewall_v1.fw_1", "fw_1", "terraform acceptance test", policyID),
				),
			},
		},
	})
}

func TestAccFWFirewallV1_timeout(t *testing.T) {
	var policyID *string

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckFWFirewallV1Destroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccFWFirewallV1_timeout,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFWFirewallV1("openstack_fw_firewall_v1.fw_1", "", "", policyID),
				),
			},
		},
	})
}

func TestAccFWFirewallV1_router(t *testing.T) {
	var firewall Firewall

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckFWFirewallV1Destroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccFWFirewallV1_router,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFWFirewallV1Exists("openstack_fw_firewall_v1.fw_1", &firewall),
					testAccCheckFWFirewallRouterCount(&firewall, 1),
				),
			},
		},
	})
}

func TestAccFWFirewallV1_no_router(t *testing.T) {
	var firewall Firewall

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckFWFirewallV1Destroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccFWFirewallV1_no_router,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFWFirewallV1Exists("openstack_fw_firewall_v1.fw_1", &firewall),
					resource.TestCheckResourceAttr("openstack_fw_firewall_v1.fw_1", "description", "firewall router test"),
					testAccCheckFWFirewallRouterCount(&firewall, 0),
				),
			},
		},
	})
}

func TestAccFWFirewallV1_router_update(t *testing.T) {
	var firewall Firewall

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckFWFirewallV1Destroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccFWFirewallV1_router,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFWFirewallV1Exists("openstack_fw_firewall_v1.fw_1", &firewall),
					testAccCheckFWFirewallRouterCount(&firewall, 1),
				),
			},
			resource.TestStep{
				Config: testAccFWFirewallV1_router_add,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFWFirewallV1Exists("openstack_fw_firewall_v1.fw_1", &firewall),
					testAccCheckFWFirewallRouterCount(&firewall, 2),
				),
			},
		},
	})
}

func TestAccFWFirewallV1_router_remove(t *testing.T) {
	var firewall Firewall

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckFWFirewallV1Destroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccFWFirewallV1_router,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFWFirewallV1Exists("openstack_fw_firewall_v1.fw_1", &firewall),
					testAccCheckFWFirewallRouterCount(&firewall, 1),
				),
			},
			resource.TestStep{
				Config: testAccFWFirewallV1_router_remove,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFWFirewallV1Exists("openstack_fw_firewall_v1.fw_1", &firewall),
					testAccCheckFWFirewallRouterCount(&firewall, 0),
				),
			},
		},
	})
}

func testAccCheckFWFirewallV1Destroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)
	networkingClient, err := config.networkingV2Client(OS_REGION_NAME)
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "openstack_firewall" {
			continue
		}

		_, err = firewalls.Get(networkingClient, rs.Primary.ID).Extract()
		if err == nil {
			return fmt.Errorf("Firewall (%s) still exists.", rs.Primary.ID)
		}
		if _, ok := err.(gophercloud.ErrDefault404); !ok {
			return err
		}
	}
	return nil
}

func testAccCheckFWFirewallV1Exists(n string, firewall *Firewall) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		config := testAccProvider.Meta().(*Config)
		networkingClient, err := config.networkingV2Client(OS_REGION_NAME)
		if err != nil {
			return fmt.Errorf("Exists) Error creating OpenStack networking client: %s", err)
		}

		var found Firewall
		err = firewalls.Get(networkingClient, rs.Primary.ID).ExtractInto(&found)
		if err != nil {
			return err
		}

		if found.ID != rs.Primary.ID {
			return fmt.Errorf("Firewall not found")
		}

		*firewall = found

		return nil
	}
}

func testAccCheckFWFirewallRouterCount(firewall *Firewall, expected int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if len(firewall.RouterIDs) != expected {
			return fmt.Errorf("Expected %d Routers, got %d", expected, len(firewall.RouterIDs))
		}

		return nil
	}
}

func testAccCheckFWFirewallV1(n, expectedName, expectedDescription string, policyID *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		config := testAccProvider.Meta().(*Config)
		networkingClient, err := config.networkingV2Client(OS_REGION_NAME)
		if err != nil {
			return fmt.Errorf("Exists) Error creating OpenStack networking client: %s", err)
		}

		var found *firewalls.Firewall
		for i := 0; i < 5; i++ {
			// Firewall creation is asynchronous. Retry some times
			// if we get a 404 error. Fail on any other error.
			found, err = firewalls.Get(networkingClient, rs.Primary.ID).Extract()
			if err != nil {
				if _, ok := err.(gophercloud.ErrDefault404); ok {
					time.Sleep(time.Second)
					continue
				}
				return err
			}
			break
		}

		switch {
		case found.Name != expectedName:
			err = fmt.Errorf("Expected Name to be <%s> but found <%s>", expectedName, found.Name)
		case found.Description != expectedDescription:
			err = fmt.Errorf("Expected Description to be <%s> but found <%s>",
				expectedDescription, found.Description)
		case found.PolicyID == "":
			err = fmt.Errorf("Policy should not be empty")
		case policyID != nil && found.PolicyID == *policyID:
			err = fmt.Errorf("Policy had not been correctly updated. Went from <%s> to <%s>",
				expectedName, found.Name)
		}

		if err != nil {
			return err
		}

		policyID = &found.PolicyID

		return nil
	}
}

const testAccFWFirewallV1_basic_1 = `
resource "openstack_fw_firewall_v1" "fw_1" {
  policy_id = "${openstack_fw_policy_v1.policy_1.id}"
}

resource "openstack_fw_policy_v1" "policy_1" {
  name = "policy_1"
}
`

const testAccFWFirewallV1_basic_2 = `
resource "openstack_fw_firewall_v1" "fw_1" {
  name = "fw_1"
  description = "terraform acceptance test"
  policy_id = "${openstack_fw_policy_v1.policy_2.id}"
  admin_state_up = true
}

resource "openstack_fw_policy_v1" "policy_2" {
  name = "policy_2"
}
`

const testAccFWFirewallV1_timeout = `
resource "openstack_fw_firewall_v1" "fw_1" {
  policy_id = "${openstack_fw_policy_v1.policy_1.id}"

  timeouts {
    create = "5m"
    update = "5m"
    delete = "5m"
  }
}

resource "openstack_fw_policy_v1" "policy_1" {
  name = "policy_1"
}
`

const testAccFWFirewallV1_router = `
resource "openstack_networking_router_v2" "router_1" {
  name = "router_1"
  admin_state_up = "true"
  distributed = "false"
}

resource "openstack_fw_policy_v1" "policy_1" {
  name = "policy_1"
}

resource "openstack_fw_firewall_v1" "fw_1" {
  name = "firewall_1"
  description = "firewall router test"
  policy_id = "${openstack_fw_policy_v1.policy_1.id}"
  associated_routers = ["${openstack_networking_router_v2.router_1.id}"]
}
`

const testAccFWFirewallV1_router_add = `
resource "openstack_networking_router_v2" "router_1" {
  name = "router_1"
  admin_state_up = "true"
  distributed = "false"
}

resource "openstack_networking_router_v2" "router_2" {
  name = "router_2"
  admin_state_up = "true"
  distributed = "false"
}

resource "openstack_fw_policy_v1" "policy_1" {
  name = "policy_1"
}

resource "openstack_fw_firewall_v1" "fw_1" {
  name = "firewall_1"
  description = "firewall router test"
  policy_id = "${openstack_fw_policy_v1.policy_1.id}"
  associated_routers = [
    "${openstack_networking_router_v2.router_1.id}",
    "${openstack_networking_router_v2.router_2.id}"
  ]
}
`

const testAccFWFirewallV1_router_remove = `
resource "openstack_fw_policy_v1" "policy_1" {
  name = "policy_1"
}

resource "openstack_fw_firewall_v1" "fw_1" {
  name = "firewall_1"
  description = "firewall router test"
  policy_id = "${openstack_fw_policy_v1.policy_1.id}"
  no_routers = true
}
`

const testAccFWFirewallV1_no_router = `
resource "openstack_fw_policy_v1" "policy_1" {
  name = "policy_1"
}

resource "openstack_fw_firewall_v1" "fw_1" {
  name = "firewall_1"
  description = "firewall router test"
  policy_id = "${openstack_fw_policy_v1.policy_1.id}"
  no_routers = true
}
`
