package google

import (
	"encoding/json"
	"fmt"
	"log"
	"sort"

	"github.com/hashicorp/terraform/helper/schema"
	"google.golang.org/api/cloudresourcemanager/v1"
)

func resourceGoogleProjectIamPolicy() *schema.Resource {
	return &schema.Resource{
		Create: resourceGoogleProjectIamPolicyCreate,
		Read:   resourceGoogleProjectIamPolicyRead,
		Update: resourceGoogleProjectIamPolicyUpdate,
		Delete: resourceGoogleProjectIamPolicyDelete,

		Schema: map[string]*schema.Schema{
			"project": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"policy_data": &schema.Schema{
				Type:             schema.TypeString,
				Required:         true,
				DiffSuppressFunc: jsonPolicyDiffSuppress,
			},
			"authoritative": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"etag": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"restore_policy": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"disable_project": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
		},
	}
}

func resourceGoogleProjectIamPolicyCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	pid := d.Get("project").(string)
	// Get the policy in the template
	p, err := getResourceIamPolicy(d)
	if err != nil {
		return fmt.Errorf("Could not get valid 'policy_data' from resource: %v", err)
	}

	// An authoritative policy is applied without regard for any existing IAM
	// policy.
	if v, ok := d.GetOk("authoritative"); ok && v.(bool) {
		log.Printf("[DEBUG] Setting authoritative IAM policy for project %q", pid)
		err := setProjectIamPolicy(p, config, pid)
		if err != nil {
			return err
		}
	} else {
		log.Printf("[DEBUG] Setting non-authoritative IAM policy for project %q", pid)
		// This is a non-authoritative policy, meaning it should be merged with
		// any existing policy
		ep, err := getProjectIamPolicy(pid, config)
		if err != nil {
			return err
		}

		// First, subtract the policy defined in the template from the
		// current policy in the project, and save the result. This will
		// allow us to restore the original policy at some point (which
		// assumes that Terraform owns any common policy that exists in
		// the template and project at create time.
		rp := subtractIamPolicy(ep, p)
		rps, err := json.Marshal(rp)
		if err != nil {
			return fmt.Errorf("Error marshaling restorable IAM policy: %v", err)
		}
		d.Set("restore_policy", string(rps))

		// Merge the policies together
		mb := mergeBindings(append(p.Bindings, rp.Bindings...))
		ep.Bindings = mb
		if err = setProjectIamPolicy(ep, config, pid); err != nil {
			return fmt.Errorf("Error applying IAM policy to project: %v", err)
		}
	}
	d.SetId(pid)
	return resourceGoogleProjectIamPolicyRead(d, meta)
}

func resourceGoogleProjectIamPolicyRead(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG]: Reading google_project_iam_policy")
	config := meta.(*Config)
	pid := d.Get("project").(string)

	p, err := getProjectIamPolicy(pid, config)
	if err != nil {
		return err
	}

	var bindings []*cloudresourcemanager.Binding
	if v, ok := d.GetOk("restore_policy"); ok {
		var restored cloudresourcemanager.Policy
		// if there's a restore policy, subtract it from the policy_data
		err := json.Unmarshal([]byte(v.(string)), &restored)
		if err != nil {
			return fmt.Errorf("Error unmarshaling restorable IAM policy: %v", err)
		}
		subtracted := subtractIamPolicy(p, &restored)
		bindings = subtracted.Bindings
	} else {
		bindings = p.Bindings
	}
	// we only marshal the bindings, because only the bindings get set in the config
	pBytes, err := json.Marshal(&cloudresourcemanager.Policy{Bindings: bindings})
	if err != nil {
		return fmt.Errorf("Error marshaling IAM policy: %v", err)
	}
	log.Printf("[DEBUG]: Setting etag=%s", p.Etag)
	d.Set("etag", p.Etag)
	d.Set("policy_data", string(pBytes))
	return nil
}

