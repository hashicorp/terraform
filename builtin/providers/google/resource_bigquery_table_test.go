package google

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccBigQueryTable_Basic(t *testing.T) {
	datasetID := fmt.Sprintf("tf_test_%s", acctest.RandString(10))
	tableID := fmt.Sprintf("tf_test_%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckBigQueryTableDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccBigQueryTable(datasetID, tableID),
				Check: resource.ComposeTestCheckFunc(
					testAccBigQueryTableExists(
						"google_bigquery_table.test"),
				),
			},

			{
				Config: testAccBigQueryTableUpdated(datasetID, tableID),
				Check: resource.ComposeTestCheckFunc(
					testAccBigQueryTableExists(
						"google_bigquery_table.test"),
				),
			},
		},
	})
}

func testAccCheckBigQueryTableDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "google_bigquery_table" {
			continue
		}

		config := testAccProvider.Meta().(*Config)
		_, err := config.clientBigQuery.Tables.Get(config.Project, rs.Primary.Attributes["dataset_id"], rs.Primary.Attributes["name"]).Do()
		if err == nil {
			return fmt.Errorf("Table still present")
		}
	}

	return nil
}

func testAccBigQueryTableExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}
		config := testAccProvider.Meta().(*Config)
		_, err := config.clientBigQuery.Tables.Get(config.Project, rs.Primary.Attributes["dataset_id"], rs.Primary.Attributes["name"]).Do()
		if err != nil {
			return fmt.Errorf("BigQuery Table not present")
		}

		return nil
	}
}

func testAccBigQueryTable(datasetID, tableID string) string {
	return fmt.Sprintf(`
resource "google_bigquery_dataset" "test" {
  dataset_id = "%s"
}

resource "google_bigquery_table" "test" {
  table_id   = "%s"
  dataset_id = "${google_bigquery_dataset.test.dataset_id}"

  time_partitioning {
    type = "DAY"
  }

  schema = <<EOH
[
  {
    "name": "city",
    "type": "RECORD",
    "fields": [
      {
        "name": "id",
        "type": "INTEGER"
      },
      {
        "name": "coord",
        "type": "RECORD",
        "fields": [
          {
            "name": "lon",
            "type": "FLOAT"
          }
        ]
      }
    ]
  }
]
EOH
}`, datasetID, tableID)
}

func testAccBigQueryTableUpdated(datasetID, tableID string) string {
	return fmt.Sprintf(`
resource "google_bigquery_dataset" "test" {
  dataset_id = "%s"
}

resource "google_bigquery_table" "test" {
  table_id   = "%s"
  dataset_id = "${google_bigquery_dataset.test.dataset_id}"

  time_partitioning {
    type = "DAY"
  }

  schema = <<EOH
[
  {
    "name": "city",
    "type": "RECORD",
    "fields": [
      {
        "name": "id",
        "type": "INTEGER"
      },
      {
        "name": "coord",
        "type": "RECORD",
        "fields": [
          {
            "name": "lon",
            "type": "FLOAT"
          },
          {
            "name": "lat",
            "type": "FLOAT"
          }
        ]
      }
    ]
  },
  {
    "name": "country",
    "type": "RECORD",
    "fields": [
      {
        "name": "id",
        "type": "INTEGER"
      },
      {
        "name": "name",
        "type": "STRING"
      }
    ]
  }
]
EOH
}`, datasetID, tableID)
}
