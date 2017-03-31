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

const orgDataResource = `

resource "cf_org" "myorg" {
	name = "myorg"
}

data "cf_org" "dd" {
    name = "${cf_org.myorg.name}"
}
`

func TestAccDataSourceOrg_normal(t *testing.T) {

	_, filename, _, _ := runtime.Caller(0)
	ut := os.Getenv("UNIT_TEST")
	if !testAccEnvironmentSet() || (len(ut) > 0 && ut != filepath.Base(filename)) {
		fmt.Printf("Skipping tests in '%s'.\n", filepath.Base(filename))
		return
	}

	ref := "data.cf_org.dd"

	resource.Test(t,
		resource.TestCase{
			PreCheck:  func() { testAccPreCheck(t) },
			Providers: testAccProviders,
			Steps: []resource.TestStep{

				resource.TestStep{
					Config: orgDataResource,
					Check: resource.ComposeTestCheckFunc(
						checkDataSourceOrgExists(ref),
						resource.TestCheckResourceAttr(
							ref, "name", "myorg"),
					),
				},
			},
		})
}

func checkDataSourceOrgExists(resource string) resource.TestCheckFunc {

	return func(s *terraform.State) error {

		session := testAccProvider.Meta().(*cfapi.Session)

		rs, ok := s.RootModule().Resources[resource]
		if !ok {
			return fmt.Errorf("org '%s' not found in terraform state", resource)
		}

		session.Log.DebugMessage(
			"terraform state for resource '%s': %# v",
			resource, rs)

		id := rs.Primary.ID
		name := rs.Primary.Attributes["name"]

		var (
			err error
			org cfapi.CCOrg
		)

		org, err = session.OrgManager().FindOrg(name)
		if err != nil {
			return err
		}
		if err := assertSame(id, org.ID); err != nil {
			return err
		}

		return nil
	}
}
