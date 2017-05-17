package ns1

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	ns1 "gopkg.in/ns1/ns1-go.v2/rest"
	"gopkg.in/ns1/ns1-go.v2/rest/model/account"
)

func TestAccTeam_basic(t *testing.T) {
	var team account.Team

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckTeamDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccTeamBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTeamExists("ns1_team.foobar", &team),
					testAccCheckTeamName(&team, "terraform test"),
					testAccCheckTeamDNSPermission(&team, "view_zones", true),
					testAccCheckTeamDNSPermission(&team, "zones_allow_by_default", true),
					testAccCheckTeamDNSPermissionZones(&team, "zones_allow", []string{"mytest.zone"}),
					testAccCheckTeamDNSPermissionZones(&team, "zones_deny", []string{"myother.zone"}),
					testAccCheckTeamDataPermission(&team, "manage_datasources", true),
				),
			},
		},
	})
}

func TestAccTeam_updated(t *testing.T) {
	var team account.Team

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckTeamDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccTeamBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTeamExists("ns1_team.foobar", &team),
					testAccCheckTeamName(&team, "terraform test"),
				),
			},
			resource.TestStep{
				Config: testAccTeamUpdated,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTeamExists("ns1_team.foobar", &team),
					testAccCheckTeamName(&team, "terraform test updated"),
					testAccCheckTeamDNSPermission(&team, "view_zones", true),
					testAccCheckTeamDNSPermission(&team, "zones_allow_by_default", true),
					testAccCheckTeamDNSPermissionZones(&team, "zones_allow", []string{}),
					testAccCheckTeamDNSPermissionZones(&team, "zones_deny", []string{}),
					testAccCheckTeamDataPermission(&team, "manage_datasources", false),
				),
			},
		},
	})
}

func testAccCheckTeamExists(n string, team *account.Team) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("NoID is set")
		}

		client := testAccProvider.Meta().(*ns1.Client)

		foundTeam, _, err := client.Teams.Get(rs.Primary.Attributes["id"])
		if err != nil {
			return err
		}

		if foundTeam.Name != rs.Primary.Attributes["name"] {
			return fmt.Errorf("Team not found")
		}

		*team = *foundTeam

		return nil
	}
}

func testAccCheckTeamDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*ns1.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "ns1_team" {
			continue
		}

		team, _, err := client.Teams.Get(rs.Primary.Attributes["id"])
		if err == nil {
			return fmt.Errorf("Team still exists: %#v: %#v", err, team.Name)
		}
	}

	return nil
}

func testAccCheckTeamName(team *account.Team, expected string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if team.Name != expected {
			return fmt.Errorf("Name: got: %s want: %s", team.Name, expected)
		}
		return nil
	}
}

func testAccCheckTeamDNSPermission(team *account.Team, perm string, expected bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		dns := team.Permissions.DNS

		switch perm {
		case "view_zones":
			if dns.ViewZones != expected {
				return fmt.Errorf("DNS.ViewZones: got: %t want: %t", dns.ViewZones, expected)
			}
		case "manage_zones":
			if dns.ManageZones != expected {
				return fmt.Errorf("DNS.ManageZones: got: %t want: %t", dns.ManageZones, expected)
			}
		case "zones_allow_by_default":
			if dns.ZonesAllowByDefault != expected {
				return fmt.Errorf("DNS.ZonesAllowByDefault: got: %t want: %t", dns.ZonesAllowByDefault, expected)
			}
		}

		return nil
	}
}

func testAccCheckTeamDataPermission(team *account.Team, perm string, expected bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		data := team.Permissions.Data

		switch perm {
		case "push_to_datafeeds":
			if data.PushToDatafeeds != expected {
				return fmt.Errorf("Data.PushToDatafeeds: got: %t want: %t", data.PushToDatafeeds, expected)
			}
		case "manage_datasources":
			if data.ManageDatasources != expected {
				return fmt.Errorf("Data.ManageDatasources: got: %t want: %t", data.ManageDatasources, expected)
			}
		case "manage_datafeeds":
			if data.ManageDatafeeds != expected {
				return fmt.Errorf("Data.ManageDatafeeds: got: %t want: %t", data.ManageDatafeeds, expected)
			}
		}

		return nil
	}
}

func testAccCheckTeamDNSPermissionZones(team *account.Team, perm string, expected []string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		dns := team.Permissions.DNS

		switch perm {
		case "zones_allow":
			if !reflect.DeepEqual(dns.ZonesAllow, expected) {
				return fmt.Errorf("DNS.ZonesAllow: got: %v want: %v", dns.ZonesAllow, expected)
			}
		case "zones_deny":
			if !reflect.DeepEqual(dns.ZonesDeny, expected) {
				return fmt.Errorf("DNS.ZonesDeny: got: %v want: %v", dns.ZonesDeny, expected)
			}
		}

		return nil
	}
}

const testAccTeamBasic = `
resource "ns1_team" "foobar" {
  name = "terraform test"

  dns_view_zones = true
  dns_zones_allow_by_default = true
  dns_zones_allow = ["mytest.zone"]
  dns_zones_deny = ["myother.zone"]

  data_manage_datasources = true
}`

const testAccTeamUpdated = `
resource "ns1_team" "foobar" {
  name = "terraform test updated"

  dns_view_zones = true
  dns_zones_allow_by_default = true

  data_manage_datasources = false
}`
