package ns1

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	ns1 "gopkg.in/ns1/ns1-go.v2/rest"
	"gopkg.in/ns1/ns1-go.v2/rest/model/monitor"
)

func TestAccNS1Monitor_Basic(t *testing.T) {
	var mj monitor.Job

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckNS1MonitorDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccNS1Monitor_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNS1MonitorExists("ns1_monitor.foobar", &mj),
					testAccCheckNS1MonitorAttributes(&mj),
					resource.TestCheckResourceAttr("ns1_monitor.foobar", "name", "terraform TCP test"),
					resource.TestCheckResourceAttr("ns1_monitor.foobar", "active", "true"),
					resource.TestCheckResourceAttr("ns1_monitor.foobar", "regions.#", "1"),
					resource.TestCheckResourceAttr("ns1_monitor.foobar", "regions.0", "lga"),
					resource.TestCheckResourceAttr("ns1_monitor.foobar", "type", "tcp"),
					resource.TestCheckResourceAttr("ns1_monitor.foobar", "frequency", "60"),
					resource.TestCheckResourceAttr("ns1_monitor.foobar", "rapid_recheck", "true"),
					resource.TestCheckResourceAttr("ns1_monitor.foobar", "policy", "all"),
					resource.TestCheckResourceAttr("ns1_monitor.foobar", "config.send", "HEAD / HTTP/1.0\r\n\r\n"),
					resource.TestCheckResourceAttr("ns1_monitor.foobar", "config.port", "80"),
					resource.TestCheckResourceAttr("ns1_monitor.foobar", "config.host", "1.1.1.1"),
					resource.TestCheckResourceAttr("ns1_monitor.foobar", "config.ssl", "1"),
					resource.TestCheckResourceAttr("ns1_monitor.foobar", "rules.#", "1"),
					resource.TestCheckResourceAttr("ns1_monitor.foobar", "rules.0.key", "output"),
					resource.TestCheckResourceAttr("ns1_monitor.foobar", "rules.0.value", "200 OK"),
					resource.TestCheckResourceAttr("ns1_monitor.foobar", "rules.0.comparison", "contains"),
				),
			},
		},
	})
}

func TestAccNS1Monitor_Updated(t *testing.T) {
	var mj monitor.Job

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckNS1MonitorDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccNS1Monitor_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNS1MonitorExists("ns1_monitor.foobar", &mj),
					testAccCheckNS1MonitorAttributes(&mj),
				),
			},
			resource.TestStep{
				Config: testAccNS1Monitor_updated,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNS1MonitorExists("ns1_monitor.foobar", &mj),
					testAccCheckNS1MonitorAttributesUpdated(&mj),
					resource.TestCheckResourceAttr("ns1_monitor.foobar", "name", "terraform TCP test"),
					resource.TestCheckResourceAttr("ns1_monitor.foobar", "active", "false"),
					resource.TestCheckResourceAttr("ns1_monitor.foobar", "regions.#", "1"),
					resource.TestCheckResourceAttr("ns1_monitor.foobar", "regions.0", "sjc"),
					resource.TestCheckResourceAttr("ns1_monitor.foobar", "type", "tcp"),
					resource.TestCheckResourceAttr("ns1_monitor.foobar", "frequency", "120"),
					resource.TestCheckResourceAttr("ns1_monitor.foobar", "rapid_recheck", "false"),
					resource.TestCheckResourceAttr("ns1_monitor.foobar", "policy", "quorum"),
					resource.TestCheckResourceAttr("ns1_monitor.foobar", "config.send", "HEAD / HTTP/1.0\r\n\r\n"),
					resource.TestCheckResourceAttr("ns1_monitor.foobar", "config.port", "443"),
					resource.TestCheckResourceAttr("ns1_monitor.foobar", "config.host", "2.2.2.2"),
					resource.TestCheckResourceAttr("ns1_monitor.foobar", "config.ssl", "0"),
					resource.TestCheckResourceAttr("ns1_monitor.foobar", "rules.#", "1"),
					resource.TestCheckResourceAttr("ns1_monitor.foobar", "rules.0.key", "output"),
					resource.TestCheckResourceAttr("ns1_monitor.foobar", "rules.0.value", "200 OK"),
					resource.TestCheckResourceAttr("ns1_monitor.foobar", "rules.0.comparison", "contains"),
				),
			},
		},
	})
}

func testAccCheckNS1MonitorExists(n string, j *monitor.Job) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("NoID is set")
		}

		client := testAccProvider.Meta().(*ns1.Client)

		foundMj, _, err := client.Jobs.Get(rs.Primary.Attributes["id"])
		if err != nil {
			return err
		}

		if foundMj.ID != rs.Primary.Attributes["id"] {
			return fmt.Errorf("Monitoring Job not found")
		}

		*j = *foundMj

		return nil
	}
}

func testAccCheckNS1MonitorDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*ns1.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "ns1_monitor" {
			continue
		}

		mj, _, err := client.Jobs.Get(rs.Primary.Attributes["id"])

		if err == nil {
			return fmt.Errorf("Monitor Job still exists %#v: %#v", err, mj)
		}
	}

	return nil
}

