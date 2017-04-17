package google

import (
	"encoding/json"
	"strconv"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
	"google.golang.org/api/cloudresourcemanager/v1"
)

var iamBinding *schema.Schema = &schema.Schema{
	Type:     schema.TypeSet,
	Required: true,
	Elem: &schema.Resource{
		Schema: map[string]*schema.Schema{
			"role": {
				Type:     schema.TypeString,
				Required: true,
			},
			"members": {
				Type:     schema.TypeSet,
				Required: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
		},
	},
}

// dataSourceGoogleIamPolicy returns a *schema.Resource that allows a customer
// to express a Google Cloud IAM policy in a data resource. This is an example
// of how the schema would be used in a config:
//
// data "google_iam_policy" "admin" {
//   binding {
//     role = "roles/storage.objectViewer"
//     members = [
//       "user:evanbrown@google.com",
//     ]
//   }
// }
func dataSourceGoogleIamPolicy() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceGoogleIamPolicyRead,
		Schema: map[string]*schema.Schema{
			"binding": iamBinding,
			"policy_data": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

// dataSourceGoogleIamPolicyRead reads a data source from config and writes it
// to state.
func dataSourceGoogleIamPolicyRead(d *schema.ResourceData, meta interface{}) error {
	var policy cloudresourcemanager.Policy
	var bindings []*cloudresourcemanager.Binding

	// The schema supports multiple binding{} blocks
	bset := d.Get("binding").(*schema.Set)

	// All binding{} blocks will be converted and stored in an array
	bindings = make([]*cloudresourcemanager.Binding, bset.Len())
	policy.Bindings = bindings

	// Convert each config binding into a cloudresourcemanager.Binding
	for i, v := range bset.List() {
		binding := v.(map[string]interface{})
		policy.Bindings[i] = &cloudresourcemanager.Binding{
			Role:    binding["role"].(string),
			Members: dataSourceGoogleIamPolicyMembers(binding["members"].(*schema.Set)),
		}
	}

	// Marshal cloudresourcemanager.Policy to JSON suitable for storing in state
	pjson, err := json.Marshal(&policy)
	if err != nil {
		// should never happen if the above code is correct
		return err
	}
	pstring := string(pjson)

	d.Set("policy_data", pstring)
	d.SetId(strconv.Itoa(hashcode.String(pstring)))

	return nil
}

// dataSourceGoogleIamPolicyMembers converts a set of members in a binding
// (a member is a principal, usually an e-mail address) into an array of
// string.
func dataSourceGoogleIamPolicyMembers(d *schema.Set) []string {
	var members []string
	members = make([]string, d.Len())

	for i, v := range d.List() {
		members[i] = v.(string)
	}
	return members
}
