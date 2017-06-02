package runscope

import (
	"fmt"
	"github.com/ewilde/go-runscope"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"os"
	"testing"
)

func TestAccSchedule_basic(t *testing.T) {
	teamId := os.Getenv("RUNSCOPE_TEAM_ID")
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckScheduleDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testRunscopeScheduleConfigA, teamId),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScheduleExists("runscope_schedule.daily"),
					resource.TestCheckResourceAttr(
						"runscope_schedule.daily", "note", "This is a daily schedule"),
					resource.TestCheckResourceAttr(
						"runscope_schedule.daily", "interval", "1d")),
			},
		},
	})
}

func testAccCheckScheduleDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*runscope.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "runscope_schedule" {
			continue
		}

		var err error
		bucketId := rs.Primary.Attributes["bucket_id"]
		testId := rs.Primary.Attributes["test_id"]
		err = client.DeleteSchedule(&runscope.Schedule{ID: rs.Primary.ID}, bucketId, testId)

		if err == nil {
			return fmt.Errorf("Record %s still exists", rs.Primary.ID)
		}
	}

	return nil
}

func testAccCheckScheduleExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Record ID is set")
		}

		client := testAccProvider.Meta().(*runscope.Client)

		var foundRecord *runscope.Schedule
		var err error

		schedule := new(runscope.Schedule)
		schedule.ID = rs.Primary.ID
		bucketId := rs.Primary.Attributes["bucket_id"]
		testId := rs.Primary.Attributes["test_id"]

		foundRecord, err = client.ReadSchedule(schedule, bucketId, testId)

		if err != nil {
			return err
		}

		if foundRecord.ID != rs.Primary.ID {
			return fmt.Errorf("Record not found")
		}

		return nil
	}
}

const testRunscopeScheduleConfigA = `
resource "runscope_schedule" "daily" {
  bucket_id      = "${runscope_bucket.bucket.id}"
  test_id        = "${runscope_test.test.id}"
  interval       = "1d"
  note           = "This is a daily schedule"
  environment_id = "${runscope_environment.environment.id}"
}

resource "runscope_test" "test" {
  bucket_id   = "${runscope_bucket.bucket.id}"
  name        = "runscope test"
  description = "This is a test test..."
}

resource "runscope_bucket" "bucket" {
  name      = "terraform-provider-test"
  team_uuid = "%s"
}

resource "runscope_environment" "environment" {
  bucket_id = "${runscope_bucket.bucket.id}"
  name      = "test-environment"

  initial_variables {
    var1 = "true",
    var2 = "value2"
  }
}
`