func testAccCheckNS1MonitorAttributes(mj *monitor.Job) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if mj.Name != "terraform TCP test" {
			return fmt.Errorf("Bad value mj.Name: %s", mj.Name)
		}

		if !mj.Active {
			return fmt.Errorf("Bad value mj.Active: %b", mj.Active)
		}

		if len(mj.Regions) != 1 {
			return fmt.Errorf("Bad number of mj.Regions: %d", len(mj.Regions))
		}

		if mj.Regions[0] != "lga" {
			return fmt.Errorf("Bad value mj.Regions[0]: %s", len(mj.Regions[0]))
		}

		if mj.Type != "tcp" {
			return fmt.Errorf("Bad value mj.Type: %s", mj.Type)
		}

		if mj.Frequency != 60 {
			return fmt.Errorf("Bad value mj.Frequency: %d", mj.Frequency)
		}

		if !mj.RapidRecheck {
			return fmt.Errorf("Bad value mj.RapidRecheck: %s", mj.RapidRecheck)
		}

		if mj.Policy != "all" {
			return fmt.Errorf("Bad value mj.Policy: %s", mj.Policy)
		}

		if mj.Config["send"].(string) != "HEAD / HTTP/1.0\r\n\r\n" {
			return fmt.Errorf("Bad value mj.Config['send']: %s", mj.Config["send"].(string))
		}

		if mj.Config["port"].(string) != "80" {
			return fmt.Errorf("Bad value mj.Config['port']: %s", mj.Config["port"].(string))
		}

		if mj.Config["host"].(string) != "1.1.1.1" {
			return fmt.Errorf("Bad value mj.Config['host']: %s", mj.Config["host"].(string))
		}

		if mj.Config["ssl"].(string) != "1" {
			return fmt.Errorf("Bad value mj.Config['ssl']: %s", mj.Config["ssl"].(string))
		}

		if len(mj.Rules) != 1 {
			return fmt.Errorf("Bad number of mj.Rules: %d", len(mj.Rules))
		}

		if mj.Rules[0].Key != "output" {
			return fmt.Errorf("Bad value mj.Rules[0].Key: %s", mj.Rules[0].Key)
		}

		if mj.Rules[0].Value.(string) != "200 OK" {
			return fmt.Errorf("Bad value mj.Rules[0].Value: %s", mj.Rules[0].Value.(string))
		}

		if mj.Rules[0].Comparison != "contains" {
			return fmt.Errorf("Bad value mj.Rules[0].Comparison: %s", mj.Rules[0].Comparison)
		}

		return nil
	}
}

func testAccCheckNS1MonitorAttributesUpdated(mj *monitor.Job) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if mj.Name != "terraform TCP test" {
			return fmt.Errorf("Bad value mj.Name: %s", mj.Name)
		}

		if mj.Active {
			return fmt.Errorf("Bad value mj.Active: %b", mj.Active)
		}

		if len(mj.Regions) != 1 {
			return fmt.Errorf("Bad number of mj.Regions: %d", len(mj.Regions))
		}

		if mj.Regions[0] != "sjc" {
			return fmt.Errorf("Bad value mj.Regions[0]: %s", len(mj.Regions[0]))
		}

		if mj.Type != "tcp" {
			return fmt.Errorf("Bad value mj.Type: %s", mj.Type)
		}

		if mj.Frequency != 120 {
			return fmt.Errorf("Bad value mj.Frequency: %d", mj.Frequency)
		}

		if mj.RapidRecheck {
			return fmt.Errorf("Bad value mj.RapidRecheck: %s", mj.RapidRecheck)
		}

		if mj.Policy != "quorum" {
			return fmt.Errorf("Bad value mj.Policy: %s", mj.Policy)
		}

		if mj.Config["send"].(string) != "HEAD / HTTP/1.0\r\n\r\n" {
			return fmt.Errorf("Bad value mj.Config['send']: %s", mj.Config["send"].(string))
		}

		if mj.Config["port"].(string) != "443" {
			return fmt.Errorf("Bad value mj.Config['port']: %s", mj.Config["port"].(string))
		}

		if mj.Config["host"].(string) != "2.2.2.2" {
			return fmt.Errorf("Bad value mj.Config['host']: %s", mj.Config["host"].(string))
		}

		if mj.Config["response_timeout"].(string) != "60000" {
			return fmt.Errorf("Bad value mj.Config['response_timeout']: %s", mj.Config["response_timeout"].(string))
		}

		if mj.Config["ssl"].(string) != "0" {
			return fmt.Errorf("Bad value mj.Config['ssl']: %s", mj.Config["ssl"].(string))
		}

		if len(mj.Rules) != 1 {
			return fmt.Errorf("Bad number of mj.Rules: %d", len(mj.Rules))
		}

		if mj.Rules[0].Key != "output" {
			return fmt.Errorf("Bad value mj.Rules[0].Key: %s", mj.Rules[0].Key)
		}

		if mj.Rules[0].Value.(string) != "200 OK" {
			return fmt.Errorf("Bad value mj.Rules[0].Value: %s", mj.Rules[0].Value.(string))
		}

		if mj.Rules[0].Comparison != "contains" {
			return fmt.Errorf("Bad value mj.Rules[0].Comparison: %s", mj.Rules[0].Comparison)
		}

		return nil
	}
}

const testAccNS1Monitor_basic = `
resource "ns1_monitor" "foobar" {
  name = "terraform TCP test"
  active = true
  regions = ["lga"]
  type = "tcp"
  frequency = 60
  rapid_recheck = true
  policy = "all"
  config {
    send = "HEAD / HTTP/1.0\r\n\r\n"
    port = 80
    host = "1.1.1.1"
    ssl = true
  }
  rules {
    key = "output"
    value = "200 OK"
    comparison =  "contains"
  }
}`

const testAccNS1Monitor_updated = `
  resource "ns1_monitor" "foobar" {
  name = "terraform TCP test"
  active = false
  regions = ["sjc"]
  type = "tcp"
  frequency = 120
  rapid_recheck = false
  policy = "quorum"
  config {
    send = "HEAD / HTTP/1.0\r\n\r\n"
    port = 443
    host = "2.2.2.2"
    response_timeout = 60000
    ssl = false
  }
  rules {
    key = "output"
    value = "200 OK"
    comparison =  "contains"
  }
}`
