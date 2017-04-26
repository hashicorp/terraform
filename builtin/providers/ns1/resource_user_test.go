package ns1

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	ns1 "gopkg.in/ns1/ns1-go.v2/rest"
	"gopkg.in/ns1/ns1-go.v2/rest/model/account"
)

func TestAccUser_basic(t *testing.T) {
	var user account.User
	rString := acctest.RandStringFromCharSet(15, acctest.CharSetAlphaNum)
	name := fmt.Sprintf("terraform acc test user %s", rString)
	username := fmt.Sprintf("tf_acc_test_user_%s", rString)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckUserDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccUserBasic(rString),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckUserExists("ns1_user.u", &user),
					resource.TestCheckResourceAttr("ns1_user.u", "email", "tf_acc_test_ns1@hashicorp.com"),
					resource.TestCheckResourceAttr("ns1_user.u", "name", name),
					resource.TestCheckResourceAttr("ns1_user.u", "teams.#", "1"),
					resource.TestCheckResourceAttr("ns1_user.u", "notify.%", "1"),
					resource.TestCheckResourceAttr("ns1_user.u", "notify.billing", "true"),
					resource.TestCheckResourceAttr("ns1_user.u", "username", username),
				),
			},
		},
	})
}

func testAccCheckUserDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*ns1.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "ns1_user" {
			continue
		}

		user, _, err := client.Users.Get(rs.Primary.Attributes["id"])
		if err == nil {
			return fmt.Errorf("User still exists: %#v: %#v", err, user.Name)
		}
	}

	return nil
}

func testAccCheckUserExists(n string, user *account.User) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		client := testAccProvider.Meta().(*ns1.Client)

		foundUser, _, err := client.Users.Get(rs.Primary.ID)
		if err != nil {
			return err
		}

		if foundUser.Username != rs.Primary.ID {
			return fmt.Errorf("User not found (%#v != %s)", foundUser, rs.Primary.ID)
		}

		*user = *foundUser

		return nil
	}
}

func testAccUserBasic(rString string) string {
	return fmt.Sprintf(`resource "ns1_team" "t" {
  name = "terraform acc test team %s"
}

resource "ns1_user" "u" {
  name = "terraform acc test user %s"
  username = "tf_acc_test_user_%s"
  email = "tf_acc_test_ns1@hashicorp.com"
  teams = ["${ns1_team.t.id}"]
  notify {
  	billing = true
  }
}
`, rString, rString, rString)
}
