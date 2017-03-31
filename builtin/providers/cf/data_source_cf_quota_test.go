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

const quotaDataResource = `

resource "cf_quota" "q" {
	name = "50g"
    allow_paid_service_plans = false
    instance_memory = 2048
    total_memory = 51200
    total_app_instances = 100
    total_routes = 50
    total_services = 200
    total_route_ports = 5
}

data "cf_quota" "qq" {
    name = "${cf_quota.q.name}"
}
`

const spaceQuotaDataResource = `

resource "cf_quota" "q" {
	name = "20g"
    allow_paid_service_plans = false
    instance_memory = 512
    total_memory = 10240
    total_app_instances = 10
    total_routes = 5
    total_services = 20
	org = "%s"
}

data "cf_quota" "qq" {
    name = "${cf_quota.q.name}"
	org = "%s"
}
`

func TestAccDataSourceQuota_normal(t *testing.T) {

	_, filename, _, _ := runtime.Caller(0)
	ut := os.Getenv("UNIT_TEST")
	if !testAccEnvironmentSet() || (len(ut) > 0 && ut != filepath.Base(filename)) {
		fmt.Printf("Skipping tests in '%s'.\n", filepath.Base(filename))
		return
	}

	ref := "data.cf_quota.qq"

	resource.Test(t,
		resource.TestCase{
			PreCheck:  func() { testAccPreCheck(t) },
			Providers: testAccProviders,
			Steps: []resource.TestStep{

				resource.TestStep{
					Config: quotaDataResource,
					Check: resource.ComposeTestCheckFunc(
						checkDataSourceQuotaExists(ref),
						resource.TestCheckResourceAttr(
							ref, "name", "50g"),
					),
				},
			},
		})
}

func TestAccDataSourceSpaceQuota_normal(t *testing.T) {

	_, filename, _, _ := runtime.Caller(0)
	ut := os.Getenv("UNIT_TEST")
	if !testAccEnvironmentSet() || (len(ut) > 0 && ut != filepath.Base(filename)) {
		fmt.Printf("Skipping tests in '%s'.\n", filepath.Base(filename))
		return
	}

	ref := "data.cf_quota.qq"
	orgID := defaultPcfDevOrgID()

	resource.Test(t,
		resource.TestCase{
			PreCheck:  func() { testAccPreCheck(t) },
			Providers: testAccProviders,
			Steps: []resource.TestStep{

				resource.TestStep{
					Config: fmt.Sprintf(spaceQuotaDataResource, orgID, orgID),
					Check: resource.ComposeTestCheckFunc(
						checkDataSourceQuotaExists(ref),
						resource.TestCheckResourceAttr(
							ref, "name", "20g"),
						resource.TestCheckResourceAttr(
							ref, "org", orgID),
					),
				},
			},
		})
}

func checkDataSourceQuotaExists(resource string) resource.TestCheckFunc {

	return func(s *terraform.State) error {

		session := testAccProvider.Meta().(*cfapi.Session)

		rs, ok := s.RootModule().Resources[resource]
		if !ok {
			return fmt.Errorf("asg '%s' not found in terraform state", resource)
		}

		session.Log.DebugMessage(
			"terraform state for resource '%s': %# v",
			resource, rs)

		id := rs.Primary.ID
		attributes := rs.Primary.Attributes

		var (
			err   error
			quota cfapi.CCQuota
		)

		quota, err = session.QuotaManager().ReadQuota(id)
		if err != nil {
			return err
		}
		if err := assertEquals(attributes, "org", quota.OrgGUID); err != nil {
			return err
		}
		return nil
	}
}
