package cloudfoundry

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/hashicorp/terraform/builtin/providers/cf/cfapi"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

const spaceDataResource = `

resource "cf_org" "org1" {
	name = "organization-one"
	quota = "${cf_quota.dev.id}"
}
resource "cf_quota" "dev" {
	name = "50g"
    allow_paid_service_plans = true
    instance_memory = 1024
    total_memory = 51200
    total_app_instances = 100
    total_routes = 100
	total_services = 150
	total_route_ports = 5
}
resource "cf_quota" "dev-space" {
	name = "50g"
    allow_paid_service_plans = true
    instance_memory = 1024
    total_memory = 51200
    total_app_instances = 100
    total_routes = 100
	total_services = 150
	org = "${cf_org.org1.id}"
}	

resource "cf_space" "space1" {
	name = "space-one"
	org = "${cf_org.org1.id}"
	quota = "${cf_quota.dev-space.id}"
}

data "cf_space" "myspace" {
    name = "${cf_space.space1.name}"
	org = "${cf_org.org1.id}"
}
`

func TestAccDataSourceSpace_normal(t *testing.T) {

	_, filename, _, _ := runtime.Caller(0)
	ut := os.Getenv("UNIT_TEST")
	if !testAccEnvironmentSet() || (len(ut) > 0 && ut != filepath.Base(filename)) {
		fmt.Printf("Skipping tests in '%s'.\n", filepath.Base(filename))
		return
	}

	ref := "data.cf_space.myspace"

	resource.Test(t,
		resource.TestCase{
			PreCheck:  func() { testAccPreCheck(t) },
			Providers: testAccProviders,
			Steps: []resource.TestStep{

				resource.TestStep{
					Config: spaceDataResource,
					Check: resource.ComposeTestCheckFunc(
						checkDataSourceSpaceExists(ref),
						resource.TestCheckResourceAttr(
							ref, "name", "space-one"),
					),
				},
			},
		})
}

func checkDataSourceSpaceExists(resource string) resource.TestCheckFunc {

	return func(s *terraform.State) error {

		session := testAccProvider.Meta().(*cfapi.Session)

		rs, ok := s.RootModule().Resources[resource]
		if !ok {
			return fmt.Errorf("space '%s' not found in terraform state", resource)
		}

		session.Log.DebugMessage(
			"terraform state for resource '%s': %# v",
			resource, rs)

		id := rs.Primary.ID
		name := rs.Primary.Attributes["name"]
		org := rs.Primary.Attributes["org"]

		var (
			err   error
			space cfapi.CCSpace
		)

		space, err = session.SpaceManager().FindSpaceInOrg(name, org)
		if err != nil {
			return err
		}
		if err := assertSame(id, space.ID); err != nil {
			return err
		}

		return nil
	}
}
