package openstack

import (
	"fmt"
	"testing"

	"github.com/haklop/gophercloud-extensions/network"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/racker/perigee"
	"github.com/rackspace/gophercloud"
)

func TestAccOpenstackSecurityGroup(t *testing.T) {
	var group network.SecurityGroup

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckOpenstackSecurityGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testSecurityGroupConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckOpenstackSecurityGroupExists("openstack_security_group.accept_test", &group),
					resource.TestCheckResourceAttr(
						"openstack_security_group.accept_test", "name", "http_rules"),
					resource.TestCheckResourceAttr(
						"openstack_security_group.accept_test", "rule.0.direction", "ingress"),
					resource.TestCheckResourceAttr(
						"openstack_security_group.accept_test", "rule.0.port_range_min", "80"),
					resource.TestCheckResourceAttr(
						"openstack_security_group.accept_test", "rule.0.port_range_max", "80"),
					resource.TestCheckResourceAttr(
						"openstack_security_group.accept_test", "rule.0.protocol", "tcp"),
					resource.TestCheckResourceAttr(
						"openstack_security_group.accept_test", "rule.0.remote_ip_prefix", "0.0.0.0/0"),
					resource.TestCheckResourceAttr(
						"openstack_security_group.accept_test", "rule.1.direction", "ingress"),
					resource.TestCheckResourceAttr(
						"openstack_security_group.accept_test", "rule.1.port_range_min", "443"),
					resource.TestCheckResourceAttr(
						"openstack_security_group.accept_test", "rule.1.port_range_max", "443"),
					resource.TestCheckResourceAttr(
						"openstack_security_group.accept_test", "rule.1.protocol", "tcp"),
					resource.TestCheckResourceAttr(
						"openstack_security_group.accept_test", "rule.1.remote_ip_prefix", "0.0.0.0/0"),
				),
			},
		},
	})
}

func testAccCheckOpenstackSecurityGroupDestroy(s *terraform.State) error {
	client := testAccProvider.client

	for _, rs := range s.Resources {
		if rs.Type != "openstack_security_group" {
			continue
		}

		networksApi, err := network.NetworksApi(client.AccessProvider, gophercloud.ApiCriteria{
			Name:      "neutron",
			UrlChoice: gophercloud.PublicURL,
		})
		if err != nil {
			return err
		}

		_, err = networksApi.GetSecurityGroup(rs.ID)
		if err == nil {
			return fmt.Errorf("Security Group (%s) still exists.", rs.ID)
		}

		httpError, ok := err.(*perigee.UnexpectedResponseCodeError)
		if !ok {
			return fmt.Errorf("Unkonw Security Group error")
		}

		if httpError.Actual == 404 {
			return nil
		}

		return err
	}

	return nil
}

func testAccCheckOpenstackSecurityGroupExists(n string, securityGroup *network.SecurityGroup) resource.TestCheckFunc {

	return func(s *terraform.State) error {
		client := testAccProvider.client
		rs, ok := s.Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.ID == "" {
			return fmt.Errorf("No security group is set")
		}

		networksApi, err := network.NetworksApi(client.AccessProvider, gophercloud.ApiCriteria{
			Name:      "neutron",
			UrlChoice: gophercloud.PublicURL,
		})
		if err != nil {
			return err
		}

		securityGroup, err = networksApi.GetSecurityGroup(rs.ID)
		return err
	}
}

const testSecurityGroupConfig = `
resource "openstack_security_group" "accept_test" {
    name = "http_rules"
    rule {
      direction = "ingress"
      port_range_min = 80
      port_range_max = 80
      protocol = tcp
      remote_ip_prefix = "0.0.0.0/0"
    }
    rule {
      direction = "ingress"
      port_range_min = 443
      port_range_max = 443
      protocol = tcp
      remote_ip_prefix = "0.0.0.0/0"
    }
}
`
