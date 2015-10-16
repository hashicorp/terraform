package google

import (
	"os"
	"fmt"
	"time"
	"testing"
	"math/rand"

	"github.com/22acacia/terraform-gcloud"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccDataflowCreate(t *testing.T) {

	if os.Getenv("GOOGLE_GCLOUD_TESTS") == "TRUE" {
		resource.Test(t, resource.TestCase{
			PreCheck:     func() { testAccPreCheck(t) },
			Providers:    testAccProviders,
			CheckDestroy: testAccCheckDataflowDestroy,
			Steps: []resource.TestStep{
				resource.TestStep{
					Config: testAccDataflow,
					Check: resource.ComposeTestCheckFunc(
						testAccDataflowExists(
							"google_dataflow.foobar"),
					),
				},
			},
		})
	}
}

var disallowedDeletedStates = map[string]bool {
	"JOB_STATE_RUNNING": true,
	"JOB_STATE_UNKNOWN": true,
	"JOB_STATE_FAILED": true,
	"JOB_STATE_UPDATED": true,
}
func testAccCheckDataflowDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "google_dataflow" {
			continue
		}

		jobstate, err := terraformGcloud.ReadDataflow(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("Failed to read dataflow list")
		}

		if jobstate == "" {
			return fmt.Errorf("Dataflow jobs never started ")
		}

		if _, ok := disallowedDeletedStates[jobstate]; ok {
			return fmt.Errorf("Dataflow job in disallowed state: %q", jobstate)
		}
	}

	return nil
}

var disallowedCreatedStates = map[string] bool {
	"JOB_STATE_FAILED": true,
	"JOB_STATE_STOPPED": true,
	"JOB_STATE_UPDATED": true,
	"JOB_STATE_UNKNOWN": true,
}

func testAccDataflowExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		jobstate, err := terraformGcloud.ReadDataflow(rs.Primary.Attributes["jobids.0"])
		if err != nil {
			return fmt.Errorf("Command line read has errored: %q with rs.Primary hash: %q", err, rs.Primary)
		}

		if jobstate == "" {
			return fmt.Errorf("Dataflow jobs never started")
		}

		if _, ok := disallowedCreatedStates[jobstate]; ok {
			return fmt.Errorf("Dataflow job in disallowed state: %q", jobstate)
		}

		return nil
	}
}

var randInt = rand.New(rand.NewSource(time.Now().UnixNano())).Int()

var testAccDataflow = fmt.Sprintf(`
resource "google_storage_bucket" "bucket" {
	name = "tf-test-bucket-%d"
	force_destroy = true
}
resource "google_dataflow" "foobar" {
	name = "foobar-%d"
	jarfile = "test-fixtures/google-cloud-dataflow-java-examples-all-bundled-1.1.1-SNAPSHOT.jar"
	class = "com.google.cloud.dataflow.examples.WordCount"
	optional_args = {
		stagingLocation = "gs://${google_storage_bucket.bucket.name}"
		runner = "DataflowPipelineRunner"
	}
}
`, randInt, randInt)
