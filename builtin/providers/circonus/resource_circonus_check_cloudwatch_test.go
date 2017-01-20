package circonus

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccCirconusCheckCloudWatch_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDestroyCirconusCheckBundle,
		Steps: []resource.TestStep{
			{
				Config: testAccCirconusCheckCloudWatchConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "active", "true"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "collector.#", "1"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "collector.2388330941.id", "/broker/1"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "cloudwatch.#", "1"),
					// resource.TestCheckResourceAttr("circonus_check.rds_metrics", "cloudwatch.2270818665.api_key", ""),
					// resource.TestCheckResourceAttr("circonus_check.rds_metrics", "cloudwatch.2270818665.api_secret", ""),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "cloudwatch.2270818665.dimmensions.%", "1"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "cloudwatch.2270818665.dimmensions.DBInstanceIdentifier", "atlas-production"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "cloudwatch.2270818665.metric.#", "17"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "cloudwatch.2270818665.metric.990896688", "CPUUtilization"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "cloudwatch.2270818665.metric.3895259375", "DatabaseConnections"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "cloudwatch.2270818665.metric.1328149445", "DiskQueueDepth"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "cloudwatch.2270818665.metric.4218650584", "FreeStorageSpace"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "cloudwatch.2270818665.metric.1835248983", "FreeableMemory"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "cloudwatch.2270818665.metric.2757008135", "MaximumUsedTransactionIDs"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "cloudwatch.2270818665.metric.915415866", "NetworkReceiveThroughput"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "cloudwatch.2270818665.metric.1852047735", "NetworkTransmitThroughput"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "cloudwatch.2270818665.metric.3518416306", "ReadIOPS"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "cloudwatch.2270818665.metric.114013313", "ReadLatency"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "cloudwatch.2270818665.metric.1284099341", "ReadThroughput"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "cloudwatch.2270818665.metric.4205329773", "SwapUsage"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "cloudwatch.2270818665.metric.3550163941", "TransactionLogsDiskUsage"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "cloudwatch.2270818665.metric.2231806695", "TransactionLogsGeneration"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "cloudwatch.2270818665.metric.335777904", "WriteIOPS"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "cloudwatch.2270818665.metric.3894876280", "WriteLatency"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "cloudwatch.2270818665.metric.1569904650", "WriteThroughput"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "cloudwatch.2270818665.namespace", "AWS/RDS"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "cloudwatch.2270818665.version", "2010-08-01"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "cloudwatch.2270818665.url", "https://monitoring.us-east-1.amazonaws.com"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "name", "Terraform test: RDS Metrics via CloudWatch"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "notes", "Collect all the things exposed"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "period", "60s"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.#", "17"),

					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.11714944.active", "true"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.11714944.name", "ReadLatency"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.11714944.tags.#", "4"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.11714944.tags.1313458811", "app:rds"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.11714944.tags.1543130091", "lifecycle:unittests"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.11714944.tags.2964981562", "app:postgresql"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.11714944.tags.4259413593", "source:cloudwatch"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.11714944.type", "numeric"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.11714944.unit", "seconds"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.1436709022.active", "true"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.1436709022.name", "TransactionLogsGeneration"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.1436709022.tags.#", "4"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.1436709022.tags.1313458811", "app:rds"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.1436709022.tags.1543130091", "lifecycle:unittests"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.1436709022.tags.2964981562", "app:postgresql"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.1436709022.tags.4259413593", "source:cloudwatch"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.1436709022.type", "numeric"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.1436709022.unit", ""),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.1444027024.active", "true"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.1444027024.name", "WriteIOPS"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.1444027024.tags.#", "4"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.1444027024.tags.1313458811", "app:rds"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.1444027024.tags.1543130091", "lifecycle:unittests"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.1444027024.tags.2964981562", "app:postgresql"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.1444027024.tags.4259413593", "source:cloudwatch"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.1444027024.type", "numeric"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.1444027024.unit", "iops"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.1604797265.active", "true"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.1604797265.name", "FreeStorageSpace"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.1604797265.tags.#", "4"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.1604797265.tags.1313458811", "app:rds"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.1604797265.tags.1543130091", "lifecycle:unittests"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.1604797265.tags.2964981562", "app:postgresql"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.1604797265.tags.4259413593", "source:cloudwatch"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.1604797265.type", "numeric"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.1604797265.unit", ""),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.1605952596.active", "true"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.1605952596.name", "WriteLatency"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.1605952596.tags.#", "4"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.1605952596.tags.1313458811", "app:rds"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.1605952596.tags.1543130091", "lifecycle:unittests"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.1605952596.tags.2964981562", "app:postgresql"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.1605952596.tags.4259413593", "source:cloudwatch"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.1605952596.type", "numeric"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.1605952596.unit", "seconds"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.1714840347.active", "true"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.1714840347.name", "DatabaseConnections"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.1714840347.tags.#", "4"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.1714840347.tags.1313458811", "app:rds"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.1714840347.tags.1543130091", "lifecycle:unittests"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.1714840347.tags.2964981562", "app:postgresql"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.1714840347.tags.4259413593", "source:cloudwatch"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.1714840347.type", "numeric"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.1714840347.unit", "connections"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.2132240407.active", "true"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.2132240407.name", "FreeableMemory"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.2132240407.tags.#", "4"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.2132240407.tags.1313458811", "app:rds"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.2132240407.tags.1543130091", "lifecycle:unittests"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.2132240407.tags.2964981562", "app:postgresql"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.2132240407.tags.4259413593", "source:cloudwatch"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.2132240407.type", "numeric"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.2132240407.unit", "bytes"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.2395338478.active", "true"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.2395338478.name", "MaximumUsedTransactionIDs"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.2395338478.tags.#", "4"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.2395338478.tags.1313458811", "app:rds"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.2395338478.tags.1543130091", "lifecycle:unittests"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.2395338478.tags.2964981562", "app:postgresql"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.2395338478.tags.4259413593", "source:cloudwatch"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.2395338478.type", "numeric"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.2395338478.unit", ""),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.2968437811.active", "true"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.2968437811.name", "ReadThroughput"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.2968437811.tags.#", "4"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.2968437811.tags.1313458811", "app:rds"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.2968437811.tags.1543130091", "lifecycle:unittests"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.2968437811.tags.2964981562", "app:postgresql"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.2968437811.tags.4259413593", "source:cloudwatch"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.2968437811.type", "numeric"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.2968437811.unit", "bytes"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.3023676211.active", "true"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.3023676211.name", "ReadIOPS"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.3023676211.tags.#", "4"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.3023676211.tags.1313458811", "app:rds"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.3023676211.tags.1543130091", "lifecycle:unittests"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.3023676211.tags.2964981562", "app:postgresql"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.3023676211.tags.4259413593", "source:cloudwatch"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.3023676211.type", "numeric"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.3023676211.unit", "iops"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.3053289991.active", "true"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.3053289991.name", "NetworkReceiveThroughput"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.3053289991.tags.#", "4"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.3053289991.tags.1313458811", "app:rds"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.3053289991.tags.1543130091", "lifecycle:unittests"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.3053289991.tags.2964981562", "app:postgresql"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.3053289991.tags.4259413593", "source:cloudwatch"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.3053289991.type", "numeric"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.3053289991.unit", "bytes"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.3187210440.active", "true"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.3187210440.name", "TransactionLogsDiskUsage"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.3187210440.tags.#", "4"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.3187210440.tags.1313458811", "app:rds"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.3187210440.tags.1543130091", "lifecycle:unittests"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.3187210440.tags.2964981562", "app:postgresql"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.3187210440.tags.4259413593", "source:cloudwatch"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.3187210440.type", "numeric"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.3187210440.unit", "bytes"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.3202842729.active", "true"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.3202842729.name", "CPUUtilization"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.3202842729.tags.#", "4"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.3202842729.tags.1313458811", "app:rds"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.3202842729.tags.1543130091", "lifecycle:unittests"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.3202842729.tags.2964981562", "app:postgresql"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.3202842729.tags.4259413593", "source:cloudwatch"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.3202842729.type", "numeric"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.3202842729.unit", "%"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.3527192726.active", "true"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.3527192726.name", "SwapUsage"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.3527192726.tags.#", "4"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.3527192726.tags.1313458811", "app:rds"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.3527192726.tags.1543130091", "lifecycle:unittests"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.3527192726.tags.2964981562", "app:postgresql"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.3527192726.tags.4259413593", "source:cloudwatch"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.3527192726.type", "numeric"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.3527192726.unit", "bytes"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.3740424181.active", "true"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.3740424181.name", "NetworkTransmitThroughput"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.3740424181.tags.#", "4"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.3740424181.tags.1313458811", "app:rds"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.3740424181.tags.1543130091", "lifecycle:unittests"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.3740424181.tags.2964981562", "app:postgresql"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.3740424181.tags.4259413593", "source:cloudwatch"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.3740424181.type", "numeric"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.3740424181.unit", "bytes"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.53704089.active", "true"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.53704089.name", "DiskQueueDepth"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.53704089.tags.#", "4"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.53704089.tags.1313458811", "app:rds"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.53704089.tags.1543130091", "lifecycle:unittests"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.53704089.tags.2964981562", "app:postgresql"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.53704089.tags.4259413593", "source:cloudwatch"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.53704089.type", "numeric"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.53704089.unit", ""),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.823122139.active", "true"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.823122139.name", "WriteThroughput"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.823122139.tags.#", "4"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.823122139.tags.1313458811", "app:rds"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.823122139.tags.1543130091", "lifecycle:unittests"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.823122139.tags.2964981562", "app:postgresql"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.823122139.tags.4259413593", "source:cloudwatch"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.823122139.type", "numeric"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "stream.823122139.unit", "bytes"),

					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "tags.#", "4"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "tags.2964981562", "app:postgresql"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "tags.1313458811", "app:rds"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "tags.1543130091", "lifecycle:unittests"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "tags.4259413593", "source:cloudwatch"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "target", "atlas-production.us-east-1.rds._aws"),
					resource.TestCheckResourceAttr("circonus_check.rds_metrics", "type", "cloudwatch"),
				),
			},
		},
	})
}

