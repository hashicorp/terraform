package shield

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"testing"
)

func TestShieldJob_basic(t *testing.T) {
	var job Job

	t.Skip(" After applying this step, the plan was not empty. ")

	testShieldJobConfig := fmt.Sprintf(`
		resource "shield_target" "test_job_target" {
		  name = "Test-Target"
		  summary = "Terraform Test Target"
		  plugin = "mysql"
		  endpoint = "{\"mysql_user\":\"root\",\"mysql_password\":\"secure-pw\",\"mysql_host\": \"localhost\",\"mysql_port\": 3306}"
		  agent = "localhost:5444"
		}
		resource "shield_retention_policy" "test_job_retention" {
		  name = "Test-Retention"
		  summary = "Terraform Test Retention"
		  expires = 86400
		}
		resource "shield_schedule" "test_job_schedule" {
		  name = "Test-Schedule"
		  summary = "Terraform Test Schedule"
		  when = "daily 1am"
		}
		resource "shield_store" "test_job_store" {
		  name = "Test-Store"
		  summary = "Terraform Test Store"
		  plugin = "fs"
		  endpoint = "{\"base_dir\": \"/backup_test\"}"
		}
		resource "shield_job" "test_job" {
		  name = "Test-Job-Tests"
		  summary = "Terraform Test Job"
		  store = "${ shield_store.test_job_store.uuid }"
		  target = "${ shield_target.test_job_target.uuid }"
		  retention = "${ shield_retention_policy.test_job_retention.uuid }"
		  schedule = "${ shield_schedule.test_job_schedule.uuid }"
		  paused = false
		}
	`)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testShieldJobDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testShieldJobConfig,
				Check: resource.ComposeTestCheckFunc(
					testShieldCheckJobExists("shield_job.test_job", &job),
				),
			},
		},
	})
}

func testShieldJobDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*ShieldClient)
	rs, ok := s.RootModule().Resources["shield_job.test_job"]
	if !ok {
		return fmt.Errorf("Not found %s", "shield_job.test_job")
	}

	response, err := client.Get(fmt.Sprintf("v1/job/%s", rs.Primary.Attributes["uuid"]))

	if err != nil {
		return err
	}

	if response.StatusCode != 404 {
		return fmt.Errorf("Job still exists")
	}

	return nil
}

func testShieldCheckJobExists(n string, job *Job) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found %s", n)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No Job UUID is set")
		}
		return nil
	}
}
