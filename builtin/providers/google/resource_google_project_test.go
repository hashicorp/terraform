package google

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"sort"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"google.golang.org/api/cloudresourcemanager/v1"
)

var (
	projectId = multiEnvSearch([]string{
		"GOOGLE_PROJECT",
		"GCLOUD_PROJECT",
		"CLOUDSDK_CORE_PROJECT",
	})
)

func multiEnvSearch(ks []string) string {
	for _, k := range ks {
		if v := os.Getenv(k); v != "" {
			return v
		}
	}
	return ""
}

// Test that a Project resource can be created and destroyed
func TestAccGoogleProject_associate(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(testAccGoogleProject_basic, projectId),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGoogleProjectExists("google_project.acceptance"),
				),
			},
		},
	})
}

// Test that a Project resource can be created, an IAM Policy
// associated with it, and then destroyed
func TestAccGoogleProject_iamPolicy1(t *testing.T) {
	var policy *cloudresourcemanager.Policy
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckGoogleProjectDestroy,
		Steps: []resource.TestStep{
			// First step inventories the project's existing IAM policy
			resource.TestStep{
				Config: fmt.Sprintf(testAccGoogleProject_basic, projectId),
				Check: resource.ComposeTestCheckFunc(
					testAccGoogleProjectExistingPolicy(policy),
				),
			},
			// Second step applies an IAM policy from a data source. The application
			// merges policies, so we validate the expected state.
			resource.TestStep{
				Config: fmt.Sprintf(testAccGoogleProject_policy1, projectId),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGoogleProjectExists("google_project.acceptance"),
					testAccCheckGoogleProjectIamPolicyIsMerged("google_project.acceptance", "data.google_iam_policy.admin", policy),
				),
			},
			// Finally, remove the custom IAM policy from config and apply, then
			// confirm that the project is in its original state.
			resource.TestStep{
				Config: fmt.Sprintf(testAccGoogleProject_basic, projectId),
			},
		},
	})
}

func testAccCheckGoogleProjectDestroy(s *terraform.State) error {
	return nil
}

// Retrieve the existing policy (if any) for a GCP Project
func testAccGoogleProjectExistingPolicy(p *cloudresourcemanager.Policy) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		c := testAccProvider.Meta().(*Config)
		var err error
		p, err = getProjectIamPolicy(projectId, c)
		if err != nil {
			return fmt.Errorf("Failed to retrieve IAM Policy for project %q: %s", projectId, err)
		}
		if len(p.Bindings) == 0 {
			return fmt.Errorf("Refuse to run test against project with zero IAM Bindings. This is likely an error in the test code that is not properly identifying the IAM policy of a project.")
		}
		return nil
	}
}

func testAccCheckGoogleProjectExists(r string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[r]
		if !ok {
			return fmt.Errorf("Not found: %s", r)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		if rs.Primary.ID != projectId {
			return fmt.Errorf("Expected project %q to match ID %q in state", projectId, rs.Primary.ID)
		}

		return nil
	}
}

func testAccCheckGoogleProjectIamPolicyIsMerged(projectRes, policyRes string, original *cloudresourcemanager.Policy) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Get the project resource
		project, ok := s.RootModule().Resources[projectRes]
		if !ok {
			return fmt.Errorf("Not found: %s", projectRes)
		}
		// The project ID should match the config's project ID
		if project.Primary.ID != projectId {
			return fmt.Errorf("Expected project %q to match ID %q in state", projectId, project.Primary.ID)
		}

		var projectP, policyP cloudresourcemanager.Policy
		// The project should have a policy
		ps, ok := project.Primary.Attributes["policy_data"]
		if !ok {
			return fmt.Errorf("Project resource %q did not have a 'policy_data' attribute. Attributes were %#v", project.Primary.Attributes["id"], project.Primary.Attributes)
		}
		if err := json.Unmarshal([]byte(ps), &projectP); err != nil {
			return err
		}

		// The data policy resource should have a policy
		policy, ok := s.RootModule().Resources[policyRes]
		if !ok {
			return fmt.Errorf("Not found: %s", policyRes)
		}
		ps, ok = policy.Primary.Attributes["policy_data"]
		if !ok {
			return fmt.Errorf("Data policy resource %q did not have a 'policy_data' attribute. Attributes were %#v", policy.Primary.Attributes["id"], project.Primary.Attributes)
		}
		if err := json.Unmarshal([]byte(ps), &policyP); err != nil {
			return err
		}

		// The bindings in both policies should be identical
		if !reflect.DeepEqual(derefBindings(projectP.Bindings), derefBindings(policyP.Bindings)) {
			return fmt.Errorf("Project and data source policies do not match: project policy is %+v, data resource policy is  %+v", derefBindings(projectP.Bindings), derefBindings(policyP.Bindings))
		}

		// Merge the project policy in Terrafomr state with the policy the project had before the config was applied
		expected := make([]*cloudresourcemanager.Binding, 0)
		expected = append(expected, original.Bindings...)
		expected = append(expected, projectP.Bindings...)
		expectedM := mergeBindings(expected)

		// Retrieve the actual policy from the project
		c := testAccProvider.Meta().(*Config)
		actual, err := getProjectIamPolicy(projectId, c)
		if err != nil {
			return fmt.Errorf("Failed to retrieve IAM Policy for project %q: %s", projectId, err)
		}
		actualM := mergeBindings(actual.Bindings)

		// The bindings should match, indicating the policy was successfully applied and merged
		if !reflect.DeepEqual(derefBindings(actualM), derefBindings(expectedM)) {
			return fmt.Errorf("Actual and expected project policies do not match: actual policy is %+v, expected policy is  %+v", derefBindings(actualM), derefBindings(expectedM))
		}

		return nil
	}
}

