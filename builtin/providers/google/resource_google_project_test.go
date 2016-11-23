package google

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"google.golang.org/api/cloudresourcemanager/v1"
)

var (
	org = multiEnvSearch([]string{
		"GOOGLE_ORG",
	})

	pname          = "Terraform Acceptance Tests"
	originalPolicy *cloudresourcemanager.Policy
)

func multiEnvSearch(ks []string) string {
	for _, k := range ks {
		if v := os.Getenv(k); v != "" {
			return v
		}
	}
	return ""
}

// Test that a Project resource can be created and an IAM policy
// associated
func TestAccGoogleProject_create(t *testing.T) {
	pid := "terraform-" + acctest.RandString(10)
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			// This step imports an existing project
			resource.TestStep{
				Config: testAccGoogleProject_create(pid, pname, org),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGoogleProjectExists("google_project.acceptance", pid),
				),
			},
		},
	})
}

func testAccCheckGoogleProjectExists(r, pid string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[r]
		if !ok {
			return fmt.Errorf("Not found: %s", r)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		if rs.Primary.ID != pid {
			return fmt.Errorf("Expected project %q to match ID %q in state", pid, rs.Primary.ID)
		}

		return nil
	}
}

func testAccGoogleProjectImportExisting(pid string) string {
	return fmt.Sprintf(`
resource "google_project" "acceptance" {
    project_id = "%s"

}
`, pid)
}

func testAccGoogleProjectImportExistingWithIam(pid string) string {
	return fmt.Sprintf(`
resource "google_project" "acceptance" {
    project_id = "%v"
    policy_data = "${data.google_iam_policy.admin.policy_data}"
}
data "google_iam_policy" "admin" {
  binding {
    role = "roles/storage.objectViewer"
    members = [
      "user:evanbrown@google.com",
    ]
  }
  binding {
    role = "roles/compute.instanceAdmin"
    members = [
      "user:evanbrown@google.com",
      "user:evandbrown@gmail.com",
    ]
  }
}`, pid)
}
