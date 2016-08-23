package google

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/hashicorp/terraform/helper/schema"
	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/googleapi"
)

// resourceGoogleProject returns a *schema.Resource that allows a customer
// to declare a Google Cloud Project resource. //
// Only the 'policy' property of a project may be updated. All other properties
// are computed.
//
// This example shows a project with a policy declared in config:
//
// resource "google_project" "my-project" {
//    project = "a-project-id"
//    policy = "${data.google_iam_policy.admin.policy}"
// }
func resourceGoogleProject() *schema.Resource {
	return &schema.Resource{
		Create: resourceGoogleProjectCreate,
		Read:   resourceGoogleProjectRead,
		Update: resourceGoogleProjectUpdate,
		Delete: resourceGoogleProjectDelete,

		Schema: map[string]*schema.Schema{
			"id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"policy_data": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"number": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

// This resource supports creation, but not in the traditional sense.
// A new Google Cloud Project can not be created. Instead, an existing Project
// is initialized and made available as a Terraform resource.
func resourceGoogleProjectCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	d.SetId(project)
	if err := resourceGoogleProjectRead(d, meta); err != nil {
		return err
	}

	// Apply the IAM policy if it is set
	if pString, ok := d.GetOk("policy_data"); ok {
		// The policy string is just a marshaled cloudresourcemanager.Policy.
		// Unmarshal it to a struct.
		var policy cloudresourcemanager.Policy
		if err = json.Unmarshal([]byte(pString.(string)), &policy); err != nil {
			return err
		}

		// Retrieve existing IAM policy from project. This will be merged
		// with the policy defined here.
		// TODO(evanbrown): Add an 'authoritative' flag that allows policy
		// in manifest to overwrite existing policy.
		p, err := getProjectIamPolicy(project, config)
		if err != nil {
			return err
		}
		log.Printf("[DEBUG] Got existing bindings from project: %#v", p.Bindings)

		// Merge the existing policy bindings with those defined in this manifest.
		p.Bindings = mergeBindings(append(p.Bindings, policy.Bindings...))

		// Apply the merged policy
		log.Printf("[DEBUG] Setting new policy for project: %#v", p)
		_, err = config.clientResourceManager.Projects.SetIamPolicy(project,
			&cloudresourcemanager.SetIamPolicyRequest{Policy: p}).Do()

		if err != nil {
			return fmt.Errorf("Error applying IAM policy for project %q: %s", project, err)
		}
	}
	return nil
}

func resourceGoogleProjectRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	project, err := getProject(d, config)
	if err != nil {
		return err
	}
	d.SetId(project)

	// Confirm the project exists.
	// TODO(evanbrown): Support project creation
	p, err := config.clientResourceManager.Projects.Get(project).Do()
	if err != nil {
		if v, ok := err.(*googleapi.Error); ok && v.Code == http.StatusNotFound {
			return fmt.Errorf("Project %q does not exist. The Google provider does not currently support new project creation.", project)
		}
		return fmt.Errorf("Error checking project %q: %s", project, err)
	}

	d.Set("number", strconv.FormatInt(int64(p.ProjectNumber), 10))
	d.Set("name", p.Name)

	return nil
}

func resourceGoogleProjectUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	// Policy has changed
	if ok := d.HasChange("policy_data"); ok {
		// The policy string is just a marshaled cloudresourcemanager.Policy.
		// Unmarshal it to a struct that contains the old and new policies
		oldP, newP := d.GetChange("policy_data")
		oldPString := oldP.(string)
		newPString := newP.(string)

		// JSON Unmarshaling would fail
		if oldPString == "" {
			oldPString = "{}"
		}
		if newPString == "" {
			newPString = "{}"
		}

		oldPStringf, _ := json.MarshalIndent(oldPString, "", "   ")
		newPStringf, _ := json.MarshalIndent(newPString, "", "   ")
		log.Printf("[DEBUG]: Old policy: %v\nNew policy: %v", string(oldPStringf), string(newPStringf))

		var oldPolicy, newPolicy cloudresourcemanager.Policy
		if err = json.Unmarshal([]byte(newPString), &newPolicy); err != nil {
			return err
		}
		if err = json.Unmarshal([]byte(oldPString), &oldPolicy); err != nil {
			return err
		}

		// Find any Roles and Members that were removed (i.e., those that are present
		// in the old but absent in the new
		oldMap := rolesToMembersMap(oldPolicy.Bindings)
		newMap := rolesToMembersMap(newPolicy.Bindings)
		deleted := make(map[string]map[string]bool)

		// Get each role and its associated members in the old state
		for role, members := range oldMap {
			// Initialize map for role
			if _, ok := deleted[role]; !ok {
				deleted[role] = make(map[string]bool)
			}
			// The role exists in the new state
			if _, ok := newMap[role]; ok {
				// Check each memeber
				for member, _ := range members {
					// Member does not exist in new state, so it was deleted
					if _, ok = newMap[role][member]; !ok {
						deleted[role][member] = true
					}
				}
			} else {
				// This indicates an entire role was deleted. Mark all members
				// for delete.
				for member, _ := range members {
					deleted[role][member] = true
				}
			}
		}
		log.Printf("[DEBUG] Roles and Members to be deleted: %#v", deleted)

		// Retrieve existing IAM policy from project. This will be merged
		// with the policy in the current state
		// TODO(evanbrown): Add an 'authoritative' flag that allows policy
		// in manifest to overwrite existing policy.
		p, err := getProjectIamPolicy(project, config)
		if err != nil {
			return err
		}
		log.Printf("[DEBUG] Got existing bindings from project: %#v", p.Bindings)

		// Merge existing policy with policy in the current state
		log.Printf("[DEBUG] Merging new bindings from project: %#v", newPolicy.Bindings)
		mergedBindings := mergeBindings(append(p.Bindings, newPolicy.Bindings...))

		// Remove any roles and members that were explicitly deleted
		mergedBindingsMap := rolesToMembersMap(mergedBindings)
		for role, members := range deleted {
			for member, _ := range members {
				delete(mergedBindingsMap[role], member)
			}
		}

		p.Bindings = rolesToMembersBinding(mergedBindingsMap)
		log.Printf("[DEBUG] Setting new policy for project: %#v", p)

		dump, _ := json.MarshalIndent(p.Bindings, " ", "  ")
		log.Printf(string(dump))
		_, err = config.clientResourceManager.Projects.SetIamPolicy(project,
			&cloudresourcemanager.SetIamPolicyRequest{Policy: p}).Do()

		if err != nil {
			return fmt.Errorf("Error applying IAM policy for project %q: %s", project, err)
		}
	}

	return nil
}

