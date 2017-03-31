package cloudfoundry

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"

	"code.cloudfoundry.org/cli/cf/errors"

	"github.com/hashicorp/terraform/builtin/providers/cf/cfapi"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

const quotaResource = `

resource "cf_quota" "50g" {
	name = "50g"
    allow_paid_service_plans = false
    instance_memory = 2048
    total_memory = 51200
    total_app_instances = 100
    total_routes = 50
    total_services = 200
    total_route_ports = 5
}
`

const quotaResourceUpdate = `

resource "cf_quota" "50g" {
	name = "50g"
    allow_paid_service_plans = true
    instance_memory = 1024
    total_memory = 51200
    total_app_instances = 100
    total_routes = 100
    total_services = 150
    total_route_ports = 10
}
`

const spaceQuotaResource = `

resource "cf_quota" "10g" {
	name = "10g"
    allow_paid_service_plans = false
    instance_memory = 512
    total_memory = 10240
    total_app_instances = 10
    total_routes = 5
    total_services = 20
	org = "%s"
}
`

func TestAccQuota_normal(t *testing.T) {

	_, filename, _, _ := runtime.Caller(0)
	ut := os.Getenv("UNIT_TEST")
	if !testAccEnvironmentSet() || (len(ut) > 0 && ut != filepath.Base(filename)) {
		fmt.Printf("Skipping tests in '%s'.\n", filepath.Base(filename))
		return
	}

	ref := "cf_quota.50g"
	quotaname := "50g"

	resource.Test(t,
		resource.TestCase{
			PreCheck:     func() { testAccPreCheck(t) },
			Providers:    testAccProviders,
			CheckDestroy: testAccCheckQuotaResourceDestroy(quotaname),
			Steps: []resource.TestStep{

				resource.TestStep{
					Config: quotaResource,
					Check: resource.ComposeTestCheckFunc(
						checkQuotaExists(ref),
						resource.TestCheckResourceAttr(
							ref, "name", "50g"),
						resource.TestCheckResourceAttr(
							ref, "allow_paid_service_plans", "false"),
						resource.TestCheckResourceAttr(
							ref, "instance_memory", "2048"),
						resource.TestCheckResourceAttr(
							ref, "total_memory", "51200"),
						resource.TestCheckResourceAttr(
							ref, "total_app_instances", "100"),
						resource.TestCheckResourceAttr(
							ref, "total_routes", "50"),
						resource.TestCheckResourceAttr(
							ref, "total_services", "200"),
						resource.TestCheckResourceAttr(
							ref, "total_route_ports", "5"),
					),
				},

				resource.TestStep{
					Config: quotaResourceUpdate,
					Check: resource.ComposeTestCheckFunc(
						checkQuotaExists(ref),
						resource.TestCheckResourceAttr(
							ref, "name", "50g"),
						resource.TestCheckResourceAttr(
							ref, "allow_paid_service_plans", "true"),
						resource.TestCheckResourceAttr(
							ref, "instance_memory", "1024"),
						resource.TestCheckResourceAttr(
							ref, "total_memory", "51200"),
						resource.TestCheckResourceAttr(
							ref, "total_app_instances", "100"),
						resource.TestCheckResourceAttr(
							ref, "total_routes", "100"),
						resource.TestCheckResourceAttr(
							ref, "total_services", "150"),
						resource.TestCheckResourceAttr(
							ref, "total_route_ports", "10"),
					),
				},
			},
		})
}

