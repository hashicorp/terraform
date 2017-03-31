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

const userProvidedServiceResourceCreate = `

resource "cf_org" "org1" {
	name = "organization-one"
}
resource "cf_quota" "dev" {
	name = "50g"
	org = "${cf_org.org1.id}"
    allow_paid_service_plans = true
    instance_memory = 1024
    total_memory = 51200
    total_app_instances = 100
    total_routes = 100
    total_services = 150

}

resource "cf_space" "space1" {
	name = "space-one"
	org = "${cf_org.org1.id}"
	quota = "${cf_quota.dev.id}"
	
	allow_ssh = true
}

resource "cf_user_provided_service" "mq" {
	name = "mq"
    space = "${cf_space.space1.id}"
    credentials = {
		"url" = "mq://localhost:9000"
		"username" = "user"
		"password" = "pwd"
	}	
}
`

const userProvidedServiceResourceUpdate = `

resource "cf_org" "org1" {
	name = "organization-one"
}
resource "cf_quota" "dev" {
	name = "50g"
	org = "${cf_org.org1.id}"
    allow_paid_service_plans = true
    instance_memory = 1024
    total_memory = 51200
    total_app_instances = 100
    total_routes = 100
    total_services = 150

}

resource "cf_space" "space1" {
	name = "space-one"
	org = "${cf_org.org1.id}"
	quota = "${cf_quota.dev.id}"
	
	allow_ssh = true
}

resource "cf_user_provided_service" "mq" {
	name = "mq"
    space = "${cf_space.space1.id}"
    credentials = {
		"url" = "mq://localhost:9000"
		"username" = "new-user"
		"password" = "new-pwd"
	}
	syslogDrainURL = "http://localhost/syslog"
	routeServiceURL = "https://localhost/route"	
}
`

func TestAccUserProvidedService_normal(t *testing.T) {

	_, filename, _, _ := runtime.Caller(0)
	ut := os.Getenv("UNIT_TEST")
	if !testAccEnvironmentSet() || (len(ut) > 0 && ut != filepath.Base(filename)) {
		fmt.Printf("Skipping tests in '%s'.\n", filepath.Base(filename))
		return
	}

	ref := "cf_user_provided_service.mq"

	resource.Test(t,
		resource.TestCase{
			PreCheck:     func() { testAccPreCheck(t) },
			Providers:    testAccProviders,
			CheckDestroy: testAccCheckUserProvidedServiceDestroyed("mq", "cf_space.space1"),
			Steps: []resource.TestStep{

				resource.TestStep{
					Config: userProvidedServiceResourceCreate,
					Check: resource.ComposeTestCheckFunc(
						testAccCheckUserProvidedServiceExists(ref),
						resource.TestCheckResourceAttr(
							ref, "name", "mq"),
						resource.TestCheckResourceAttr(
							ref, "credentials.url", "mq://localhost:9000"),
						resource.TestCheckResourceAttr(
							ref, "credentials.username", "user"),
						resource.TestCheckResourceAttr(
							ref, "credentials.password", "pwd"),
						resource.TestCheckNoResourceAttr(
							ref, "syslogDrainURL"),
						resource.TestCheckNoResourceAttr(
							ref, "routeServiceURL"),
					),
				},

				resource.TestStep{
					Config: userProvidedServiceResourceUpdate,
					Check: resource.ComposeTestCheckFunc(
						testAccCheckUserProvidedServiceExists(ref),
						resource.TestCheckResourceAttr(
							ref, "name", "mq"),
						resource.TestCheckResourceAttr(
							ref, "credentials.url", "mq://localhost:9000"),
						resource.TestCheckResourceAttr(
							ref, "credentials.username", "new-user"),
						resource.TestCheckResourceAttr(
							ref, "credentials.password", "new-pwd"),
						resource.TestCheckResourceAttr(
							ref, "syslogDrainURL", "http://localhost/syslog"),
						resource.TestCheckResourceAttr(
							ref, "routeServiceURL", "https://localhost/route"),
					),
				},
			},
		})
}

func testAccCheckUserProvidedServiceExists(resource string) resource.TestCheckFunc {

	return func(s *terraform.State) (err error) {

		session := testAccProvider.Meta().(*cfapi.Session)

		rs, ok := s.RootModule().Resources[resource]
		if !ok {
			return fmt.Errorf("user provided service '%s' not found in terraform state", resource)
		}

		session.Log.DebugMessage(
			"terraform state for resource '%s': %# v",
			resource, rs)

		id := rs.Primary.ID

		var (
			serviceInstance cfapi.CCUserProvidedService
		)

		sm := session.ServiceManager()
		if serviceInstance, err = sm.ReadUserProvidedService(id); err != nil {
			return
		}
		session.Log.DebugMessage(
			"retrieved user provided service for resource '%s' with id '%s': %# v",
			resource, id, serviceInstance)

		return
	}
}

func testAccCheckUserProvidedServiceDestroyed(name string, spaceResource string) resource.TestCheckFunc {

	return func(s *terraform.State) error {

		session := testAccProvider.Meta().(*cfapi.Session)

		rs, ok := s.RootModule().Resources[spaceResource]
		if !ok {
			return fmt.Errorf("space '%s' not found in terraform state", spaceResource)
		}

		session.Log.DebugMessage("checking User Provided Service is Destroyed %s", name)

		if _, err := session.ServiceManager().FindServiceInstance(name, rs.Primary.ID); err != nil {
			switch err.(type) {
			case *errors.ModelNotFoundError:
				return nil

			default:
				return err
			}
		}
		return fmt.Errorf("user provided service with name '%s' still exists in cloud foundry", name)
	}
}
