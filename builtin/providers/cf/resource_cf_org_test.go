package cloudfoundry

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"code.cloudfoundry.org/cli/cf/errors"

	"github.com/hashicorp/terraform/builtin/providers/cf/cfapi"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

const orgResource = `

data "cf_quota" "default" {
    name = "default"
}
resource "cf_quota" "runaway" {
	name = "runaway"
    allow_paid_service_plans = true
    instance_memory = -1
    total_app_instances = -1
    total_memory = 204800
    total_routes = 2000
    total_services = -1
    total_route_ports = 0
}
resource "cf_user" "manager1" {
    name = "manager1@acme.com"
}
resource "cf_user" "dev1" {
    name = "developer1@acme.com"
}
resource "cf_user" "dev2" {
    name = "developer2@acme.com"
}
resource "cf_user" "auditor1" {
    name = "auditor1@acme.com"
}
resource "cf_user" "auditor2" {
    name = "auditor2@acme.com"
}
resource "cf_user" "auditor3" {
    name = "auditor3@acme.com"
}

resource "cf_org" "org1" {

	name = "organization-one"

	members = [
        "${cf_user.dev1.id}",
        "${cf_user.dev2.id}" 		
	]
    managers = [ 
        "${cf_user.manager1.id}" 
    ]
    auditors = [ 
        "${cf_user.auditor1.id}",
		"${cf_user.auditor2.id}" 
    ]

    quota = "${cf_quota.runaway.id}"
}
`

const orgResourceUpdate = `

data "cf_quota" "default" {
    name = "default"
}
resource "cf_quota" "runaway" {
	name = "runaway"
    allow_paid_service_plans = true
    instance_memory = -1
    total_app_instances = -1
    total_memory = 204800
    total_routes = 2000
    total_services = -1
    total_route_ports = 0
}
resource "cf_user" "manager1" {
    name = "manager1@acme.com"
}
resource "cf_user" "dev1" {
    name = "developer1@acme.com"
}
resource "cf_user" "dev2" {
    name = "developer2@acme.com"
}
resource "cf_user" "auditor1" {
    name = "auditor1@acme.com"
}
resource "cf_user" "auditor2" {
    name = "auditor2@acme.com"
}
resource "cf_user" "auditor3" {
    name = "auditor3@acme.com"
}

resource "cf_org" "org1" {

	name = "organization-one-updated"

	members = [
        "${cf_user.dev2.id}" 		
	]
    managers = [ 
        "${cf_user.manager1.id}" 
    ]
    auditors = [ 
        "${cf_user.auditor2.id}",
		"${cf_user.auditor3.id}" 
    ]

    quota = "${data.cf_quota.default.id}"
}
`

func TestAccOrg_normal(t *testing.T) {

	_, filename, _, _ := runtime.Caller(0)
	ut := os.Getenv("UNIT_TEST")
	if !testAccEnvironmentSet() || (ut != "" && ut != filepath.Base(filename)) {
		fmt.Printf("Skipping tests in '%s'.\n", filepath.Base(filename))
		return
	}

	ref := "cf_org.org1"

	resource.Test(t,
		resource.TestCase{
			PreCheck:     func() { testAccPreCheck(t) },
			Providers:    testAccProviders,
			CheckDestroy: testAccCheckOrgDestroyed("organization-one-updated"),
			Steps: []resource.TestStep{

				resource.TestStep{
					Config: orgResource,
					Check: resource.ComposeTestCheckFunc(
						testAccCheckOrgExists(ref),
						resource.TestCheckResourceAttr(
							ref, "name", "organization-one"),
						resource.TestCheckResourceAttr(
							ref, "managers.#", "1"),
						resource.TestCheckResourceAttr(
							ref, "auditors.#", "2"),
					),
				},

				resource.TestStep{
					Config: orgResourceUpdate,
					Check: resource.ComposeTestCheckFunc(
						testAccCheckOrgExists(ref),
						resource.TestCheckResourceAttr(
							ref, "name", "organization-one-updated"),
						resource.TestCheckResourceAttr(
							ref, "managers.#", "1"),
						resource.TestCheckResourceAttr(
							ref, "auditors.#", "2"),
					),
				},
			},
		})
}

func testAccCheckOrgExists(resource string) resource.TestCheckFunc {

	return func(s *terraform.State) (err error) {

		session := testAccProvider.Meta().(*cfapi.Session)

		rs, ok := s.RootModule().Resources[resource]
		if !ok {
			return fmt.Errorf("quota '%s' not found in terraform state", resource)
		}

		session.Log.DebugMessage(
			"terraform state for resource '%s': %# v",
			resource, rs)

		id := rs.Primary.ID
		attributes := rs.Primary.Attributes

		var (
			org cfapi.CCOrg

			members, managers, billingManagers, auditors []interface{}
		)

		om := session.OrgManager()
		if org, err = om.ReadOrg(id); err != nil {
			return
		}
		session.Log.DebugMessage(
			"retrieved org for resource '%s' with id '%s': %# v",
			resource, id, org)

		if err := assertEquals(attributes, "name", org.Name); err != nil {
			return err
		}
		if err := assertEquals(attributes, "quota", org.QuotaGUID); err != nil {
			return err
		}

		if members, err = om.ListUsers(id, cfapi.OrgRoleMember); err != nil {
			return
		}
		session.Log.DebugMessage(
			"retrieved members of org identified resource '%s': %# v",
			resource, members)

		if err := assertSetEquals(attributes, "members", members); err != nil {
			return err
		}

		if managers, err = om.ListUsers(id, cfapi.OrgRoleManager); err != nil {
			return
		}
		session.Log.DebugMessage(
			"retrieved managers of org identified resource '%s': %# v",
			resource, managers)

		if err := assertSetEquals(attributes, "managers", managers); err != nil {
			return err
		}

		if billingManagers, err = om.ListUsers(id, cfapi.OrgRoleBillingManager); err != nil {
			return
		}
		session.Log.DebugMessage(
			"retrieved billing managers of org identified resource '%s': %# v",
			resource, billingManagers)

		if err := assertSetEquals(attributes, "billing_managers", billingManagers); err != nil {
			return err
		}

		if auditors, err = om.ListUsers(id, cfapi.OrgRoleAuditor); err != nil {
			return
		}
		session.Log.DebugMessage(
			"retrieved managers of org identified resource '%s': %# v",
			resource, auditors)

		if err := assertSetEquals(attributes, "auditors", auditors); err != nil {
			return err
		}

		return
	}
}

func testAccCheckOrgDestroyed(orgname string) resource.TestCheckFunc {

	return func(s *terraform.State) error {

		session := testAccProvider.Meta().(*cfapi.Session)
		if _, err := session.OrgManager().FindOrg(orgname); err != nil {
			switch err.(type) {
			case *errors.ModelNotFoundError:
				return nil
			default:
				return err
			}
		}
		return fmt.Errorf("org with name '%s' still exists in cloud foundry", orgname)
	}
}
