package github

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccGithubRepositoryDeployKey_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckGithubRepositoryDeployKeyDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccGithubRepositoryDeployKeyConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGithubRepositoryDeployKeyExists("github_repository_deploy_key.test_repo_deploy_key"),
					resource.TestCheckResourceAttr("github_repository_deploy_key.test_repo_deploy_key", "read_only", "false"),
					resource.TestCheckResourceAttr("github_repository_deploy_key.test_repo_deploy_key", "repository", testRepo),
					resource.TestCheckResourceAttr("github_repository_deploy_key.test_repo_deploy_key", "key", testAccGithubRepositoryDeployKeytestDeployKey),
					resource.TestCheckResourceAttr("github_repository_deploy_key.test_repo_deploy_key", "title", "title"),
				),
			},
		},
	})
}

func TestAccGithubRepositoryDeployKey_importBasic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckGithubRepositoryDeployKeyDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccGithubRepositoryDeployKeyConfig,
			},
			resource.TestStep{
				ResourceName:      "github_repository_deploy_key.test_repo_deploy_key",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckGithubRepositoryDeployKeyDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*Organization).client

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "github_repository_deploy_key" {
			continue
		}

		o := testAccProvider.Meta().(*Organization).name
		r, i := parseTwoPartID(rs.Primary.ID)
		id, err := strconv.Atoi(i)
		if err != nil {
			return err
		}

		_, resp, err := conn.Repositories.GetKey(context.TODO(), o, r, id)

		if err != nil && resp.Response.StatusCode != 404 {
			return err
		}
		return nil
	}

	return nil
}

func testAccCheckGithubRepositoryDeployKeyExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not Found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No membership ID is set")
		}

		conn := testAccProvider.Meta().(*Organization).client
		o := testAccProvider.Meta().(*Organization).name
		r, i := parseTwoPartID(rs.Primary.ID)
		id, err := strconv.Atoi(i)
		if err != nil {
			return err
		}

		_, _, err = conn.Repositories.GetKey(context.TODO(), o, r, id)
		if err != nil {
			return err
		}

		return nil
	}
}

const testAccGithubRepositoryDeployKeytestDeployKey = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQDnDk1liOxXwE27fjOVVHl6RNVgQznGqGIfhsoa5QNfLOcoWJR3EIv44dSUx1GSvxQ7uR9qBY/i/SEdAbKdupo3Ru5sykc0GqaMRVys+Cin/Lgnl6+ntmTZOudNjIbz10Vfu/dKmexSzqlD3XWzPGXRI5WyKWzvc2XKjRdfnOOzogJpqJ5kh/CN0ZhCzBPTu/b4mJl2ionTEzEeLK2g4Re4IuU/dGoyf0LGLidjmqhSY7dQtL+mfte9m3x/BQTrDf0+AW3kGWXR8EL0EyIJ2HRtHW67YnoOcTAFK0hDCuKgvt78rqdUQ2bVjcsIhNqnvQMPf3ZeZ5bP2JqB9zKaFl8uaRJv+TdxEeFTkgnbYb85M+aBggBYr6xxeb24g7WlU0iPxJ8GmjvCizxe2I1DOJDRDozn1sinKjapNRdJy00iuo46TJC5Wgmid0vnMJ7SMZtubz+btxhoFLt4F4U2JnILaYG4/buJg4H/GkqmkE8G3hr4b4mgsFXBtBFgK6uCTFQSvvV7TyyWkZxHL6DRCxL/Dp0bSj+EM8Tw1K304EvkBEO3rMyvPs4nXL7pepyKWalmUI8U4Qp2xMXSq7fmfZY55osb03MUAtKl0wJ/ykyKOwYWeLbubSVcc6VPx5bXZmnM5bTcZdYW9+vNt86X2F2b0h/sIkGNEPpqQQBzElY+fQ=="

var testAccGithubRepositoryDeployKeyConfig = fmt.Sprintf(`
  resource "github_repository_deploy_key" "test_repo_deploy_key" {
    key = "%s"
    read_only = "false"
    repository = "%s"
    title = "title"
  }
`, testAccGithubRepositoryDeployKeytestDeployKey, testRepo)
