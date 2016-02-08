package google

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccGoogleSqlUser_basic(t *testing.T) {
	user := acctest.RandString(10)
	instance := acctest.RandString(10)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccGoogleSqlUserDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testGoogleSqlUser_basic(instance, user),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGoogleSqlUserExists("google_sql_user.user"),
				),
			},
		},
	})
}

func TestAccGoogleSqlUser_update(t *testing.T) {
	user := acctest.RandString(10)
	instance := acctest.RandString(10)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccGoogleSqlUserDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testGoogleSqlUser_basic(instance, user),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGoogleSqlUserExists("google_sql_user.user"),
				),
			},

			resource.TestStep{
				Config: testGoogleSqlUser_basic2(instance, user),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGoogleSqlUserExists("google_sql_user.user"),
				),
			},
		},
	})
}

func testAccCheckGoogleSqlUserExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		config := testAccProvider.Meta().(*Config)
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Resource not found: %s", n)
		}

		name := rs.Primary.Attributes["name"]
		instance := rs.Primary.Attributes["instance"]
		host := rs.Primary.Attributes["host"]
		users, err := config.clientSqlAdmin.Users.List(config.Project,
			instance).Do()

		for _, user := range users.Items {
			if user.Name == name && user.Host == host {
				return nil
			}
		}

		return fmt.Errorf("Not found: %s: %s", n, err)
	}
}

func testAccGoogleSqlUserDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		config := testAccProvider.Meta().(*Config)
		if rs.Type != "google_sql_database" {
			continue
		}

		name := rs.Primary.Attributes["name"]
		instance := rs.Primary.Attributes["instance"]
		host := rs.Primary.Attributes["host"]
		users, err := config.clientSqlAdmin.Users.List(config.Project,
			instance).Do()

		for _, user := range users.Items {
			if user.Name == name && user.Host == host {
				return fmt.Errorf("User still %s exists %s", name, err)
			}
		}

		return nil
	}

	return nil
}

func testGoogleSqlUser_basic(instance, user string) string {
	return fmt.Sprintf(`
	resource "google_sql_database_instance" "instance" {
		name = "i%s"
		region = "us-central"
		settings {
			tier = "D0"
		}
	}

	resource "google_sql_user" "user" {
		name = "user%s"
		instance = "${google_sql_database_instance.instance.name}"
		host = "google.com"
		password = "hunter2"
	}
	`, instance, user)
}

func testGoogleSqlUser_basic2(instance, user string) string {
	return fmt.Sprintf(`
	resource "google_sql_database_instance" "instance" {
		name = "i%s"
		region = "us-central"
		settings {
			tier = "D0"
		}
	}

	resource "google_sql_user" "user" {
		name = "user%s"
		instance = "${google_sql_database_instance.instance.name}"
		host = "google.com"
		password = "oops"
	}
	`, instance, user)
}