func TestAccSpaceQuota_normal(t *testing.T) {

	_, filename, _, _ := runtime.Caller(0)
	ut := os.Getenv("UNIT_TEST")
	if !testAccEnvironmentSet() || (len(ut) > 0 && ut != filepath.Base(filename)) {
		fmt.Printf("Skipping tests in '%s'.\n", filepath.Base(filename))
		return
	}

	ref := "cf_quota.10g"
	quotaname := "10g"
	orgID := defaultPcfDevOrgID()

	resource.Test(t,
		resource.TestCase{
			PreCheck:     func() { testAccPreCheck(t) },
			Providers:    testAccProviders,
			CheckDestroy: testAccCheckSpaceQuotaResourceDestroy(quotaname),
			Steps: []resource.TestStep{

				resource.TestStep{
					Config: fmt.Sprintf(spaceQuotaResource, orgID),
					Check: resource.ComposeTestCheckFunc(
						checkQuotaExists(ref),
						resource.TestCheckResourceAttr(
							ref, "name", "10g"),
						resource.TestCheckResourceAttr(
							ref, "allow_paid_service_plans", "false"),
						resource.TestCheckResourceAttr(
							ref, "instance_memory", "512"),
						resource.TestCheckResourceAttr(
							ref, "total_memory", "10240"),
						resource.TestCheckResourceAttr(
							ref, "total_app_instances", "10"),
						resource.TestCheckResourceAttr(
							ref, "total_routes", "5"),
						resource.TestCheckResourceAttr(
							ref, "total_services", "20"),
						resource.TestCheckResourceAttr(
							ref, "org", orgID),
					),
				},
			},
		})
}

func checkQuotaExists(resource string) resource.TestCheckFunc {

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

		var quota cfapi.CCQuota
		if quota, err = session.QuotaManager().ReadQuota(id); err != nil {
			return
		}

		session.Log.DebugMessage(
			"quota detail read from cloud foundry '%s': %# v",
			resource, quota)

		if err := assertEquals(attributes, "name", quota.Name); err != nil {
			return err
		}
		if err := assertEquals(attributes, "org", quota.OrgGUID); err != nil {
			return err
		}
		if err := assertEquals(attributes, "allow_paid_service_plans", strconv.FormatBool(quota.NonBasicServicesAllowed)); err != nil {
			return err
		}
		if err := assertEquals(attributes, "instance_memory", strconv.Itoa(int(quota.InstanceMemoryLimit))); err != nil {
			return err
		}
		if err := assertEquals(attributes, "total_memory", strconv.Itoa(int(quota.MemoryLimit))); err != nil {
			return err
		}
		if err := assertEquals(attributes, "total_app_instances", strconv.Itoa(quota.AppInstanceLimit)); err != nil {
			return err
		}
		if err := assertEquals(attributes, "total_services", strconv.Itoa(quota.TotalServices)); err != nil {
			return err
		}
		if err := assertEquals(attributes, "total_routes", strconv.Itoa(quota.TotalRoutes)); err != nil {
			return err
		}
		if len(quota.OrgGUID) == 0 {
			if err := assertEquals(attributes, "total_route_ports", strconv.Itoa(quota.TotalReserveredPorts)); err != nil {
				return err
			}
		}
		return
	}
}

func testAccCheckQuotaResourceDestroy(quotaname string) resource.TestCheckFunc {

	return func(s *terraform.State) (err error) {

		session := testAccProvider.Meta().(*cfapi.Session)
		if _, err := session.QuotaManager().FindQuota(quotaname); err != nil {
			switch err.(type) {
			case *errors.ModelNotFoundError:
				return nil
			default:
				return err
			}
		}
		return fmt.Errorf("quota with name '%s' still exists in cloud foundry", quotaname)
	}
}

func testAccCheckSpaceQuotaResourceDestroy(quotaname string) resource.TestCheckFunc {

	return func(s *terraform.State) (err error) {

		session := testAccProvider.Meta().(*cfapi.Session)
		if _, err := session.QuotaManager().FindSpaceQuota(quotaname, defaultPcfDevOrgID()); err != nil {
			switch err.(type) {
			case *errors.ModelNotFoundError:
				return nil
			default:
				return err
			}
		}
		return fmt.Errorf("space quota with name '%s' still exists in cloud foundry", quotaname)
	}
}
