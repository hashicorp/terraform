package cloudfoundry

import (
	"fmt"
	"testing"

	"code.cloudfoundry.org/cli/cf/errors"

	"github.com/hashicorp/terraform/builtin/providers/cf/cfapi"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

const sbResource = `

resource "cf_service_broker" "mysql" {
	name = "test-mysql"
	url = "http://mysql-broker.local.pcfdev.io"
	username = "admin"
	password = "admin"
}
`

const sbResourceUpdate = `

resource "cf_service_broker" "mysql" {
	name = "test-mysql-renamed"
	url = "http://mysql-broker.local.pcfdev.io"
	username = "admin"
	password = "admin"
}
`

func TestAccServiceBroker_normal(t *testing.T) {

	deleteMySQLServiceBroker("p-mysql")

	ref := "cf_service_broker.mysql"

	resource.Test(t,
		resource.TestCase{
			PreCheck:     func() { testAccPreCheck(t) },
			Providers:    testAccProviders,
			CheckDestroy: testAccCheckServiceBrokerDestroyed("test-mysql"),
			Steps: []resource.TestStep{

				resource.TestStep{
					Config: sbResource,
					Check: resource.ComposeTestCheckFunc(
						testAccCheckServiceBrokerExists(ref),
						resource.TestCheckResourceAttr(
							ref, "name", "test-mysql"),
						resource.TestCheckResourceAttr(
							ref, "url", "http://mysql-broker.local.pcfdev.io"),
						resource.TestCheckResourceAttr(
							ref, "username", "admin"),
						resource.TestCheckResourceAttrSet(
							ref, "service_plans.p-mysql/512mb"),
						resource.TestCheckResourceAttrSet(
							ref, "service_plans.p-mysql/1gb"),
					),
				},

				resource.TestStep{
					Config: sbResourceUpdate,
					Check: resource.ComposeTestCheckFunc(
						testAccCheckServiceBrokerExists(ref),
						resource.TestCheckResourceAttr(
							ref, "name", "test-mysql-renamed"),
					),
				},
			},
		})
}

func testAccCheckServiceBrokerExists(resource string) resource.TestCheckFunc {

	return func(s *terraform.State) (err error) {

		session := testAccProvider.Meta().(*cfapi.Session)

		rs, ok := s.RootModule().Resources[resource]
		if !ok {
			return fmt.Errorf("service broker '%s' not found in terraform state", resource)
		}

		session.Log.DebugMessage(
			"terraform state for resource '%s': %# v",
			resource, rs)

		id := rs.Primary.ID
		attributes := rs.Primary.Attributes

		var (
			serviceBroker cfapi.CCServiceBroker
		)

		sm := session.ServiceManager()
		if serviceBroker, err = sm.ReadServiceBroker(id); err != nil {
			return
		}

		if err := assertEquals(attributes, "name", serviceBroker.Name); err != nil {
			return err
		}
		if err := assertEquals(attributes, "url", serviceBroker.BrokerURL); err != nil {
			return err
		}
		if err := assertEquals(attributes, "username", serviceBroker.AuthUserName); err != nil {
			return err
		}

		return
	}
}

func testAccCheckServiceBrokerDestroyed(name string) resource.TestCheckFunc {

	return func(s *terraform.State) error {

		session := testAccProvider.Meta().(*cfapi.Session)
		if _, err := session.ServiceManager().GetServiceBrokerID(name); err != nil {
			switch err.(type) {
			case *errors.ModelNotFoundError:
				return nil
			default:
				return err
			}
		}

		return fmt.Errorf("service broker with name '%s' still exists in cloud foundry", name)
	}
}
