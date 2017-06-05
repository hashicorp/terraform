package librato

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/henrikhodne/go-librato/librato"
)

func TestAccLibratoMetrics(t *testing.T) {
	var metric librato.Metric

	name := fmt.Sprintf("tftest-metric-%s", acctest.RandString(10))
	typ := "counter"
	desc1 := fmt.Sprintf("A test %s metric", typ)
	desc2 := fmt.Sprintf("An updated test %s metric", typ)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckLibratoMetricDestroy,
		Steps: []resource.TestStep{
			{
				Config: counterMetricConfig(name, typ, desc1),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckLibratoMetricExists("librato_metric.foobar", &metric),
					testAccCheckLibratoMetricName(&metric, name),
					testAccCheckLibratoMetricType(&metric, typ),
					resource.TestCheckResourceAttr(
						"librato_metric.foobar", "name", name),
				),
			},
			{
				PreConfig: sleep(t, 5),
				Config:    counterMetricConfig(name, typ, desc2),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckLibratoMetricExists("librato_metric.foobar", &metric),
					testAccCheckLibratoMetricName(&metric, name),
					testAccCheckLibratoMetricType(&metric, typ),
					testAccCheckLibratoMetricDescription(&metric, desc2),
					resource.TestCheckResourceAttr(
						"librato_metric.foobar", "name", name),
				),
			},
		},
	})

	name = fmt.Sprintf("tftest-metric-%s", acctest.RandString(10))
	typ = "gauge"
	desc1 = fmt.Sprintf("A test %s metric", typ)
	desc2 = fmt.Sprintf("An updated test %s metric", typ)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckLibratoMetricDestroy,
		Steps: []resource.TestStep{
			{
				Config: gaugeMetricConfig(name, typ, desc1),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckLibratoMetricExists("librato_metric.foobar", &metric),
					testAccCheckLibratoMetricName(&metric, name),
					testAccCheckLibratoMetricType(&metric, typ),
					resource.TestCheckResourceAttr(
						"librato_metric.foobar", "name", name),
				),
			},
			{
				PreConfig: sleep(t, 5),
				Config:    gaugeMetricConfig(name, typ, desc2),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckLibratoMetricExists("librato_metric.foobar", &metric),
					testAccCheckLibratoMetricName(&metric, name),
					testAccCheckLibratoMetricType(&metric, typ),
					testAccCheckLibratoMetricDescription(&metric, desc2),
					resource.TestCheckResourceAttr(
						"librato_metric.foobar", "name", name),
				),
			},
		},
	})

	name = fmt.Sprintf("tftest-metric-%s", acctest.RandString(10))
	typ = "composite"
	desc1 = fmt.Sprintf("A test %s metric", typ)
	desc2 = fmt.Sprintf("An updated test %s metric", typ)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckLibratoMetricDestroy,
		Steps: []resource.TestStep{
			{
				Config: compositeMetricConfig(name, typ, desc1),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckLibratoMetricExists("librato_metric.foobar", &metric),
					testAccCheckLibratoMetricName(&metric, name),
					testAccCheckLibratoMetricType(&metric, typ),
					resource.TestCheckResourceAttr(
						"librato_metric.foobar", "name", name),
				),
			},
			{
				PreConfig: sleep(t, 5),
				Config:    compositeMetricConfig(name, typ, desc2),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckLibratoMetricExists("librato_metric.foobar", &metric),
					testAccCheckLibratoMetricName(&metric, name),
					testAccCheckLibratoMetricType(&metric, typ),
					testAccCheckLibratoMetricDescription(&metric, desc2),
					resource.TestCheckResourceAttr(
						"librato_metric.foobar", "name", name),
				),
			},
		},
	})
}

func testAccCheckLibratoMetricDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*librato.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "librato_metric" {
			continue
		}

		_, _, err := client.Metrics.Get(rs.Primary.ID)
		if err == nil {
			return fmt.Errorf("Metric still exists")
		}
	}

	return nil
}

func testAccCheckLibratoMetricName(metric *librato.Metric, name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if metric.Name == nil || *metric.Name != name {
			return fmt.Errorf("Bad name: %s", *metric.Name)
		}

		return nil
	}
}

func testAccCheckLibratoMetricDescription(metric *librato.Metric, desc string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if metric.Description == nil || *metric.Description != desc {
			return fmt.Errorf("Bad description: %s", *metric.Description)
		}

		return nil
	}
}

func testAccCheckLibratoMetricType(metric *librato.Metric, wantType string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if metric.Type == nil || *metric.Type != wantType {
			return fmt.Errorf("Bad metric type: %s", *metric.Type)
		}

		return nil
	}
}

func testAccCheckLibratoMetricExists(n string, metric *librato.Metric) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Metric ID is set")
		}

		client := testAccProvider.Meta().(*librato.Client)

		foundMetric, _, err := client.Metrics.Get(rs.Primary.ID)

		if err != nil {
			return err
		}

		if foundMetric.Name == nil || *foundMetric.Name != rs.Primary.ID {
			return fmt.Errorf("Metric not found")
		}

		*metric = *foundMetric

		return nil
	}
}

func counterMetricConfig(name, typ, desc string) string {
	return strings.TrimSpace(fmt.Sprintf(`
    resource "librato_metric" "foobar" {
        name = "%s"
        type = "%s"
        description = "%s"
        attributes {
          display_stacked = true
        }
    }`, name, typ, desc))
}

func gaugeMetricConfig(name, typ, desc string) string {
	return strings.TrimSpace(fmt.Sprintf(`
    resource "librato_metric" "foobar" {
        name = "%s"
        type = "%s"
        description = "%s"
        attributes {
          display_stacked = true
        }
    }`, name, typ, desc))
}

func compositeMetricConfig(name, typ, desc string) string {
	return strings.TrimSpace(fmt.Sprintf(`
    resource "librato_metric" "foobar" {
        name = "%s"
        type = "%s"
        description = "%s"
        composite = "s(\"librato.cpu.percent.user\", {\"environment\" : \"prod\", \"service\": \"api\"})"
        attributes {
          display_stacked = true
        }
    }`, name, typ, desc))
}
