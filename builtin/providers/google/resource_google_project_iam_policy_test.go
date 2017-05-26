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

// Test that a non-collapsed IAM policy doesn't perpetually diff
func TestAccGoogleProjectIamPolicy_expanded(t *testing.T) {
	pid := "terraform-" + acctest.RandString(10)
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccGoogleProjectAssociatePolicyExpanded(pid, pname, org),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGoogleProjectIamPolicyExists("google_project_iam_policy.acceptance", "data.google_iam_policy.expanded", pid),
				),
			},
		},
	})
}

func getStatePrimaryResource(s *terraform.State, res, expectedID string) (*terraform.InstanceState, error) {
	// Get the project resource
	resource, ok := s.RootModule().Resources[res]
	if !ok {
		return nil, fmt.Errorf("Not found: %s", res)
	}
	if resource.Primary.Attributes["id"] != expectedID && expectedID != "" {
		return nil, fmt.Errorf("Expected project %q to match ID %q in state", resource.Primary.ID, expectedID)
	}
	return resource.Primary, nil
}

func getGoogleProjectIamPolicyFromResource(resource *terraform.InstanceState) (cloudresourcemanager.Policy, error) {
	var p cloudresourcemanager.Policy
	ps, ok := resource.Attributes["policy_data"]
	if !ok {
		return p, fmt.Errorf("Resource %q did not have a 'policy_data' attribute. Attributes were %#v", resource.ID, resource.Attributes)
	}
	if err := json.Unmarshal([]byte(ps), &p); err != nil {
		return p, fmt.Errorf("Could not unmarshal %s:\n: %v", ps, err)
	}
	return p, nil
}

func getGoogleProjectIamPolicyFromState(s *terraform.State, res, expectedID string) (cloudresourcemanager.Policy, error) {
	project, err := getStatePrimaryResource(s, res, expectedID)
	if err != nil {
		return cloudresourcemanager.Policy{}, err
	}
	return getGoogleProjectIamPolicyFromResource(project)
}

func compareBindings(a, b []*cloudresourcemanager.Binding) bool {
	a = mergeBindings(a)
	b = mergeBindings(b)
	sort.Sort(sortableBindings(a))
	sort.Sort(sortableBindings(b))
	return reflect.DeepEqual(derefBindings(a), derefBindings(b))
}

func testAccCheckGoogleProjectIamPolicyExists(projectRes, policyRes, pid string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		projectPolicy, err := getGoogleProjectIamPolicyFromState(s, projectRes, pid)
		if err != nil {
			return fmt.Errorf("Error retrieving IAM policy for project from state: %s", err)
		}
		policyPolicy, err := getGoogleProjectIamPolicyFromState(s, policyRes, "")
		if err != nil {
			return fmt.Errorf("Error retrieving IAM policy for data_policy from state: %s", err)
		}

		// The bindings in both policies should be identical
		if !compareBindings(projectPolicy.Bindings, policyPolicy.Bindings) {
			return fmt.Errorf("Project and data source policies do not match: project policy is %+v, data resource policy is  %+v", derefBindings(projectPolicy.Bindings), derefBindings(policyPolicy.Bindings))
		}
		return nil
	}
}

func testAccCheckGoogleProjectIamPolicyIsMerged(projectRes, policyRes, pid string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		err := testAccCheckGoogleProjectIamPolicyExists(projectRes, policyRes, pid)(s)
		if err != nil {
			return err
		}

		projectPolicy, err := getGoogleProjectIamPolicyFromState(s, projectRes, pid)
		if err != nil {
			return fmt.Errorf("Error retrieving IAM policy for project from state: %s", err)
		}

		// Merge the project policy in Terraform state with the policy the project had before the config was applied
		var expected []*cloudresourcemanager.Binding
		expected = append(expected, originalPolicy.Bindings...)
		expected = append(expected, projectPolicy.Bindings...)
		expected = mergeBindings(expected)

		// Retrieve the actual policy from the project
		c := testAccProvider.Meta().(*Config)
		actual, err := getProjectIamPolicy(pid, c)
		if err != nil {
			return fmt.Errorf("Failed to retrieve IAM Policy for project %q: %s", pid, err)
		}
		// The bindings should match, indicating the policy was successfully applied and merged
		if !compareBindings(actual.Bindings, expected) {
			return fmt.Errorf("Actual and expected project policies do not match: actual policy is %+v, expected policy is  %+v", derefBindings(actual.Bindings), derefBindings(expected))
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

func testAccGoogleProject_createBilling(pid, name, org, billing string) string {
	return fmt.Sprintf(`
resource "google_project" "acceptance" {
    project_id = "%s"
    name = "%s"
    org_id = "%s"
    billing_account = "%s"
}`, pid, name, org, billing)
}

func testAccGoogleProjectAssociatePolicyExpanded(pid, name, org string) string {
	return fmt.Sprintf(`
resource "google_project" "acceptance" {
    project_id = "%s"
    name = "%s"
    org_id = "%s"
}
resource "google_project_iam_policy" "acceptance" {
    project = "${google_project.acceptance.id}"
    policy_data = "${data.google_iam_policy.expanded.policy_data}"
    authoritative = false
}
data "google_iam_policy" "expanded" {
    binding {
        role = "roles/viewer"
        members = [
            "user:paddy@carvers.co",
        ]
    }
    
    binding {
        role = "roles/viewer"
        members = [
            "user:paddy@hashicorp.com",
        ]
    }
}`, pid, name, org)
}