func TestIamRolesToMembersBinding(t *testing.T) {
	table := []struct {
		expect []*cloudresourcemanager.Binding
		input  map[string]map[string]bool
	}{
		{
			expect: []*cloudresourcemanager.Binding{
				{
					Role: "role-1",
					Members: []string{
						"member-1",
						"member-2",
					},
				},
			},
			input: map[string]map[string]bool{
				"role-1": map[string]bool{
					"member-1": true,
					"member-2": true,
				},
			},
		},
		{
			expect: []*cloudresourcemanager.Binding{
				{
					Role: "role-1",
					Members: []string{
						"member-1",
						"member-2",
					},
				},
			},
			input: map[string]map[string]bool{
				"role-1": map[string]bool{
					"member-1": true,
					"member-2": true,
				},
			},
		},
		{
			expect: []*cloudresourcemanager.Binding{
				{
					Role:    "role-1",
					Members: []string{},
				},
			},
			input: map[string]map[string]bool{
				"role-1": map[string]bool{},
			},
		},
	}

	for _, test := range table {
		got := rolesToMembersBinding(test.input)

		sort.Sort(Binding(got))
		for i, _ := range got {
			sort.Strings(got[i].Members)
		}

		if !reflect.DeepEqual(derefBindings(got), derefBindings(test.expect)) {
			t.Errorf("got %+v, expected %+v", derefBindings(got), derefBindings(test.expect))
		}
	}
}
func TestIamRolesToMembersMap(t *testing.T) {
	table := []struct {
		input  []*cloudresourcemanager.Binding
		expect map[string]map[string]bool
	}{
		{
			input: []*cloudresourcemanager.Binding{
				{
					Role: "role-1",
					Members: []string{
						"member-1",
						"member-2",
					},
				},
			},
			expect: map[string]map[string]bool{
				"role-1": map[string]bool{
					"member-1": true,
					"member-2": true,
				},
			},
		},
		{
			input: []*cloudresourcemanager.Binding{
				{
					Role: "role-1",
					Members: []string{
						"member-1",
						"member-2",
						"member-1",
						"member-2",
					},
				},
			},
			expect: map[string]map[string]bool{
				"role-1": map[string]bool{
					"member-1": true,
					"member-2": true,
				},
			},
		},
		{
			input: []*cloudresourcemanager.Binding{
				{
					Role: "role-1",
				},
			},
			expect: map[string]map[string]bool{
				"role-1": map[string]bool{},
			},
		},
	}

	for _, test := range table {
		got := rolesToMembersMap(test.input)
		if !reflect.DeepEqual(got, test.expect) {
			t.Errorf("got %+v, expected %+v", got, test.expect)
		}
	}
}

func TestIamMergeBindings(t *testing.T) {
	table := []struct {
		input  []*cloudresourcemanager.Binding
		expect []cloudresourcemanager.Binding
	}{
		{
			input: []*cloudresourcemanager.Binding{
				{
					Role: "role-1",
					Members: []string{
						"member-1",
						"member-2",
					},
				},
				{
					Role: "role-1",
					Members: []string{
						"member-3",
					},
				},
			},
			expect: []cloudresourcemanager.Binding{
				{
					Role: "role-1",
					Members: []string{
						"member-1",
						"member-2",
						"member-3",
					},
				},
			},
		},
		{
			input: []*cloudresourcemanager.Binding{
				{
					Role: "role-1",
					Members: []string{
						"member-3",
						"member-4",
					},
				},
				{
					Role: "role-1",
					Members: []string{
						"member-2",
						"member-1",
					},
				},
				{
					Role: "role-2",
					Members: []string{
						"member-1",
					},
				},
				{
					Role: "role-1",
					Members: []string{
						"member-5",
					},
				},
				{
					Role: "role-3",
					Members: []string{
						"member-1",
					},
				},
				{
					Role: "role-2",
					Members: []string{
						"member-2",
					},
				},
			},
			expect: []cloudresourcemanager.Binding{
				{
					Role: "role-1",
					Members: []string{
						"member-1",
						"member-2",
						"member-3",
						"member-4",
						"member-5",
					},
				},
				{
					Role: "role-2",
					Members: []string{
						"member-1",
						"member-2",
					},
				},
				{
					Role: "role-3",
					Members: []string{
						"member-1",
					},
				},
			},
		},
	}

	for _, test := range table {
		got := mergeBindings(test.input)
		sort.Sort(Binding(got))
		for i, _ := range got {
			sort.Strings(got[i].Members)
		}

		if !reflect.DeepEqual(derefBindings(got), test.expect) {
			t.Errorf("\ngot %+v\nexpected %+v", derefBindings(got), test.expect)
		}
	}
}

func derefBindings(b []*cloudresourcemanager.Binding) []cloudresourcemanager.Binding {
	db := make([]cloudresourcemanager.Binding, len(b))

	for i, v := range b {
		db[i] = *v
	}
	return db
}

type Binding []*cloudresourcemanager.Binding

func (b Binding) Len() int {
	return len(b)
}
func (b Binding) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}
func (b Binding) Less(i, j int) bool {
	return b[i].Role < b[j].Role
}

var testAccGoogleProject_basic = `
resource "google_project" "acceptance" {
    id = "%v"
}`

var testAccGoogleProject_policy1 = `
resource "google_project" "acceptance" {
    id = "%v"
    policy_data = "${data.google_iam_policy.admin.policy_data}"
}

data "google_iam_policy" "admin" {
  binding {
    role = "roles/storage.objectViewer"
    members = [
      "user:evanbrown@google.com",
    ]
  }
  binding {
    role = "roles/compute.instanceAdmin"
    members = [
      "user:evanbrown@google.com",
      "user:evandbrown@gmail.com",
    ]
  }

}`