func resourceGoogleProjectDelete(d *schema.ResourceData, meta interface{}) error {
	d.SetId("")
	return nil
}

// Retrieve the existing IAM Policy for a Project
func getProjectIamPolicy(project string, config *Config) (*cloudresourcemanager.Policy, error) {
	p, err := config.clientResourceManager.Projects.GetIamPolicy(project,
		&cloudresourcemanager.GetIamPolicyRequest{}).Do()

	if err != nil {
		return nil, fmt.Errorf("Error retrieving IAM policy for project %q: %s", project, err)
	}
	return p, nil
}

// Convert a map of roles->members to a list of Binding
func rolesToMembersBinding(m map[string]map[string]bool) []*cloudresourcemanager.Binding {
	bindings := make([]*cloudresourcemanager.Binding, 0)
	for role, members := range m {
		b := cloudresourcemanager.Binding{
			Role:    role,
			Members: make([]string, 0),
		}
		for m, _ := range members {
			b.Members = append(b.Members, m)
		}
		bindings = append(bindings, &b)
	}
	return bindings
}

// Map a role to a map of members, allowing easy merging of multiple bindings.
func rolesToMembersMap(bindings []*cloudresourcemanager.Binding) map[string]map[string]bool {
	bm := make(map[string]map[string]bool)
	// Get each binding
	for _, b := range bindings {
		// Initialize members map
		if _, ok := bm[b.Role]; !ok {
			bm[b.Role] = make(map[string]bool)
		}
		// Get each member (user/principal) for the binding
		for _, m := range b.Members {
			// Add the member
			bm[b.Role][m] = true
		}
	}
	return bm
}

// Merge multiple Bindings such that Bindings with the same Role result in
// a single Binding with combined Members
func mergeBindings(bindings []*cloudresourcemanager.Binding) []*cloudresourcemanager.Binding {
	bm := rolesToMembersMap(bindings)
	rb := make([]*cloudresourcemanager.Binding, 0)

	for role, members := range bm {
		var b cloudresourcemanager.Binding
		b.Role = role
		b.Members = make([]string, 0)
		for m, _ := range members {
			b.Members = append(b.Members, m)
		}
		rb = append(rb, &b)
	}

	return rb
}
