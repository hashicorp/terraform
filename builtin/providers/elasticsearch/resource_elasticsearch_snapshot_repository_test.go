package elasticsearch

import (
	"context"
	"fmt"
	"testing"

	elastic "gopkg.in/olivere/elastic.v5"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccElasticsearchSnapshotRepository(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckElasticsearchSnapshotRepositoryDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccElasticsearchSnapshotRepository,
				Check: resource.ComposeTestCheckFunc(
					testCheckElasticsearchSnapshotRepositoryExists("elasticsearch_snapshot_repository.test"),
				),
			},
		},
	})
}

func testCheckElasticsearchSnapshotRepositoryExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No snapshot repository ID is set")
		}

		conn := testAccProvider.Meta().(*elastic.Client)
		_, err := conn.SnapshotGetRepository(rs.Primary.ID).Do(context.TODO())
		if err != nil {
			return err
		}

		return nil
	}
}

func testCheckElasticsearchSnapshotRepositoryDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*elastic.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "elasticsearch_snapshot_repository" {
			continue
		}

		_, err := conn.SnapshotGetRepository(rs.Primary.ID).Do(context.TODO())
		if err != nil {
			return nil
		}

		return fmt.Errorf("Snapshot repository %q still exists", rs.Primary.ID)
	}

	return nil
}

var testAccElasticsearchSnapshotRepository = `
resource "elasticsearch_snapshot_repository" "test" {
  name = "terraform-test"
  type = "fs"

  settings {
    location = "/tmp/elasticsearch"
  }
}
`
