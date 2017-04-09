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
resource "cf_org" "org1" {

	name = "organization-one"
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
resource "cf_org" "org1" {

	name = "organization-one-updated"
    quota = "${data.cf_quota.default.id}"
}
`

func TestAccOrg_normal(t *testing.T) {

	_, filename, _, _ := runtime.Caller(0)
	ut := os.Getenv("UNIT_TEST")
	if !testAccEnvironmentSet() || (len(ut) > 0 && ut != filepath.Base(filename)) {
		fmt.Printf("Skipping tests in '%s'.\n", filepath.Base(filename))
		return
	}

	refOrg := "cf_org.org1"
	refQuotaRunway := "cf_quota.runaway"
	refQuotaDefault := "data.cf_quota.default"

	resource.Test(t,
		resource.TestCase{
			PreCheck:     func() { testAccPreCheck(t) },
			Providers:    testAccProviders,
			CheckDestroy: testAccCheckOrgDestroyed("organization-one-updated"),
			Steps: []resource.TestStep{

				resource.TestStep{
					Config: orgResource,
					Check: resource.ComposeTestCheckFunc(
						testAccCheckOrgExists(refOrg, refQuotaRunway),
						resource.TestCheckResourceAttr(
							refOrg, "name", "organization-one"),
					),
				},

				resource.TestStep{
					Config: orgResourceUpdate,
					Check: resource.ComposeTestCheckFunc(
						testAccCheckOrgExists(refOrg, refQuotaDefault),
						resource.TestCheckResourceAttr(
							refOrg, "name", "organization-one-updated"),
					),
				},
			},
		})
}

func testAccCheckOrgExists(resOrg, resQuota string) resource.TestCheckFunc {

	return func(s *terraform.State) (err error) {

		session := testAccProvider.Meta().(*cfapi.Session)

		rs, ok := s.RootModule().Resources[resOrg]
		if !ok {
			return fmt.Errorf("org '%s' not found in terraform state", resOrg)
		}

		session.Log.DebugMessage(
			"terraform state for resource '%s': %# v",
			resOrg, rs)

		id := rs.Primary.ID
		attributes := rs.Primary.Attributes

		var org cfapi.CCOrg
		om := session.OrgManager()
		if org, err = om.ReadOrg(id); err != nil {
			return
		}
		session.Log.DebugMessage(
			"retrieved org for resource '%s' with id '%s': %# v",
			resOrg, id, org)

		if err := assertEquals(attributes, "name", org.Name); err != nil {
			return err
		}
		if err := assertEquals(attributes, "quota", org.QuotaGUID); err != nil {
			return err
		}

		rs, ok = s.RootModule().Resources[resQuota]
		if org.QuotaGUID != rs.Primary.ID {
			return fmt.Errorf("expected org '%s' to be associated with quota '%s' but it was not", resOrg, resQuota)
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
