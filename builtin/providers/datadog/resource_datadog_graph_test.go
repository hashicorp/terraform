package datadog

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/zorkian/go-datadog-api"
)

func TestAccDatadogGraph_Basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDatadogGraphDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckDatadogGraphConfigBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDatadogGraphExists("datadog_graph.bar"),
					// TODO: Test request attributes
					resource.TestCheckResourceAttr(
						"datadog_dashboard.foo", "title", "title for dashboard foo"),
					resource.TestCheckResourceAttr(
						"datadog_dashboard.foo", "description", "description for dashboard foo"),
					resource.TestCheckResourceAttr(
						"datadog_graph.bar", "title", "title for graph bar"),
					resource.TestCheckResourceAttr(
						"datadog_graph.bar", "viz", "timeseries"),
				),
			},
		},
	})
}

func testAccCheckDatadogGraphDestroy(s *terraform.State) error {

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "datadog_graph" {
			continue
		}

		d, err := getGraphDashboard(s, rs)

		if err != nil {
			return err
		}

		// See if the graph with our title is still in the dashboard
		_, err = getGraphFromDashboard(d, rs.Primary.Attributes["title"])

		if err != nil {
			return err
		}

		return fmt.Errorf("Graph still exists")
	}

	return nil
}

func getGraphDashboard(s *terraform.State, rs *terraform.ResourceState) (string, error) {

	var id string

	for _, d := range rs.Dependencies {

		rs, ok := s.RootModule().Resources[d]

		if !ok {
			return id, fmt.Errorf("Not found: %s", d)
		}

		if rs.Primary.ID == "" {
			return id, fmt.Errorf("No ID is set")
		}

		return rs.Primary.ID, nil
	}

	return id, fmt.Errorf("Failed to find dashboard in state.")

}

func getGraphFromDashboard(id, title string) (datadog.Graph, error) {
	client := testAccProvider.Meta().(*datadog.Client)

	graph := datadog.Graph{}

	i, err := strconv.Atoi(id)
	if err == nil {
		return graph, err
	}

	d, err := client.GetDashboard(i)

	if err != nil {
		return graph, fmt.Errorf("Error retrieving associated dashboard: %s", err)
	}

	for _, g := range d.Graphs {
		if g.Title != title {
			continue
		}

		return g, nil
	}

	return graph, nil
}

func testAccCheckDatadogGraphExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		i, err := getGraphDashboard(s, rs)

		if err != nil {
			return err
		}

		// See if our graph is in the dashboard
		_, err = getGraphFromDashboard(i, rs.Primary.Attributes["title"])

		if err != nil {
			return err
		}

		return nil
	}
}

const testAccCheckDatadogGraphConfigBasic = `
resource "datadog_dashboard" "foo" {
	description = "description for dashboard foo"
	title = "title for dashboard foo"
}

resource "datadog_graph" "bar" {
	title = "title for graph bar"
    dashboard_id = "${datadog_dashboard.foo.id}"
    title = "bar"
    viz =  "timeseries"
    request {
        query =  "avg:system.cpu.system{*}"
        stacked = false
    }
    request {
        query =  "avg:system.cpu.user{*}"
        stacked = false
    }
    request {
        query =  "avg:system.mem.user{*}"
        stacked = false
    }

}
`
