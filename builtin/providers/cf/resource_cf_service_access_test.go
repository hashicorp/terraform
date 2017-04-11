package cloudfoundry

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/builtin/providers/cf/cfapi"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

const saResource = `

resource "cf_service_broker" "mysql" {
	name = "test-mysql"
	url = "http://mysql-broker.local.pcfdev.io"
	username = "admin"
	password = "admin"
}

resource "cf_service_access" "mysql-access" {
	plan = "${cf_service_broker.mysql.service_plans["p-mysql/512mb"]}"
	org = "%s"
}
`

const saResourceUpdated = `

resource "cf_service_broker" "mysql" {
	name = "test-mysql"
	url = "http://mysql-broker.local.pcfdev.io"
	username = "admin"
	password = "admin"
}

resource "cf_service_access" "mysql-access" {
	plan = "${cf_service_broker.mysql.service_plans["p-mysql/1gb"]}"
	org = "%s"
}
`

func TestAccServiceAccess_normal(t *testing.T) {

	deleteMySQLServiceBroker("p-redis")

	var servicePlanAccessGUID string
	ref := "cf_service_access.mysql-access"

	resource.Test(t,
		resource.TestCase{
			PreCheck:     func() { testAccPreCheck(t) },
			Providers:    testAccProviders,
			CheckDestroy: testAccCheckServiceAccessDestroyed(servicePlanAccessGUID),
			Steps: []resource.TestStep{

				resource.TestStep{
					Config: fmt.Sprintf(saResource, defaultPcfDevOrgID()),
					Check: resource.ComposeTestCheckFunc(
						testAccCheckServiceAccessExists(ref,
							func(guid string) {
								servicePlanAccessGUID = guid
							}),
						resource.TestCheckResourceAttrSet(
							ref, "plan"),
						resource.TestCheckResourceAttr(
							ref, "org", defaultPcfDevOrgID()),
					),
				},

				resource.TestStep{
					Config: fmt.Sprintf(saResourceUpdated, defaultPcfDevOrgID()),
					Check: resource.ComposeTestCheckFunc(
						testAccCheckServiceAccessExists(ref,
							func(guid string) {
								servicePlanAccessGUID = guid
							}),
						resource.TestCheckResourceAttrSet(
							ref, "plan"),
						resource.TestCheckResourceAttr(
							ref, "org", defaultPcfDevOrgID()),
					),
				},
			},
		})
}

func testAccCheckServiceAccessExists(resource string,
	setServicePlanAccessGUID func(string)) resource.TestCheckFunc {

	return func(s *terraform.State) (err error) {

		session := testAccProvider.Meta().(*cfapi.Session)
		sm := session.ServiceManager()

		rs, ok := s.RootModule().Resources[resource]
		if !ok {
			return fmt.Errorf("service access resource '%s' not found in terraform state", rs)
		}

		id := rs.Primary.ID
		attributes := rs.Primary.Attributes

		setServicePlanAccessGUID(id)

		plan, org, err := sm.ReadServicePlanAccess(id)
		if err != nil {
			return err
		}
		if err := assertEquals(attributes, "plan", plan); err != nil {
			return err
		}
		if err := assertEquals(attributes, "org", org); err != nil {
			return err
		}

		return
	}
}

func testAccCheckServiceAccessDestroyed(servicePlanAccessGUID string) resource.TestCheckFunc {

	return func(s *terraform.State) error {

		session := testAccProvider.Meta().(*cfapi.Session)

		_, _, err := session.ServiceManager().ReadServicePlanAccess(servicePlanAccessGUID)
		if err == nil {
			return fmt.Errorf("service plan access with guid '%s' still exists in cloud foundry", servicePlanAccessGUID)
		}
		return nil
	}
}
