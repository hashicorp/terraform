package runscope

import (
	"fmt"
	"github.com/ewilde/go-runscope"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"os"
	"testing"
)

func TestAccStep_basic(t *testing.T) {
	teamId := os.Getenv("RUNSCOPE_TEAM_ID")
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckStepDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testRunscopeStepConfigA, teamId),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckStepExists("runscope_step.main_page"),
					resource.TestCheckResourceAttr(
						"runscope_step.main_page", "url", "http://example.com")),
			},
		},
	})
}

func testAccCheckStepDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*runscope.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "runscope_step" {
			continue
		}

		var err error
		bucketId := rs.Primary.Attributes["bucket_id"]
		testId := rs.Primary.Attributes["test_id"]
		err = client.DeleteTestStep(&runscope.TestStep{ID: rs.Primary.ID}, bucketId, testId)

		if err == nil {
			return fmt.Errorf("Record %s still exists", rs.Primary.ID)
		}
	}

	return nil
}

func testAccCheckStepExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Record ID is set")
		}

		client := testAccProvider.Meta().(*runscope.Client)

		var foundRecord *runscope.TestStep
		var err error

		step := new(runscope.TestStep)
		step.ID = rs.Primary.ID
		bucketId := rs.Primary.Attributes["bucket_id"]
		testId := rs.Primary.Attributes["test_id"]

		foundRecord, err = client.ReadTestStep(step, bucketId, testId)

		if err != nil {
			return err
		}

		if foundRecord.ID != rs.Primary.ID {
			return fmt.Errorf("Record not found")
		}

		if len(foundRecord.Variables) != 2 {
			return fmt.Errorf("Expected %d variables, actual %d", 2, len(foundRecord.Variables))
		}

		variable := foundRecord.Variables[1]
		if variable.Name != "httpContentEncoding" {
			return fmt.Errorf("Expected %s variables, actual %s", "httpContentEncoding", variable.Name)
		}

		if len(foundRecord.Assertions) != 2 {
			return fmt.Errorf("Expected %d assertions, actual %d", 2, len(foundRecord.Assertions))
		}

		assertion := foundRecord.Assertions[1]
		if assertion.Source != "response_json" {
			return fmt.Errorf("Expected assertion source %s, actual %s",
				"response_json", assertion.Source)
		}

		if len(foundRecord.Headers) != 2 {
			return fmt.Errorf("Expected %d headers, actual %d", 1, len(foundRecord.Headers))
		}

		if header, ok := foundRecord.Headers["Accept-Encoding"]; ok {
			if len(header) != 2 {
				return fmt.Errorf("Expected %d values for header %s, actual %d",
					2, "Accept-Encoding", len(header))

			}

			if header[1] != "application/xml" {
				return fmt.Errorf("Expected header value %s, actual %s",
					"application/xml", header[1])
			}
		} else {
			return fmt.Errorf("Expected header %s to exist", "Accept-Encoding")
		}

		return nil
	}
}

const testRunscopeStepConfigA = `
resource "runscope_step" "main_page" {
  bucket_id      = "${runscope_bucket.bucket.id}"
  test_id        = "${runscope_test.test.id}"
  step_type      = "request"
  url            = "http://example.com"
  method         = "GET"
  variables      = [
  	{
  	   name     = "httpStatus"
  	   source   = "response_status"
  	},
  	{
  	   name     = "httpContentEncoding"
  	   source   = "response_header"
  	   property = "Content-Encoding"
  	},
  ]
  assertions     = [
  	{
  	   source     = "response_status"
           comparison = "equal_number"
           value      = "200"
  	},
  	{
  	   source     = "response_json"
           comparison = "equal"
           value      = "c5baeb4a-2379-478a-9cda-1b671de77cf9",
           property   = "data.id"
  	},
  ],
  headers        = [
  	{
  		header = "Accept-Encoding",
  		value  = "application/json"
  	},
  	{
  		header = "Accept-Encoding",
  		value  = "application/xml"
  	},
  	{
  		header = "Authorization",
  		value  = "Bearer bb74fe7b-b9f2-48bd-9445-bdc60e1edc6a",
	}
  ]
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
`
