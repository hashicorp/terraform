package datadog

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/zorkian/go-datadog-api"
)

const config1 = `
resource "datadog_timeboard" "acceptance_test" {
  title = "Acceptance Test Timeboard"
  description = "Created using the Datadog prodivider in Terraform"
  read_only = true
  graph {
    title = "Top System CPU by Docker container"
    viz = "toplist"
    request {
      q = "top(avg:docker.cpu.system{*} by {container_name}, 10, 'mean', 'desc')"
    }
  }
}
`

const config2 = `
resource "datadog_timeboard" "acceptance_test" {
  title = "Acceptance Test Timeboard"
  description = "Created using the Datadog prodivider in Terraform"
  graph {
    title = "Redis latency (ms)"
    viz = "timeseries"
    request {
      q = "avg:redis.info.latency_ms{$host}"
    }
  }
  graph {
    title = "Redis memory usage"
    viz = "timeseries"
    request {
      q = "avg:redis.mem.used{$host} - avg:redis.mem.lua{$host}, avg:redis.mem.lua{$host}"
      stacked = true
    }
    request {
      q = "avg:redis.mem.rss{$host}"
    }
    request {
      q = "avg:redis.mem.rss{$host}"
      type = "bars"
      style {
        palette = "warm"
      }
    }
  }
  template_variable {
    name = "host"
    prefix = "host"
  }
}
`

func TestAccDatadogTimeboard_update(t *testing.T) {

	step1 := resource.TestStep{
		Config: config1,
		Check: resource.ComposeTestCheckFunc(
			checkExists,
			resource.TestCheckResourceAttr("datadog_timeboard.acceptance_test", "title", "Acceptance Test Timeboard"),
			resource.TestCheckResourceAttr("datadog_timeboard.acceptance_test", "description", "Created using the Datadog prodivider in Terraform"),
			resource.TestCheckResourceAttr("datadog_timeboard.acceptance_test", "read_only", "true"),
			resource.TestCheckResourceAttr("datadog_timeboard.acceptance_test", "graph.0.title", "Top System CPU by Docker container"),
			resource.TestCheckResourceAttr("datadog_timeboard.acceptance_test", "graph.0.viz", "toplist"),
			resource.TestCheckResourceAttr("datadog_timeboard.acceptance_test", "graph.0.request.0.q", "top(avg:docker.cpu.system{*} by {container_name}, 10, 'mean', 'desc')"),
		),
	}

	step2 := resource.TestStep{
		Config: config2,
		Check: resource.ComposeTestCheckFunc(
			checkExists,
			resource.TestCheckResourceAttr("datadog_timeboard.acceptance_test", "title", "Acceptance Test Timeboard"),
			resource.TestCheckResourceAttr("datadog_timeboard.acceptance_test", "description", "Created using the Datadog prodivider in Terraform"),
			resource.TestCheckResourceAttr("datadog_timeboard.acceptance_test", "graph.0.title", "Redis latency (ms)"),
			resource.TestCheckResourceAttr("datadog_timeboard.acceptance_test", "graph.0.viz", "timeseries"),
			resource.TestCheckResourceAttr("datadog_timeboard.acceptance_test", "graph.0.request.0.q", "avg:redis.info.latency_ms{$host}"),
			resource.TestCheckResourceAttr("datadog_timeboard.acceptance_test", "graph.1.title", "Redis memory usage"),
			resource.TestCheckResourceAttr("datadog_timeboard.acceptance_test", "graph.1.viz", "timeseries"),
			resource.TestCheckResourceAttr("datadog_timeboard.acceptance_test", "graph.1.request.0.q", "avg:redis.mem.used{$host} - avg:redis.mem.lua{$host}, avg:redis.mem.lua{$host}"),
			resource.TestCheckResourceAttr("datadog_timeboard.acceptance_test", "graph.1.request.0.stacked", "true"),
			resource.TestCheckResourceAttr("datadog_timeboard.acceptance_test", "graph.1.request.1.q", "avg:redis.mem.rss{$host}"),
			resource.TestCheckResourceAttr("datadog_timeboard.acceptance_test", "template_variable.0.name", "host"),
			resource.TestCheckResourceAttr("datadog_timeboard.acceptance_test", "template_variable.0.prefix", "host"),
			resource.TestCheckResourceAttr("datadog_timeboard.acceptance_test", "graph.1.request.2.type", "bars"),
			resource.TestCheckResourceAttr("datadog_timeboard.acceptance_test", "graph.1.request.2.style.palette", "warm"),
		),
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: checkDestroy,
		Steps:        []resource.TestStep{step1, step2},
	})
}

func checkExists(s *terraform.State) error {
	client := testAccProvider.Meta().(*datadog.Client)
	for _, r := range s.RootModule().Resources {
		i, _ := strconv.Atoi(r.Primary.ID)
		if _, err := client.GetDashboard(i); err != nil {
			return fmt.Errorf("Received an error retrieving monitor %s", err)
		}
	}
	return nil
}

func checkDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*datadog.Client)
	for _, r := range s.RootModule().Resources {
		i, _ := strconv.Atoi(r.Primary.ID)
		if _, err := client.GetDashboard(i); err != nil {
			if strings.Contains(err.Error(), "404 Not Found") {
				continue
			}
			return fmt.Errorf("Received an error retrieving timeboard %s", err)
		}
		return fmt.Errorf("Timeboard still exists")
	}
	return nil
}
