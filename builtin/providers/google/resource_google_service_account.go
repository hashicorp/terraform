package google

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"google.golang.org/api/iam/v1"
)

func resourceGoogleServiceAccount() *schema.Resource {
	return &schema.Resource{
		Create: resourceGoogleServiceAccountCreate,
		Read:   resourceGoogleServiceAccountRead,
		Delete: resourceGoogleServiceAccountDelete,
		Update: resourceGoogleServiceAccountUpdate,
		Schema: map[string]*schema.Schema{
			"email": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"unique_id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"account_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"display_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"project": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"policy_data": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func resourceGoogleServiceAccountCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	project, err := getProject(d, config)
	if err != nil {
		return err
	}
	aid := d.Get("account_id").(string)
	displayName := d.Get("display_name").(string)

	sa := &iam.ServiceAccount{
		DisplayName: displayName,
	}

	r := &iam.CreateServiceAccountRequest{
		AccountId:      aid,
		ServiceAccount: sa,
	}

	sa, err = config.clientIAM.Projects.ServiceAccounts.Create("projects/"+project, r).Do()
	if err != nil {
		return fmt.Errorf("Error creating service account: %s", err)
	}

	d.SetId(sa.Name)

	// Apply the IAM policy if it is set
	if pString, ok := d.GetOk("policy_data"); ok {
		// The policy string is just a marshaled cloudresourcemanager.Policy.
		// Unmarshal it to a struct.
		var policy iam.Policy
		if err = json.Unmarshal([]byte(pString.(string)), &policy); err != nil {
			return err
		}

		// Retrieve existing IAM policy from project. This will be merged
		// with the policy defined here.
		// TODO(evanbrown): Add an 'authoritative' flag that allows policy
		// in manifest to overwrite existing policy.
		p, err := getServiceAccountIamPolicy(sa.Name, config)
		if err != nil {
			return fmt.Errorf("Could not find service account %q when applying IAM policy: %s", sa.Name, err)
		}
		log.Printf("[DEBUG] Got existing bindings for service account: %#v", p.Bindings)

		// Merge the existing policy bindings with those defined in this manifest.
		p.Bindings = saMergeBindings(append(p.Bindings, policy.Bindings...))

		// Apply the merged policy
		log.Printf("[DEBUG] Setting new policy for service account: %#v", p)
		_, err = config.clientIAM.Projects.ServiceAccounts.SetIamPolicy(sa.Name,
			&iam.SetIamPolicyRequest{Policy: p}).Do()

		if err != nil {
			return fmt.Errorf("Error applying IAM policy for service account %q: %s", sa.Name, err)
		}
	}
	return resourceGoogleServiceAccountRead(d, meta)
}

func resourceGoogleServiceAccountRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	// Confirm the service account exists
	sa, err := config.clientIAM.Projects.ServiceAccounts.Get(d.Id()).Do()
	if err != nil {
		return handleNotFoundError(err, d, fmt.Sprintf("Service Account %q", d.Id()))
	}

	d.Set("email", sa.Email)
	d.Set("unique_id", sa.UniqueId)
	d.Set("name", sa.Name)
	d.Set("display_name", sa.DisplayName)
	return nil
}

func resourceGoogleServiceAccountDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	name := d.Id()
	_, err := config.clientIAM.Projects.ServiceAccounts.Delete(name).Do()
	if err != nil {
		return err
	}
	d.SetId("")
	return nil
}

func resourceGoogleServiceAccountUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	var err error
	if ok := d.HasChange("display_name"); ok {
		sa, err := config.clientIAM.Projects.ServiceAccounts.Get(d.Id()).Do()
		if err != nil {
			return fmt.Errorf("Error retrieving service account %q: %s", d.Id(), err)
		}
		_, err = config.clientIAM.Projects.ServiceAccounts.Update(d.Id(),
			&iam.ServiceAccount{
				DisplayName: d.Get("display_name").(string),
				Etag:        sa.Etag,
			}).Do()
		if err != nil {
			return fmt.Errorf("Error updating service account %q: %s", d.Id(), err)
		}
	}

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

		log.Printf("[DEBUG]: Old policy: %q\nNew policy: %q", string(oldPString), string(newPString))

		var oldPolicy, newPolicy iam.Policy
		if err = json.Unmarshal([]byte(newPString), &newPolicy); err != nil {
			return err
		}
		if err = json.Unmarshal([]byte(oldPString), &oldPolicy); err != nil {
			return err
		}

		// Find any Roles and Members that were removed (i.e., those that are present
		// in the old but absent in the new
		oldMap := saRolesToMembersMap(oldPolicy.Bindings)
		newMap := saRolesToMembersMap(newPolicy.Bindings)
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
		p, err := getServiceAccountIamPolicy(d.Id(), config)
		if err != nil {
			return err
		}
		log.Printf("[DEBUG] Got existing bindings from service account %q: %#v", d.Id(), p.Bindings)

		// Merge existing policy with policy in the current state
		log.Printf("[DEBUG] Merging new bindings from service account %q: %#v", d.Id(), newPolicy.Bindings)
		mergedBindings := saMergeBindings(append(p.Bindings, newPolicy.Bindings...))

		// Remove any roles and members that were explicitly deleted
		mergedBindingsMap := saRolesToMembersMap(mergedBindings)
		for role, members := range deleted {
			for member, _ := range members {
				delete(mergedBindingsMap[role], member)
			}
		}

		p.Bindings = saRolesToMembersBinding(mergedBindingsMap)
		log.Printf("[DEBUG] Setting new policy for project: %#v", p)

		dump, _ := json.MarshalIndent(p.Bindings, " ", "  ")
		log.Printf(string(dump))
		_, err = config.clientIAM.Projects.ServiceAccounts.SetIamPolicy(d.Id(),
			&iam.SetIamPolicyRequest{Policy: p}).Do()

		if err != nil {
			return fmt.Errorf("Error applying IAM policy for service account %q: %s", d.Id(), err)
		}
	}
	return nil
}

// Retrieve the existing IAM Policy for a service account
func getServiceAccountIamPolicy(sa string, config *Config) (*iam.Policy, error) {
	p, err := config.clientIAM.Projects.ServiceAccounts.GetIamPolicy(sa).Do()

	if err != nil {
		return nil, fmt.Errorf("Error retrieving IAM policy for service account %q: %s", sa, err)
	}
	return p, nil
}

// Convert a map of roles->members to a list of Binding
func saRolesToMembersBinding(m map[string]map[string]bool) []*iam.Binding {
	bindings := make([]*iam.Binding, 0)
	for role, members := range m {
		b := iam.Binding{
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
func saRolesToMembersMap(bindings []*iam.Binding) map[string]map[string]bool {
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
func saMergeBindings(bindings []*iam.Binding) []*iam.Binding {
	bm := saRolesToMembersMap(bindings)
	rb := make([]*iam.Binding, 0)

	for role, members := range bm {
		var b iam.Binding
		b.Role = role
		b.Members = make([]string, 0)
		for m, _ := range members {
			b.Members = append(b.Members, m)
		}
		rb = append(rb, &b)
	}

	return rb
}
