package clc

import (
	"fmt"
	"testing"

	clc "github.com/CenturyLinkCloud/clc-sdk"
	"github.com/CenturyLinkCloud/clc-sdk/group"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

// things to test:
//   resolves to existing group
//   does not nuke a group w/ no parents (root group)
//   change a name on a group

func TestAccGroupBasic(t *testing.T) {
	var resp group.Response
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckGroupConfigBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGroupExists("clc_group.acc_test_group", &resp),
					testAccCheckGroupParent(&resp, "Default Group"),
					resource.TestCheckResourceAttr(
						"clc_group.acc_test_group", "name", "okcomputer"),
					resource.TestCheckResourceAttr(
						"clc_group.acc_test_group", "location_id", testAccDC),
				),
			},
			resource.TestStep{
				Config: testAccCheckGroupConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGroupExists("clc_group.acc_test_group", &resp),
					testAccCheckGroupParent(&resp, "Default Group"),
					resource.TestCheckResourceAttr(
						"clc_group.acc_test_group", "name", "foobar"),
					resource.TestCheckResourceAttr(
						"clc_group.acc_test_group", "location_id", testAccDC),
				),
			},
			resource.TestStep{
				Config: testAccCheckGroupConfigReparent,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGroupExists("clc_group.acc_test_group", &resp),
					testAccCheckGroupParent(&resp, "reparent"),
					resource.TestCheckResourceAttr(
						"clc_group.acc_test_group", "name", "foobar"),
					resource.TestCheckResourceAttr(
						"clc_group.acc_test_group", "location_id", testAccDC),
				),
			},
		},
	})
}

func testAccCheckGroupDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*clc.Client)
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "clc_group" {
			continue
		}
		_, err := client.Group.Get(rs.Primary.ID)
		if err == nil {
			return fmt.Errorf("Group still exists")
		}
	}
	return nil
}

func testAccCheckGroupParent(resp *group.Response, expectedName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*clc.Client)
		ok, l := resp.Links.GetLink("parentGroup")
		if !ok {
			return fmt.Errorf("Missing parent group: %v", resp)
		}
		parent, err := client.Group.Get(l.ID)
		if err != nil {
			return fmt.Errorf("Failed fetching parent %v: %v", l.ID, err)
		}
		if parent.Name != expectedName {
			return fmt.Errorf("Incorrect parent found:'%v' expected:'%v'", parent.Name, expectedName)
		}
		// would be good to test parent but we'd have to make a bunch of calls
		return nil
	}
}

func testAccCheckGroupExists(n string, resp *group.Response) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No Group ID is set")
		}

		client := testAccProvider.Meta().(*clc.Client)
		g, err := client.Group.Get(rs.Primary.ID)
		if err != nil {
			return err
		}

		if g.ID != rs.Primary.ID {
			return fmt.Errorf("Group not found")
		}
		*resp = *g
		return nil
	}
}

const testAccCheckGroupConfigBasic = `
variable "dc" { default = "IL1" }

resource "clc_group" "acc_test_group" {
  location_id	= "${var.dc}"
  name		= "okcomputer"
  description	= "mishaps happening"
  parent	= "Default Group"
}`

const testAccCheckGroupConfigUpdate = `
variable "dc" { default = "IL1" }

resource "clc_group" "acc_test_group" {
  location_id	= "${var.dc}"
  name		= "foobar"
  description	= "update test"
  parent	= "Default Group"
}`

const testAccCheckGroupConfigReparent = `
variable "dc" { default = "IL1" }

resource "clc_group" "acc_test_group_reparent" {
  location_id	= "${var.dc}"
  name		= "reparent"
  description	= "introduce a parent group in place"
  parent	= "Default Group"
}

resource "clc_group" "acc_test_group" {
  location_id	= "${var.dc}"
  name		= "foobar"
  description	= "update test"
  parent	= "${clc_group.acc_test_group_reparent.id}"
}
`
