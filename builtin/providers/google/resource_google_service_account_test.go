package google

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

var (
	projectId = multiEnvSearch([]string{
		"GOOGLE_PROJECT",
		"GCLOUD_PROJECT",
		"CLOUDSDK_CORE_PROJECT",
	})
)

// Test that a service account resource can be created, updated, and destroyed
func TestAccGoogleServiceAccount_basic(t *testing.T) {
	accountId := "a" + acctest.RandString(10)
	displayName := "Terraform Test"
	displayName2 := "Terraform Test Update"
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			// The first step creates a basic service account
			resource.TestStep{
				Config: testAccGoogleServiceAccountBasic(accountId, displayName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGoogleServiceAccountExists("google_service_account.acceptance"),
				),
			},
			// The second step updates the service account
			resource.TestStep{
				Config: testAccGoogleServiceAccountBasic(accountId, displayName2),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGoogleServiceAccountNameModified("google_service_account.acceptance", displayName2),
				),
			},
		},
	})
}

// Test that a service account resource can be created with a policy, updated,
// and destroyed.
func TestAccGoogleServiceAccount_createPolicy(t *testing.T) {
	accountId := "a" + acctest.RandString(10)
	displayName := "Terraform Test"
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			// The first step creates a basic service account with an IAM policy
			resource.TestStep{
				Config: testAccGoogleServiceAccountPolicy(accountId, projectId),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGoogleServiceAccountPolicyCount("google_service_account.acceptance", 1),
				),
			},
			// The second step updates the service account with no IAM policy
			resource.TestStep{
				Config: testAccGoogleServiceAccountBasic(accountId, displayName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGoogleServiceAccountPolicyCount("google_service_account.acceptance", 0),
				),
			},
			// The final step re-applies the IAM policy
			resource.TestStep{
				Config: testAccGoogleServiceAccountPolicy(accountId, projectId),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGoogleServiceAccountPolicyCount("google_service_account.acceptance", 1),
				),
			},
		},
	})
}

func testAccCheckGoogleServiceAccountPolicyCount(r string, n int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		c := testAccProvider.Meta().(*Config)
		p, err := getServiceAccountIamPolicy(s.RootModule().Resources[r].Primary.ID, c)
		if err != nil {
			return fmt.Errorf("Failed to retrieve IAM Policy for service account: %s", err)
		}
		if len(p.Bindings) != n {
			return fmt.Errorf("The service account has %v bindings but %v were expected", len(p.Bindings), n)
		}
		return nil
	}
}

func testAccCheckGoogleServiceAccountExists(r string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[r]
		if !ok {
			return fmt.Errorf("Not found: %s", r)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		return nil
	}
}

func testAccCheckGoogleServiceAccountNameModified(r, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[r]
		if !ok {
			return fmt.Errorf("Not found: %s", r)
		}

		if rs.Primary.Attributes["display_name"] != n {
			return fmt.Errorf("display_name is %q expected %q", rs.Primary.Attributes["display_name"], n)
		}

		return nil
	}
}

func testAccGoogleServiceAccountBasic(account, name string) string {
	t := `resource "google_service_account" "acceptance" {
    account_id = "%v"
	display_name = "%v"
 }`
	return fmt.Sprintf(t, account, name)
}

func testAccGoogleServiceAccountPolicy(account, name string) string {

	t := `resource "google_service_account" "acceptance" {
    account_id = "%v"
    display_name = "%v"
    policy_data = "${data.google_iam_policy.service_account.policy_data}"
}

data "google_iam_policy" "service_account" {
  binding {
    role = "roles/iam.serviceAccountActor"
    members = [
      "serviceAccount:%v@%v.iam.gserviceaccount.com",
    ]
  }
}`

	return fmt.Sprintf(t, account, name, account, projectId)
}
