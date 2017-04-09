package cloudfoundry

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/builtin/providers/cf/cfapi"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

const userOrgRoleAssoc = `

resource "cf_user" "u" {
    name = "user"
}
resource "cf_org" "o1" {
    name = "org1"
}
resource "cf_org" "o2" {
    name = "org2"
}
resource "cf_org" "o3" {
    name = "org3"
}
resource "cf_org" "o4" {
    name = "org4"
}
resource "cf_org" "o5" {
    name = "org5"
}
resource "cf_org" "o6" {
    name = "org6"
}
resource "cf_org" "o7" {
    name = "org7"
}
resource "cf_org" "o8" {
    name = "org8"
}

resource "cf_user_org_role" "u" {
    user = "${cf_user.u.id}"

	role {
		type = "manager"
		org = "${cf_org.o1.id}"
	}
	role {
		type = "manager"
		org = "${cf_org.o2.id}"
	}
	role {
		type = "billing_manager"
		org = "${cf_org.o3.id}"
	}
	role {
		type = "billing_manager"
		org = "${cf_org.o4.id}"
	}
	role {
		type = "auditor"
		org = "${cf_org.o5.id}"
	}
	role {
		type = "auditor"
		org = "${cf_org.o6.id}"
	}
	role {
		type = "member"
		org = "${cf_org.o7.id}"
	}
	role {
		type = "member"
		org = "${cf_org.o8.id}"
	}
}
`

const userOrgRoleAssocUpdate = `

resource "cf_user" "u" {
    name = "user"
}
resource "cf_org" "o1" {
    name = "org1"
}
resource "cf_org" "o2" {
    name = "org2"
}
resource "cf_org" "o3" {
    name = "org3"
}
resource "cf_org" "o4" {
    name = "org4"
}
resource "cf_org" "o5" {
    name = "org5"
}
resource "cf_org" "o6" {
    name = "org6"
}
resource "cf_org" "o7" {
    name = "org7"
}
resource "cf_org" "o8" {
    name = "org8"
}

resource "cf_user_org_role" "u" {
    user = "${cf_user.u.id}"

	role {
		type = "manager"
		org = "${cf_org.o2.id}"
	}
	role {
		type = "manager"
		org = "${cf_org.o5.id}"
	}
	role {
		type = "billing_manager"
		org = "${cf_org.o3.id}"
	}
	role {
		type = "billing_manager"
		org = "${cf_org.o4.id}"
	}
	role {
		type = "billing_manager"
		org = "${cf_org.o6.id}"
	}
	role {
		type = "member"
		org = "${cf_org.o1.id}"
	}
	role {
		type = "member"
		org = "${cf_org.o7.id}"
	}
	role {
		type = "member"
		org = "${cf_org.o8.id}"
	}
}
`

const userOrgRoleAssocDeleted = `

resource "cf_user" "u" {
    name = "user"
}
resource "cf_org" "o1" {
    name = "org1"
}
resource "cf_org" "o2" {
    name = "org2"
}
resource "cf_org" "o3" {
    name = "org3"
}
resource "cf_org" "o4" {
    name = "org4"
}
resource "cf_org" "o5" {
    name = "org5"
}
resource "cf_org" "o6" {
    name = "org6"
}
resource "cf_org" "o7" {
    name = "org7"
}
resource "cf_org" "o8" {
    name = "org8"
}
`

func TestAccUserOrgRoleAssoc_normal(t *testing.T) {

	ref := "cf_user_org_role.u"
	username := "user"

	resource.Test(t,
		resource.TestCase{
			PreCheck:     func() { testAccPreCheck(t) },
			Providers:    testAccProviders,
			CheckDestroy: testAccCheckUserDestroy(username),
			Steps: []resource.TestStep{

				resource.TestStep{
					Config: userOrgRoleAssoc,
					Check: resource.ComposeTestCheckFunc(
						testAccCheckUserOrgRoleAssoc(ref),
						resource.TestCheckResourceAttr(
							ref, "role.#", "8"),
					),
				},

				resource.TestStep{
					Config: userOrgRoleAssocUpdate,
					Check: resource.ComposeTestCheckFunc(
						testAccCheckUserOrgRoleAssoc(ref),
						resource.TestCheckResourceAttr(
							ref, "role.#", "8"),
					),
				},

				resource.TestStep{
					Config: userOrgRoleAssocDeleted,
					Check: resource.ComposeTestCheckFunc(
						testAccCheckUserOrgRoleAssocDeleted(ref, username),
					),
				},
			},
		})
}

func testAccCheckUserOrgRoleAssoc(resource string) resource.TestCheckFunc {

	return func(s *terraform.State) error {

		session := testAccProvider.Meta().(*cfapi.Session)

		rs, ok := s.RootModule().Resources[resource]
		if !ok {
			return fmt.Errorf("user org role '%s' not found in terraform state", resource)
		}

		session.Log.DebugMessage(
			"terraform state for resource '%s': %# v",
			resource, rs)

		userID := getUserIDFromUORID(rs.Primary.ID)
		attributes := rs.Primary.Attributes

		um := session.UserManager()

		roles := make(map[string]bool)
		for r, t := range userOrgRoleToTypeMap {

			orgIDs, err := um.ListOrgsForUser(userID, r)
			if err != nil {
				return err
			}
			for _, o := range orgIDs {
				roles[t+"/"+o] = true
			}
		}

		if err := assertListEquals(attributes, "role", len(roles),
			func(values map[string]string, i int) (match bool) {

				t := values["type"]
				o := values["org"]

				_, exists := roles[t+"/"+o]
				return exists

			}); err != nil {
			return err
		}
		return nil
	}
}

func testAccCheckUserOrgRoleAssocDeleted(resource string, username string) resource.TestCheckFunc {

	return func(s *terraform.State) error {

		_, ok := s.RootModule().Resources[resource]
		if ok {
			return fmt.Errorf("user org role '%s' still found in terraform state after deletion", resource)
		}

		session := testAccProvider.Meta().(*cfapi.Session)
		um := session.UserManager()

		user, err := um.FindByUsername(username)
		if err != nil {
			return err
		}

		orgIds, err := um.ListOrgsForUser(user.GUID, cfapi.UserIsOrgManager)
		if err != nil {
			return err
		}
		if len(orgIds) > 0 {
			return fmt.Errorf("user with username '%s' still has some org associations", username)
		}
		orgIds, err = um.ListOrgsForUser(user.GUID, cfapi.UserIsOrgBillingManager)
		if err != nil {
			return err
		}
		if len(orgIds) > 0 {
			return fmt.Errorf("user with username '%s' still has some org associations", username)
		}
		orgIds, err = um.ListOrgsForUser(user.GUID, cfapi.UserIsOrgAuditor)
		if err != nil {
			return err
		}
		if len(orgIds) > 0 {
			return fmt.Errorf("user with username '%s' still has some org associations", username)
		}
		orgIds, err = um.ListOrgsForUser(user.GUID, cfapi.UserIsOrgMember)
		if err != nil {
			return err
		}
		if len(orgIds) > 0 {
			return fmt.Errorf("user with username '%s' still has some org associations", username)
		}
		return nil
	}
}
