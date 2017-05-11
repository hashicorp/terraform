package github

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/google/go-github/github"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccGithubBranchProtection_basic(t *testing.T) {
	var protection github.Protection

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccGithubBranchProtectionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccGithubBranchProtectionConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGithubProtectedBranchExists("github_branch_protection.master", &protection),
					testAccCheckGithubBranchProtectionRequiredStatusChecks(&protection, true, true, []string{"github/foo"}),
					testAccCheckGithubBranchProtectionRestrictions(&protection, []string{testUser}, []string{}),
					resource.TestCheckResourceAttr("github_branch_protection.master", "repository", testRepo),
					resource.TestCheckResourceAttr("github_branch_protection.master", "branch", "master"),
					resource.TestCheckResourceAttr("github_branch_protection.master", "required_status_checks.0.include_admins", "true"),
					resource.TestCheckResourceAttr("github_branch_protection.master", "required_status_checks.0.strict", "true"),
					resource.TestCheckResourceAttr("github_branch_protection.master", "required_status_checks.0.contexts.#", "1"),
					resource.TestCheckResourceAttr("github_branch_protection.master", "required_status_checks.0.contexts.0", "github/foo"),
					resource.TestCheckResourceAttr("github_branch_protection.master", "required_pull_request_reviews.0.include_admins", "true"),
					resource.TestCheckResourceAttr("github_branch_protection.master", "restrictions.0.users.#", "1"),
					resource.TestCheckResourceAttr("github_branch_protection.master", "restrictions.0.users.0", testUser),
					resource.TestCheckResourceAttr("github_branch_protection.master", "restrictions.0.teams.#", "0"),
				),
			},
			{
				Config: testAccGithubBranchProtectionUpdateConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGithubProtectedBranchExists("github_branch_protection.master", &protection),
					testAccCheckGithubBranchProtectionRequiredStatusChecks(&protection, false, false, []string{"github/bar"}),
					testAccCheckGithubBranchProtectionNoRestrictionsExist(&protection),
					resource.TestCheckResourceAttr("github_branch_protection.master", "repository", testRepo),
					resource.TestCheckResourceAttr("github_branch_protection.master", "branch", "master"),
					resource.TestCheckResourceAttr("github_branch_protection.master", "required_status_checks.0.include_admins", "false"),
					resource.TestCheckResourceAttr("github_branch_protection.master", "required_status_checks.0.strict", "false"),
					resource.TestCheckResourceAttr("github_branch_protection.master", "required_status_checks.0.contexts.#", "1"),
					resource.TestCheckResourceAttr("github_branch_protection.master", "required_status_checks.0.contexts.0", "github/bar"),
					resource.TestCheckResourceAttr("github_branch_protection.master", "required_pull_request_reviews.#", "0"),
					resource.TestCheckResourceAttr("github_branch_protection.master", "restrictions.#", "0"),
				),
			},
		},
	})
}

func TestAccGithubBranchProtection_importBasic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccGithubBranchProtectionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccGithubBranchProtectionConfig,
			},
			{
				ResourceName:      "github_branch_protection.master",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckGithubProtectedBranchExists(n string, protection *github.Protection) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not Found: %s", n)
		}

		if rs.Primary.ID != "test-repo:master" {
			return fmt.Errorf("Expected ID to be %v, got %v", "test-repo:master", rs.Primary.ID)
		}

		conn := testAccProvider.Meta().(*Organization).client
		o := testAccProvider.Meta().(*Organization).name
		r, b := parseTwoPartID(rs.Primary.ID)

		githubProtection, _, err := conn.Repositories.GetBranchProtection(context.TODO(), o, r, b)
		if err != nil {
			return err
		}

		*protection = *githubProtection
		return nil
	}
}

func testAccCheckGithubBranchProtectionRequiredStatusChecks(protection *github.Protection, expectedIncludeAdmins bool, expectedStrict bool, expectedContexts []string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rsc := protection.RequiredStatusChecks
		if rsc == nil {
			return fmt.Errorf("Expected RequiredStatusChecks to be present, but was nil")
		}

		if rsc.IncludeAdmins != expectedIncludeAdmins {
			return fmt.Errorf("Expected RequiredStatusChecks.IncludeAdmins to be %v, got %v", expectedIncludeAdmins, rsc.IncludeAdmins)
		}
		if rsc.Strict != expectedStrict {
			return fmt.Errorf("Expected RequiredStatusChecks.Strict to be %v, got %v", expectedStrict, rsc.Strict)
		}

		if !reflect.DeepEqual(rsc.Contexts, expectedContexts) {
			return fmt.Errorf("Expected RequiredStatusChecks.Contexts to be %v, got %v", expectedContexts, rsc.Contexts)
		}

		return nil
	}
}

func testAccCheckGithubBranchProtectionRestrictions(protection *github.Protection, expectedUserLogins []string, expectedTeamNames []string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		restrictions := protection.Restrictions
		if restrictions == nil {
			return fmt.Errorf("Expected Restrictions to be present, but was nil")
		}

		userLogins := []string{}
		for _, u := range restrictions.Users {
			userLogins = append(userLogins, *u.Login)
		}
		if !reflect.DeepEqual(userLogins, expectedUserLogins) {
			return fmt.Errorf("Expected Restrictions.Users to be %v, got %v", expectedUserLogins, userLogins)
		}

		teamLogins := []string{}
		for _, t := range restrictions.Teams {
			teamLogins = append(teamLogins, *t.Name)
		}
		if !reflect.DeepEqual(teamLogins, expectedTeamNames) {
			return fmt.Errorf("Expected Restrictions.Teams to be %v, got %v", expectedTeamNames, teamLogins)
		}

		return nil
	}
}

func testAccCheckGithubBranchProtectionNoRestrictionsExist(protection *github.Protection) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if protection.Restrictions != nil {
			return fmt.Errorf("Expected Restrictions to be nil, but was %v", protection.Restrictions)
		}

		return nil

	}
}

func testAccGithubBranchProtectionDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*Organization).client

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "github_branch_protection" {
			continue
		}

		o := testAccProvider.Meta().(*Organization).name
		r, b := parseTwoPartID(rs.Primary.ID)
		protection, res, err := conn.Repositories.GetBranchProtection(context.TODO(), o, r, b)

		if err == nil {
			if protection != nil {
				return fmt.Errorf("Branch protection still exists")
			}
		}
		if res.StatusCode != 404 {
			return err
		}
		return nil
	}
	return nil
}

var testAccGithubBranchProtectionConfig string = fmt.Sprintf(`
resource "github_branch_protection" "master" {
  repository = "%s"
  branch     = "master"

  required_status_checks = {
    include_admins = true
    strict         = true
    contexts       = ["github/foo"]
  }

  required_pull_request_reviews {
    include_admins = true
  }

  restrictions {
    users = ["%s"]
  }
}
`, testRepo, testUser)

var testAccGithubBranchProtectionUpdateConfig string = fmt.Sprintf(`
resource "github_branch_protection" "master" {
  repository = "%s"
  branch     = "master"

  required_status_checks = {
    include_admins = false
    strict         = false
    contexts       = ["github/bar"]
  }
}
`, testRepo)
