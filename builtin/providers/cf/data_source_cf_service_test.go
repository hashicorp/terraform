package cloudfoundry

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"code.cloudfoundry.org/cli/cf/models"

	"github.com/hashicorp/terraform/builtin/providers/cf/cfapi"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

const serviceDataResource = `

data "cf_service" "redis" {
    name = "p-redis"
}
`

func TestAccDataSourceService_normal(t *testing.T) {

	_, filename, _, _ := runtime.Caller(0)
	ut := os.Getenv("UNIT_TEST")
	if !testAccEnvironmentSet() || (len(ut) > 0 && ut != filepath.Base(filename)) {
		fmt.Printf("Skipping tests in '%s'.\n", filepath.Base(filename))
		return
	}

	ref := "data.cf_service.redis"

	resource.Test(t,
		resource.TestCase{
			PreCheck:  func() { testAccPreCheck(t) },
			Providers: testAccProviders,
			Steps: []resource.TestStep{

				resource.TestStep{
					Config: serviceDataResource,
					Check: resource.ComposeTestCheckFunc(
						checkDataSourceServiceExists(ref),
						resource.TestCheckResourceAttr(
							ref, "name", "p-redis"),
					),
				},
			},
		})
}

func checkDataSourceServiceExists(resource string) resource.TestCheckFunc {

	return func(s *terraform.State) error {

		session := testAccProvider.Meta().(*cfapi.Session)

		rs, ok := s.RootModule().Resources[resource]
		if !ok {
			return fmt.Errorf("service '%s' not found in terraform state", resource)
		}

		session.Log.DebugMessage(
			"terraform state for resource '%s': %# v",
			resource, rs)

		id := rs.Primary.ID
		name := rs.Primary.Attributes["name"]

		var (
			err     error
			service models.ServiceOffering
		)

		service, err = session.ServiceManager().FindServiceByName(name)
		if err != nil {
			return err
		}
		if err := assertSame(id, service.GUID); err != nil {
			return err
		}

		return nil
	}
}
