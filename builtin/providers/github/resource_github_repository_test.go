package github

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-github/github"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccGithubRepository_basic(t *testing.T) {
	var repo github.Repository
	randString := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	name := fmt.Sprintf("tf-acc-test-%s", randString)
	description := fmt.Sprintf("Terraform acceptance tests %s", randString)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckGithubRepositoryDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccGithubRepositoryConfig(randString),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGithubRepositoryExists("github_repository.foo", &repo),
					testAccCheckGithubRepositoryAttributes(&repo, &testAccGithubRepositoryExpectedAttributes{
						Name:          name,
						Description:   description,
						Homepage:      "http://example.com/",
						HasIssues:     true,
						HasWiki:       true,
						HasDownloads:  true,
						DefaultBranch: "master",
					}),
				),
			},
			{
				Config: testAccGithubRepositoryUpdateConfig(randString),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGithubRepositoryExists("github_repository.foo", &repo),
					testAccCheckGithubRepositoryAttributes(&repo, &testAccGithubRepositoryExpectedAttributes{
						Name:          name,
						Description:   "Updated " + description,
						Homepage:      "http://example.com/",
						DefaultBranch: "master",
					}),
				),
			},
		},
	})
}

func TestAccGithubRepository_importBasic(t *testing.T) {
	randString := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckGithubRepositoryDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccGithubRepositoryConfig(randString),
			},
			{
				ResourceName:      "github_repository.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckGithubRepositoryExists(n string, repo *github.Repository) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not Found: %s", n)
		}

		repoName := rs.Primary.ID
		if repoName == "" {
			return fmt.Errorf("No repository name is set")
		}

		org := testAccProvider.Meta().(*Organization)
		conn := org.client
		gotRepo, _, err := conn.Repositories.Get(context.TODO(), org.name, repoName)
		if err != nil {
			return err
		}
		*repo = *gotRepo
		return nil
	}
}

type testAccGithubRepositoryExpectedAttributes struct {
	Name         string
	Description  string
	Homepage     string
	Private      bool
	HasIssues    bool
	HasWiki      bool
	HasDownloads bool

	DefaultBranch string
}

func testAccCheckGithubRepositoryAttributes(repo *github.Repository, want *testAccGithubRepositoryExpectedAttributes) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if *repo.Name != want.Name {
			return fmt.Errorf("got repo %q; want %q", *repo.Name, want.Name)
		}
		if *repo.Description != want.Description {
			return fmt.Errorf("got description %q; want %q", *repo.Description, want.Description)
		}
		if *repo.Homepage != want.Homepage {
			return fmt.Errorf("got homepage URL %q; want %q", *repo.Homepage, want.Homepage)
		}
		if *repo.Private != want.Private {
			return fmt.Errorf("got private %#v; want %#v", *repo.Private, want.Private)
		}
		if *repo.HasIssues != want.HasIssues {
			return fmt.Errorf("got has issues %#v; want %#v", *repo.HasIssues, want.HasIssues)
		}
		if *repo.HasWiki != want.HasWiki {
			return fmt.Errorf("got has wiki %#v; want %#v", *repo.HasWiki, want.HasWiki)
		}
		if *repo.HasDownloads != want.HasDownloads {
			return fmt.Errorf("got has downloads %#v; want %#v", *repo.HasDownloads, want.HasDownloads)
		}

		if *repo.DefaultBranch != want.DefaultBranch {
			return fmt.Errorf("got default branch %q; want %q", *repo.DefaultBranch, want.DefaultBranch)
		}

		// For the rest of these, we just want to make sure they've been
		// populated with something that seems somewhat reasonable.
		if !strings.HasSuffix(*repo.FullName, "/"+want.Name) {
			return fmt.Errorf("got full name %q; want to end with '/%s'", *repo.FullName, want.Name)
		}
		if !strings.HasSuffix(*repo.CloneURL, "/"+want.Name+".git") {
			return fmt.Errorf("got Clone URL %q; want to end with '/%s.git'", *repo.CloneURL, want.Name)
		}
		if !strings.HasPrefix(*repo.CloneURL, "https://") {
			return fmt.Errorf("got Clone URL %q; want to start with 'https://'", *repo.CloneURL)
		}
		if !strings.HasSuffix(*repo.SSHURL, "/"+want.Name+".git") {
			return fmt.Errorf("got SSH URL %q; want to end with '/%s.git'", *repo.SSHURL, want.Name)
		}
		if !strings.HasPrefix(*repo.SSHURL, "git@github.com:") {
			return fmt.Errorf("got SSH URL %q; want to start with 'git@github.com:'", *repo.SSHURL)
		}
		if !strings.HasSuffix(*repo.GitURL, "/"+want.Name+".git") {
			return fmt.Errorf("got git URL %q; want to end with '/%s.git'", *repo.GitURL, want.Name)
		}
		if !strings.HasPrefix(*repo.GitURL, "git://") {
			return fmt.Errorf("got git URL %q; want to start with 'git://'", *repo.GitURL)
		}
		if !strings.HasSuffix(*repo.SVNURL, "/"+want.Name) {
			return fmt.Errorf("got svn URL %q; want to end with '/%s'", *repo.SVNURL, want.Name)
		}
		if !strings.HasPrefix(*repo.SVNURL, "https://") {
			return fmt.Errorf("got svn URL %q; want to start with 'https://'", *repo.SVNURL)
		}

		return nil
	}
}

func testAccCheckGithubRepositoryDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*Organization).client
	orgName := testAccProvider.Meta().(*Organization).name

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "github_repository" {
			continue
		}

		gotRepo, resp, err := conn.Repositories.Get(context.TODO(), orgName, rs.Primary.ID)
		if err == nil {
			if gotRepo != nil && *gotRepo.Name == rs.Primary.ID {
				return fmt.Errorf("Repository %s/%s still exists", orgName, *gotRepo.Name)
			}
		}
		if resp.StatusCode != 404 {
			return err
		}
		return nil
	}
	return nil
}

func testAccGithubRepositoryConfig(randString string) string {
	return fmt.Sprintf(`
resource "github_repository" "foo" {
  name = "tf-acc-test-%s"
  description = "Terraform acceptance tests %s"
  homepage_url = "http://example.com/"

  # So that acceptance tests can be run in a github organization
  # with no billing
  private = false

  has_issues = true
  has_wiki = true
  has_downloads = true
}
`, randString, randString)
}

func testAccGithubRepositoryUpdateConfig(randString string) string {
	return fmt.Sprintf(`
resource "github_repository" "foo" {
  name = "tf-acc-test-%s"
  description = "Updated Terraform acceptance tests %s"
  homepage_url = "http://example.com/"

  # So that acceptance tests can be run in a github organization
  # with no billing
  private = false

  has_issues = false
  has_wiki = false
  has_downloads = false
}
`, randString, randString)
}
