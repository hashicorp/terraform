package circonus

import (
	"fmt"
	"strings"
	"testing"

	"github.com/circonus-labs/circonus-gometrics/api"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccCirconusStreamGroup_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDestroyCirconusStreamGroup,
		Steps: []resource.TestStep{
			{
				Config: testAccCirconusStreamGroupConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("circonus_stream_group.nomad-job1", "description", `Stream Group (a.k.a. "metric cluster") Description`),
					resource.TestCheckResourceAttr("circonus_stream_group.nomad-job1", "name", "job1-stream-agg"),
					resource.TestCheckResourceAttr("circonus_stream_group.nomad-job1", "group.1688061877.query", "*`nomad-jobname`memory`rss"),
					resource.TestCheckResourceAttr("circonus_stream_group.nomad-job1", "group.1688061877.type", "average"),
					resource.TestCheckResourceAttr("circonus_stream_group.nomad-job1", "tags.2087084518", "author:terraform"),
					resource.TestCheckResourceAttr("circonus_stream_group.nomad-job1", "tags.3354173695", "source:nomad"),
				),
			},
		},
	})
}

func testAccCheckDestroyCirconusStreamGroup(s *terraform.State) error {
	ctxt := testAccProvider.Meta().(*_ProviderContext)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "circonus_stream_group" {
			continue
		}

		cid := rs.Primary.ID
		exists, err := checkStreamGroupExists(ctxt, api.CIDType(&cid))
		switch {
		case !exists:
			// noop
		case exists:
			return fmt.Errorf("stream group still exists after destroy")
		case err != nil:
			return fmt.Errorf("Error checking stream group %s", err)
		}
	}

	return nil
}

func testAccStreamGroupExists(n string, streamGroupID api.CIDType) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Resource not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		ctxt := testAccProvider.Meta().(*_ProviderContext)
		cid := rs.Primary.ID
		exists, err := checkStreamGroupExists(ctxt, api.CIDType(&cid))
		switch {
		case !exists:
			// noop
		case exists:
			return fmt.Errorf("stream group still exists after destroy")
		case err != nil:
			return fmt.Errorf("Error checking stream group %s", err)
		}

		return nil
	}
}

func checkStreamGroupExists(c *_ProviderContext, streamGroupID api.CIDType) (bool, error) {
	sg, err := c.client.FetchMetricCluster(streamGroupID, "")
	if err != nil {
		if strings.Contains(err.Error(), defaultCirconus404ErrorString) {
			return false, nil
		} else {
			return false, err
		}
	}

	if api.CIDType(&sg.CID) == streamGroupID {
		return true, nil
	} else {
		return false, nil
	}
}

const testAccCirconusStreamGroupConfig = `
resource "circonus_stream_group" "nomad-job1" {
  description = <<EOF
Stream Group (a.k.a. "metric cluster") Description
EOF
  name = "job1-stream-agg"

  group {
    query = "*` + "`" + `nomad-jobname` + "`" + `memory` + "`" + `rss"
    type = "average"
  }

  tags = [
    "author:terraform",
    "source:nomad",
  ]
}
`
