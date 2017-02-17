package github

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

const (
	testTitleSSHKey = "sshkey title"
)

func TestAccGithubRepositorySSHKey_basic(t *testing.T) {
	testSSHKey := os.Getenv("GITHUB_TEST_SSHKEY")
	testAccGithubRepositorySSHKeyConfig := fmt.Sprintf(`
		resource "github_repository_sshkey" "test_repo_sshkey" {
			title = "%s"
			sshkey = "%s"
		}
	`, testTitleSSHKey, testSSHKey)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckGithubRepositorySSHKeyDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccGithubRepositorySSHKeyConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGithubRepositorySSHKeyExists("github_repository_sshkey.test_repo_sshkey"),
				),
			},
		},
	})
}

func testAccCheckGithubRepositorySSHKeyDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*Clients).UserClient

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "github_repository_sshkey" {
			continue
		}

		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return err
		}

		key, _, err := conn.Users.GetKey(id)
		if err != nil {
			return err
		}

		if key != nil {
			return fmt.Errorf("Ssh key still exists")
		}
	}

	return nil
}

func testAccCheckGithubRepositorySSHKeyExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not Found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("ID is not set")
		}

		conn := testAccProvider.Meta().(*Clients).UserClient
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return err
		}

		key, _, err := conn.Users.GetKey(id)
		if err != nil {
			return err
		}

		if key == nil {
			return fmt.Errorf("ssh key does not exist")
		}

		return nil
	}
}

var testSSHKey = `
-----BEGIN PUBLIC KEY-----
MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQCCXIfYjGfxeB1XGMD8rtcC8BKl
s/PScavn2szY04//P0hD+hnrsyFreixT7GwG3AOac6qR7rUr6/w7Pe/bypBUXWNp
Ff4bqLrHezaVbDnMiSMA2m62/o4nX5cHoRqQyIObKfRu8tPwdHKO1pFuZE32Bgyq
qHmOHG4aIsOqSHZfVwIDAQAB
-----END PUBLIC KEY-----
`
