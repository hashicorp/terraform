package nsone

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	nsone "gopkg.in/ns1/ns1-go.v2/rest"
	"gopkg.in/ns1/ns1-go.v2/rest/model/monitor"
)

func TestAccMonitoringJob_basic(t *testing.T) {
	var mj monitor.Job
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckMonitoringJobDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccMonitoringJobBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMonitoringJobState("name", "terraform test"),
					testAccCheckMonitoringJobExists("nsone_monitoringjob.it", &mj),

					testAccCheckMonitoringJobName(&mj, "terraform test"),
					testAccCheckMonitoringJobActive(&mj, true),
					testAccCheckMonitoringJobRegions(&mj, []string{"lga"}),
					testAccCheckMonitoringJobType(&mj, "tcp"),
					testAccCheckMonitoringJobFrequency(&mj, 60),
					testAccCheckMonitoringJobRapidRecheck(&mj, false),
					testAccCheckMonitoringJobPolicy(&mj, "quorum"),
					testAccCheckMonitoringJobConfigSend(&mj, "HEAD / HTTP/1.0\r\n\r\n"),
					testAccCheckMonitoringJobConfigPort(&mj, 80),
					testAccCheckMonitoringJobConfigHost(&mj, "1.1.1.1"),
				),
			},
		},
	})
}

func TestAccMonitoringJob_updated(t *testing.T) {
	var mj monitor.Job
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckMonitoringJobDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccMonitoringJobBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMonitoringJobState("name", "terraform test"),
					testAccCheckMonitoringJobExists("nsone_monitoringjob.it", &mj),

					testAccCheckMonitoringJobName(&mj, "terraform test"),
					testAccCheckMonitoringJobActive(&mj, true),
					testAccCheckMonitoringJobRegions(&mj, []string{"lga"}),
					testAccCheckMonitoringJobType(&mj, "tcp"),
					testAccCheckMonitoringJobFrequency(&mj, 60),
					testAccCheckMonitoringJobRapidRecheck(&mj, false),
					testAccCheckMonitoringJobPolicy(&mj, "quorum"),
					testAccCheckMonitoringJobConfigSend(&mj, "HEAD / HTTP/1.0\r\n\r\n"),
					testAccCheckMonitoringJobConfigPort(&mj, 80),
					testAccCheckMonitoringJobConfigHost(&mj, "1.1.1.1"),
				),
			},
			resource.TestStep{
				Config: testAccMonitoringJobUpdated,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMonitoringJobState("name", "terraform test"),
					testAccCheckMonitoringJobExists("nsone_monitoringjob.it", &mj),
					testAccCheckMonitoringJobName(&mj, "terraform test"),
					testAccCheckMonitoringJobActive(&mj, true),
					testAccCheckMonitoringJobRegions(&mj, []string{"lga"}),
					testAccCheckMonitoringJobType(&mj, "tcp"),
					testAccCheckMonitoringJobFrequency(&mj, 120),
					testAccCheckMonitoringJobRapidRecheck(&mj, true),
					testAccCheckMonitoringJobPolicy(&mj, "all"),
					testAccCheckMonitoringJobConfigSend(&mj, "HEAD / HTTP/1.0\r\n\r\n"),
					testAccCheckMonitoringJobConfigPort(&mj, 443),
					testAccCheckMonitoringJobConfigHost(&mj, "1.1.1.1"),
				),
			},
		},
	})
}

func testAccCheckMonitoringJobState(key, value string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources["nsone_monitoringjob.it"]
		if !ok {
			return fmt.Errorf("Not found: %s", "nsone_monitoringjob.it")
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

func testAccCheckMonitoringJobExists(n string, monitoringJob *monitor.Job) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Resource not found: %v", n)
		}

		id := rs.Primary.ID
		if id == "" {
			return fmt.Errorf("ID is not set")
		}

		client := testAccProvider.Meta().(*nsone.Client)

		foundMj, _, err := client.Jobs.Get(id)

		if err != nil {
			return err
		}

		if foundMj.ID != id {
			return fmt.Errorf("Monitoring Job not found want: %#v, got %#v", id, foundMj)
		}

		*monitoringJob = *foundMj

		return nil
	}
}

func testAccCheckMonitoringJobDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*nsone.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "nsone_monitoringjob" {
			continue
		}

		mj, _, err := client.Jobs.Get(rs.Primary.Attributes["id"])

		if err == nil {
			return fmt.Errorf("Monitoring Job still exists %#v: %#v", err, mj)
		}
	}

	return nil
}

func testAccCheckMonitoringJobName(mj *monitor.Job, expected string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if mj.Name != expected {
			return fmt.Errorf("Name: got: %#v want: %#v", mj.Name, expected)
		}
		return nil
	}
}

func testAccCheckMonitoringJobActive(mj *monitor.Job, expected bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if mj.Active != expected {
			return fmt.Errorf("Active: got: %#v want: %#v", mj.Active, expected)
		}
		return nil
	}
}

func testAccCheckMonitoringJobRegions(mj *monitor.Job, expected []string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if !reflect.DeepEqual(mj.Regions, expected) {
			return fmt.Errorf("Regions: got: %#v want: %#v", mj.Regions, expected)
		}
		return nil
	}
}

func testAccCheckMonitoringJobType(mj *monitor.Job, expected string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if mj.Type != expected {
			return fmt.Errorf("Type: got: %#v want: %#v", mj.Type, expected)
		}
		return nil
	}
}

func testAccCheckMonitoringJobFrequency(mj *monitor.Job, expected int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if mj.Frequency != expected {
			return fmt.Errorf("Frequency: got: %#v want: %#v", mj.Frequency, expected)
		}
		return nil
	}
}

func testAccCheckMonitoringJobRapidRecheck(mj *monitor.Job, expected bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if mj.RapidRecheck != expected {
			return fmt.Errorf("RapidRecheck: got: %#v want: %#v", mj.RapidRecheck, expected)
		}
		return nil
	}
}

func testAccCheckMonitoringJobPolicy(mj *monitor.Job, expected string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if mj.Policy != expected {
			return fmt.Errorf("Policy: got: %#v want: %#v", mj.Policy, expected)
		}
		return nil
	}
}

func testAccCheckMonitoringJobConfigSend(mj *monitor.Job, expected string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if mj.Config["send"].(string) != expected {
			return fmt.Errorf("Config.send: got: %#v want: %#v", mj.Config["send"].(string), expected)
		}
		return nil
	}
}

func testAccCheckMonitoringJobConfigPort(mj *monitor.Job, expected float64) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if mj.Config["port"].(float64) != expected {
			return fmt.Errorf("Config.port: got: %#v want: %#v", mj.Config["port"].(float64), expected)
		}
		return nil
	}
}

func testAccCheckMonitoringJobConfigHost(mj *monitor.Job, expected string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if mj.Config["host"].(string) != expected {
			return fmt.Errorf("Config.host: got: %#v want: %#v", mj.Config["host"].(string), expected)
		}
		return nil
	}
}

const testAccMonitoringJobBasic = `
resource "nsone_monitoringjob" "it" {
  job_type = "tcp"
  name     = "terraform test"

  regions   = ["lga"]
  frequency = 60

  config {
    send = "HEAD / HTTP/1.0\r\n\r\n"
    port = 80
    host = "1.1.1.1"
  }
}
`

const testAccMonitoringJobUpdated = `
resource "nsone_monitoringjob" "it" {
  job_type = "tcp"
  name     = "terraform test"

  active        = true
  regions       = ["lga"]
  frequency     = 120
  rapid_recheck = true
  policy        = "all"

  config {
    send = "HEAD / HTTP/1.0\r\n\r\n"
    port = 443
    host = "1.1.1.1"
  }
}
`
