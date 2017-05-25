package google

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccBigQueryDataset_basic(t *testing.T) {
	datasetID := fmt.Sprintf("tf_test_%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckBigQueryDatasetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccBigQueryDataset(datasetID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBigQueryDatasetExists(
						"google_bigquery_dataset.test"),
				),
			},

			{
				Config: testAccBigQueryDatasetUpdated(datasetID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBigQueryDatasetExists(
						"google_bigquery_dataset.test"),
				),
			},
		},
	})
}

func testAccCheckBigQueryDatasetDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "google_bigquery_dataset" {
			continue
		}

		_, err := config.clientBigQuery.Datasets.Get(config.Project, rs.Primary.Attributes["dataset_id"]).Do()
		if err == nil {
			return fmt.Errorf("Dataset still exists")
		}
	}

	return nil
}

func testAccCheckBigQueryDatasetExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		config := testAccProvider.Meta().(*Config)

		found, err := config.clientBigQuery.Datasets.Get(config.Project, rs.Primary.Attributes["dataset_id"]).Do()
		if err != nil {
			return err
		}

		if found.Id != rs.Primary.ID {
			return fmt.Errorf("Dataset not found")
		}

		return nil
	}
}

func testAccBigQueryDataset(datasetID string) string {
	return fmt.Sprintf(`
resource "google_bigquery_dataset" "test" {
  dataset_id                  = "%s"
  friendly_name               = "foo"
  description                 = "This is a foo description"
  location                    = "EU"
  default_table_expiration_ms = 3600000

  labels {
    env                         = "foo"
    default_table_expiration_ms = 3600000
  }
}`, datasetID)
}

func testAccBigQueryDatasetUpdated(datasetID string) string {
	return fmt.Sprintf(`
resource "google_bigquery_dataset" "test" {
  dataset_id                  = "%s"
  friendly_name               = "bar"
  description                 = "This is a bar description"
  location                    = "EU"
  default_table_expiration_ms = 7200000

  labels {
    env                         = "bar"
    default_table_expiration_ms = 7200000
  }
}`, datasetID)
}
