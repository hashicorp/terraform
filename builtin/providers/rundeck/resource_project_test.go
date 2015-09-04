package rundeck

import (
	"fmt"
	"testing"

	"github.com/apparentlymart/go-rundeck-api/rundeck"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccProject_basic(t *testing.T) {
	var project rundeck.Project

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccProjectCheckDestroy(&project),
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccProjectConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccProjectCheckExists("rundeck_project.main", &project),
					func(s *terraform.State) error {
						if expected := "terraform-acc-test-basic"; project.Name != expected {
							return fmt.Errorf("wrong name; expected %v, got %v", expected, project.Name)
						}
						if expected := "baz"; project.Config["foo.bar"] != expected {
							return fmt.Errorf("wrong foo.bar config; expected %v, got %v", expected, project.Config["foo.bar"])
						}
						if expected := "file"; project.Config["resources.source.1.type"] != expected {
							return fmt.Errorf("wrong resources.source.1.type config; expected %v, got %v", expected, project.Config["resources.source.1.type"])
						}
						return nil
					},
				),
			},
		},
	})
}

func testAccProjectCheckDestroy(project *rundeck.Project) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*rundeck.Client)
		_, err := client.GetProject(project.Name)
		if err == nil {
			return fmt.Errorf("project still exists")
		}
		if _, ok := err.(*rundeck.NotFoundError); !ok {
			return fmt.Errorf("got something other than NotFoundError (%v) when getting project", err)
		}

		return nil
	}
}

func testAccProjectCheckExists(rn string, project *rundeck.Project) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("resource not found: %s", rn)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("project id not set")
		}

		client := testAccProvider.Meta().(*rundeck.Client)
		gotProject, err := client.GetProject(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("error getting project: %s", err)
		}

		*project = *gotProject

		return nil
	}
}

const testAccProjectConfig_basic = `
resource "rundeck_project" "main" {
  name = "terraform-acc-test-basic"
  description = "Terraform Acceptance Tests Basic Project"

  resource_model_source {
    type = "file"
    config = {
        format = "resourcexml"
        file = "/tmp/terraform-acc-tests.xml"
    }
  }

  extra_config = {
    "foo/bar" = "baz"
  }
}
`
