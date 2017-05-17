package circonus

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccCirconusCheckCloudWatch_basic(t *testing.T) {
	checkName := fmt.Sprintf("Terraform test: RDS Metrics via CloudWatch - %s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDestroyCirconusCheckBundle,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccCirconusCheckCloudWatchConfigFmt, checkName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "active", "true"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "collector.#", "1"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "collector.2388330941.id", "/broker/1"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "cloudwatch.#", "1"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "cloudwatch.88938937.dimmensions.%", "1"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "cloudwatch.88938937.dimmensions.DBInstanceIdentifier", "atlas-production"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "cloudwatch.88938937.metric.#", "17"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "cloudwatch.88938937.metric.990896688", "CPUUtilization"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "cloudwatch.88938937.metric.3895259375", "DatabaseConnections"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "cloudwatch.88938937.metric.1328149445", "DiskQueueDepth"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "cloudwatch.88938937.metric.4218650584", "FreeStorageSpace"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "cloudwatch.88938937.metric.1835248983", "FreeableMemory"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "cloudwatch.88938937.metric.2757008135", "MaximumUsedTransactionIDs"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "cloudwatch.88938937.metric.915415866", "NetworkReceiveThroughput"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "cloudwatch.88938937.metric.1852047735", "NetworkTransmitThroughput"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "cloudwatch.88938937.metric.3518416306", "ReadIOPS"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "cloudwatch.88938937.metric.114013313", "ReadLatency"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "cloudwatch.88938937.metric.1284099341", "ReadThroughput"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "cloudwatch.88938937.metric.4205329773", "SwapUsage"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "cloudwatch.88938937.metric.3550163941", "TransactionLogsDiskUsage"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "cloudwatch.88938937.metric.2231806695", "TransactionLogsGeneration"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "cloudwatch.88938937.metric.335777904", "WriteIOPS"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "cloudwatch.88938937.metric.3894876280", "WriteLatency"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "cloudwatch.88938937.metric.1569904650", "WriteThroughput"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "cloudwatch.88938937.namespace", "AWS/RDS"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "cloudwatch.88938937.version", "2010-08-01"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "cloudwatch.88938937.url", "https://monitoring.us-east-1.amazonaws.com"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "name", checkName),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "notes", "Collect all the things exposed"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "period", "60s"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.#", "17"),

					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3038868367.active", "true"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3038868367.name", "ReadLatency"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3038868367.tags.#", "4"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3038868367.tags.1313458811", "app:rds"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3038868367.tags.1401442048", "lifecycle:unittest"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3038868367.tags.2964981562", "app:postgresql"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3038868367.tags.4259413593", "source:cloudwatch"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3038868367.type", "numeric"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3038868367.unit", "seconds"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3699049608.active", "true"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3699049608.name", "TransactionLogsGeneration"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3699049608.tags.#", "4"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3699049608.tags.1313458811", "app:rds"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3699049608.tags.1401442048", "lifecycle:unittest"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3699049608.tags.2964981562", "app:postgresql"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3699049608.tags.4259413593", "source:cloudwatch"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3699049608.type", "numeric"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3699049608.unit", ""),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3932256294.active", "true"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3932256294.name", "WriteIOPS"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3932256294.tags.#", "4"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3932256294.tags.1313458811", "app:rds"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3932256294.tags.1401442048", "lifecycle:unittest"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3932256294.tags.2964981562", "app:postgresql"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3932256294.tags.4259413593", "source:cloudwatch"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3932256294.type", "numeric"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3932256294.unit", "iops"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3655789574.active", "true"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3655789574.name", "FreeStorageSpace"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3655789574.tags.#", "4"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3655789574.tags.1313458811", "app:rds"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3655789574.tags.1401442048", "lifecycle:unittest"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3655789574.tags.2964981562", "app:postgresql"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3655789574.tags.4259413593", "source:cloudwatch"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3655789574.type", "numeric"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3655789574.unit", ""),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3129782198.active", "true"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3129782198.name", "WriteLatency"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3129782198.tags.#", "4"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3129782198.tags.1313458811", "app:rds"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3129782198.tags.1401442048", "lifecycle:unittest"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3129782198.tags.2964981562", "app:postgresql"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3129782198.tags.4259413593", "source:cloudwatch"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3129782198.type", "numeric"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3129782198.unit", "seconds"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.4177148265.active", "true"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.4177148265.name", "DatabaseConnections"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.4177148265.tags.#", "4"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.4177148265.tags.1313458811", "app:rds"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.4177148265.tags.1401442048", "lifecycle:unittest"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.4177148265.tags.2964981562", "app:postgresql"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.4177148265.tags.4259413593", "source:cloudwatch"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.4177148265.type", "numeric"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.4177148265.unit", "connections"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.2882920904.active", "true"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.2882920904.name", "FreeableMemory"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.2882920904.tags.#", "4"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.2882920904.tags.1313458811", "app:rds"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.2882920904.tags.1401442048", "lifecycle:unittest"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.2882920904.tags.2964981562", "app:postgresql"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.2882920904.tags.4259413593", "source:cloudwatch"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.2882920904.type", "numeric"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.2882920904.unit", "bytes"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3449506.active", "true"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3449506.name", "MaximumUsedTransactionIDs"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3449506.tags.#", "4"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3449506.tags.1313458811", "app:rds"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3449506.tags.1401442048", "lifecycle:unittest"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3449506.tags.2964981562", "app:postgresql"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3449506.tags.4259413593", "source:cloudwatch"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3449506.type", "numeric"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3449506.unit", ""),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3907804414.active", "true"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3907804414.name", "ReadThroughput"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3907804414.tags.#", "4"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3907804414.tags.1313458811", "app:rds"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3907804414.tags.1401442048", "lifecycle:unittest"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3907804414.tags.2964981562", "app:postgresql"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3907804414.tags.4259413593", "source:cloudwatch"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3907804414.type", "numeric"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3907804414.unit", "bytes"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.2040713199.active", "true"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.2040713199.name", "ReadIOPS"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.2040713199.tags.#", "4"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.2040713199.tags.1313458811", "app:rds"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.2040713199.tags.1401442048", "lifecycle:unittest"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.2040713199.tags.2964981562", "app:postgresql"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.2040713199.tags.4259413593", "source:cloudwatch"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.2040713199.type", "numeric"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.2040713199.unit", "iops"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.4227693369.active", "true"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.4227693369.name", "NetworkReceiveThroughput"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.4227693369.tags.#", "4"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.4227693369.tags.1313458811", "app:rds"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.4227693369.tags.1401442048", "lifecycle:unittest"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.4227693369.tags.2964981562", "app:postgresql"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.4227693369.tags.4259413593", "source:cloudwatch"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.4227693369.type", "numeric"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.4227693369.unit", "bytes"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.2478479732.active", "true"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.2478479732.name", "TransactionLogsDiskUsage"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.2478479732.tags.#", "4"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.2478479732.tags.1313458811", "app:rds"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.2478479732.tags.1401442048", "lifecycle:unittest"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.2478479732.tags.2964981562", "app:postgresql"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.2478479732.tags.4259413593", "source:cloudwatch"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.2478479732.type", "numeric"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.2478479732.unit", "bytes"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3437310515.active", "true"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3437310515.name", "CPUUtilization"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3437310515.tags.#", "4"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3437310515.tags.1313458811", "app:rds"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3437310515.tags.1401442048", "lifecycle:unittest"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3437310515.tags.2964981562", "app:postgresql"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3437310515.tags.4259413593", "source:cloudwatch"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3437310515.type", "numeric"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3437310515.unit", "%"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3884263074.active", "true"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3884263074.name", "SwapUsage"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3884263074.tags.#", "4"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3884263074.tags.1313458811", "app:rds"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3884263074.tags.1401442048", "lifecycle:unittest"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3884263074.tags.2964981562", "app:postgresql"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3884263074.tags.4259413593", "source:cloudwatch"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3884263074.type", "numeric"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3884263074.unit", "bytes"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.2675539305.active", "true"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.2675539305.name", "NetworkTransmitThroughput"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.2675539305.tags.#", "4"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.2675539305.tags.1313458811", "app:rds"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.2675539305.tags.1401442048", "lifecycle:unittest"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.2675539305.tags.2964981562", "app:postgresql"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.2675539305.tags.4259413593", "source:cloudwatch"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.2675539305.type", "numeric"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.2675539305.unit", "bytes"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.4273610941.active", "true"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.4273610941.name", "DiskQueueDepth"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.4273610941.tags.#", "4"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.4273610941.tags.1313458811", "app:rds"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.4273610941.tags.1401442048", "lifecycle:unittest"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.4273610941.tags.2964981562", "app:postgresql"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.4273610941.tags.4259413593", "source:cloudwatch"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.4273610941.type", "numeric"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.4273610941.unit", ""),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3085334826.active", "true"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3085334826.name", "WriteThroughput"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3085334826.tags.#", "4"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3085334826.tags.1313458811", "app:rds"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3085334826.tags.1401442048", "lifecycle:unittest"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3085334826.tags.2964981562", "app:postgresql"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3085334826.tags.4259413593", "source:cloudwatch"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3085334826.type", "numeric"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "metric.3085334826.unit", "bytes"),

					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "tags.#", "4"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "tags.2964981562", "app:postgresql"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "tags.1313458811", "app:rds"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "tags.1401442048", "lifecycle:unittest"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "tags.4259413593", "source:cloudwatch"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "target", "atlas-production.us-east-1.rds._aws"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "type", "cloudwatch"),
				),
			},
		},
	})
}

