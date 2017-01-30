package google

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"google.golang.org/api/cloudresourcemanager/v1"
)

func TestSubtractIamPolicy(t *testing.T) {
	table := []struct {
		a      *cloudresourcemanager.Policy
		b      *cloudresourcemanager.Policy
		expect cloudresourcemanager.Policy
	}{
		{
			a: &cloudresourcemanager.Policy{
				Bindings: []*cloudresourcemanager.Binding{
					{
						Role: "a",
						Members: []string{
							"1",
							"2",
						},
					},
					{
						Role: "b",
						Members: []string{
							"1",
							"2",
						},
					},
				},
			},
			b: &cloudresourcemanager.Policy{
				Bindings: []*cloudresourcemanager.Binding{
					{
						Role: "a",
						Members: []string{
							"3",
							"4",
						},
					},
					{
						Role: "b",
						Members: []string{
							"1",
							"2",
						},
					},
				},
			},
			expect: cloudresourcemanager.Policy{
				Bindings: []*cloudresourcemanager.Binding{
					{
						Role: "a",
						Members: []string{
							"1",
							"2",
						},
					},
				},
			},
		},
		{
			a: &cloudresourcemanager.Policy{
				Bindings: []*cloudresourcemanager.Binding{
					{
						Role: "a",
						Members: []string{
							"1",
							"2",
						},
					},
					{
						Role: "b",
						Members: []string{
							"1",
							"2",
						},
					},
				},
			},
			b: &cloudresourcemanager.Policy{
				Bindings: []*cloudresourcemanager.Binding{
					{
						Role: "a",
						Members: []string{
							"1",
							"2",
						},
					},
					{
						Role: "b",
						Members: []string{
							"1",
							"2",
						},
					},
				},
			},
			expect: cloudresourcemanager.Policy{
				Bindings: []*cloudresourcemanager.Binding{},
			},
		},
		{
			a: &cloudresourcemanager.Policy{
				Bindings: []*cloudresourcemanager.Binding{
					{
						Role: "a",
						Members: []string{
							"1",
							"2",
							"3",
						},
					},
					{
						Role: "b",
						Members: []string{
							"1",
							"2",
							"3",
						},
					},
				},
			},
			b: &cloudresourcemanager.Policy{
				Bindings: []*cloudresourcemanager.Binding{
					{
						Role: "a",
						Members: []string{
							"1",
							"3",
						},
					},
					{
						Role: "b",
						Members: []string{
							"1",
							"2",
							"3",
						},
					},
				},
			},
			expect: cloudresourcemanager.Policy{
				Bindings: []*cloudresourcemanager.Binding{
					{
						Role: "a",
						Members: []string{
							"2",
						},
					},
				},
			},
		},
		{
			a: &cloudresourcemanager.Policy{
				Bindings: []*cloudresourcemanager.Binding{
					{
						Role: "a",
						Members: []string{
							"1",
							"2",
							"3",
						},
					},
					{
						Role: "b",
						Members: []string{
							"1",
							"2",
							"3",
						},
					},
				},
			},
			b: &cloudresourcemanager.Policy{
				Bindings: []*cloudresourcemanager.Binding{
					{
						Role: "a",
						Members: []string{
							"1",
							"2",
							"3",
						},
					},
					{
						Role: "b",
						Members: []string{
							"1",
							"2",
							"3",
						},
					},
				},
			},
			expect: cloudresourcemanager.Policy{
				Bindings: []*cloudresourcemanager.Binding{},
			},
		},
	}

	for _, test := range table {
		c := subtractIamPolicy(test.a, test.b)
		sort.Sort(sortableBindings(c.Bindings))
		for i, _ := range c.Bindings {
			sort.Strings(c.Bindings[i].Members)
		}

		if !reflect.DeepEqual(derefBindings(c.Bindings), derefBindings(test.expect.Bindings)) {
			t.Errorf("\ngot %+v\nexpected %+v", derefBindings(c.Bindings), derefBindings(test.expect.Bindings))
		}
	}
}

