package circonus

import (
	"fmt"
	"strings"
	"testing"

	"github.com/circonus-labs/circonus-gometrics/api"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccCirconusMetricCluster_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDestroyCirconusMetricCluster,
		Steps: []resource.TestStep{
			{
				Config: testAccCirconusMetricClusterConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("circonus_metric_cluster.nomad-job1", "description", `Metric Cluster Description`),
					resource.TestCheckResourceAttrSet("circonus_metric_cluster.nomad-job1", "id"),
					resource.TestCheckResourceAttr("circonus_metric_cluster.nomad-job1", "name", "job1-stream-agg"),
					resource.TestCheckResourceAttr("circonus_metric_cluster.nomad-job1", "query.236803225.definition", "*`nomad-jobname`memory`rss"),
					resource.TestCheckResourceAttr("circonus_metric_cluster.nomad-job1", "query.236803225.type", "average"),
					resource.TestCheckResourceAttr("circonus_metric_cluster.nomad-job1", "tags.2087084518", "author:terraform"),
					resource.TestCheckResourceAttr("circonus_metric_cluster.nomad-job1", "tags.3354173695", "source:nomad"),
				),
			},
		},
	})
}

func testAccCheckDestroyCirconusMetricCluster(s *terraform.State) error {
	ctxt := testAccProvider.Meta().(*providerContext)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "circonus_metric_cluster" {
			continue
		}

		cid := rs.Primary.ID
		exists, err := checkMetricClusterExists(ctxt, api.CIDType(&cid))
		switch {
		case !exists:
			// noop
		case exists:
			return fmt.Errorf("metric cluster still exists after destroy")
		case err != nil:
			return fmt.Errorf("Error checking metric cluster: %v", err)
		}
	}

	return nil
}

func testAccMetricClusterExists(n string, metricClusterCID api.CIDType) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Resource not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		ctxt := testAccProvider.Meta().(*providerContext)
		cid := rs.Primary.ID
		exists, err := checkMetricClusterExists(ctxt, api.CIDType(&cid))
		switch {
		case !exists:
			// noop
		case exists:
			return fmt.Errorf("metric cluster still exists after destroy")
		case err != nil:
			return fmt.Errorf("Error checking metric cluster: %v", err)
		}

		return nil
	}
}

func checkMetricClusterExists(c *providerContext, metricClusterCID api.CIDType) (bool, error) {
	cmc, err := c.client.FetchMetricCluster(metricClusterCID, "")
	if err != nil {
		if strings.Contains(err.Error(), defaultCirconus404ErrorString) {
			return false, nil
		}

		return false, err
	}

	if api.CIDType(&cmc.CID) == metricClusterCID {
		return true, nil
	}

	return false, nil
}

const testAccCirconusMetricClusterConfig = `
resource "circonus_metric_cluster" "nomad-job1" {
  description = <<EOF
Metric Cluster Description
EOF
  name = "job1-stream-agg"

  query {
    definition = "*` + "`" + `nomad-jobname` + "`" + `memory` + "`" + `rss"
    type = "average"
  }

  tags = [
    "author:terraform",
    "source:nomad",
  ]
}
`
