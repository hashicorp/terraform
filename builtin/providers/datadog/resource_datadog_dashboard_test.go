package datadog

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/zorkian/go-datadog-api"
)

func TestAccDatadogDashboard_Basic(t *testing.T) {
	var resp datadog.Dashboard

	testTemplateVariables := func(*terraform.State) error {
		var resp datadog.Dashboard

		if len(resp.TemplateVariables) != 2 {
			return fmt.Errorf("bad template variables: %#v", resp.TemplateVariables)
		}

		variables := make(map[string]datadog.TemplateVariable)
		for _, r := range resp.TemplateVariables {
			variables[r.Name] = r
		}

		if _, ok := variables["host1"]; !ok {
			return fmt.Errorf("bad template variables: %#v", resp.TemplateVariables)
		}
		if _, ok := variables["host2"]; !ok {
			return fmt.Errorf("bad template variables: %#v", resp.TemplateVariables)
		}

		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDatadogDashboardDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckDatadogDashboardConfigBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDatadogDashboardExists("datadog_dashboard.foo", &resp),
					resource.TestCheckResourceAttr(
						"datadog_dashboard.foo", "title", "title for dashboard foo"),
					resource.TestCheckResourceAttr(
						"datadog_dashboard.foo", "description", "description for dashboard foo"),
					testTemplateVariables,
					// TODO: Add tests to verify change
				),
			},
		},
	})

}

func testAccCheckDatadogDashboardDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*datadog.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "datadog_dashboard" {
			continue
		}

		intID, intErr := strconv.Atoi(rs.Primary.ID)
		if intErr == nil {
			return intErr
		}

		_, err := client.GetDashboard(intID)

		if err == nil {
			return fmt.Errorf("Dashboard still exists")
		}
	}

	return nil
}

func testAccCheckDatadogDashboardExists(n string, DashboardResp *datadog.Dashboard) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Dashboard ID is set")
		}

		client := testAccProvider.Meta().(*datadog.Client)

		intID, intErr := strconv.Atoi(rs.Primary.ID)

		if intErr != nil {
			return intErr
		}

		resp, err := client.GetDashboard(intID)

		if err != nil {
			return err
		}

		// TODO: fix this one.
		//if resp.Dashboard.name != rs.Primary.ID {
		//return fmt.Errorf("Domain not found")
		//}

		DashboardResp = resp

		return nil
	}
}

const testAccCheckDatadogDashboardConfigBasic = `
resource "datadog_dashboard" "foo" {
       description = "description for dashboard foo"
       title = "title for dashboard foo"
       template_variable {
		name = "host1"
		prefix = "host"
		default = "host:foo.example.com"
       }
       template_variable {
		name = "host2"
		prefix = "host"
		default = "host:bar.example.com"
       }
   }
`