// Test that an IAM policy can be applied to a project
func TestAccGoogleProjectIamPolicy_basic(t *testing.T) {
	pid := "terraform-" + acctest.RandString(10)
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			// Create a new project
			resource.TestStep{
				Config: testAccGoogleProject_create(pid, pname, org),
				Check: resource.ComposeTestCheckFunc(
					testAccGoogleProjectExistingPolicy(pid),
				),
			},
			// Apply an IAM policy from a data source. The application
			// merges policies, so we validate the expected state.
			resource.TestStep{
				Config: testAccGoogleProjectAssociatePolicyBasic(pid, pname, org),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGoogleProjectIamPolicyIsMerged("google_project_iam_policy.acceptance", "data.google_iam_policy.admin", pid),
				),
			},
			// Finally, remove the custom IAM policy from config and apply, then
			// confirm that the project is in its original state.
			resource.TestStep{
				Config: testAccGoogleProject_create(pid, pname, org),
				Check: resource.ComposeTestCheckFunc(
					testAccGoogleProjectExistingPolicy(pid),
				),
			},
		},
	})
}

func testAccCheckGoogleProjectIamPolicyIsMerged(projectRes, policyRes, pid string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Get the project resource
		project, ok := s.RootModule().Resources[projectRes]
		if !ok {
			return fmt.Errorf("Not found: %s", projectRes)
		}
		// The project ID should match the config's project ID
		if project.Primary.ID != pid {
			return fmt.Errorf("Expected project %q to match ID %q in state", pid, project.Primary.ID)
		}

		var projectP, policyP cloudresourcemanager.Policy
		// The project should have a policy
		ps, ok := project.Primary.Attributes["policy_data"]
		if !ok {
			return fmt.Errorf("Project resource %q did not have a 'policy_data' attribute. Attributes were %#v", project.Primary.Attributes["id"], project.Primary.Attributes)
		}
		if err := json.Unmarshal([]byte(ps), &projectP); err != nil {
			return fmt.Errorf("Could not unmarshal %s:\n: %v", ps, err)
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
		sort.Sort(sortableBindings(projectP.Bindings))
		sort.Sort(sortableBindings(policyP.Bindings))
		if !reflect.DeepEqual(derefBindings(projectP.Bindings), derefBindings(policyP.Bindings)) {
			return fmt.Errorf("Project and data source policies do not match: project policy is %+v, data resource policy is  %+v", derefBindings(projectP.Bindings), derefBindings(policyP.Bindings))
		}

		// Merge the project policy in Terraform state with the policy the project had before the config was applied
		expected := make([]*cloudresourcemanager.Binding, 0)
		expected = append(expected, originalPolicy.Bindings...)
		expected = append(expected, projectP.Bindings...)
		expectedM := mergeBindings(expected)

		// Retrieve the actual policy from the project
		c := testAccProvider.Meta().(*Config)
		actual, err := getProjectIamPolicy(pid, c)
		if err != nil {
			return fmt.Errorf("Failed to retrieve IAM Policy for project %q: %s", pid, err)
		}
		actualM := mergeBindings(actual.Bindings)

		sort.Sort(sortableBindings(actualM))
		sort.Sort(sortableBindings(expectedM))
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

		sort.Sort(sortableBindings(got))
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
		sort.Sort(sortableBindings(got))
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
		sort.Strings(db[i].Members)
	}
	return db
}

// Confirm that a project has an IAM policy with at least 1 binding
func testAccGoogleProjectExistingPolicy(pid string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		c := testAccProvider.Meta().(*Config)
		var err error
		originalPolicy, err = getProjectIamPolicy(pid, c)
		if err != nil {
			return fmt.Errorf("Failed to retrieve IAM Policy for project %q: %s", pid, err)
		}
		if len(originalPolicy.Bindings) == 0 {
			return fmt.Errorf("Refuse to run test against project with zero IAM Bindings. This is likely an error in the test code that is not properly identifying the IAM policy of a project.")
		}
		return nil
	}
}

func testAccGoogleProjectAssociatePolicyBasic(pid, name, org string) string {
	return fmt.Sprintf(`
resource "google_project" "acceptance" {
    project_id = "%s"
	name = "%s"
	org_id = "%s"
}
resource "google_project_iam_policy" "acceptance" {
    project = "${google_project.acceptance.id}"
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
}
`, pid, name, org)
}

func testAccGoogleProject_create(pid, name, org string) string {
	return fmt.Sprintf(`
resource "google_project" "acceptance" {
    project_id = "%s"
	name = "%s"
	org_id = "%s"
}`, pid, name, org)
}