const testAccCirconusCheckCloudWatchConfigFmt = `
variable "cloudwatch_rds_tags" {
  type = "list"
  default = [
    "app:postgresql",
    "app:rds",
    "lifecycle:unittest",
    "source:cloudwatch",
  ]
}

resource "circonus_check" "rds_metrics" {
  active = true
  name = "%s"
  notes = "Collect all the things exposed"
  period = "60s"

  collector {
    id = "/broker/1"
  }

  cloudwatch {
    dimmensions = {
      DBInstanceIdentifier = "atlas-production",
    }

    metric = [
      "CPUUtilization",
      "DatabaseConnections",
      "DiskQueueDepth",
      "FreeStorageSpace",
      "FreeableMemory",
      "MaximumUsedTransactionIDs",
      "NetworkReceiveThroughput",
      "NetworkTransmitThroughput",
      "ReadIOPS",
      "ReadLatency",
      "ReadThroughput",
      "SwapUsage",
      "TransactionLogsDiskUsage",
      "TransactionLogsGeneration",
      "WriteIOPS",
      "WriteLatency",
      "WriteThroughput",
    ]

    namespace = "AWS/RDS"
    url = "https://monitoring.us-east-1.amazonaws.com"
  }

  metric {
    name = "CPUUtilization"
    tags = [ "${var.cloudwatch_rds_tags}" ]
    type = "numeric"
    unit = "%%"
  }

  metric {
    name = "DatabaseConnections"
    tags = [ "${var.cloudwatch_rds_tags}" ]
    type = "numeric"
    unit = "connections"
  }

  metric {
    name = "DiskQueueDepth"
    tags = [ "${var.cloudwatch_rds_tags}" ]
    type = "numeric"
  }

  metric {
    name = "FreeStorageSpace"
    tags = [ "${var.cloudwatch_rds_tags}" ]
    type = "numeric"
  }

  metric {
    name = "FreeableMemory"
    tags = [ "${var.cloudwatch_rds_tags}" ]
    type = "numeric"
    unit = "bytes"
  }

  metric {
    name = "MaximumUsedTransactionIDs"
    tags = [ "${var.cloudwatch_rds_tags}" ]
    type = "numeric"
  }

  metric {
    name = "NetworkReceiveThroughput"
    tags = [ "${var.cloudwatch_rds_tags}" ]
    type = "numeric"
    unit = "bytes"
  }

  metric {
    name = "NetworkTransmitThroughput"
    tags = [ "${var.cloudwatch_rds_tags}" ]
    type = "numeric"
    unit = "bytes"
  }

  metric {
    name = "ReadIOPS"
    tags = [ "${var.cloudwatch_rds_tags}" ]
    type = "numeric"
    unit = "iops"
  }

  metric {
    name = "ReadLatency"
    tags = [ "${var.cloudwatch_rds_tags}" ]
    type = "numeric"
    unit = "seconds"
  }

  metric {
    name = "ReadThroughput"
    tags = [ "${var.cloudwatch_rds_tags}" ]
    type = "numeric"
    unit = "bytes"
  }

  metric {
    name = "SwapUsage"
    tags = [ "${var.cloudwatch_rds_tags}" ]
    type = "numeric"
    unit = "bytes"
  }

  metric {
    name = "TransactionLogsDiskUsage"
    tags = [ "${var.cloudwatch_rds_tags}" ]
    type = "numeric"
    unit = "bytes"
  }

  metric {
    name = "TransactionLogsGeneration"
    tags = [ "${var.cloudwatch_rds_tags}" ]
    type = "numeric"
  }

  metric {
    name = "WriteIOPS"
    tags = [ "${var.cloudwatch_rds_tags}" ]
    type = "numeric"
    unit = "iops"
  }

  metric {
    name = "WriteLatency"
    tags = [ "${var.cloudwatch_rds_tags}" ]
    type = "numeric"
    unit = "seconds"
  }

  metric {
    name = "WriteThroughput"
    tags = [ "${var.cloudwatch_rds_tags}" ]
    type = "numeric"
    unit = "bytes"
  }

  tags = [ "${var.cloudwatch_rds_tags}" ]
}
`
