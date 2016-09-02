package google

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccBigqueryDatasetCreate(t *testing.T) {

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckBigQueryDatasetDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccBigQueryDataset,
				Check: resource.ComposeTestCheckFunc(
					testAccBigQueryDatasetExists(
						"google_bigquery_dataset.foobar"),
				),
			},
		},
	})
}

func testAccCheckBigQueryDatasetDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "google_bigquery_dataset" {
			continue
		}

		config := testAccProvider.Meta().(*Config)
		_, err := config.clientBigQuery.Datasets.Get(config.Project, rs.Primary.Attributes["datasetId"]).Do()
		if err != nil {
			fmt.Errorf("Dataset still present")
		}
	}

	return nil
}

func testAccBigQueryDatasetExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}
		config := testAccProvider.Meta().(*Config)
		_, err := config.clientBigQuery.Datasets.Get(config.Project, rs.Primary.Attributes["datasetId"]).Do()
		if err != nil {
			fmt.Errorf("BigQuery Dataset not present")
		}

		return nil
	}
}

const testAccBigQueryDataset = `
resource "google_bigquery_dataset" "foobar" {
	datasetId = "foobar"
	friendlyName = "hi"
}`
