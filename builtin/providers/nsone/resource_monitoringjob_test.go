package nsone

import (
	"fmt"
	"testing"

	"github.com/bobtfish/go-nsone-api"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccMonitoringJob_basic(t *testing.T) {
	var mj nsone.MonitoringJob
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckMonitoringJobDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccMonitoringJobBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMonitoringJobState("name", "terraform test"),
					testAccCheckMonitoringJobExists("nsone_monitoringjob.foobar", &mj),
					testAccCheckMonitoringJobAttributes(&mj),
				),
			},
		},
	})
}

func TestAccMonitoringJob_updated(t *testing.T) {
	var mj nsone.MonitoringJob
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckMonitoringJobDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccMonitoringJobBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMonitoringJobState("name", "terraform test"),
					testAccCheckMonitoringJobExists("nsone_monitoringjob.foobar", &mj),
					testAccCheckMonitoringJobAttributes(&mj),
				),
			},
			resource.TestStep{
				Config: testAccMonitoringJobUpdated,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMonitoringJobState("name", "terraform test"),
					testAccCheckMonitoringJobExists("nsone_monitoringjob.foobar", &mj),
					testAccCheckMonitoringJobAttributesUpdated(&mj),
				),
			},
		},
	})
}

func testAccCheckMonitoringJobState(key, value string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources["nsone_monitoringjob.foobar"]
		if !ok {
			return fmt.Errorf("Not found: %s", "nsone_monitoringjob.foobar")
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		p := rs.Primary
		if p.Attributes[key] != value {
			return fmt.Errorf(
				"%s != %s (actual: %s)", key, value, p.Attributes[key])
		}

		return nil
	}
}

func testAccCheckMonitoringJobExists(n string, monitoringJob *nsone.MonitoringJob) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("NoID is set")
		}

		client := testAccProvider.Meta().(*nsone.APIClient)

		foundMj, err := client.GetMonitoringJob(rs.Primary.Attributes["id"])

		p := rs.Primary

		if err != nil {
			return err
		}

		if foundMj.Id != p.Attributes["id"] {
			return fmt.Errorf("Monitoring Job not found")
		}

		*monitoringJob = foundMj

		return nil
	}
}

func testAccCheckMonitoringJobDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*nsone.APIClient)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "nsone_monitoringjob" {
			continue
		}

		_, err := client.GetMonitoringJob(rs.Primary.Attributes["id"])

		if err == nil {
			return fmt.Errorf("Record still exists")
		}
	}

	return nil
}

func testAccCheckMonitoringJobAttributes(mj *nsone.MonitoringJob) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if mj.Frequency != 60 {
			return fmt.Errorf("Bad value : %d", mj.Frequency)
		}

		if mj.RapidRecheck != true {
			return fmt.Errorf("Bad value : %s", mj.RapidRecheck)
		}

		if mj.Policy != "all" {
			return fmt.Errorf("Bad value : %s", mj.Policy)
		}

		if mj.Config["port"].(float64) != 80 {
			return fmt.Errorf("Bad value : %b", mj.Config["port"].(float64))
		}

		return nil
	}
}

func testAccCheckMonitoringJobAttributesUpdated(mj *nsone.MonitoringJob) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if mj.Frequency != 120 {
			return fmt.Errorf("Bad value : %d", mj.Frequency)
		}

		if mj.RapidRecheck != false {
			return fmt.Errorf("Bad value : %s", mj.RapidRecheck)
		}

		if mj.Policy != "quorum" {
			return fmt.Errorf("Bad value : %s", mj.Policy)
		}

		if mj.Config["port"].(float64) != 443 {
			return fmt.Errorf("Bad value : %b", mj.Config["port"].(float64))
		}

		return nil
	}
}

const testAccMonitoringJobBasic = `
resource "nsone_monitoringjob" "foobar" {
  name = "terraform test"
  active = true
  regions = [ "lga" ]
  job_type = "tcp"
  frequency = 60
  rapid_recheck = true
  policy = "all"
  config {
    send = "HEAD / HTTP/1.0\r\n\r\n"
    port = 80
    host = "1.1.1.1"
  }
}`

const testAccMonitoringJobUpdated = `
resource "nsone_monitoringjob" "foobar" {
	name = "terraform test"
	active = true
	regions = [ "lga" ]
	job_type = "tcp"
	frequency = 120
	rapid_recheck = false
	policy = "quorum"
	config {
		send = "HEAD / HTTP/1.0\r\n\r\n"
		port = 443
		host = "1.1.1.1"
	}
}`
