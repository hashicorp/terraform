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

const serviceInstanceResourceCreate = `

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

data "cf_service" "redis" {
    name = "p-redis"
}
data "cf_service_plan" "redis" {
    name = "shared-vm"
    service = "${data.cf_service.redis.id}"
}

resource "cf_service_instance" "redis1" {
	name = "redis1"
    space = "${cf_space.space1.id}"
    servicePlan = "${data.cf_service_plan.redis.id}"
	
}
`

const serviceInstanceResourceUpdate = `

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

data "cf_service" "redis" {
    name = "p-redis"
}
data "cf_service_plan" "redis" {
    name = "shared-vm"
    service = "${data.cf_service.redis.id}"
}

resource "cf_service_instance" "redis1" {
	name = "redis-new-name"
    space = "${cf_space.space1.id}"
    servicePlan = "${data.cf_service_plan.redis.id}"
	tags = [ "redis" , "data-grid" ]
}
`

func TestAccServiceInstance_normal(t *testing.T) {

	_, filename, _, _ := runtime.Caller(0)
	ut := os.Getenv("UNIT_TEST")
	if !testAccEnvironmentSet() || (len(ut) > 0 && ut != filepath.Base(filename)) {
		fmt.Printf("Skipping tests in '%s'.\n", filepath.Base(filename))
		return
	}

	ref := "cf_service_instance.redis1"

	resource.Test(t,
		resource.TestCase{
			PreCheck:     func() { testAccPreCheck(t) },
			Providers:    testAccProviders,
			CheckDestroy: testAccCheckServiceInstanceDestroyed("redis1", "cf_space.space1"),
			Steps: []resource.TestStep{

				resource.TestStep{
					Config: serviceInstanceResourceCreate,
					Check: resource.ComposeTestCheckFunc(
						testAccCheckServiceInstanceExists(ref),
						resource.TestCheckResourceAttr(
							ref, "name", "redis1"),
						resource.TestCheckNoResourceAttr(
							ref, "tags"),
					),
				},

				resource.TestStep{
					Config: serviceInstanceResourceUpdate,
					Check: resource.ComposeTestCheckFunc(
						testAccCheckServiceInstanceExists(ref),
						resource.TestCheckResourceAttr(
							ref, "name", "redis-new-name"),
						resource.TestCheckResourceAttr(
							ref, "tags.#", "2"),
						resource.TestCheckResourceAttr(
							ref, "tags.0", "redis"),
						resource.TestCheckResourceAttr(
							ref, "tags.1", "data-grid"),
					),
				},
			},
		})
}

func testAccCheckServiceInstanceExists(resource string) resource.TestCheckFunc {

	return func(s *terraform.State) (err error) {

		session := testAccProvider.Meta().(*cfapi.Session)

		rs, ok := s.RootModule().Resources[resource]
		if !ok {
			return fmt.Errorf("service instance '%s' not found in terraform state", resource)
		}

		session.Log.DebugMessage(
			"terraform state for resource '%s': %# v",
			resource, rs)

		id := rs.Primary.ID

		var (
			serviceInstance cfapi.CCServiceInstance
		)

		sm := session.ServiceManager()
		if serviceInstance, err = sm.ReadServiceInstance(id); err != nil {
			return
		}
		session.Log.DebugMessage(
			"retrieved service instance for resource '%s' with id '%s': %# v",
			resource, id, serviceInstance)

		return
	}
}

func testAccCheckServiceInstanceDestroyed(name string, spaceResource string) resource.TestCheckFunc {

	return func(s *terraform.State) error {

		session := testAccProvider.Meta().(*cfapi.Session)

		rs, ok := s.RootModule().Resources[spaceResource]
		if !ok {
			return fmt.Errorf("space '%s' not found in terraform state", spaceResource)
		}

		session.Log.DebugMessage("checking ServiceInstance is Destroyed %s", name)

		if _, err := session.ServiceManager().FindServiceInstance(name, rs.Primary.ID); err != nil {
			switch err.(type) {
			case *errors.ModelNotFoundError:
				return nil

			default:
				return err
			}
		}
		return fmt.Errorf("service instance with name '%s' still exists in cloud foundry", name)
	}
}
