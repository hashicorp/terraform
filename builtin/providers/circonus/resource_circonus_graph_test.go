package circonus

import (
	"fmt"
	"strings"
	"testing"

	"github.com/circonus-labs/circonus-gometrics/api"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccCirconusGraph_basic(t *testing.T) {
	graphName := fmt.Sprintf("Test Graph - %s", acctest.RandString(5))
	checkName := fmt.Sprintf("ICMP Ping check - %s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDestroyCirconusGraph,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccCirconusGraphConfigFmt, checkName, graphName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("circonus_graph.mixed-points", "name", graphName),
					resource.TestCheckResourceAttr("circonus_graph.mixed-points", "description", "Terraform Test: mixed graph"),
					resource.TestCheckResourceAttr("circonus_graph.mixed-points", "notes", "test notes"),
					resource.TestCheckResourceAttr("circonus_graph.mixed-points", "graph_style", "line"),
					resource.TestCheckResourceAttr("circonus_graph.mixed-points", "left.%", "1"),
					resource.TestCheckResourceAttr("circonus_graph.mixed-points", "left.max", "11"),
					resource.TestCheckResourceAttr("circonus_graph.mixed-points", "right.%", "3"),
					resource.TestCheckResourceAttr("circonus_graph.mixed-points", "right.logarithmic", "10"),
					resource.TestCheckResourceAttr("circonus_graph.mixed-points", "right.max", "20"),
					resource.TestCheckResourceAttr("circonus_graph.mixed-points", "right.min", "-1"),

					resource.TestCheckResourceAttr("circonus_graph.mixed-points", "line_style", "stepped"),

					resource.TestCheckResourceAttr("circonus_graph.mixed-points", "metric.#", "2"),

					resource.TestCheckResourceAttr("circonus_graph.mixed-points", "metric.0.caql", ""),
					resource.TestCheckResourceAttrSet("circonus_graph.mixed-points", "metric.0.check"),
					resource.TestCheckResourceAttr("circonus_graph.mixed-points", "metric.0.metric_name", "maximum"),
					resource.TestCheckResourceAttr("circonus_graph.mixed-points", "metric.0.metric_type", "numeric"),
					resource.TestCheckResourceAttr("circonus_graph.mixed-points", "metric.0.name", "Maximum Latency"),
					resource.TestCheckResourceAttr("circonus_graph.mixed-points", "metric.0.axis", "left"),
					resource.TestCheckResourceAttr("circonus_graph.mixed-points", "metric.0.color", "#657aa6"),
					resource.TestCheckResourceAttr("circonus_graph.mixed-points", "metric.0.function", "gauge"),
					resource.TestCheckResourceAttr("circonus_graph.mixed-points", "metric.0.active", "true"),

					resource.TestCheckResourceAttr("circonus_graph.mixed-points", "metric.1.caql", ""),
					resource.TestCheckResourceAttrSet("circonus_graph.mixed-points", "metric.1.check"),
					resource.TestCheckResourceAttr("circonus_graph.mixed-points", "metric.1.metric_name", "minimum"),
					resource.TestCheckResourceAttr("circonus_graph.mixed-points", "metric.1.metric_type", "numeric"),
					resource.TestCheckResourceAttr("circonus_graph.mixed-points", "metric.1.name", "Minimum Latency"),
					resource.TestCheckResourceAttr("circonus_graph.mixed-points", "metric.1.axis", "right"),
					resource.TestCheckResourceAttr("circonus_graph.mixed-points", "metric.1.color", "#657aa6"),
					resource.TestCheckResourceAttr("circonus_graph.mixed-points", "metric.1.function", "gauge"),
					resource.TestCheckResourceAttr("circonus_graph.mixed-points", "metric.1.active", "true"),

					resource.TestCheckResourceAttr("circonus_graph.mixed-points", "tags.#", "2"),
					resource.TestCheckResourceAttr("circonus_graph.mixed-points", "tags.2087084518", "author:terraform"),
					resource.TestCheckResourceAttr("circonus_graph.mixed-points", "tags.1401442048", "lifecycle:unittest"),
				),
			},
		},
	})
}

func testAccCheckDestroyCirconusGraph(s *terraform.State) error {
	ctxt := testAccProvider.Meta().(*providerContext)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "circonus_graph" {
			continue
		}

		cid := rs.Primary.ID
		exists, err := checkGraphExists(ctxt, api.CIDType(&cid))
		switch {
		case !exists:
			// noop
		case exists:
			return fmt.Errorf("graph still exists after destroy")
		case err != nil:
			return fmt.Errorf("Error checking graph %s", err)
		}
	}

	return nil
}

func checkGraphExists(c *providerContext, graphID api.CIDType) (bool, error) {
	g, err := c.client.FetchGraph(graphID)
	if err != nil {
		if strings.Contains(err.Error(), defaultCirconus404ErrorString) {
			return false, nil
		}

		return false, err
	}

	if api.CIDType(&g.CID) == graphID {
		return true, nil
	}

	return false, nil
}

const testAccCirconusGraphConfigFmt = `
variable "test_tags" {
  type = "list"
  default = [ "author:terraform", "lifecycle:unittest" ]
}

resource "circonus_check" "api_latency" {
  active = true
  name = "%s"
  period = "60s"

  collector {
    id = "/broker/1"
  }

  icmp_ping {
    count = 5
  }

  metric {
    name = "maximum"
    tags = [ "${var.test_tags}" ]
    type = "numeric"
    unit = "seconds"
  }

  metric {
    name = "minimum"
    tags = [ "${var.test_tags}" ]
    type = "numeric"
    unit = "seconds"
  }

  tags = [ "${var.test_tags}" ]
  target = "api.circonus.com"
}

resource "circonus_graph" "mixed-points" {
  name = "%s"
  description = "Terraform Test: mixed graph"
  notes = "test notes"
  graph_style = "line"
  line_style = "stepped"

  metric {
    # caql = "" # conflicts with metric_name/check
    check = "${circonus_check.api_latency.checks[0]}"
    metric_name = "maximum"
    metric_type = "numeric"
    name = "Maximum Latency"
    axis = "left" # right
    color = "#657aa6"
    function = "gauge"
    active = true
  }

  metric {
    # caql = "" # conflicts with metric_name/check
    check = "${circonus_check.api_latency.checks[0]}"
    metric_name = "minimum"
    metric_type = "numeric"
    name = "Minimum Latency"
    axis = "right" # left
    color = "#657aa6"
    function = "gauge"
    active = true
  }

  // metric_cluster {
  //   active = true
  //   aggregate = "average"
  //   axis = "left" # right
  //   color = "#657aa6"
  //   group = "${circonus_check.api_latency.checks[0]}"
  //   name = "Metrics Used"
  // }

  left {
    max = 11
  }

  right {
    logarithmic = 10
    max = 20
    min = -1
  }

  tags = [ "${var.test_tags}" ]
}
`
