package ns1

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	ns1 "gopkg.in/ns1/ns1-go.v2/rest"
	"gopkg.in/ns1/ns1-go.v2/rest/model/account"
)

func TestAccNS1Team_Basic(t *testing.T) {
	var team account.Team

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckNS1TeamDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccNS1Team_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNS1TeamExists("ns1_team.foobar", &team),
					testAccCheckNS1TeamAttributes(&team),
					resource.TestCheckResourceAttr("ns1_team.foobar", "name", "terraform test"),
					resource.TestCheckResourceAttr("ns1_team.foobar", "permissions.#", "1"),
					resource.TestCheckResourceAttr("ns1_team.foobar", "permissions.dns.#", "1"),
				),
			},
		},
	})
}

func TestAccNS1Team_Updated(t *testing.T) {
	var team account.Team

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckNS1TeamDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccNS1Team_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNS1TeamExists("ns1_team.foobar", &team),
					testAccCheckNS1TeamAttributes(&team),
				),
			},
			resource.TestStep{
				Config: testAccNS1Team_updated,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNS1TeamExists("ns1_zone.foobar", &team),
					testAccCheckNS1TeamAttributesUpdated(&team),
					resource.TestCheckResourceAttr("ns1_team.foobar", "name", "terraform test updated"),
					resource.TestCheckResourceAttr("ns1_team.foobar", "permissions.#", "1"),
					resource.TestCheckResourceAttr("ns1_team.foobar", "permissions.dns.#", "1"),
				),
			},
		},
	})
}

func testAccCheckNS1TeamExists(n string, team *account.Team) resource.TestCheckFunc {
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

func testAccCheckNS1TeamDestroy(s *terraform.State) error {
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

func testAccCheckNS1TeamAttributes(team *account.Team) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if team.Name != "terraform test" {
			return fmt.Errorf("Bad value team.Name: %s", team.Name)
		}

		return nil
	}
}

func testAccCheckNS1TeamAttributesUpdated(team *account.Team) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if team.Name != "terraform test updated" {
			return fmt.Errorf("Bad value team.Name: %s", team.Name)
		}

		return nil
	}
}

const testAccNS1Team_basic = `
resource "ns1_team" "foobar" {
  name = "terraform test"
    permissions = {
      dns = {
	view_zones = true
	/* manage_zones = true */
        zones_allow_by_default = true
        zones_allow = ["mytest.zone"]
        zones_deny = ["myother.zone"]
      }
    }
}`

const testAccNS1Team_updated = `
resource "ns1_team" "foobar" {
  name = "terraform test updated"
    permissions = {
      dns = {
	view_zones = true
	/* manage_zones = true */
        zones_allow_by_default = true
        zones_allow = ["mytest.zone"]
        zones_deny = ["myother.zone"]
      }
    }
}`
