package heroku

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/cyberdelia/heroku-go/v3"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccHerokuAuth_Basic(t *testing.T) {
	var auth heroku.OAuthAuthorization

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHerokuAuthorizationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuAuthorizationConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuAuthorizationExists("heroku_authorization.foobar", &auth),
					testAccCheckHerokuAuthorizationHasToken(&auth),
				),
			},
		},
	})
}

func TestAccHerokuAuth_Scopes(t *testing.T) {
	var auth heroku.OAuthAuthorization

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHerokuAuthorizationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuAuthorizationConfig_scopes(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuAuthorizationExists("heroku_authorization.foobar", &auth),
					testAccCheckHerokuAuthorizationHasToken(&auth),
					testAccCheckHerokuAuthorizationHasScope(&auth),
				),
			},
		},
	})
}

func testAccCheckHerokuAuthorizationConfig_basic() string {
	return `
resource "heroku_authorization" "foobar" {
}`
}

func testAccCheckHerokuAuthorizationConfig_scopes() string {
	return `
resource "heroku_authorization" "foobar" {
	scope = [ "identity", "read" ]
}`
}

func testAccCheckHerokuAuthorizationExists(n string, auth *heroku.OAuthAuthorization) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		client := testAccProvider.Meta().(*heroku.Service)

		foundAuth, err := client.OAuthAuthorizationInfo(context.TODO(), rs.Primary.ID)
		if err != nil {
			return err
		}

		if foundAuth.ID != rs.Primary.ID {
			return fmt.Errorf("Authorization not found")
		}

		*auth = *foundAuth

		return nil
	}
}

func testAccCheckHerokuAuthorizationHasToken(auth *heroku.OAuthAuthorization) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources["heroku_authorization.foobar"]
		if !ok {
			return fmt.Errorf("Not found: heroku_authorization.foobar")
		}

		if token, ok := rs.Primary.Attributes["token"]; ok {
			if auth.AccessToken == nil {
				return fmt.Errorf("Unable to match token: API response missing token fields")
			}

			if token != auth.AccessToken.Token {
				return fmt.Errorf("Resource token not equal to API token")
			}

			return nil
		} else {
			return fmt.Errorf("Resource had no token")
		}
	}
}

func testAccCheckHerokuAuthorizationHasScope(auth *heroku.OAuthAuthorization) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources["heroku_authorization.foobar"]
		if !ok {
			return fmt.Errorf("Not found: heroku_authorization.foobar")
		}

		if expected, found := strconv.Itoa(len(auth.Scope)), rs.Primary.Attributes["scope.#"]; expected != found {
			return fmt.Errorf("Found wrong number of scopes (expected %s, found %s)", expected, found)
		}
		for index, expected := range auth.Scope {
			key := fmt.Sprintf("scope.%d", index)
			found := rs.Primary.Attributes[key]

			if expected != found {
				return fmt.Errorf("Found unexpected scope at index %d (expected %q, found %q", index, expected, found)
			}
		}
		return nil
	}
}

func testAccCheckHerokuAuthorizationDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*heroku.Service)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "heroku_authorization" {
			continue
		}

		_, err := client.OAuthAuthorizationInfo(context.TODO(), rs.Primary.ID)

		if err == nil {
			return fmt.Errorf("Authorization still exists: %s", rs.Primary.ID)
		}
	}

	return nil
}