func resourceGoogleProjectIamPolicyUpdate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG]: Updating google_project_iam_policy")
	config := meta.(*Config)
	pid := d.Get("project").(string)

	// Get the policy in the template
	p, err := getResourceIamPolicy(d)
	if err != nil {
		return fmt.Errorf("Could not get valid 'policy_data' from resource: %v", err)
	}
	pBytes, _ := json.Marshal(p)
	log.Printf("[DEBUG] Got policy from config: %s", string(pBytes))

	// An authoritative policy is applied without regard for any existing IAM
	// policy.
	if v, ok := d.GetOk("authoritative"); ok && v.(bool) {
		log.Printf("[DEBUG] Updating authoritative IAM policy for project %q", pid)
		err := setProjectIamPolicy(p, config, pid)
		if err != nil {
			return fmt.Errorf("Error setting project IAM policy: %v", err)
		}
		d.Set("restore_policy", "")
	} else {
		log.Printf("[DEBUG] Updating non-authoritative IAM policy for project %q", pid)
		// Get the previous policy from state
		pp, err := getPrevResourceIamPolicy(d)
		if err != nil {
			return fmt.Errorf("Error retrieving previous version of changed project IAM policy: %v", err)
		}
		ppBytes, _ := json.Marshal(pp)
		log.Printf("[DEBUG] Got previous version of changed project IAM policy: %s", string(ppBytes))

		// Get the existing IAM policy from the API
		ep, err := getProjectIamPolicy(pid, config)
		if err != nil {
			return fmt.Errorf("Error retrieving IAM policy from project API: %v", err)
		}
		epBytes, _ := json.Marshal(ep)
		log.Printf("[DEBUG] Got existing version of changed IAM policy from project API: %s", string(epBytes))

		// Subtract the previous and current policies from the policy retrieved from the API
		rp := subtractIamPolicy(ep, pp)
		rpBytes, _ := json.Marshal(rp)
		log.Printf("[DEBUG] After subtracting the previous policy from the existing policy, remaining policies: %s", string(rpBytes))
		rp = subtractIamPolicy(rp, p)
		rpBytes, _ = json.Marshal(rp)
		log.Printf("[DEBUG] After subtracting the remaining policies from the config policy, remaining policies: %s", string(rpBytes))
		rps, err := json.Marshal(rp)
		if err != nil {
			return fmt.Errorf("Error marhsaling restorable IAM policy: %v", err)
		}
		d.Set("restore_policy", string(rps))

		// Merge the policies together
		mb := mergeBindings(append(p.Bindings, rp.Bindings...))
		ep.Bindings = mb
		if err = setProjectIamPolicy(ep, config, pid); err != nil {
			return fmt.Errorf("Error applying IAM policy to project: %v", err)
		}
	}

	return resourceGoogleProjectIamPolicyRead(d, meta)
}

func resourceGoogleProjectIamPolicyDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG]: Deleting google_project_iam_policy")
	config := meta.(*Config)
	pid := d.Get("project").(string)

	// Get the existing IAM policy from the API
	ep, err := getProjectIamPolicy(pid, config)
	if err != nil {
		return fmt.Errorf("Error retrieving IAM policy from project API: %v", err)
	}
	// Deleting an authoritative policy will leave the project with no policy,
	// and unaccessible by anyone without org-level privs. For this reason, the
	// "disable_project" property must be set to true, forcing the user to ack
	// this outcome
	if v, ok := d.GetOk("authoritative"); ok && v.(bool) {
		if v, ok := d.GetOk("disable_project"); !ok || !v.(bool) {
			return fmt.Errorf("You must set 'disable_project' to true before deleting an authoritative IAM policy")
		}
		ep.Bindings = make([]*cloudresourcemanager.Binding, 0)

	} else {
		// A non-authoritative policy should set the policy to the value of "restore_policy" in state
		// Get the previous policy from state
		rp, err := getRestoreIamPolicy(d)
		if err != nil {
			return fmt.Errorf("Error retrieving previous version of changed project IAM policy: %v", err)
		}
		ep.Bindings = rp.Bindings
	}
	if err = setProjectIamPolicy(ep, config, pid); err != nil {
		return fmt.Errorf("Error applying IAM policy to project: %v", err)
	}
	d.SetId("")
	return nil
}

// Subtract all bindings in policy b from policy a, and return the result
func subtractIamPolicy(a, b *cloudresourcemanager.Policy) *cloudresourcemanager.Policy {
	am := rolesToMembersMap(a.Bindings)

	for _, b := range b.Bindings {
		if _, ok := am[b.Role]; ok {
			for _, m := range b.Members {
				delete(am[b.Role], m)
			}
			if len(am[b.Role]) == 0 {
				delete(am, b.Role)
			}
		}
	}
	a.Bindings = rolesToMembersBinding(am)
	return a
}