const testAccCirconusCheckCloudWatchConfig = `
variable "cloudwatch_rds_tags" {
  type = "list"
  default = [
    "app:postgresql",
    "app:rds",
    "lifecycle:unittests",
    "source:cloudwatch",
  ]
}

resource "circonus_check" "rds_metrics" {
  active = true
  name = "Terraform test: RDS Metrics via CloudWatch"
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

  stream {
    name = "CPUUtilization"
    tags = [ "${var.cloudwatch_rds_tags}" ]
    type = "numeric"
    unit = "%"
  }

  stream {
    name = "DatabaseConnections"
    tags = [ "${var.cloudwatch_rds_tags}" ]
    type = "numeric"
    unit = "connections"
  }

  stream {
    name = "DiskQueueDepth"
    tags = [ "${var.cloudwatch_rds_tags}" ]
    type = "numeric"
  }

  stream {
    name = "FreeStorageSpace"
    tags = [ "${var.cloudwatch_rds_tags}" ]
    type = "numeric"
  }

  stream {
    name = "FreeableMemory"
    tags = [ "${var.cloudwatch_rds_tags}" ]
    type = "numeric"
    unit = "bytes"
  }

  stream {
    name = "MaximumUsedTransactionIDs"
    tags = [ "${var.cloudwatch_rds_tags}" ]
    type = "numeric"
  }

  stream {
    name = "NetworkReceiveThroughput"
    tags = [ "${var.cloudwatch_rds_tags}" ]
    type = "numeric"
    unit = "bytes"
  }

  stream {
    name = "NetworkTransmitThroughput"
    tags = [ "${var.cloudwatch_rds_tags}" ]
    type = "numeric"
    unit = "bytes"
  }

  stream {
    name = "ReadIOPS"
    tags = [ "${var.cloudwatch_rds_tags}" ]
    type = "numeric"
    unit = "iops"
  }

  stream {
    name = "ReadLatency"
    tags = [ "${var.cloudwatch_rds_tags}" ]
    type = "numeric"
    unit = "seconds"
  }

  stream {
    name = "ReadThroughput"
    tags = [ "${var.cloudwatch_rds_tags}" ]
    type = "numeric"
    unit = "bytes"
  }

  stream {
    name = "SwapUsage"
    tags = [ "${var.cloudwatch_rds_tags}" ]
    type = "numeric"
    unit = "bytes"
  }

  stream {
    name = "TransactionLogsDiskUsage"
    tags = [ "${var.cloudwatch_rds_tags}" ]
    type = "numeric"
    unit = "bytes"
  }

  stream {
    name = "TransactionLogsGeneration"
    tags = [ "${var.cloudwatch_rds_tags}" ]
    type = "numeric"
  }

  stream {
    name = "WriteIOPS"
    tags = [ "${var.cloudwatch_rds_tags}" ]
    type = "numeric"
    unit = "iops"
  }

  stream {
    name = "WriteLatency"
    tags = [ "${var.cloudwatch_rds_tags}" ]
    type = "numeric"
    unit = "seconds"
  }

  stream {
    name = "WriteThroughput"
    tags = [ "${var.cloudwatch_rds_tags}" ]
    type = "numeric"
    unit = "bytes"
  }

  tags = [ "${var.cloudwatch_rds_tags}" ]
}
`
