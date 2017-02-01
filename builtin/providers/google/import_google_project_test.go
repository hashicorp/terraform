package google

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccGoogleProject_importBasic(t *testing.T) {
	resourceName := "google_project.acceptance"
	projectId := "terraform-" + acctest.RandString(10)
	conf := testAccGoogleProject_import(projectId, org, pname)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: conf,
			},

			resource.TestStep{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccGoogleProject_import(pid, orgId, projectName string) string {
	return fmt.Sprintf(`
resource "google_project" "acceptance" {
    project_id = "%s"
    org_id = "%s"
    name = "%s"
}`, pid, orgId, projectName)
}