func setProjectIamPolicy(policy *cloudresourcemanager.Policy, config *Config, pid string) error {
	// Apply the policy
	pbytes, _ := json.Marshal(policy)
	log.Printf("[DEBUG] Setting policy %#v for project: %s", string(pbytes), pid)
	_, err := config.clientResourceManager.Projects.SetIamPolicy(pid,
		&cloudresourcemanager.SetIamPolicyRequest{Policy: policy}).Do()

	if err != nil {
		return fmt.Errorf("Error applying IAM policy for project %q. Policy is %#v, error is %s", pid, policy, err)
	}
	return nil
}

// Get a cloudresourcemanager.Policy from a schema.ResourceData
func getResourceIamPolicy(d *schema.ResourceData) (*cloudresourcemanager.Policy, error) {
	ps := d.Get("policy_data").(string)
	// The policy string is just a marshaled cloudresourcemanager.Policy.
	policy := &cloudresourcemanager.Policy{}
	if err := json.Unmarshal([]byte(ps), policy); err != nil {
		return nil, fmt.Errorf("Could not unmarshal %s:\n: %v", ps, err)
	}
	return policy, nil
}

// Get the previous cloudresourcemanager.Policy from a schema.ResourceData if the
// resource has changed
func getPrevResourceIamPolicy(d *schema.ResourceData) (*cloudresourcemanager.Policy, error) {
	var policy *cloudresourcemanager.Policy = &cloudresourcemanager.Policy{}
	if d.HasChange("policy_data") {
		v, _ := d.GetChange("policy_data")
		if err := json.Unmarshal([]byte(v.(string)), policy); err != nil {
			return nil, fmt.Errorf("Could not unmarshal previous policy %s:\n: %v", v, err)
		}
	}
	return policy, nil
}

// Get the restore_policy that can be used to restore a project's IAM policy to its
// state before it was adopted into Terraform
func getRestoreIamPolicy(d *schema.ResourceData) (*cloudresourcemanager.Policy, error) {
	if v, ok := d.GetOk("restore_policy"); ok {
		policy := &cloudresourcemanager.Policy{}
		if err := json.Unmarshal([]byte(v.(string)), policy); err != nil {
			return nil, fmt.Errorf("Could not unmarshal previous policy %s:\n: %v", v, err)
		}
		return policy, nil
	}
	return nil, fmt.Errorf("Resource does not have a 'restore_policy' attribute defined.")
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

func jsonPolicyDiffSuppress(k, old, new string, d *schema.ResourceData) bool {
	var oldPolicy, newPolicy cloudresourcemanager.Policy
	if err := json.Unmarshal([]byte(old), &oldPolicy); err != nil {
		log.Printf("[ERROR] Could not unmarshal old policy %s: %v", old, err)
		return false
	}
	if err := json.Unmarshal([]byte(new), &newPolicy); err != nil {
		log.Printf("[ERROR] Could not unmarshal new policy %s: %v", new, err)
		return false
	}
	oldPolicy.Bindings = mergeBindings(oldPolicy.Bindings)
	newPolicy.Bindings = mergeBindings(newPolicy.Bindings)
	if newPolicy.Etag != oldPolicy.Etag {
		return false
	}
	if newPolicy.Version != oldPolicy.Version {
		return false
	}
	if len(newPolicy.Bindings) != len(oldPolicy.Bindings) {
		return false
	}
	sort.Sort(sortableBindings(newPolicy.Bindings))
	sort.Sort(sortableBindings(oldPolicy.Bindings))
	for pos, newBinding := range newPolicy.Bindings {
		oldBinding := oldPolicy.Bindings[pos]
		if oldBinding.Role != newBinding.Role {
			return false
		}
		if len(oldBinding.Members) != len(newBinding.Members) {
			return false
		}
		sort.Strings(oldBinding.Members)
		sort.Strings(newBinding.Members)
		for i, newMember := range newBinding.Members {
			oldMember := oldBinding.Members[i]
			if newMember != oldMember {
				return false
			}
		}
	}
	return true
}

type sortableBindings []*cloudresourcemanager.Binding

func (b sortableBindings) Len() int {
	return len(b)
}
func (b sortableBindings) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}
func (b sortableBindings) Less(i, j int) bool {
	return b[i].Role < b[j].Role
}
