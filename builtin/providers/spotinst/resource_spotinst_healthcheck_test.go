package spotinst

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/spotinst/spotinst-sdk-go/spotinst"
)

func TestAccSpotinstHealthCheck_Basic(t *testing.T) {
	var healthCheck spotinst.HealthCheck
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckSpotinstHealthCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckSpotinstHealthCheckConfigBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSpotinstHealthCheckExists("spotinst_healthcheck.foo", &healthCheck),
					testAccCheckSpotinstHealthCheckAttributes(&healthCheck),
					resource.TestCheckResourceAttr("spotinst_healthcheck.foo", "name", "hc-foo"),
				),
			},
		},
	})
}

func TestAccSpotinstHealthCheck_Updated(t *testing.T) {
	var healthCheck spotinst.HealthCheck
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckSpotinstHealthCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckSpotinstHealthCheckConfigBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSpotinstHealthCheckExists("spotinst_healthcheck.foo", &healthCheck),
					testAccCheckSpotinstHealthCheckAttributes(&healthCheck),
					resource.TestCheckResourceAttr("spotinst_healthcheck.foo", "name", "hc-foo"),
				),
			},
			{
				Config: testAccCheckSpotinstHealthCheckConfigNewValue,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSpotinstHealthCheckExists("spotinst_healthcheck.foo", &healthCheck),
					testAccCheckSpotinstHealthCheckAttributesUpdated(&healthCheck),
					resource.TestCheckResourceAttr("spotinst_healthcheck.foo", "name", "hc-bar"),
				),
			},
		},
	})
}

func testAccCheckSpotinstHealthCheckDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*spotinst.Client)
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "spotinst_healthcheck" {
			continue
		}
		input := &spotinst.ReadHealthCheckInput{ID: spotinst.String(rs.Primary.ID)}
		resp, err := client.HealthCheckService.Read(input)
		if err == nil && resp != nil && resp.HealthCheck != nil {
			return fmt.Errorf("HealthCheck still exists")
		}
	}
	return nil
}

func testAccCheckSpotinstHealthCheckAttributes(healthCheck *spotinst.HealthCheck) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if p := spotinst.StringValue(healthCheck.Check.Protocol); p != "http" {
			return fmt.Errorf("Bad content: %s", p)
		}
		if e := spotinst.StringValue(healthCheck.Check.Endpoint); e != "http://endpoint.com" {
			return fmt.Errorf("Bad content: %s", e)
		}
		return nil
	}
}

func testAccCheckSpotinstHealthCheckAttributesUpdated(healthCheck *spotinst.HealthCheck) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if p := spotinst.StringValue(healthCheck.Check.Protocol); p != "https" {
			return fmt.Errorf("Bad content: %s", p)
		}
		if e := spotinst.StringValue(healthCheck.Check.Endpoint); e != "https://endpoint.com" {
			return fmt.Errorf("Bad content: %s", e)
		}
		return nil
	}
}

func testAccCheckSpotinstHealthCheckExists(n string, healthCheck *spotinst.HealthCheck) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No resource ID is set")
		}
		client := testAccProvider.Meta().(*spotinst.Client)
		input := &spotinst.ReadHealthCheckInput{ID: spotinst.String(rs.Primary.ID)}
		resp, err := client.HealthCheckService.Read(input)
		if err != nil {
			return err
		}
		if spotinst.StringValue(resp.HealthCheck.ID) != rs.Primary.Attributes["id"] {
			return fmt.Errorf("HealthCheck not found: %+v,\n %+v\n", resp.HealthCheck, rs.Primary.Attributes)
		}
		*healthCheck = *resp.HealthCheck
		return nil
	}
}

const testAccCheckSpotinstHealthCheckConfigBasic = `
resource "spotinst_healthcheck" "foo" {
	name = "hc-foo"
	resource_id = "sig-foo"
	check {
		protocol = "http"
		endpoint = "http://endpoint.com"
		port = 1337
		interval = 10
		timeout = 10
	}
	threshold {
		healthy = 1
		unhealthy = 1
	}
	proxy {
		addr = "http://proxy.com"
		port = 80
	}
}`

const testAccCheckSpotinstHealthCheckConfigNewValue = `
resource "spotinst_healthcheck" "foo" {
	name = "hc-bar"
	resource_id = "sig-foo"
	check {
		protocol = "https"
		endpoint = "https://endpoint.com"
		port = 3000
		interval = 10
		timeout = 10
	}
	threshold {
		healthy = 2
		unhealthy = 2
	}
	proxy {
		addr = "http://proxy.com"
		port = 8080
	}
}`
