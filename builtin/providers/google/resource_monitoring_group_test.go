package google

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"google.golang.org/api/monitoring/v3"
)

func TestAccMonitoringGroup_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckMonitoringGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccMonitoringGroup_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccMonitoringGroupExists(
						"google_monitoring_group.foobar"),
				),
			},
		},
	})
}

func TestAccMonitoringGroup_update(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckMonitoringGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccMonitoringGroup_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccMonitoringGroupExists(
						"google_monitoring_group.foobar"),
				),
			},
			resource.TestStep{
				Config: testAccMonitoringGroup_update,
				Check: resource.ComposeTestCheckFunc(
					testAccMonitoringGroupExists(
						"google_monitoring_group.foobar"),
				),
			},
		},
	})
}

func TestAccMonitoringGroupCreate(t *testing.T) {

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckMonitoringGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccMonitoringGroup_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccMonitoringGroupExists(
						"google_monitoring_group.foobar"),
				),
			},
		},
	})
}

func testAccCheckMonitoringGroupDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "google_monitoring_group" {
			continue
		}

		config := testAccProvider.Meta().(*Config)
		group, _ := config.clientMonitoring.Projects.Groups.Get(rs.Primary.ID).Do()
		if group != nil {
			return fmt.Errorf("Group still present")
		}
	}

	return nil
}

func testAccMonitoringGroupExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}
		config := testAccProvider.Meta().(*Config)
		_, err := config.clientMonitoring.Projects.Groups.Get(rs.Primary.ID).Do()
		if err != nil {
			return fmt.Errorf("Group does not exist")
		}

		return nil
	}
}

var testAccMonitoringGroup_basic = `
resource "google_monitoring_group" "foobar" {
	name = "test"
	filter = "resource.metadata.name=starts_with(\"test-\")"
}`

var testAccMonitoringGroup_update = `
resource "google_monitoring_group" "foobar" {
	name = "test"
	filter = "resource.metadata.name=has_substring(\"test\")"
}`
